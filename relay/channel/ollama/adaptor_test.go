package ollama

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertGeminiRequestToOllamaChat(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1beta/models/gpt-oss:120b:generateContent", nil)

	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-oss:120b",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-oss:120b",
		},
	}

	request := &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{
				Role: "user",
				Parts: []dto.GeminiPart{
					{Text: "Why is the sky blue?"},
				},
			},
		},
	}

	adaptor := &Adaptor{}
	converted, err := adaptor.ConvertGeminiRequest(c, info, request)
	require.NoError(t, err)

	ollamaRequest, ok := converted.(*OllamaChatRequest)
	require.True(t, ok)
	require.Equal(t, "gpt-oss:120b", ollamaRequest.Model)
	require.Len(t, ollamaRequest.Messages, 1)
	require.Equal(t, "user", ollamaRequest.Messages[0].Role)
	require.Equal(t, "Why is the sky blue?", ollamaRequest.Messages[0].Content)
}

func TestDoResponseConvertsOllamaToClaudeNonStream(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatClaude,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-oss:120b",
		},
		ClaudeConvertInfo: &relaycommon.ClaudeConvertInfo{
			LastMessagesType: relaycommon.LastMessageTypeNone,
		},
	}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body: io.NopCloser(strings.NewReader(
			`{"model":"gpt-oss:120b","created_at":"2026-04-13T00:00:00Z","message":{"role":"assistant","content":"hello from ollama"},"done":true,"done_reason":"stop","prompt_eval_count":12,"eval_count":7}`,
		)),
	}

	adaptor := &Adaptor{}
	usageAny, newAPIError := adaptor.DoResponse(c, resp, info)
	require.Nil(t, newAPIError)

	usage, ok := usageAny.(*dto.Usage)
	require.True(t, ok)
	require.Equal(t, 12, usage.PromptTokens)
	require.Equal(t, 7, usage.CompletionTokens)

	var claudeResp dto.ClaudeResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &claudeResp))
	require.Equal(t, "message", claudeResp.Type)
	require.Equal(t, "assistant", claudeResp.Role)
	require.Len(t, claudeResp.Content, 1)
	require.Equal(t, "text", claudeResp.Content[0].Type)
	require.Equal(t, "hello from ollama", claudeResp.Content[0].GetText())
	require.NotNil(t, claudeResp.Usage)
	require.Equal(t, 12, claudeResp.Usage.InputTokens)
	require.Equal(t, 7, claudeResp.Usage.OutputTokens)
}

func TestDoResponseConvertsOllamaToGeminiNonStream(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1beta/models/gpt-oss:120b:generateContent", nil)

	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatGemini,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-oss:120b",
		},
	}

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body: io.NopCloser(strings.NewReader(
			`{"model":"gpt-oss:120b","created_at":"2026-04-13T00:00:00Z","message":{"role":"assistant","content":"hello from ollama"},"done":true,"done_reason":"stop","prompt_eval_count":12,"eval_count":7}`,
		)),
	}

	adaptor := &Adaptor{}
	usageAny, newAPIError := adaptor.DoResponse(c, resp, info)
	require.Nil(t, newAPIError)

	usage, ok := usageAny.(*dto.Usage)
	require.True(t, ok)
	require.Equal(t, 19, usage.TotalTokens)

	var geminiResp dto.GeminiChatResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &geminiResp))
	require.Len(t, geminiResp.Candidates, 1)
	require.Len(t, geminiResp.Candidates[0].Content.Parts, 1)
	require.Equal(t, "hello from ollama", geminiResp.Candidates[0].Content.Parts[0].Text)
	require.Equal(t, 12, geminiResp.UsageMetadata.PromptTokenCount)
	require.Equal(t, 7, geminiResp.UsageMetadata.CandidatesTokenCount)
	require.Equal(t, 19, geminiResp.UsageMetadata.TotalTokenCount)
}

func TestDoResponseConvertsOllamaToClaudeStream(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatClaude,
		IsStream:    true,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-oss:120b",
		},
		ClaudeConvertInfo: &relaycommon.ClaudeConvertInfo{
			LastMessagesType: relaycommon.LastMessageTypeNone,
		},
	}

	streamBody := strings.Join([]string{
		`{"model":"gpt-oss:120b","created_at":"2026-04-13T00:00:00Z","message":{"role":"assistant","content":"hel"},"done":false}`,
		`{"model":"gpt-oss:120b","created_at":"2026-04-13T00:00:01Z","message":{"role":"assistant","content":"lo"},"done":false}`,
		`{"model":"gpt-oss:120b","created_at":"2026-04-13T00:00:01Z","done":true,"done_reason":"stop","prompt_eval_count":12,"eval_count":7}`,
	}, "\n")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(streamBody)),
	}

	adaptor := &Adaptor{}
	usageAny, newAPIError := adaptor.DoResponse(c, resp, info)
	require.Nil(t, newAPIError)

	usage, ok := usageAny.(*dto.Usage)
	require.True(t, ok)
	require.Equal(t, 12, usage.PromptTokens)
	require.Equal(t, 7, usage.CompletionTokens)

	body := recorder.Body.String()
	require.Contains(t, body, "event: message_start")
	require.Contains(t, body, `"type":"text_delta"`)
	require.Contains(t, body, `"text":"hel"`)
	require.Contains(t, body, "event: message_stop")
}

func TestDoResponseConvertsOllamaToGeminiStream(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1beta/models/gpt-oss:120b:streamGenerateContent", nil)

	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatGemini,
		IsStream:    true,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "gpt-oss:120b",
		},
	}

	streamBody := strings.Join([]string{
		`{"model":"gpt-oss:120b","created_at":"2026-04-13T00:00:00Z","message":{"role":"assistant","content":"hel"},"done":false}`,
		`{"model":"gpt-oss:120b","created_at":"2026-04-13T00:00:01Z","message":{"role":"assistant","content":"lo"},"done":false}`,
		`{"model":"gpt-oss:120b","created_at":"2026-04-13T00:00:01Z","done":true,"done_reason":"stop","prompt_eval_count":12,"eval_count":7}`,
	}, "\n")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(streamBody)),
	}

	adaptor := &Adaptor{}
	usageAny, newAPIError := adaptor.DoResponse(c, resp, info)
	require.Nil(t, newAPIError)

	usage, ok := usageAny.(*dto.Usage)
	require.True(t, ok)
	require.Equal(t, 19, usage.TotalTokens)

	body := recorder.Body.String()
	require.Contains(t, body, `"text":"hel"`)
	require.Contains(t, body, `"finishReason":"STOP"`)
	require.Contains(t, body, `"promptTokenCount":12`)
	require.Contains(t, body, `"candidatesTokenCount":7`)
}
