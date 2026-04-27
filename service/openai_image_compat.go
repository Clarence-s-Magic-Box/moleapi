package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
)

const gptImage2ModelPrefix = "gpt-image-2"

var (
	imagePromptMarkdownDataURLPattern = regexp.MustCompile(`!\[[^\]\r\n]*\]\(\s*data:image/[A-Za-z0-9.+-]+;base64,[^)]+\)`)
	imagePromptDataURLPattern         = regexp.MustCompile(`data:image/[A-Za-z0-9.+-]+;base64,[-A-Za-z0-9+/_=]+`)
)

func IsGPTImage2Model(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	return model == gptImage2ModelPrefix || strings.HasPrefix(model, gptImage2ModelPrefix+"-")
}

func ImageCompatTokenCountMeta(req dto.Request, model string) (*types.TokenCountMeta, bool) {
	if req == nil {
		return nil, false
	}

	switch r := req.(type) {
	case *dto.GeneralOpenAIRequest:
		if !IsGPTImage2Model(model) && !IsGPTImage2Model(r.Model) {
			return nil, false
		}
		imageReq, err := ChatCompletionsRequestToImageRequest(r)
		if err != nil {
			return nil, false
		}
		return imageReq.GetTokenCountMeta(), true
	case *dto.OpenAIResponsesRequest:
		if !IsGPTImage2Model(model) && !IsGPTImage2Model(r.Model) {
			return nil, false
		}
		imageReq, err := ResponsesRequestToImageRequest(r)
		if err != nil {
			return nil, false
		}
		return imageReq.GetTokenCountMeta(), true
	default:
		return nil, false
	}
}

func ChatCompletionsRequestToImageRequest(req *dto.GeneralOpenAIRequest) (*dto.ImageRequest, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	if strings.TrimSpace(req.Model) == "" {
		return nil, errors.New("model is required")
	}

	prompt := extractPromptFromChatRequest(req)
	if prompt == "" {
		return nil, errors.New("prompt is required")
	}

	imageReq := &dto.ImageRequest{
		Model:  req.Model,
		Prompt: prompt,
		Size:   req.Size,
		Stream: req.Stream,
		User:   req.User,
	}
	if req.N != nil && *req.N > 0 {
		imageReq.N = common.GetPointer(uint(*req.N))
	}
	ApplyImageOptionsFromRaw(req.ExtraBody, imageReq)
	return imageReq, nil
}

func ResponsesRequestToImageRequest(req *dto.OpenAIResponsesRequest) (*dto.ImageRequest, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	if strings.TrimSpace(req.Model) == "" {
		return nil, errors.New("model is required")
	}

	prompt := extractPromptFromResponsesRequest(req)
	if prompt == "" {
		return nil, errors.New("prompt is required")
	}

	imageReq := &dto.ImageRequest{
		Model:  req.Model,
		Prompt: prompt,
		Stream: req.Stream,
		User:   req.User,
	}
	ApplyImageGenerationToolOptions(req.Tools, imageReq)
	return imageReq, nil
}

func ImageUsageToUsage(imageUsage *dto.Usage, fallbackPromptTokens int) *dto.Usage {
	usage := &dto.Usage{}
	if imageUsage != nil {
		*usage = *imageUsage
	}

	if usage.PromptTokens == 0 && usage.InputTokens != 0 {
		usage.PromptTokens = usage.InputTokens
	}
	if usage.CompletionTokens == 0 && usage.OutputTokens != 0 {
		usage.CompletionTokens = usage.OutputTokens
	}
	if usage.InputTokens == 0 && usage.PromptTokens != 0 {
		usage.InputTokens = usage.PromptTokens
	}
	if usage.OutputTokens == 0 && usage.CompletionTokens != 0 {
		usage.OutputTokens = usage.CompletionTokens
	}
	if usage.InputTokensDetails != nil {
		usage.PromptTokensDetails = *usage.InputTokensDetails
	}
	if usage.CompletionTokenDetails.ImageTokens == 0 && usage.OutputTokens != 0 {
		usage.CompletionTokenDetails.ImageTokens = usage.OutputTokens
	}
	if usage.PromptTokens == 0 && fallbackPromptTokens > 0 {
		usage.PromptTokens = fallbackPromptTokens
		usage.InputTokens = fallbackPromptTokens
	}
	if usage.CompletionTokens == 0 && imageUsage == nil {
		usage.CompletionTokens = 1
		usage.OutputTokens = 1
		usage.CompletionTokenDetails.ImageTokens = 1
	}
	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	return usage
}

func FirstImageData(resp *dto.ImageResponse) (dto.ImageData, bool) {
	if resp == nil || len(resp.Data) == 0 {
		return dto.ImageData{}, false
	}
	for _, item := range resp.Data {
		if item.B64Json != "" || item.Url != "" {
			return item, true
		}
	}
	return dto.ImageData{}, false
}

func ImageMarkdownContent(image dto.ImageData, outputFormat string) string {
	if image.B64Json != "" {
		return fmt.Sprintf("![generated image](data:%s;base64,%s)", imageMimeType(outputFormat), image.B64Json)
	}
	if image.Url != "" {
		return fmt.Sprintf("![generated image](%s)", image.Url)
	}
	return ""
}

func ImageResponseToChatResponse(resp *dto.ImageResponse, model string, id string, fallbackPromptTokens int) (*dto.OpenAITextResponse, *dto.Usage, error) {
	image, ok := FirstImageData(resp)
	if !ok {
		return nil, nil, errors.New("image response does not contain generated image data")
	}

	usage := ImageUsageToUsage(resp.Usage, fallbackPromptTokens)
	created := resp.Created
	if created == 0 {
		created = time.Now().Unix()
	}

	chatResp := &dto.OpenAITextResponse{
		Id:      id,
		Object:  "chat.completion",
		Created: created,
		Model:   model,
		Choices: []dto.OpenAITextResponseChoice{
			{
				Index: 0,
				Message: dto.Message{
					Role:    "assistant",
					Content: ImageMarkdownContent(image, resp.OutputFormat),
				},
				FinishReason: "stop",
			},
		},
		Usage: *usage,
	}
	return chatResp, usage, nil
}

func ImageResponseToResponsesResponse(resp *dto.ImageResponse, model string, id string, itemID string, fallbackPromptTokens int) (map[string]any, *dto.Usage, error) {
	image, ok := FirstImageData(resp)
	if !ok {
		return nil, nil, errors.New("image response does not contain generated image data")
	}

	usage := ImageUsageToUsage(resp.Usage, fallbackPromptTokens)
	created := resp.Created
	if created == 0 {
		created = time.Now().Unix()
	}

	output := imageOutputItem(image, resp, itemID, "completed")
	response := map[string]any{
		"id":                  id,
		"object":              "response",
		"created_at":          created,
		"status":              "completed",
		"model":               model,
		"output":              []map[string]any{output},
		"parallel_tool_calls": false,
		"usage":               ResponsesUsageMap(usage),
	}
	return response, usage, nil
}

func ResponsesUsageMap(usage *dto.Usage) map[string]any {
	if usage == nil {
		return nil
	}
	inputDetails := usage.PromptTokensDetails
	if usage.InputTokensDetails != nil {
		inputDetails = *usage.InputTokensDetails
	}
	outputDetails := usage.CompletionTokenDetails
	if outputDetails.ImageTokens == 0 && usage.OutputTokens != 0 {
		outputDetails.ImageTokens = usage.OutputTokens
	}
	return map[string]any{
		"input_tokens":  usage.PromptTokens,
		"output_tokens": usage.CompletionTokens,
		"total_tokens":  usage.TotalTokens,
		"input_tokens_details": map[string]any{
			"cached_tokens": inputDetails.CachedTokens,
			"text_tokens":   inputDetails.TextTokens,
			"image_tokens":  inputDetails.ImageTokens,
			"audio_tokens":  inputDetails.AudioTokens,
		},
		"output_tokens_details": map[string]any{
			"text_tokens":      outputDetails.TextTokens,
			"image_tokens":     outputDetails.ImageTokens,
			"audio_tokens":     outputDetails.AudioTokens,
			"reasoning_tokens": outputDetails.ReasoningTokens,
		},
	}
}

func ImageOutputItemFromStream(event dto.ImageStreamEvent, itemID string, status string) map[string]any {
	image := dto.ImageData{B64Json: event.B64Json}
	resp := &dto.ImageResponse{
		Background:   event.Background,
		OutputFormat: event.OutputFormat,
		Quality:      event.Quality,
		Size:         event.Size,
	}
	return imageOutputItem(image, resp, itemID, status)
}

func imageOutputItem(image dto.ImageData, resp *dto.ImageResponse, itemID string, status string) map[string]any {
	item := map[string]any{
		"id":     itemID,
		"type":   dto.ResponsesOutputTypeImageGenerationCall,
		"status": status,
	}
	if image.B64Json != "" {
		item["result"] = image.B64Json
	}
	if image.RevisedPrompt != "" {
		item["revised_prompt"] = image.RevisedPrompt
	}
	if resp != nil {
		if resp.Quality != "" {
			item["quality"] = resp.Quality
		}
		if resp.Size != "" {
			item["size"] = resp.Size
		}
		if resp.Background != "" {
			item["background"] = resp.Background
		}
		if resp.OutputFormat != "" {
			item["output_format"] = resp.OutputFormat
		}
	}
	return item
}

func extractPromptFromChatRequest(req *dto.GeneralOpenAIRequest) string {
	var parts []string
	for _, msg := range req.Messages {
		parts = append(parts, textPartsFromChatMessage(msg)...)
	}
	if len(parts) == 0 {
		parts = append(parts, textPartsFromAny(req.Prompt)...)
		parts = append(parts, textPartsFromAny(req.Input)...)
	}
	if req.Instruction != "" {
		parts = append([]string{req.Instruction}, parts...)
	}
	return strings.TrimSpace(strings.Join(compactStrings(parts), "\n\n"))
}

func extractPromptFromResponsesRequest(req *dto.OpenAIResponsesRequest) string {
	var parts []string
	if len(req.Instructions) > 0 {
		parts = append(parts, textPartsFromRaw(req.Instructions)...)
	}
	parts = append(parts, textPartsFromRaw(req.Input)...)
	if len(req.Prompt) > 0 {
		parts = append(parts, textPartsFromRaw(req.Prompt)...)
	}
	return strings.TrimSpace(strings.Join(compactStrings(parts), "\n\n"))
}

func textPartsFromChatMessage(msg dto.Message) []string {
	if msg.Content == nil {
		return nil
	}
	if msg.IsStringContent() {
		return []string{msg.StringContent()}
	}
	var parts []string
	for _, part := range msg.ParseContent() {
		if part.Type == dto.ContentTypeText && strings.TrimSpace(part.Text) != "" {
			parts = append(parts, part.Text)
		}
	}
	return parts
}

func textPartsFromRaw(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var value any
	if err := common.Unmarshal(raw, &value); err != nil {
		return nil
	}
	return textPartsFromAny(value)
}

func textPartsFromAny(value any) []string {
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		return []string{v}
	case []any:
		var parts []string
		for _, item := range v {
			parts = append(parts, textPartsFromAny(item)...)
		}
		return parts
	case map[string]any:
		var parts []string
		itemType := common.Interface2String(v["type"])
		if text := common.Interface2String(v["text"]); text != "" && (itemType == "" || itemType == "text" || itemType == "input_text" || itemType == "output_text") {
			parts = append(parts, text)
		}
		if content, ok := v["content"]; ok {
			parts = append(parts, textPartsFromAny(content)...)
		}
		if input, ok := v["input"]; ok {
			parts = append(parts, textPartsFromAny(input)...)
		}
		return parts
	default:
		return nil
	}
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = stripImagePromptGeneratedPayloads(value)
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func stripImagePromptGeneratedPayloads(value string) string {
	if value == "" {
		return value
	}
	value = imagePromptMarkdownDataURLPattern.ReplaceAllString(value, "")
	return imagePromptDataURLPattern.ReplaceAllString(value, "")
}

func ApplyImageGenerationToolOptions(raw []byte, imageReq *dto.ImageRequest) {
	if len(raw) == 0 || imageReq == nil {
		return
	}
	var tools []map[string]any
	if err := common.Unmarshal(raw, &tools); err != nil {
		return
	}
	for _, tool := range tools {
		if common.Interface2String(tool["type"]) != dto.BuildInToolImageGeneration {
			continue
		}
		applyImageOptionsFromMap(tool, imageReq)
	}
}

func ApplyImageOptionsFromRaw(raw []byte, imageReq *dto.ImageRequest) {
	if len(raw) == 0 || imageReq == nil {
		return
	}
	var values map[string]any
	if err := common.Unmarshal(raw, &values); err != nil {
		return
	}
	applyImageOptionsFromMap(values, imageReq)
}

func applyImageOptionsFromMap(values map[string]any, imageReq *dto.ImageRequest) {
	if imageReq == nil {
		return
	}
	if size := common.Interface2String(values["size"]); size != "" {
		imageReq.Size = size
	}
	if quality := common.Interface2String(values["quality"]); quality != "" {
		imageReq.Quality = quality
	}
	assignRawOption(values, "background", &imageReq.Background)
	assignRawOption(values, "output_format", &imageReq.OutputFormat)
	assignRawOption(values, "output_compression", &imageReq.OutputCompression)
	assignRawOption(values, "partial_images", &imageReq.PartialImages)
	assignRawOption(values, "moderation", &imageReq.Moderation)
}

func assignRawOption(values map[string]any, key string, target *json.RawMessage) {
	if values == nil || target == nil {
		return
	}
	value, ok := values[key]
	if !ok {
		return
	}
	raw, err := common.Marshal(value)
	if err != nil {
		return
	}
	*target = raw
}

func imageMimeType(outputFormat string) string {
	switch strings.ToLower(strings.TrimSpace(outputFormat)) {
	case "jpeg", "jpg":
		return "image/jpeg"
	case "webp":
		return "image/webp"
	default:
		return "image/png"
	}
}
