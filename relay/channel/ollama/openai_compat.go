package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func ollamaReasoningText(raw json.RawMessage) string {
	text := strings.TrimSpace(string(raw))
	if text == "" || text == "null" {
		return ""
	}

	var decoded string
	if err := common.Unmarshal(raw, &decoded); err == nil {
		return decoded
	}
	return text
}

func ollamaToolCallsToOpenAI(toolCalls []OllamaToolCall, nextIndex *int) []dto.ToolCallResponse {
	if len(toolCalls) == 0 {
		return nil
	}

	responses := make([]dto.ToolCallResponse, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		args := toolCall.Function.Arguments
		if args == nil {
			args = map[string]any{}
		}

		argBytes, _ := common.Marshal(args)
		call := dto.ToolCallResponse{
			Type: "function",
			Function: dto.FunctionResponse{
				Name:      toolCall.Function.Name,
				Arguments: string(argBytes),
			},
		}

		if nextIndex != nil {
			index := *nextIndex
			call.ID = fmt.Sprintf("call_%d", index)
			call.SetIndex(index)
			*nextIndex = index + 1
		}

		responses = append(responses, call)
	}
	return responses
}

func ollamaFinishReason(chunk ollamaChatStreamChunk, hasToolCalls bool) string {
	if hasToolCalls {
		return constant.FinishReasonToolCalls
	}

	switch strings.ToLower(strings.TrimSpace(chunk.DoneReason)) {
	case "", "stop":
		return constant.FinishReasonStop
	case "length":
		return constant.FinishReasonLength
	case "tool_calls", "tool_call":
		return constant.FinishReasonToolCalls
	default:
		return constant.FinishReasonStop
	}
}

func ollamaChunkHasPayload(chunk ollamaChatStreamChunk) bool {
	if chunk.Message != nil {
		if chunk.Message.Content != "" {
			return true
		}
		if ollamaReasoningText(chunk.Message.Thinking) != "" {
			return true
		}
		if len(chunk.Message.ToolCalls) > 0 {
			return true
		}
	}
	return chunk.Response != ""
}

func ollamaChunkToOpenAIStreamResponse(responseID string, chunk ollamaChatStreamChunk, fallbackModel string, fallbackCreated int64, nextToolCallIndex *int) (*dto.ChatCompletionsStreamResponse, string, string) {
	if !ollamaChunkHasPayload(chunk) {
		return nil, "", ""
	}

	model := fallbackModel
	if chunk.Model != "" {
		model = chunk.Model
	}

	created := fallbackCreated
	if chunk.CreatedAt != "" {
		created = toUnix(chunk.CreatedAt)
	}

	content := chunk.Response
	reasoning := ""
	var toolCalls []dto.ToolCallResponse
	if chunk.Message != nil {
		if chunk.Message.Content != "" {
			content = chunk.Message.Content
		}
		reasoning = ollamaReasoningText(chunk.Message.Thinking)
		toolCalls = ollamaToolCallsToOpenAI(chunk.Message.ToolCalls, nextToolCallIndex)
	}

	response := &dto.ChatCompletionsStreamResponse{
		Id:      responseID,
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   model,
		Choices: []dto.ChatCompletionsStreamResponseChoice{
			{
				Index: 0,
				Delta: dto.ChatCompletionsStreamResponseChoiceDelta{},
			},
		},
	}

	if content != "" {
		response.Choices[0].Delta.SetContentString(content)
	}
	if reasoning != "" {
		response.Choices[0].Delta.SetReasoningContent(reasoning)
	}
	if len(toolCalls) > 0 {
		response.Choices[0].Delta.ToolCalls = toolCalls
	}

	return response, content, reasoning
}

func ollamaBodyToOpenAIResponse(responseID string, body []byte, info *relaycommon.RelayInfo) (*dto.OpenAITextResponse, error) {
	lines := strings.Split(string(body), "\n")
	chunks := make([]ollamaChatStreamChunk, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var chunk ollamaChatStreamChunk
		if err := common.UnmarshalJsonStr(line, &chunk); err != nil {
			if len(lines) == 1 {
				return nil, err
			}
			continue
		}
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		var chunk ollamaChatStreamChunk
		if err := common.Unmarshal(body, &chunk); err != nil {
			return nil, err
		}
		chunks = append(chunks, chunk)
	}

	var contentBuilder strings.Builder
	var reasoningBuilder strings.Builder
	var toolCalls []dto.ToolCallResponse
	nextToolCallIndex := 0
	model := info.UpstreamModelName
	created := common.GetTimestamp()

	for _, chunk := range chunks {
		if chunk.Model != "" {
			model = chunk.Model
		}
		if chunk.CreatedAt != "" {
			created = toUnix(chunk.CreatedAt)
		}
		if chunk.Message != nil {
			contentBuilder.WriteString(chunk.Message.Content)
			reasoningBuilder.WriteString(ollamaReasoningText(chunk.Message.Thinking))
			toolCalls = append(toolCalls, ollamaToolCallsToOpenAI(chunk.Message.ToolCalls, &nextToolCallIndex)...)
			continue
		}
		contentBuilder.WriteString(chunk.Response)
	}

	lastChunk := chunks[len(chunks)-1]
	message := dto.Message{
		Role: "assistant",
	}
	message.SetStringContent(contentBuilder.String())
	message.ReasoningContent = reasoningBuilder.String()
	if len(toolCalls) > 0 {
		message.SetToolCalls(toolCalls)
	}

	usage := dto.Usage{
		PromptTokens:     lastChunk.PromptEvalCount,
		CompletionTokens: lastChunk.EvalCount,
		TotalTokens:      lastChunk.PromptEvalCount + lastChunk.EvalCount,
	}

	response := &dto.OpenAITextResponse{
		Id:      responseID,
		Model:   model,
		Object:  "chat.completion",
		Created: created,
		Choices: []dto.OpenAITextResponseChoice{
			{
				Index:        0,
				Message:      message,
				FinishReason: ollamaFinishReason(lastChunk, len(toolCalls) > 0),
			},
		},
		Usage: usage,
	}
	return response, nil
}

func newOpenAIHTTPResponse(original *http.Response, payload any) (*http.Response, error) {
	data, err := common.Marshal(payload)
	if err != nil {
		return nil, err
	}

	headers := make(http.Header)
	if original != nil && original.Header != nil {
		headers = original.Header.Clone()
	}
	headers.Set("Content-Type", "application/json")

	statusCode := http.StatusOK
	if original != nil && original.StatusCode != 0 {
		statusCode = original.StatusCode
	}

	return &http.Response{
		StatusCode: statusCode,
		Header:     headers,
		Body:       io.NopCloser(bytes.NewReader(data)),
	}, nil
}
