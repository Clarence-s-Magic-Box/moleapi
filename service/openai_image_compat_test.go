package service

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
)

func TestChatCompletionsRequestToImageRequest(t *testing.T) {
	stream := true
	n := 2
	req := &dto.GeneralOpenAIRequest{
		Model:  "gpt-image-2",
		Stream: &stream,
		N:      &n,
		Size:   "1024x1024",
		Messages: []dto.Message{
			{Role: "system", Content: "Use a clean editorial style."},
			{
				Role: "user",
				Content: []any{
					map[string]any{"type": "text", "text": "Draw a red door"},
					map[string]any{"type": "image_url", "image_url": "https://example.com/input.png"},
				},
			},
		},
		ExtraBody: []byte(`{"quality":"high","background":"transparent","partial_images":3}`),
	}

	imageReq, err := ChatCompletionsRequestToImageRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if imageReq.Model != "gpt-image-2" {
		t.Fatalf("unexpected model: %s", imageReq.Model)
	}
	if imageReq.Prompt != "Use a clean editorial style.\n\nDraw a red door" {
		t.Fatalf("unexpected prompt: %q", imageReq.Prompt)
	}
	if imageReq.Stream == nil || !*imageReq.Stream {
		t.Fatal("expected stream to be preserved")
	}
	if imageReq.N == nil || *imageReq.N != 2 {
		t.Fatalf("unexpected n: %v", imageReq.N)
	}
	if imageReq.Size != "1024x1024" || imageReq.Quality != "high" {
		t.Fatalf("unexpected size or quality: %s / %s", imageReq.Size, imageReq.Quality)
	}
	if string(imageReq.Background) != `"transparent"` {
		t.Fatalf("unexpected background: %s", string(imageReq.Background))
	}
	if string(imageReq.PartialImages) != "3" {
		t.Fatalf("unexpected partial_images: %s", string(imageReq.PartialImages))
	}
}

func TestResponsesRequestToImageRequest(t *testing.T) {
	stream := true
	req := &dto.OpenAIResponsesRequest{
		Model:  "gpt-image-2",
		Stream: &stream,
		Input:  []byte(`[{"role":"user","content":[{"type":"input_text","text":"Make a launch poster"}]}]`),
		Tools:  []byte(`[{"type":"web_search_preview"},{"type":"image_generation","size":"1536x1024","quality":"medium","partial_images":2,"output_format":"webp"}]`),
	}

	imageReq, err := ResponsesRequestToImageRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if imageReq.Prompt != "Make a launch poster" {
		t.Fatalf("unexpected prompt: %q", imageReq.Prompt)
	}
	if imageReq.Stream == nil || !*imageReq.Stream {
		t.Fatal("expected stream to be preserved")
	}
	if imageReq.Size != "1536x1024" || imageReq.Quality != "medium" {
		t.Fatalf("unexpected size or quality: %s / %s", imageReq.Size, imageReq.Quality)
	}
	if string(imageReq.PartialImages) != "2" {
		t.Fatalf("unexpected partial_images: %s", string(imageReq.PartialImages))
	}
	if string(imageReq.OutputFormat) != `"webp"` {
		t.Fatalf("unexpected output_format: %s", string(imageReq.OutputFormat))
	}
}

func TestImageResponseToChatResponse(t *testing.T) {
	resp := &dto.ImageResponse{
		Created:      123,
		OutputFormat: "webp",
		Data:         []dto.ImageData{{B64Json: "abc123"}},
		Usage:        &dto.Usage{PromptTokens: 5, CompletionTokens: 7, TotalTokens: 12},
	}

	chatResp, usage, err := ImageResponseToChatResponse(resp, "gpt-image-2", "chatcmpl_test", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if chatResp.Id != "chatcmpl_test" || chatResp.Model != "gpt-image-2" {
		t.Fatalf("unexpected response identity: %s / %s", chatResp.Id, chatResp.Model)
	}
	if len(chatResp.Choices) != 1 {
		t.Fatalf("unexpected choice count: %d", len(chatResp.Choices))
	}
	if chatResp.Choices[0].Message.Content != "![generated image](data:image/webp;base64,abc123)" {
		t.Fatalf("unexpected content: %v", chatResp.Choices[0].Message.Content)
	}
	if usage.TotalTokens != 12 || chatResp.Usage.TotalTokens != 12 {
		t.Fatalf("unexpected usage: %+v / %+v", usage, chatResp.Usage)
	}
}

func TestImageResponseToResponsesResponse(t *testing.T) {
	resp := &dto.ImageResponse{
		Created:      456,
		Background:   "transparent",
		OutputFormat: "png",
		Quality:      "high",
		Size:         "1024x1024",
		Data: []dto.ImageData{{
			B64Json:       "image-b64",
			RevisedPrompt: "A revised prompt",
		}},
		Usage: &dto.Usage{PromptTokens: 3, CompletionTokens: 9, TotalTokens: 12, OutputTokens: 9},
	}

	response, usage, err := ImageResponseToResponsesResponse(resp, "gpt-image-2", "resp_test", "igc_test", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output, ok := response["output"].([]map[string]any)
	if !ok || len(output) != 1 {
		t.Fatalf("unexpected output: %#v", response["output"])
	}
	item := output[0]
	if item["type"] != dto.ResponsesOutputTypeImageGenerationCall || item["result"] != "image-b64" {
		t.Fatalf("unexpected output item: %#v", item)
	}
	if item["revised_prompt"] != "A revised prompt" || item["quality"] != "high" || item["size"] != "1024x1024" {
		t.Fatalf("unexpected image metadata: %#v", item)
	}
	if usage.TotalTokens != 12 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
	usageMap, ok := response["usage"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected usage map: %#v", response["usage"])
	}
	if usageMap["total_tokens"] != 12 {
		t.Fatalf("unexpected usage total: %#v", usageMap)
	}
}
