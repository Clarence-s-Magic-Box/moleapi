package relay

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	openaichannel "github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupImageCompatStreamTest(body string) (*gin.Context, *httptest.ResponseRecorder, *http.Response, *relaycommon.RelayInfo) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Set(common.RequestIdKey, "test")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{},
	}
	return c, recorder, resp, info
}

func TestImageStreamToChatCompletionsUsesFinalImageOnly(t *testing.T) {
	sse := strings.Join([]string{
		`data: {"type":"image_generation.partial_image","b64_json":"partial","partial_image_index":0}`,
		`data: {"type":"image_generation.completed","b64_json":"final","output_format":"png","usage":{"input_tokens":3,"output_tokens":9,"total_tokens":12}}`,
		`data: [DONE]`,
		"",
	}, "\n")
	c, recorder, resp, info := setupImageCompatStreamTest(sse)

	usage, newAPIError := imageStreamToChatCompletions(c, info, resp, "gpt-image-2", nil)
	if newAPIError != nil {
		t.Fatalf("unexpected error: %v", newAPIError)
	}
	if usage.PromptTokens != 3 || usage.CompletionTokens != 9 || usage.TotalTokens != 12 {
		t.Fatalf("unexpected usage: %+v", usage)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "chat.completion.chunk") {
		t.Fatalf("expected chat completion chunks, got: %s", body)
	}
	if !strings.Contains(body, "data:image/png;base64,final") {
		t.Fatalf("expected final image content, got: %s", body)
	}
	if strings.Contains(body, "partial") {
		t.Fatalf("chat stream should not include partial image data, got: %s", body)
	}
	if !strings.Contains(body, "[DONE]") {
		t.Fatalf("expected stream terminator, got: %s", body)
	}
}

func TestChatCompletionsViaImageGenerationStreamsAsyncJSONResponse(t *testing.T) {
	service.InitHttpClient()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/images/generations":
			_, _ = w.Write([]byte(`{"code":200,"data":[{"status":"submitted","task_id":"task_123"}]}`))
		case "/v1/tasks/task_123":
			_, _ = w.Write([]byte(`{"code":200,"data":{"id":"task_123","status":"completed","created":10,"completed":20,"result":{"images":[{"url":["https://example.com/generated.png"],"expires_at":30}]}}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	requestBody := `{"model":"gpt-image-2","stream":true,"messages":[{"role":"user","content":"draw"}]}`
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/pg/chat/completions", strings.NewReader(requestBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(common.RequestIdKey, "test")
	stream := true
	req := &dto.GeneralOpenAIRequest{
		Model:  "gpt-image-2",
		Stream: &stream,
		Messages: []dto.Message{
			{Role: "user", Content: "draw"},
		},
	}
	info := &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeChatCompletions,
		OriginModelName: "gpt-image-2",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl:    server.URL,
			UpstreamModelName: "gpt-image-2",
		},
		ShouldIncludeUsage: true,
	}
	adaptor := &openaichannel.Adaptor{}
	adaptor.Init(info)

	usage, newAPIError := chatCompletionsViaImageGeneration(c, info, adaptor, req)
	if newAPIError != nil {
		t.Fatalf("unexpected error: %v", newAPIError)
	}
	if usage == nil || usage.TotalTokens == 0 {
		t.Fatalf("expected usage, got %+v", usage)
	}
	body := recorder.Body.String()
	for _, want := range []string{
		"chat.completion.chunk",
		"https://example.com/generated.png",
		"[DONE]",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in stream body, got: %s", want, body)
		}
	}
	if got := recorder.Header().Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("expected event-stream response, got %q", got)
	}
}

func TestAsyncImageTaskErrorSkipsRetry(t *testing.T) {
	newAPIError := asyncImageTaskNewAPIError(&service.AsyncImageTaskError{
		StatusCode: http.StatusGatewayTimeout,
		OpenAIError: types.OpenAIError{
			Message: "async image task timed out",
			Type:    "upstream_task_timeout",
			Code:    "task_timeout",
		},
	})

	if newAPIError == nil {
		t.Fatal("expected error")
	}
	if !types.IsSkipRetryError(newAPIError) {
		t.Fatal("expected async task error to skip retry")
	}
}

func TestImageStreamToChatCompletionsRejectsMissingCompletedImage(t *testing.T) {
	sse := strings.Join([]string{
		`data: {"type":"image_generation.partial_image","b64_json":"partial","partial_image_index":0}`,
		`data: [DONE]`,
		"",
	}, "\n")
	c, _, resp, info := setupImageCompatStreamTest(sse)

	usage, newAPIError := imageStreamToChatCompletions(c, info, resp, "gpt-image-2", nil)
	if newAPIError == nil {
		t.Fatal("expected error for stream without completed image")
	}
	if usage != nil {
		t.Fatalf("expected no usage for failed image stream, got %+v", usage)
	}
}

func TestImageStreamToChatCompletionsRejectsCompletedEventWithoutImage(t *testing.T) {
	sse := strings.Join([]string{
		`data: {"type":"image_generation.completed","output_format":"png","usage":{"input_tokens":3,"output_tokens":9,"total_tokens":12}}`,
		`data: [DONE]`,
		"",
	}, "\n")
	c, _, resp, info := setupImageCompatStreamTest(sse)

	usage, newAPIError := imageStreamToChatCompletions(c, info, resp, "gpt-image-2", nil)
	if newAPIError == nil {
		t.Fatal("expected error for completed image stream without image data")
	}
	if usage != nil {
		t.Fatalf("expected no usage for failed image stream, got %+v", usage)
	}
}

func TestImageStreamToResponsesEmitsPartialAndCompletedEvents(t *testing.T) {
	sse := strings.Join([]string{
		`data: {"type":"image_generation.partial_image","b64_json":"partial","partial_image_index":0,"output_format":"webp"}`,
		`data: {"type":"image_generation.completed","b64_json":"final","output_format":"webp","usage":{"input_tokens":4,"output_tokens":8,"total_tokens":12}}`,
		`data: [DONE]`,
		"",
	}, "\n")
	c, recorder, resp, info := setupImageCompatStreamTest(sse)

	usage, newAPIError := imageStreamToResponses(c, info, resp, "gpt-image-2", nil)
	if newAPIError != nil {
		t.Fatalf("unexpected error: %v", newAPIError)
	}
	if usage.PromptTokens != 4 || usage.CompletionTokens != 8 || usage.TotalTokens != 12 {
		t.Fatalf("unexpected usage: %+v", usage)
	}

	body := recorder.Body.String()
	for _, want := range []string{
		"event: response.created",
		"event: response.output_item.added",
		"event: response.image_generation_call.partial_image",
		`"partial_image_b64":"partial"`,
		`"partial_image_index":0`,
		"event: response.output_item.done",
		"event: response.completed",
		`"result":"final"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in responses stream, got: %s", want, body)
		}
	}
}

func TestReadImageResponseResolvesAsyncTask(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/tasks/task_123" {
			t.Fatalf("unexpected task path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"data":{"id":"task_123","status":"completed","created":10,"completed":20,"result":{"images":[{"url":["https://example.com/generated.png"],"expires_at":30}]}}}`))
	}))
	defer server.Close()

	c, _, resp, info := setupImageCompatStreamTest(`{"code":200,"data":[{"status":"submitted","task_id":"task_123"}]}`)
	resp.Header = http.Header{"Content-Type": []string{"application/json"}}
	info.ChannelBaseUrl = server.URL
	info.ApiKey = "test-key"

	imageResp, body, newAPIError := readImageResponse(c, info, resp, nil, nil)
	if newAPIError != nil {
		t.Fatalf("unexpected error: %v", newAPIError)
	}
	if len(imageResp.Data) != 1 || imageResp.Data[0].Url != "https://example.com/generated.png" {
		t.Fatalf("unexpected image response: %+v", imageResp)
	}
	if !strings.Contains(string(body), `"url":"https://example.com/generated.png"`) {
		t.Fatalf("expected resolved image response body, got %s", string(body))
	}
}

func TestReadImageResponsePricesActualImageCount(t *testing.T) {
	c, _, resp, info := setupImageCompatStreamTest(`{"created":123,"data":[{"url":"https://example.com/a.png"},{"url":"https://example.com/b.png"}]}`)
	resp.Header = http.Header{"Content-Type": []string{"application/json"}}
	info.PriceData = types.PriceData{UsePrice: true}

	imageResp, _, newAPIError := readImageResponse(c, info, resp, nil, nil)
	if newAPIError != nil {
		t.Fatalf("unexpected error: %v", newAPIError)
	}
	if len(imageResp.Data) != 2 {
		t.Fatalf("unexpected image response: %+v", imageResp)
	}
	if got := info.PriceData.OtherRatios["n"]; got != 2 {
		t.Fatalf("expected actual image count pricing ratio 2, got %v", got)
	}
}

func TestReadImageResponseRejectsNonJSONBodyAsUpstreamError(t *testing.T) {
	c, _, resp, info := setupImageCompatStreamTest(`<html>bad gateway</html>`)
	resp.Header = http.Header{"Content-Type": []string{"text/html"}}

	imageResp, _, newAPIError := readImageResponse(c, info, resp, nil, nil)

	if newAPIError == nil {
		t.Fatal("expected non-json image response to return an error")
	}
	if newAPIError.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected bad gateway status, got %d", newAPIError.StatusCode)
	}
	if imageResp != nil {
		t.Fatalf("expected no image response, got %+v", imageResp)
	}
}

func TestReadImageResponseSplitsAsyncImageRequestsAndEstimatesOutputTokens(t *testing.T) {
	service.InitHttpClient()
	var extraSubmitSeen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/images/generations":
			extraSubmitSeen = true
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"n":1`) {
				t.Fatalf("expected additional upstream request to use n=1, got %s", string(body))
			}
			_, _ = w.Write([]byte(`{"code":200,"data":[{"status":"submitted","task_id":"task_2"}]}`))
		case "/v1/tasks/task_1":
			_, _ = w.Write([]byte(`{"code":200,"data":{"id":"task_1","status":"completed","created":10,"completed":20,"result":{"images":[{"url":["https://example.com/a.png"],"expires_at":30}]}}}`))
		case "/v1/tasks/task_2":
			_, _ = w.Write([]byte(`{"code":200,"data":{"id":"task_2","status":"completed","created":11,"completed":21,"result":{"images":[{"url":["https://example.com/b.png"],"expires_at":31}]}}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c, _, resp, info := setupImageCompatStreamTest(`{"code":200,"data":[{"status":"submitted","task_id":"task_1"}]}`)
	resp.Header = http.Header{"Content-Type": []string{"application/json"}}
	info.ChannelBaseUrl = server.URL
	info.SetEstimatePromptTokens(5)
	n := uint(2)
	imageReq := &dto.ImageRequest{Model: "gpt-image-2", Prompt: "draw", N: &n, Size: "1024x1024", Quality: "low"}

	imageResp, body, newAPIError := readImageResponse(c, info, resp, &openaichannel.Adaptor{}, imageReq)
	if newAPIError != nil {
		t.Fatalf("unexpected error: %v", newAPIError)
	}
	if !extraSubmitSeen {
		t.Fatal("expected an additional upstream task submission")
	}
	if len(imageResp.Data) != 2 {
		t.Fatalf("expected two images, got %+v", imageResp)
	}
	if imageResp.Usage == nil || imageResp.Usage.PromptTokens != 5 || imageResp.Usage.CompletionTokens != 392 || imageResp.Usage.TotalTokens != 397 {
		t.Fatalf("unexpected usage: %+v", imageResp.Usage)
	}
	if !strings.Contains(string(body), "https://example.com/a.png") || !strings.Contains(string(body), "https://example.com/b.png") {
		t.Fatalf("expected both image urls in response, got %s", string(body))
	}
}
