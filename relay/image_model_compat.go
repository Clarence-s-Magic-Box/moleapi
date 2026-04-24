package relay

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func chatCompletionsViaImageGeneration(c *gin.Context, info *relaycommon.RelayInfo, adaptor channel.Adaptor, request *dto.GeneralOpenAIRequest) (*dto.Usage, *types.NewAPIError) {
	imageReq, err := service.ChatCompletionsRequestToImageRequest(request)
	if err != nil {
		return nil, types.NewErrorWithStatusCode(err, types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	applyImageOptionsFromRequestBody(c, imageReq)
	if imageReq.Stream != nil && *imageReq.Stream {
		// Chat Completions is append-only text streaming, so partial images would
		// become permanent extra images in the final message. Stream only the final image.
		imageReq.PartialImages = []byte("0")
	}

	httpResp, newAPIError := doImageGenerationRequest(c, info, adaptor, imageReq)
	if newAPIError != nil {
		return nil, newAPIError
	}
	if imageReq.IsStream(c) || strings.HasPrefix(httpResp.Header.Get("Content-Type"), "text/event-stream") {
		return imageStreamToChatCompletions(c, info, httpResp, imageReq.Model)
	}
	return imageResponseToChatCompletions(c, info, httpResp, imageReq.Model)
}

func responsesViaImageGeneration(c *gin.Context, info *relaycommon.RelayInfo, adaptor channel.Adaptor, request *dto.OpenAIResponsesRequest) (*dto.Usage, *types.NewAPIError) {
	imageReq, err := service.ResponsesRequestToImageRequest(request)
	if err != nil {
		return nil, types.NewErrorWithStatusCode(err, types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	applyImageOptionsFromRequestBody(c, imageReq)
	service.ApplyImageGenerationToolOptions(request.Tools, imageReq)

	httpResp, newAPIError := doImageGenerationRequest(c, info, adaptor, imageReq)
	if newAPIError != nil {
		return nil, newAPIError
	}
	if imageReq.IsStream(c) || strings.HasPrefix(httpResp.Header.Get("Content-Type"), "text/event-stream") {
		return imageStreamToResponses(c, info, httpResp, imageReq.Model)
	}
	return imageResponseToResponses(c, info, httpResp, imageReq.Model)
}

func doImageGenerationRequest(c *gin.Context, info *relaycommon.RelayInfo, adaptor channel.Adaptor, imageReq *dto.ImageRequest) (*http.Response, *types.NewAPIError) {
	if imageReq == nil {
		return nil, types.NewErrorWithStatusCode(fmt.Errorf("image request is nil"), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	savedRelayMode := info.RelayMode
	savedRequestURLPath := info.RequestURLPath
	savedRequest := info.Request
	defer func() {
		info.RelayMode = savedRelayMode
		info.RequestURLPath = savedRequestURLPath
		info.Request = savedRequest
	}()

	info.RelayMode = relayconstant.RelayModeImagesGenerations
	info.RequestURLPath = "/v1/images/generations"
	info.Request = imageReq

	convertedRequest, err := adaptor.ConvertImageRequest(c, info, *imageReq)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}
	relaycommon.AppendRequestConversionFromRequest(info, convertedRequest)

	var requestBody io.Reader
	switch v := convertedRequest.(type) {
	case io.Reader:
		requestBody = v
	default:
		jsonData, err := common.Marshal(convertedRequest)
		if err != nil {
			return nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}
		jsonData, err = relaycommon.RemoveDisabledFields(jsonData, info.ChannelOtherSettings, info.ChannelSetting.PassThroughBodyEnabled)
		if err != nil {
			return nil, types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}
		if len(info.ParamOverride) > 0 {
			jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
			if err != nil {
				return nil, newAPIErrorFromParamOverride(err)
			}
		}
		requestBody = bytes.NewBuffer(jsonData)
	}

	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}
	if resp == nil {
		return nil, types.NewOpenAIError(nil, types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}

	httpResp := resp.(*http.Response)
	info.IsStream = info.IsStream || imageReq.IsStream(c) || strings.HasPrefix(httpResp.Header.Get("Content-Type"), "text/event-stream")
	if httpResp.StatusCode != http.StatusOK {
		newAPIError := service.RelayErrorHandler(c.Request.Context(), httpResp, false)
		service.ResetStatusCode(newAPIError, c.GetString("status_code_mapping"))
		return nil, newAPIError
	}
	return httpResp, nil
}

func imageResponseToChatCompletions(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response, model string) (*dto.Usage, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)

	imageResp, body, newAPIError := readImageResponse(resp)
	if newAPIError != nil {
		return nil, newAPIError
	}

	chatResp, usage, err := service.ImageResponseToChatResponse(imageResp, model, helper.GetResponseID(c), info.GetEstimatePromptTokens())
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	body, err = common.Marshal(chatResp)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeJsonMarshalFailed, http.StatusInternalServerError)
	}
	service.IOCopyBytesGracefully(c, resp, body)
	return usage, nil
}

func imageResponseToResponses(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response, model string) (*dto.Usage, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)

	imageResp, body, newAPIError := readImageResponse(resp)
	if newAPIError != nil {
		return nil, newAPIError
	}

	response, usage, err := service.ImageResponseToResponsesResponse(imageResp, model, getResponsesCompatID(c), getImageCompatItemID(c), info.GetEstimatePromptTokens())
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	body, err = common.Marshal(response)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeJsonMarshalFailed, http.StatusInternalServerError)
	}
	service.IOCopyBytesGracefully(c, resp, body)
	return usage, nil
}

func imageStreamToChatCompletions(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response, model string) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		return nil, types.NewOpenAIError(fmt.Errorf("invalid response"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}

	responseID := helper.GetResponseID(c)
	created := time.Now().Unix()
	usage := &dto.Usage{}
	var streamErr *types.NewAPIError
	sentStart := false
	sentStop := false
	sentFinalImage := false

	sendStart := func() bool {
		if sentStart {
			return true
		}
		if err := helper.ObjectData(c, helper.GenerateStartEmptyResponse(responseID, created, model, nil)); err != nil {
			streamErr = types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
			return false
		}
		sentStart = true
		return true
	}
	sendContent := func(content string) bool {
		if content == "" {
			return true
		}
		if !sendStart() {
			return false
		}
		chunk := &dto.ChatCompletionsStreamResponse{
			Id:      responseID,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   model,
			Choices: []dto.ChatCompletionsStreamResponseChoice{
				{
					Index: 0,
					Delta: dto.ChatCompletionsStreamResponseChoiceDelta{
						Content: &content,
					},
				},
			},
		}
		if err := helper.ObjectData(c, chunk); err != nil {
			streamErr = types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
			return false
		}
		return true
	}
	sendStop := func() bool {
		if sentStop {
			return true
		}
		if !sendStart() {
			return false
		}
		if err := helper.ObjectData(c, helper.GenerateStopResponse(responseID, created, model, "stop")); err != nil {
			streamErr = types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
			return false
		}
		sentStop = true
		return true
	}

	helper.StreamScannerHandler(c, resp, info, func(data string, sr *helper.StreamResult) {
		if streamErr != nil {
			sr.Stop(streamErr)
			return
		}
		var event dto.ImageStreamEvent
		if err := common.UnmarshalJsonStr(data, &event); err != nil {
			streamErr = types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
			sr.Stop(streamErr)
			return
		}
		if event.Type != "image_generation.completed" {
			return
		}
		usage = service.ImageUsageToUsage(event.Usage, info.GetEstimatePromptTokens())
		content := service.ImageMarkdownContent(dto.ImageData{B64Json: event.B64Json}, event.OutputFormat)
		if !sendContent(content) {
			sr.Stop(streamErr)
			return
		}
		sentFinalImage = true
		if !sendStop() {
			sr.Stop(streamErr)
			return
		}
		sr.Done()
	})

	if streamErr != nil {
		return nil, streamErr
	}
	if !sentFinalImage {
		usage = service.ImageUsageToUsage(nil, info.GetEstimatePromptTokens())
	}
	if !sentStop && !sendStop() {
		return nil, streamErr
	}
	if info.ShouldIncludeUsage {
		if err := helper.ObjectData(c, helper.GenerateFinalUsageResponse(responseID, created, model, *usage)); err != nil {
			return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
		}
	}
	helper.Done(c)
	return usage, nil
}

func imageStreamToResponses(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response, model string) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		return nil, types.NewOpenAIError(fmt.Errorf("invalid response"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}

	responseID := getResponsesCompatID(c)
	itemID := getImageCompatItemID(c)
	created := time.Now().Unix()
	usage := &dto.Usage{}
	var streamErr *types.NewAPIError
	sentStart := false
	sentCompleted := false

	sendEvent := func(eventType string, payload map[string]any) bool {
		payload["type"] = eventType
		data, err := common.Marshal(payload)
		if err != nil {
			streamErr = types.NewOpenAIError(err, types.ErrorCodeJsonMarshalFailed, http.StatusInternalServerError)
			return false
		}
		helper.ResponseChunkData(c, dto.ResponsesStreamResponse{Type: eventType}, string(data))
		return true
	}
	sendStart := func() bool {
		if sentStart {
			return true
		}
		if !sendEvent("response.created", map[string]any{
			"response": map[string]any{
				"id":         responseID,
				"object":     "response",
				"created_at": created,
				"status":     "in_progress",
				"model":      model,
				"output":     []any{},
			},
		}) {
			return false
		}
		if !sendEvent(dto.ResponsesOutputTypeItemAdded, map[string]any{
			"output_index": 0,
			"item": map[string]any{
				"id":     itemID,
				"type":   dto.ResponsesOutputTypeImageGenerationCall,
				"status": "in_progress",
			},
		}) {
			return false
		}
		sentStart = true
		return true
	}

	helper.StreamScannerHandler(c, resp, info, func(data string, sr *helper.StreamResult) {
		if streamErr != nil {
			sr.Stop(streamErr)
			return
		}
		var event dto.ImageStreamEvent
		if err := common.UnmarshalJsonStr(data, &event); err != nil {
			streamErr = types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
			sr.Stop(streamErr)
			return
		}
		if !sendStart() {
			sr.Stop(streamErr)
			return
		}
		switch event.Type {
		case "image_generation.partial_image":
			payload := map[string]any{
				"output_index":      0,
				"item_id":           itemID,
				"partial_image_b64": event.B64Json,
			}
			if event.PartialImageIndex != nil {
				payload["partial_image_index"] = *event.PartialImageIndex
			}
			if event.Size != "" {
				payload["size"] = event.Size
			}
			if event.Quality != "" {
				payload["quality"] = event.Quality
			}
			if event.Background != "" {
				payload["background"] = event.Background
			}
			if event.OutputFormat != "" {
				payload["output_format"] = event.OutputFormat
			}
			if !sendEvent("response.image_generation_call.partial_image", payload) {
				sr.Stop(streamErr)
				return
			}
		case "image_generation.completed":
			usage = service.ImageUsageToUsage(event.Usage, info.GetEstimatePromptTokens())
			outputItem := service.ImageOutputItemFromStream(event, itemID, "completed")
			if !sendEvent(dto.ResponsesOutputTypeItemDone, map[string]any{
				"output_index": 0,
				"item":         outputItem,
			}) {
				sr.Stop(streamErr)
				return
			}
			if !sendEvent("response.completed", map[string]any{
				"response": map[string]any{
					"id":                  responseID,
					"object":              "response",
					"created_at":          created,
					"status":              "completed",
					"model":               model,
					"output":              []map[string]any{outputItem},
					"parallel_tool_calls": false,
					"usage":               service.ResponsesUsageMap(usage),
				},
			}) {
				sr.Stop(streamErr)
				return
			}
			sentCompleted = true
			sr.Done()
		}
	})

	if streamErr != nil {
		return nil, streamErr
	}
	if !sentCompleted {
		usage = service.ImageUsageToUsage(nil, info.GetEstimatePromptTokens())
	}
	return usage, nil
}

func readImageResponse(resp *http.Response) (*dto.ImageResponse, []byte, *types.NewAPIError) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	var imageResp dto.ImageResponse
	if err := common.Unmarshal(body, &imageResp); err != nil {
		return nil, nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	return &imageResp, body, nil
}

func applyImageOptionsFromRequestBody(c *gin.Context, imageReq *dto.ImageRequest) {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return
	}
	body, err := storage.Bytes()
	if err != nil {
		return
	}
	service.ApplyImageOptionsFromRaw(body, imageReq)
}

func getResponsesCompatID(c *gin.Context) string {
	return fmt.Sprintf("resp_%s", c.GetString(common.RequestIdKey))
}

func getImageCompatItemID(c *gin.Context) string {
	return fmt.Sprintf("igc_%s", c.GetString(common.RequestIdKey))
}
