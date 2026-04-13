package ollama

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type ollamaChatStreamChunk struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	// chat
	Message *struct {
		Role      string           `json:"role"`
		Content   string           `json:"content"`
		Thinking  json.RawMessage  `json:"thinking"`
		ToolCalls []OllamaToolCall `json:"tool_calls"`
	} `json:"message"`
	// generate
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	DoneReason         string `json:"done_reason"`
	TotalDuration      int64  `json:"total_duration"`
	LoadDuration       int64  `json:"load_duration"`
	PromptEvalCount    int    `json:"prompt_eval_count"`
	EvalCount          int    `json:"eval_count"`
	PromptEvalDuration int64  `json:"prompt_eval_duration"`
	EvalDuration       int64  `json:"eval_duration"`
}

func toUnix(ts string) int64 {
	if ts == "" {
		return time.Now().Unix()
	}
	// try time.RFC3339 or with nanoseconds
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		t2, err2 := time.Parse(time.RFC3339, ts)
		if err2 == nil {
			return t2.Unix()
		}
		return time.Now().Unix()
	}
	return t.Unix()
}

func ollamaStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		return nil, types.NewOpenAIError(fmt.Errorf("empty response"), types.ErrorCodeBadResponse, http.StatusBadRequest)
	}
	defer service.CloseResponseBodyGracefully(resp)

	helper.SetEventStreamHeaders(c)
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	usage := &dto.Usage{}
	model := info.UpstreamModelName
	responseID := helper.GetResponseID(c)
	created := time.Now().Unix()
	nextToolCallIndex := 0
	toolCallCount := 0
	sawToolCalls := false
	sentStart := false
	var responseTextBuilder strings.Builder
	var finalStreamResponse *dto.ChatCompletionsStreamResponse

	sendStartResponse := func() *types.NewAPIError {
		if sentStart {
			return nil
		}
		start := helper.GenerateStartEmptyResponse(responseID, created, model, nil)
		startData, err := common.Marshal(start)
		if err != nil {
			return types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
		}
		if err := openai.HandleStreamFormat(c, info, string(startData), info.ChannelSetting.ForceFormat, info.ChannelSetting.ThinkingToContent); err != nil {
			return types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
		}
		sentStart = true
		return nil
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var chunk ollamaChatStreamChunk
		if err := common.UnmarshalJsonStr(line, &chunk); err != nil {
			logger.LogError(c, "ollama stream json decode error: "+err.Error()+" line="+line)
			return usage, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
		}
		if chunk.Model != "" {
			model = chunk.Model
		}
		created = toUnix(chunk.CreatedAt)

		if (chunk.Done || ollamaChunkHasPayload(chunk)) && !sentStart {
			if err := sendStartResponse(); err != nil {
				return usage, err
			}
		}

		streamChunk, contentDelta, reasoningDelta := ollamaChunkToOpenAIStreamResponse(responseID, chunk, model, created, &nextToolCallIndex)
		if streamChunk != nil {
			responseTextBuilder.WriteString(contentDelta)
			responseTextBuilder.WriteString(reasoningDelta)
			if len(streamChunk.Choices) > 0 && len(streamChunk.Choices[0].Delta.ToolCalls) > 0 {
				sawToolCalls = true
				toolCallCount += len(streamChunk.Choices[0].Delta.ToolCalls)
			}

			chunkData, err := common.Marshal(streamChunk)
			if err != nil {
				return usage, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
			}
			if err := openai.HandleStreamFormat(c, info, string(chunkData), info.ChannelSetting.ForceFormat, info.ChannelSetting.ThinkingToContent); err != nil {
				return usage, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
			}
		}

		if !chunk.Done {
			continue
		}

		usage.PromptTokens = chunk.PromptEvalCount
		usage.CompletionTokens = chunk.EvalCount
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
		finalStreamResponse = helper.GenerateStopResponse(responseID, created, model, ollamaFinishReason(chunk, sawToolCalls))
		break
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		logger.LogError(c, "ollama stream scan error: "+err.Error())
		return usage, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	if finalStreamResponse == nil {
		if !sentStart {
			if err := sendStartResponse(); err != nil {
				return usage, err
			}
		}
		finalStreamResponse = helper.GenerateStopResponse(responseID, created, model, constant.FinishReasonStop)
	}

	if usage.TotalTokens == 0 {
		estimatedUsage := service.ResponseText2Usage(c, responseTextBuilder.String(), model, info.GetEstimatePromptTokens())
		estimatedUsage.CompletionTokens += toolCallCount * 7
		estimatedUsage.TotalTokens = estimatedUsage.PromptTokens + estimatedUsage.CompletionTokens
		usage = estimatedUsage
	}

	finalWithUsage := finalStreamResponse.Copy()
	finalWithUsage.Usage = usage
	finalData, err := common.Marshal(finalWithUsage)
	if err != nil {
		return usage, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	containStreamUsage := false
	if info.RelayFormat == types.RelayFormatOpenAI {
		openAIFinal := finalStreamResponse.Copy()
		if info.ShouldIncludeUsage {
			openAIFinal.Usage = usage
			containStreamUsage = true
		}
		openAIFinalData, err := common.Marshal(openAIFinal)
		if err != nil {
			return usage, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
		}
		if err := openai.HandleStreamFormat(c, info, string(openAIFinalData), info.ChannelSetting.ForceFormat, info.ChannelSetting.ThinkingToContent); err != nil {
			return usage, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
		}
	}

	openai.HandleFinalResponse(c, info, string(finalData), responseID, created, model, "", usage, containStreamUsage)
	return usage, nil
}

// non-stream handler for chat/generate
func ollamaChatHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	service.CloseResponseBodyGracefully(resp)
	if common.DebugEnabled {
		println("ollama non-stream raw resp:", string(body))
	}

	openAIResponse, err := ollamaBodyToOpenAIResponse(common.GetUUID(), body, info)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	wrappedResp, err := newOpenAIHTTPResponse(resp, openAIResponse)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}

	openaiAdaptor := openai.Adaptor{}
	usageAny, newAPIError := openaiAdaptor.DoResponse(c, wrappedResp, info)
	if newAPIError != nil {
		return nil, newAPIError
	}

	usage, ok := usageAny.(*dto.Usage)
	if !ok {
		return nil, types.NewOpenAIError(fmt.Errorf("unexpected usage type: %T", usageAny), types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	return usage, nil
}
