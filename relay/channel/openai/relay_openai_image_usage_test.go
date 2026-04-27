package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
)

func setupImageUsageHandlerTest(body string) (*gin.Context, *httptest.ResponseRecorder, *http.Response, *common.RelayInfo) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	info := &common.RelayInfo{
		RelayMode:   relayconstant.RelayModeImagesGenerations,
		ChannelMeta: &common.ChannelMeta{},
	}
	return c, recorder, resp, info
}

func TestOpenaiHandlerWithUsageRejectsImageErrorBody(t *testing.T) {
	c, recorder, resp, info := setupImageUsageHandlerTest(`{"error":{"message":"generation failed","type":"invalid_request_error"}}`)

	usage, newAPIError := OpenaiHandlerWithUsage(c, info, resp)

	if newAPIError == nil {
		t.Fatal("expected image error body to return an error")
	}
	if usage != nil {
		t.Fatalf("expected no usage for failed image response, got %+v", usage)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("failed image response should not be written as success, got %q", recorder.Body.String())
	}
}

func TestOpenaiHandlerWithUsageRejectsImageResponseWithoutData(t *testing.T) {
	c, recorder, resp, info := setupImageUsageHandlerTest(`{"created":123,"data":[]}`)

	usage, newAPIError := OpenaiHandlerWithUsage(c, info, resp)

	if newAPIError == nil {
		t.Fatal("expected empty image data response to return an error")
	}
	if usage != nil {
		t.Fatalf("expected no usage for empty image response, got %+v", usage)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("empty image response should not be written as success, got %q", recorder.Body.String())
	}
}

func TestOpenaiHandlerWithUsageAcceptsImageResponseWithData(t *testing.T) {
	c, recorder, resp, info := setupImageUsageHandlerTest(`{"created":123,"data":[{"b64_json":"abc"}],"usage":{"input_tokens":3,"output_tokens":9,"total_tokens":12}}`)

	usage, newAPIError := OpenaiHandlerWithUsage(c, info, resp)

	if newAPIError != nil {
		t.Fatalf("unexpected error: %v", newAPIError)
	}
	if usage == nil {
		t.Fatal("expected usage")
	}
	if usage.PromptTokens != 3 || usage.CompletionTokens != 9 || usage.TotalTokens != 12 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
	if !strings.Contains(recorder.Body.String(), `"b64_json":"abc"`) {
		t.Fatalf("expected successful image response body, got %q", recorder.Body.String())
	}
}
