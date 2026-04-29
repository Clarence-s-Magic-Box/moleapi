package openai

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"

	"github.com/gin-gonic/gin"
)

func TestConvertImageRequestUsesAPIMartGenerationCompatibilityForGPTImage2Edits(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("model", "gpt-image-2"); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("prompt", "make it brighter"); err != nil {
		t.Fatal(err)
	}
	part, err := writer.CreateFormFile("image", "input.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(part, strings.NewReader("png-data")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	n := uint(1)
	info := &relaycommon.RelayInfo{
		RelayMode:      relayconstant.RelayModeImagesEdits,
		RequestURLPath: "/v1/images/edits",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://api.apimart.ai",
		},
	}

	convertedAny, err := (&Adaptor{}).ConvertImageRequest(c, info, dto.ImageRequest{
		Model:  "gpt-image-2",
		Prompt: "make it brighter",
		N:      &n,
		Size:   "1024x1024",
	})
	if err != nil {
		t.Fatalf("ConvertImageRequest returned error: %v", err)
	}
	converted, ok := convertedAny.(*dto.ImageRequest)
	if !ok {
		t.Fatalf("expected *dto.ImageRequest, got %T", convertedAny)
	}
	if info.RelayMode != relayconstant.RelayModeImagesGenerations {
		t.Fatalf("expected relay mode to switch to generations, got %d", info.RelayMode)
	}
	if info.RequestURLPath != "/v1/images/generations" {
		t.Fatalf("unexpected request path: %s", info.RequestURLPath)
	}
	if c.Request.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("expected JSON content type, got %s", c.Request.Header.Get("Content-Type"))
	}
	if converted.Size != "1:1" {
		t.Fatalf("expected APIMart ratio size, got %s", converted.Size)
	}

	jsonBody, err := common.Marshal(converted)
	if err != nil {
		t.Fatal(err)
	}
	bodyText := string(jsonBody)
	if !strings.Contains(bodyText, `"image_urls":["data:image/png;base64,`) {
		t.Fatalf("expected image file to be converted to image_urls data URI, got %s", bodyText)
	}
	if strings.Contains(bodyText, `"image":`) {
		t.Fatalf("expected legacy image field to be removed, got %s", bodyText)
	}

	requestURL, err := (&Adaptor{}).GetRequestURL(info)
	if err != nil {
		t.Fatal(err)
	}
	if requestURL != "https://api.apimart.ai/v1/images/generations" {
		t.Fatalf("unexpected upstream URL: %s", requestURL)
	}
}

func TestConvertImageRequestKeepsOfficialGPTImage2EditsAsMultipart(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("model", "gpt-image-2"); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("prompt", "make it brighter"); err != nil {
		t.Fatal(err)
	}
	part, err := writer.CreateFormFile("image", "input.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(part, strings.NewReader("png-data")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	n := uint(1)
	info := &relaycommon.RelayInfo{
		RelayMode:      relayconstant.RelayModeImagesEdits,
		RequestURLPath: "/v1/images/edits",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://api.openai.com",
		},
	}

	convertedAny, err := (&Adaptor{}).ConvertImageRequest(c, info, dto.ImageRequest{
		Model:  "gpt-image-2",
		Prompt: "make it brighter",
		N:      &n,
		Size:   "1024x1024",
	})
	if err != nil {
		t.Fatalf("ConvertImageRequest returned error: %v", err)
	}
	if _, ok := convertedAny.(*bytes.Buffer); !ok {
		t.Fatalf("expected multipart request body, got %T", convertedAny)
	}
	if info.RelayMode != relayconstant.RelayModeImagesEdits {
		t.Fatalf("expected relay mode to remain edits, got %d", info.RelayMode)
	}
	if info.RequestURLPath != "/v1/images/edits" {
		t.Fatalf("unexpected request path: %s", info.RequestURLPath)
	}
	if !strings.HasPrefix(c.Request.Header.Get("Content-Type"), "multipart/form-data; boundary=") {
		t.Fatalf("expected multipart content type, got %s", c.Request.Header.Get("Content-Type"))
	}

	requestURL, err := (&Adaptor{}).GetRequestURL(info)
	if err != nil {
		t.Fatal(err)
	}
	if requestURL != "https://api.openai.com/v1/images/edits" {
		t.Fatalf("unexpected upstream URL: %s", requestURL)
	}
}

func TestConvertImageRequestUsesConfiguredGenerationCompatibilityForFutureProviders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("model", "gpt-image-2"); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("prompt", "make it brighter"); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("image", "https://example.com/input.png"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	n := uint(1)
	info := &relaycommon.RelayInfo{
		RelayMode:      relayconstant.RelayModeImagesEdits,
		RequestURLPath: "/v1/images/edits",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://images.example.com",
			ChannelOtherSettings: dto.ChannelOtherSettings{
				ImageEditUseGenerationEndpoint: true,
			},
		},
	}

	convertedAny, err := (&Adaptor{}).ConvertImageRequest(c, info, dto.ImageRequest{
		Model:  "gpt-image-2",
		Prompt: "make it brighter",
		N:      &n,
	})
	if err != nil {
		t.Fatalf("ConvertImageRequest returned error: %v", err)
	}
	converted, ok := convertedAny.(*dto.ImageRequest)
	if !ok {
		t.Fatalf("expected *dto.ImageRequest, got %T", convertedAny)
	}
	if info.RelayMode != relayconstant.RelayModeImagesGenerations {
		t.Fatalf("expected relay mode to switch to generations, got %d", info.RelayMode)
	}
	if !strings.Contains(string(converted.ImageUrls), "https://example.com/input.png") {
		t.Fatalf("expected configured provider image URL to be forwarded, got %s", string(converted.ImageUrls))
	}
}
