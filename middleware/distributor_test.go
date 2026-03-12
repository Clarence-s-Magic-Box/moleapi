package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newDistributorTestContext(path string, body string) *gin.Context {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: path},
		Header: make(http.Header),
		Body:   io.NopCloser(strings.NewReader(body)),
	}
	c.Request.Header.Set("Content-Type", "application/json")
	return c
}

func TestGetModelRequest_SystemRedirect_MolePrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := newDistributorTestContext("/v1/chat/completions", `{"model":"mole-gpt-4o"}`)

	req, shouldSelectChannel, err := getModelRequest(c)
	require.NoError(t, err)
	require.True(t, shouldSelectChannel)
	require.Equal(t, "gpt-4o", req.Model)
}

func TestGetModelRequest_SystemRedirect_ResponsesCompact(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := newDistributorTestContext("/v1/responses/compact", `{"model":"mole-gpt-4.1"}`)

	req, shouldSelectChannel, err := getModelRequest(c)
	require.NoError(t, err)
	require.True(t, shouldSelectChannel)
	require.Equal(t, ratio_setting.WithCompactModelSuffix("gpt-4.1"), req.Model)
}

func TestGetModelRequest_SystemRedirect_IgnoreEmptyTarget(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := newDistributorTestContext("/v1/chat/completions", `{"model":"mole-"}`)

	req, _, err := getModelRequest(c)
	require.NoError(t, err)
	require.Equal(t, "mole-", req.Model)
}

func TestGetModelRequest_SystemRedirect_NonMolePrefixNotRedirected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := newDistributorTestContext("/v1/chat/completions", `{"model":"moleapi-gpt-4o"}`)

	req, shouldSelectChannel, err := getModelRequest(c)
	require.NoError(t, err)
	require.True(t, shouldSelectChannel)
	require.Equal(t, "moleapi-gpt-4o", req.Model)
}
