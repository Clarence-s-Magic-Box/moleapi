package openaicompat

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestResponsesResponseToChatCompletionsResponseTracksImageTokens(t *testing.T) {
	t.Parallel()

	resp := &dto.OpenAIResponsesResponse{
		Model: "gpt-4.1",
		Usage: &dto.Usage{
			InputTokens:  120,
			OutputTokens: 360,
			TotalTokens:  480,
			InputTokensDetails: &dto.InputTokenDetails{
				CachedTokens: 10,
				TextTokens:   40,
				ImageTokens:  70,
			},
			CompletionTokenDetails: dto.OutputTokenDetails{
				TextTokens:      180,
				ImageTokens:     150,
				ReasoningTokens: 30,
			},
		},
		Output: []dto.ResponsesOutput{
			{
				Type: "message",
				Role: "assistant",
				Content: []dto.ResponsesOutputContent{
					{Type: "output_text", Text: "ok"},
				},
			},
		},
	}

	chatResp, usage, err := ResponsesResponseToChatCompletionsResponse(resp, "resp_1")
	require.NoError(t, err)
	require.NotNil(t, chatResp)
	require.NotNil(t, usage)
	require.Equal(t, 70, usage.PromptTokensDetails.ImageTokens)
	require.Equal(t, 40, usage.PromptTokensDetails.TextTokens)
	require.Equal(t, 150, usage.CompletionTokenDetails.ImageTokens)
	require.Equal(t, 180, usage.CompletionTokenDetails.TextTokens)
	require.Equal(t, 30, usage.CompletionTokenDetails.ReasoningTokens)
	require.Equal(t, usage.CompletionTokenDetails.ImageTokens, chatResp.Usage.CompletionTokenDetails.ImageTokens)
}
