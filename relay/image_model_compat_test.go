package relay

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
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

	usage, newAPIError := imageStreamToChatCompletions(c, info, resp, "gpt-image-2")
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

func TestImageStreamToResponsesEmitsPartialAndCompletedEvents(t *testing.T) {
	sse := strings.Join([]string{
		`data: {"type":"image_generation.partial_image","b64_json":"partial","partial_image_index":0,"output_format":"webp"}`,
		`data: {"type":"image_generation.completed","b64_json":"final","output_format":"webp","usage":{"input_tokens":4,"output_tokens":8,"total_tokens":12}}`,
		`data: [DONE]`,
		"",
	}, "\n")
	c, recorder, resp, info := setupImageCompatStreamTest(sse)

	usage, newAPIError := imageStreamToResponses(c, info, resp, "gpt-image-2")
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
