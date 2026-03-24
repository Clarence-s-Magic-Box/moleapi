package helper

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newModelMappedContext(modelMapping string) *gin.Context {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	if modelMapping != "" {
		c.Set("model_mapping", modelMapping)
	}
	return c
}

func TestModelMappedHelper_SystemRedirect_MolePrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := newModelMappedContext("")
	info := &relaycommon.RelayInfo{
		OriginModelName: "mole-gpt-4o",
	}
	request := &dto.GeneralOpenAIRequest{Model: "mole-gpt-4o"}

	err := ModelMappedHelper(c, info, request)
	require.NoError(t, err)
	require.True(t, info.IsModelMapped)
	require.Equal(t, "gpt-4o", info.UpstreamModelName)
	require.Equal(t, "gpt-4o", request.Model)
}

func TestModelMappedHelper_SystemRedirect_WithChannelMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := newModelMappedContext(`{"gpt-4o":"o3"}`)
	info := &relaycommon.RelayInfo{
		OriginModelName: "mole-gpt-4o",
	}
	request := &dto.GeneralOpenAIRequest{Model: "mole-gpt-4o"}

	err := ModelMappedHelper(c, info, request)
	require.NoError(t, err)
	require.True(t, info.IsModelMapped)
	require.Equal(t, "o3", info.UpstreamModelName)
	require.Equal(t, "o3", request.Model)
}

func TestModelMappedHelper_SystemRedirect_NonMolePrefixNotRedirected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := newModelMappedContext("")
	info := &relaycommon.RelayInfo{
		OriginModelName: "moleapi-gpt-4o",
	}
	request := &dto.GeneralOpenAIRequest{Model: "moleapi-gpt-4o"}

	err := ModelMappedHelper(c, info, request)
	require.NoError(t, err)
	require.False(t, info.IsModelMapped)
	require.Equal(t, "", info.UpstreamModelName)
	require.Equal(t, "moleapi-gpt-4o", request.Model)
}

func TestModelMappedHelper_SystemRedirect_GptAlias(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := newModelMappedContext("")
	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt5.4",
	}
	request := &dto.GeneralOpenAIRequest{Model: "gpt5.4"}

	err := ModelMappedHelper(c, info, request)
	require.NoError(t, err)
	require.True(t, info.IsModelMapped)
	require.Equal(t, "gpt-5.4", info.UpstreamModelName)
	require.Equal(t, "gpt-5.4", request.Model)
}
