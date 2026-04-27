package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestResolveAsyncImageTaskResponseCompleted(t *testing.T) {
	var sawAuthorization bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/tasks/task_123" {
			t.Fatalf("unexpected task path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("language") != "en" {
			t.Fatalf("expected language query, got %q", r.URL.RawQuery)
		}
		sawAuthorization = r.Header.Get("Authorization") == "Bearer test-key"
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"data":{"id":"task_123","status":"completed","created":10,"completed":20,"result":{"images":[{"url":["https://example.com/a.png","https://example.com/b.png"],"expires_at":30}]}}}`))
	}))
	defer server.Close()

	imageResp, body, ok, err := ResolveAsyncImageTaskResponse(context.Background(), []byte(`{"code":200,"data":[{"status":"submitted","task_id":"task_123"}]}`), AsyncImageTaskPollOptions{
		BaseURL:    server.URL,
		APIKey:     "test-key",
		HTTPClient: server.Client(),
		Interval:   time.Millisecond,
		Timeout:    time.Second,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected async task response to be detected")
	}
	if !sawAuthorization {
		t.Fatal("expected polling request to include bearer token")
	}
	if imageResp.Created != 20 || len(imageResp.Data) != 2 || imageResp.Data[0].Url != "https://example.com/a.png" {
		t.Fatalf("unexpected image response: %+v", imageResp)
	}
	if !strings.Contains(string(body), `"url":"https://example.com/a.png"`) {
		t.Fatalf("expected OpenAI image body, got %s", string(body))
	}
}

func TestResolveAsyncImageTaskResponsesCombinesCompletedTasks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/tasks/task_a":
			_, _ = w.Write([]byte(`{"code":200,"data":{"id":"task_a","status":"completed","created":10,"completed":20,"result":{"images":[{"url":["https://example.com/a.png"],"expires_at":30}]}}}`))
		case "/v1/tasks/task_b":
			_, _ = w.Write([]byte(`{"code":200,"data":{"id":"task_b","status":"completed","created":11,"completed":21,"result":{"images":[{"url":["https://example.com/b.png"],"expires_at":31}]}}}`))
		default:
			t.Fatalf("unexpected task path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	imageResp, body, ok, err := ResolveAsyncImageTaskResponses(context.Background(), [][]byte{
		[]byte(`{"code":200,"data":[{"status":"submitted","task_id":"task_a"}]}`),
		[]byte(`{"code":200,"data":[{"status":"submitted","task_id":"task_b"}]}`),
	}, AsyncImageTaskPollOptions{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		Interval:   time.Millisecond,
		Timeout:    time.Second,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected async task responses to be detected")
	}
	if len(imageResp.Data) != 2 || imageResp.Data[0].Url != "https://example.com/a.png" || imageResp.Data[1].Url != "https://example.com/b.png" {
		t.Fatalf("unexpected image response: %+v", imageResp)
	}
	if !strings.Contains(string(body), `"url":"https://example.com/a.png"`) || !strings.Contains(string(body), `"url":"https://example.com/b.png"`) {
		t.Fatalf("expected combined image body, got %s", string(body))
	}
}

func TestResolveAsyncImageTaskResponseFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"data":{"id":"task_123","status":"failed","error":{"message":"blocked by safety filter","type":"invalid_request_error","code":"content_policy"}}}`))
	}))
	defer server.Close()

	_, _, ok, err := ResolveAsyncImageTaskResponse(context.Background(), []byte(`{"code":200,"data":[{"status":"submitted","task_id":"task_123"}]}`), AsyncImageTaskPollOptions{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		Interval:   time.Millisecond,
		Timeout:    time.Second,
	})
	if !ok {
		t.Fatal("expected async task response to be detected")
	}
	var taskErr *AsyncImageTaskError
	if !errors.As(err, &taskErr) {
		t.Fatalf("expected AsyncImageTaskError, got %T: %v", err, err)
	}
	if taskErr.OpenAIError.Message != "blocked by safety filter" {
		t.Fatalf("unexpected task error: %+v", taskErr.OpenAIError)
	}
}

func TestResolveAsyncImageTaskResponseIgnoresNormalImageBody(t *testing.T) {
	_, _, ok, err := ResolveAsyncImageTaskResponse(context.Background(), []byte(`{"created":123,"data":[{"url":"https://example.com/a.png"}]}`), AsyncImageTaskPollOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("normal image body should not be detected as async task")
	}
}
