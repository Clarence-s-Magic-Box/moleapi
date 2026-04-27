package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
)

const (
	defaultAsyncImageTaskPollInterval = 5 * time.Second
	defaultAsyncImageTaskPollTimeout  = 300 * time.Second
)

type AsyncImageTaskPollOptions struct {
	BaseURL      string
	APIKey       string
	Proxy        string
	HTTPClient   *http.Client
	InitialDelay time.Duration
	Interval     time.Duration
	Timeout      time.Duration
}

type AsyncImageTaskError struct {
	StatusCode  int
	OpenAIError types.OpenAIError
}

func (e *AsyncImageTaskError) Error() string {
	if e == nil {
		return ""
	}
	if e.OpenAIError.Message != "" {
		return e.OpenAIError.Message
	}
	return "async image task failed"
}

type asyncImageTaskStatusResponse struct {
	Code int `json:"code"`
	Data struct {
		ID            string `json:"id"`
		Status        string `json:"status"`
		Progress      int    `json:"progress"`
		Created       int64  `json:"created"`
		Completed     int64  `json:"completed"`
		EstimatedTime int64  `json:"estimated_time"`
		ActualTime    int64  `json:"actual_time"`
		Result        struct {
			Images []struct {
				URL       []string `json:"url"`
				ExpiresAt int64    `json:"expires_at"`
			} `json:"images"`
		} `json:"result"`
		Error *types.OpenAIError `json:"error,omitempty"`
	} `json:"data"`
	Error *types.OpenAIError `json:"error,omitempty"`
}

func ResolveAsyncImageTaskResponse(ctx context.Context, submitBody []byte, opts AsyncImageTaskPollOptions) (*dto.ImageResponse, []byte, bool, error) {
	taskID, ok, err := ExtractAsyncImageTaskID(submitBody)
	if err != nil || !ok {
		return nil, nil, ok, err
	}

	statusURL, err := asyncImageTaskStatusURL(opts.BaseURL, taskID)
	if err != nil {
		return nil, nil, true, err
	}

	client, err := asyncImageTaskHTTPClient(opts)
	if err != nil {
		return nil, nil, true, err
	}

	interval := opts.Interval
	if interval <= 0 {
		interval = defaultAsyncImageTaskPollInterval
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultAsyncImageTaskPollTimeout
	}
	deadline := time.Now().Add(timeout)

	if err := sleepAsyncImageTaskPoll(ctx, opts.InitialDelay); err != nil {
		return nil, nil, true, err
	}

	for {
		imageResp, rawBody, done, err := fetchAsyncImageTaskStatus(ctx, client, statusURL, opts.APIKey)
		if done || err != nil {
			if err != nil {
				return nil, nil, true, err
			}
			imageResp.Metadata = rawBody
			body, err := common.Marshal(imageResp)
			if err != nil {
				return nil, nil, true, err
			}
			return imageResp, body, true, nil
		}

		if time.Now().Add(interval).After(deadline) {
			return nil, nil, true, &AsyncImageTaskError{
				StatusCode: http.StatusGatewayTimeout,
				OpenAIError: types.OpenAIError{
					Message: fmt.Sprintf("async image task %s timed out", taskID),
					Type:    "upstream_task_timeout",
					Code:    "task_timeout",
				},
			}
		}

		if err := sleepAsyncImageTaskPoll(ctx, interval); err != nil {
			return nil, nil, true, err
		}
	}
}

func ExtractAsyncImageTaskID(body []byte) (string, bool, error) {
	if len(body) == 0 {
		return "", false, nil
	}
	var raw struct {
		Code  int                `json:"code"`
		Data  json.RawMessage    `json:"data"`
		Error *types.OpenAIError `json:"error,omitempty"`
	}
	if err := common.Unmarshal(body, &raw); err != nil {
		return "", false, nil
	}
	if raw.Error != nil && raw.Error.Message != "" {
		return "", false, nil
	}
	if len(raw.Data) == 0 {
		return "", false, nil
	}

	var items []struct {
		Status string `json:"status"`
		TaskID string `json:"task_id"`
		ID     string `json:"id"`
	}
	if err := common.Unmarshal(raw.Data, &items); err == nil {
		for _, item := range items {
			taskID := strings.TrimSpace(item.TaskID)
			if taskID == "" {
				taskID = strings.TrimSpace(item.ID)
			}
			if taskID != "" && isAsyncImageSubmittedStatus(item.Status) {
				return taskID, true, nil
			}
		}
	}

	var item struct {
		Status string `json:"status"`
		TaskID string `json:"task_id"`
		ID     string `json:"id"`
	}
	if err := common.Unmarshal(raw.Data, &item); err == nil {
		taskID := strings.TrimSpace(item.TaskID)
		if taskID == "" {
			taskID = strings.TrimSpace(item.ID)
		}
		if taskID != "" && isAsyncImageSubmittedStatus(item.Status) {
			return taskID, true, nil
		}
	}

	return "", false, nil
}

func asyncImageTaskHTTPClient(opts AsyncImageTaskPollOptions) (*http.Client, error) {
	if opts.HTTPClient != nil {
		return opts.HTTPClient, nil
	}
	client, err := GetHttpClientWithProxy(opts.Proxy)
	if client == nil && err == nil {
		client = http.DefaultClient
	}
	return client, err
}

func asyncImageTaskStatusURL(baseURL string, taskID string) (string, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return "", fmt.Errorf("async image task base url is empty")
	}
	baseURL = strings.TrimRight(baseURL, "/")
	baseURL = strings.TrimSuffix(baseURL, "/images/generations")
	if !strings.HasSuffix(baseURL, "/v1") {
		baseURL += "/v1"
	}
	u, err := url.Parse(baseURL + "/tasks/" + url.PathEscape(taskID))
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("language", "en")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func fetchAsyncImageTaskStatus(ctx context.Context, client *http.Client, statusURL string, apiKey string) (*dto.ImageResponse, []byte, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
	if err != nil {
		return nil, nil, false, err
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, nil, false, err
		}
		return nil, nil, false, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, false, err
	}

	if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
		return nil, body, false, asyncImageTaskHTTPError(resp.StatusCode, body)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, body, false, nil
	}

	var statusResp asyncImageTaskStatusResponse
	if err := common.Unmarshal(body, &statusResp); err != nil {
		return nil, body, false, err
	}
	if statusResp.Error != nil && statusResp.Error.Message != "" {
		return nil, body, false, &AsyncImageTaskError{StatusCode: resp.StatusCode, OpenAIError: *statusResp.Error}
	}

	status := strings.ToLower(strings.TrimSpace(statusResp.Data.Status))
	switch status {
	case "completed", "succeeded", "success":
		imageResp, err := asyncImageTaskToImageResponse(&statusResp, body)
		if err != nil {
			return nil, body, false, err
		}
		return imageResp, body, true, nil
	case "failed", "failure", "cancelled", "canceled":
		openAIError := types.OpenAIError{
			Message: "async image task failed",
			Type:    "upstream_task_failed",
			Code:    "task_failed",
		}
		if statusResp.Data.Error != nil && statusResp.Data.Error.Message != "" {
			openAIError = *statusResp.Data.Error
		}
		return nil, body, false, &AsyncImageTaskError{StatusCode: http.StatusBadGateway, OpenAIError: openAIError}
	case "submitted", "pending", "processing", "in_progress", "running", "queued", "":
		return nil, body, false, nil
	default:
		return nil, body, false, nil
	}
}

func asyncImageTaskHTTPError(statusCode int, body []byte) error {
	var errResp struct {
		Error *types.OpenAIError `json:"error,omitempty"`
	}
	if common.Unmarshal(body, &errResp) == nil && errResp.Error != nil && errResp.Error.Message != "" {
		return &AsyncImageTaskError{StatusCode: statusCode, OpenAIError: *errResp.Error}
	}
	return &AsyncImageTaskError{
		StatusCode: statusCode,
		OpenAIError: types.OpenAIError{
			Message: fmt.Sprintf("async image task polling failed with status %d", statusCode),
			Type:    "upstream_task_error",
			Code:    statusCode,
		},
	}
}

func asyncImageTaskToImageResponse(statusResp *asyncImageTaskStatusResponse, rawBody []byte) (*dto.ImageResponse, error) {
	if statusResp == nil {
		return nil, fmt.Errorf("async image task response is nil")
	}
	imageResp := &dto.ImageResponse{
		Created:  statusResp.Data.Completed,
		Metadata: rawBody,
	}
	if imageResp.Created == 0 {
		imageResp.Created = statusResp.Data.Created
	}
	if imageResp.Created == 0 {
		imageResp.Created = time.Now().Unix()
	}
	for _, image := range statusResp.Data.Result.Images {
		for _, rawURL := range image.URL {
			rawURL = strings.TrimSpace(rawURL)
			if rawURL == "" {
				continue
			}
			imageResp.Data = append(imageResp.Data, dto.ImageData{Url: rawURL})
		}
	}
	if len(imageResp.Data) == 0 {
		return nil, fmt.Errorf("async image task completed without image urls")
	}
	return imageResp, nil
}

func isAsyncImageSubmittedStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "submitted", "pending", "processing", "in_progress", "running", "queued":
		return true
	default:
		return false
	}
}

func sleepAsyncImageTaskPoll(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
