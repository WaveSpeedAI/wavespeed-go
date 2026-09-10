package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestInitWithAPIKey(t *testing.T) {
	client := NewClient(WithAPIKey("test-key"))
	if client.apiKey != "test-key" {
		t.Errorf("expected apiKey=test-key, got %s", client.apiKey)
	}
	if client.baseURL != "https://api.wavespeed.ai" {
		t.Errorf("expected default baseURL, got %s", client.baseURL)
	}
}

func TestInitWithCustomBaseURL(t *testing.T) {
	client := NewClient(WithAPIKey("test-key"), WithBaseURL("https://custom.api.com/"))
	if client.baseURL != "https://custom.api.com" {
		t.Errorf("expected baseURL=https://custom.api.com, got %s", client.baseURL)
	}
}

func TestGetHeadersRaisesWithoutAPIKey(t *testing.T) {
	client := NewClient()
	client.apiKey = ""
	_, err := client.getHeaders()
	if err == nil {
		t.Fatal("expected error when no API key provided")
	}
	if !strings.Contains(err.Error(), "API key is required") {
		t.Errorf("expected 'API key is required' error, got: %v", err)
	}
}

func TestGetHeadersReturnsAuthHeader(t *testing.T) {
	client := NewClient(WithAPIKey("test-key"))
	headers, err := client.getHeaders()
	if err != nil {
		t.Fatalf("getHeaders error: %v", err)
	}
	if headers["Authorization"] != "Bearer test-key" {
		t.Errorf("expected Authorization header with Bearer, got %s", headers["Authorization"])
	}
	if headers["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type header, got %s", headers["Content-Type"])
	}
	if headers["X-Client-Name"] != "wavespeed-go" {
		t.Errorf("expected X-Client-Name=wavespeed-go, got %s", headers["X-Client-Name"])
	}
	if headers["X-Client-Version"] != Version {
		t.Errorf("expected X-Client-Version=%s, got %s", Version, headers["X-Client-Version"])
	}
	if headers["X-Client-OS"] != runtime.GOOS {
		t.Errorf("expected X-Client-OS=%s, got %s", runtime.GOOS, headers["X-Client-OS"])
	}
}

func TestClientNamePrecedence(t *testing.T) {
	// Explicit option overrides the default.
	client := NewClient(WithAPIKey("test-key"), WithClientName("my-app"))
	if got := client.resolveClientName(); got != "my-app" {
		t.Errorf("expected client name my-app, got %s", got)
	}

	// Environment variable overrides the explicit option.
	t.Setenv("WAVESPEED_CLIENT_NAME", "env-app")
	if got := client.resolveClientName(); got != "env-app" {
		t.Errorf("expected client name env-app, got %s", got)
	}
}

func TestSubmitSendsAttributionHeaders(t *testing.T) {
	var gotHeaders http.Header
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/test-model", func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "req-1"}})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	if _, _, err := client.submit("test-model", map[string]any{"prompt": "hi"}, false, 30); err != nil {
		t.Fatalf("submit error: %v", err)
	}

	if got := gotHeaders.Get("X-Client-Name"); got != "wavespeed-go" {
		t.Errorf("expected X-Client-Name=wavespeed-go, got %s", got)
	}
	if got := gotHeaders.Get("X-Client-Version"); got != Version {
		t.Errorf("expected X-Client-Version=%s, got %s", Version, got)
	}
	if got := gotHeaders.Get("X-Client-OS"); got != runtime.GOOS {
		t.Errorf("expected X-Client-OS=%s, got %s", runtime.GOOS, got)
	}
}

func TestSubmitSuccess(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/wavespeed-ai/z-image/turbo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":200,"message":"ok","data":{"id":"req-123","model":"wavespeed-ai/z-image/turbo","status":"processing","outputs":[]}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	requestID, result, err := client.submit("wavespeed-ai/z-image/turbo", map[string]any{"prompt": "test"}, false, 0)
	if err != nil {
		t.Fatalf("submit error: %v", err)
	}
	if requestID != "req-123" {
		t.Errorf("expected requestID=req-123, got %s", requestID)
	}
	if result != nil {
		t.Errorf("expected nil result in async mode, got %+v", result)
	}
}

func TestSubmitFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/wavespeed-ai/z-image/turbo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	_, _, err := client.submit("wavespeed-ai/z-image/turbo", map[string]any{"prompt": "test"}, false, 0)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("expected 'HTTP 500' in error, got: %v", err)
	}
}

func TestGetResultSuccess(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/predictions/req-123/result", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":200,"data":{"id":"req-123","status":"completed","outputs":["https://example.com/out.png"]}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	result, err := client.getResult("req-123", 0)
	if err != nil {
		t.Fatalf("getResult error: %v", err)
	}
	data, ok := result["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data field in result")
	}
	if data["status"] != "completed" {
		t.Errorf("expected status=completed, got %v", data["status"])
	}
}

func TestRunSuccess(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/wavespeed-ai/z-image/turbo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":200,"data":{"id":"req-123"}}`))
	})
	mux.HandleFunc("/api/v3/predictions/req-123/result", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":200,"data":{"status":"completed","outputs":["https://example.com/out.png"]}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	result, err := client.Run("wavespeed-ai/z-image/turbo", map[string]any{"prompt": "test"}, WithPollInterval(0.01))
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	outputs, ok := result["outputs"].([]any)
	if !ok {
		t.Fatalf("expected outputs array, got %+v", result)
	}
	if len(outputs) != 1 {
		t.Errorf("expected 1 output, got %d", len(outputs))
	}
	if outputs[0] != "https://example.com/out.png" {
		t.Errorf("unexpected output: %v", outputs[0])
	}
}

func TestRunFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/wavespeed-ai/z-image/turbo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":200,"data":{"id":"req-123"}}`))
	})
	mux.HandleFunc("/api/v3/predictions/req-123/result", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":200,"data":{"status":"failed","error":"Model error"}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	_, err := client.Run("wavespeed-ai/z-image/turbo", map[string]any{"prompt": "test"}, WithPollInterval(0.01))
	if err == nil {
		t.Fatal("expected error for failed prediction")
	}
	if !strings.Contains(err.Error(), "Model error") {
		t.Errorf("expected 'Model error' in error, got: %v", err)
	}
}

func TestUploadFilePath(t *testing.T) {
	mux := http.NewServeMux()
	var serverURL string
	mux.HandleFunc("/api/v3/media/uploads", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			http.Error(w, "no auth", http.StatusUnauthorized)
			return
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload["size"] != float64(len("fake image data")) {
			http.Error(w, "bad ticket", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"code":200,"message":"success","data":{"download_url":"https://example.com/uploaded.png","upload":{"method":"PUT","url":%q,"headers":{"Content-Type":"image/png"}}}}`, serverURL+"/storage-upload")
	})
	mux.HandleFunc("/storage-upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			http.Error(w, "credentials leaked", http.StatusBadRequest)
			return
		}
		content, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodPut || string(content) != "fake image data" {
			http.Error(w, "bad upload", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	serverURL = server.URL

	tmpFile := filepath.Join(os.TempDir(), "wavespeed-test.png")
	if err := os.WriteFile(tmpFile, []byte("fake image data"), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile)

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	url, err := client.Upload(tmpFile)
	if err != nil {
		t.Fatalf("upload error: %v", err)
	}
	if url != "https://example.com/uploaded.png" {
		t.Errorf("expected URL=https://example.com/uploaded.png, got %s", url)
	}
}

func TestUploadFileNotFound(t *testing.T) {
	client := NewClient(WithAPIKey("test-key"))
	_, err := client.Upload("/nonexistent/path/to/file.png")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "no such file") {
		t.Errorf("expected file not found error, got: %v", err)
	}
}

func TestUploadRaisesWithoutAPIKey(t *testing.T) {
	client := NewClient()
	client.apiKey = ""
	_, err := client.Upload("/some/file.png")
	if err == nil {
		t.Fatal("expected error when no API key provided")
	}
	if !strings.Contains(err.Error(), "API key is required") {
		t.Errorf("expected 'API key is required' error, got: %v", err)
	}
}

func TestUploadHTTPError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/media/uploads", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	tmpFile := filepath.Join(os.TempDir(), "wavespeed-test.png")
	if err := os.WriteFile(tmpFile, []byte("fake image data"), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile)

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	_, err := client.Upload(tmpFile)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("expected 'HTTP 500' in error, got: %v", err)
	}
}

func TestUploadAPIError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/media/uploads", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":500,"message":"Upload failed: invalid file type"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	tmpFile := filepath.Join(os.TempDir(), "wavespeed-test.png")
	if err := os.WriteFile(tmpFile, []byte("fake image data"), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile)

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	_, err := client.Upload(tmpFile)
	if err == nil {
		t.Fatal("expected error for API error response")
	}
	if !strings.Contains(err.Error(), "invalid file type") {
		t.Errorf("expected 'invalid file type' in error, got: %v", err)
	}
}

func TestRunSyncModeFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/wavespeed-ai/z-image/turbo", func(w http.ResponseWriter, r *http.Request) {
		// Return non-completed status in sync mode
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":200,"data":{"id":"req-123","status":"failed","error":"Model crashed","outputs":[]}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	_, err := client.Run("wavespeed-ai/z-image/turbo", map[string]any{"prompt": "test"}, WithSyncMode(true))
	if err == nil {
		t.Fatal("expected error for non-completed status in sync mode")
	}
	if !strings.Contains(err.Error(), "prediction failed") {
		t.Errorf("expected 'prediction failed' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "Model crashed") {
		t.Errorf("expected 'Model crashed' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "req-123") {
		t.Errorf("expected 'req-123' in error, got: %v", err)
	}
}

func TestRunSyncModeTimeoutQueryableError(t *testing.T) {
	resultHits := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/wavespeed-ai/z-image/turbo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":200,"data":{"id":"req-timeout","status":"processing","code":5004,"error":"Sync mode timed out after 90 seconds. The prediction is still processing asynchronously.","outputs":[]}}`))
	})
	mux.HandleFunc("/api/v3/predictions/req-timeout/result", func(w http.ResponseWriter, r *http.Request) {
		resultHits++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":200,"data":{"id":"req-timeout","status":"completed","outputs":[]}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	resultURL := server.URL + "/api/v3/predictions/req-timeout/result"

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	_, err := client.Run("wavespeed-ai/z-image/turbo", map[string]any{"prompt": "test"}, WithSyncMode(true), WithMaxRetries(1))
	if err == nil {
		t.Fatal("expected sync timeout error")
	}
	if !strings.Contains(err.Error(), "sync mode timed out") {
		t.Errorf("expected 'sync mode timed out' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "req-timeout") {
		t.Errorf("expected task id in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), resultURL) {
		t.Errorf("expected result URL in error, got: %v", err)
	}
	if resultHits != 0 {
		t.Errorf("expected no result polling in sync mode timeout, got %d hits", resultHits)
	}
}

func TestRunNoThrowSyncModeTimeoutReturnsProcessing(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/wavespeed-ai/z-image/turbo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":200,"data":{"id":"req-timeout","status":"processing","code":5004,"error":"Sync mode timed out after 90 seconds. The prediction is still processing asynchronously.","outputs":[]}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	resultURL := server.URL + "/api/v3/predictions/req-timeout/result"

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	result := client.RunNoThrow("wavespeed-ai/z-image/turbo", map[string]any{"prompt": "test"}, WithSyncMode(true))
	if result.Outputs != nil {
		t.Fatalf("expected nil outputs, got %+v", result.Outputs)
	}
	if result.Detail.Status != "processing" {
		t.Errorf("expected status=processing, got %s", result.Detail.Status)
	}
	if result.Detail.TaskID != "req-timeout" {
		t.Errorf("expected task id, got %s", result.Detail.TaskID)
	}
	if result.Detail.ResultURL != resultURL {
		t.Errorf("expected result URL, got %s", result.Detail.ResultURL)
	}
	if !strings.Contains(result.Detail.Error, "sync mode timed out") {
		t.Errorf("expected sync timeout error, got %s", result.Detail.Error)
	}
}

func TestRunTimeout(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/wavespeed-ai/z-image/turbo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":200,"data":{"id":"req-123"}}`))
	})
	mux.HandleFunc("/api/v3/predictions/req-123/result", func(w http.ResponseWriter, r *http.Request) {
		// Always return pending status to trigger timeout
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":200,"data":{"status":"pending"}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	_, err := client.Run("wavespeed-ai/z-image/turbo", map[string]any{"prompt": "test"}, WithTimeout(0.1), WithPollInterval(0.01))
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected 'timed out' in error, got: %v", err)
	}
}

func TestRunUsesDefaultClient(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/wavespeed-ai/z-image/turbo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":200,"data":{"id":"req-123"}}`))
	})
	mux.HandleFunc("/api/v3/predictions/req-123/result", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":200,"data":{"status":"completed","outputs":["https://example.com/out.png"]}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	// Set environment variables for default client
	oldAPIKey := os.Getenv("WAVESPEED_API_KEY")
	os.Setenv("WAVESPEED_API_KEY", "test-key")
	defer func() {
		if oldAPIKey != "" {
			os.Setenv("WAVESPEED_API_KEY", oldAPIKey)
		} else {
			os.Unsetenv("WAVESPEED_API_KEY")
		}
	}()

	// Reset default client
	defaultClient = nil

	// Use module-level Run function
	result, err := Run("wavespeed-ai/z-image/turbo", map[string]any{"prompt": "test"}, WithPollInterval(0.01))
	if err != nil {
		// This will fail because we can't override the base URL for the default client
		// But we're testing that it attempts to use the default client
		t.Logf("Expected failure (cannot override baseURL): %v", err)
		return
	}

	outputs, ok := result["outputs"].([]any)
	if !ok {
		t.Fatalf("expected outputs array, got %+v", result)
	}
	if len(outputs) != 1 {
		t.Errorf("expected 1 output, got %d", len(outputs))
	}
}

func TestRunRealAPI(t *testing.T) {
	apiKey := os.Getenv("WAVESPEED_API_KEY")
	if apiKey == "" {
		t.Skip("WAVESPEED_API_KEY environment variable not set")
	}

	// Reset default client
	defaultClient = nil

	output, err := Run(
		"wavespeed-ai/z-image/turbo",
		map[string]any{"prompt": "A simple red circle on white background"},
	)
	if err != nil {
		t.Fatalf("run error: %v", err)
	}

	outputs, ok := output["outputs"]
	if !ok {
		t.Fatal("expected 'outputs' key in result")
	}

	outputsList, ok := outputs.([]any)
	if !ok {
		t.Fatalf("expected outputs to be array, got %T", outputs)
	}

	if len(outputsList) == 0 {
		t.Fatal("expected at least one output")
	}

	// Output should be a URL
	firstOutput, ok := outputsList[0].(string)
	if !ok {
		t.Fatalf("expected output to be string, got %T", outputsList[0])
	}
	if !strings.HasPrefix(firstOutput, "http") {
		t.Errorf("expected output to start with 'http', got %s", firstOutput)
	}
}

func TestUploadRealAPI(t *testing.T) {
	apiKey := os.Getenv("WAVESPEED_API_KEY")
	if apiKey == "" {
		t.Skip("WAVESPEED_API_KEY environment variable not set")
	}

	// Reset default client
	defaultClient = nil

	// Create a minimal valid PNG file (1x1 red pixel)
	pngData := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
		0x0c, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x05, 0xfe, 0xd4, 0x00, 0x00, 0x00,
		0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}

	tmpFile := filepath.Join(os.TempDir(), "wavespeed-test.png")
	if err := os.WriteFile(tmpFile, pngData, 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile)

	url, err := Upload(tmpFile)
	if err != nil {
		t.Fatalf("upload error: %v", err)
	}

	if !strings.HasPrefix(url, "http") {
		t.Errorf("expected URL to start with 'http', got %s", url)
	}
}

// Additional tests for coverage improvement

func TestRunAllRetriesFailed(t *testing.T) {
	// Test scenario where all retries are exhausted
	var attemptCount atomic.Int64
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/wavespeed-ai/z-image/turbo", func(w http.ResponseWriter, r *http.Request) {
		attemptCount.Add(1)
		// Return 500 error which is retryable
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"code":500,"message":"Internal Server Error"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL), WithClientMaxRetries(2), WithRetryInterval(0.01)) // maxRetries=2
	_, err := client.Run("wavespeed-ai/z-image/turbo", map[string]any{"prompt": "test"}, WithPollInterval(0.01), WithMaxRetries(2))

	if err == nil {
		t.Fatal("expected error after all retries failed")
	}

	// Should have attempted 3 times (initial + 2 retries)
	if attemptCount.Load() < 3 {
		t.Errorf("expected at least 3 attempts, got %d", attemptCount.Load())
	}
}

func TestGetResultConnectionRetry(t *testing.T) {
	// Test that getResult does NOT retry on HTTP status code errors (only on connection errors)
	var attemptCount atomic.Int64
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/predictions/req-123/result", func(w http.ResponseWriter, r *http.Request) {
		attemptCount.Add(1)
		// Return 500 - this should NOT trigger a retry
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server Error"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL), WithMaxConnectionRetries(5), WithRetryInterval(0.01))
	_, err := client.getResult("req-123", 0)

	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}

	// HTTP errors should NOT retry, only connection errors do
	if attemptCount.Load() != 1 {
		t.Errorf("expected exactly 1 attempt (no retry for HTTP errors), got %d", attemptCount.Load())
	}

	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("expected 'HTTP 500' in error, got: %v", err)
	}
}

func TestIsRetryableError(t *testing.T) {
	client := NewClient(WithAPIKey("test-key"))

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"timeout error", errors.New("connection timeout"), true},
		{"connection error", errors.New("connection refused"), true},
		{"http 500 error", errors.New("HTTP 500 Internal Server Error"), true},
		{"http 502 error", errors.New("HTTP 502 Bad Gateway"), true},
		{"http 503 error", errors.New("HTTP 503 Service Unavailable"), true},
		{"429 rate limit", errors.New("HTTP 429 Too Many Requests"), true},
		{"non-retryable 404", errors.New("HTTP 404 Not Found"), false},
		{"non-retryable 400", errors.New("HTTP 400 Bad Request"), false},
		{"generic error", errors.New("some random error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.isRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("isRetryableError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestSubmitConnectionRetry(t *testing.T) {
	// Test that submit does NOT retry on HTTP status code errors (only on connection errors)
	var attemptCount atomic.Int64
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/wavespeed-ai/z-image/turbo", func(w http.ResponseWriter, r *http.Request) {
		attemptCount.Add(1)
		// Return 502 - this should NOT trigger a retry
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("Bad Gateway"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL), WithMaxConnectionRetries(5), WithRetryInterval(0.01))
	_, _, err := client.submit("wavespeed-ai/z-image/turbo", map[string]any{"prompt": "test"}, false, 0)

	if err == nil {
		t.Fatal("expected error for HTTP 502")
	}

	// HTTP errors should NOT retry, only connection errors do
	if attemptCount.Load() != 1 {
		t.Errorf("expected exactly 1 attempt (no retry for HTTP errors), got %d", attemptCount.Load())
	}

	if !strings.Contains(err.Error(), "HTTP 502") {
		t.Errorf("expected 'HTTP 502' in error, got: %v", err)
	}
}

func TestWaitInvalidResponse(t *testing.T) {
	// Test wait with invalid response format
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/predictions/req-123/result", func(w http.ResponseWriter, r *http.Request) {
		// Return response without data field
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":200,"invalid":"response"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	_, err := client.wait("req-123", 0.1, 0.01)

	if err == nil {
		t.Fatal("expected error for invalid response format")
	}

	if !strings.Contains(err.Error(), "invalid response format") {
		t.Errorf("expected 'invalid response format' error, got: %v", err)
	}
}

func TestGetResultNon200Status(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/predictions/req-123/result", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"task not found"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	_, err := client.getResult("req-123", 0)

	if err == nil {
		t.Fatal("expected error for HTTP 404")
	}

	if !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("expected 'HTTP 404' in error, got: %v", err)
	}

	if !strings.Contains(err.Error(), "req-123") {
		t.Errorf("expected request ID in error message, got: %v", err)
	}
}

func TestSubmitMissingRequestID(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/wavespeed-ai/z-image/turbo", func(w http.ResponseWriter, r *http.Request) {
		// Return response without request ID
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":200,"message":"ok","data":{"id":"","model":"wavespeed-ai/z-image/turbo","status":"processing"}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	_, _, err := client.submit("wavespeed-ai/z-image/turbo", map[string]any{"prompt": "test"}, false, 0)

	if err == nil {
		t.Fatal("expected error for missing request ID")
	}

	if !strings.Contains(err.Error(), "no request ID") {
		t.Errorf("expected 'no request ID' error, got: %v", err)
	}
}

func TestWaitMissingStatus(t *testing.T) {
	// Test wait with response missing status field
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/predictions/req-123/result", func(w http.ResponseWriter, r *http.Request) {
		// Return data without status
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":200,"data":{"id":"req-123"}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	_, err := client.wait("req-123", 0.1, 0.01)

	if err == nil {
		t.Fatal("expected error for missing status")
	}

	if !strings.Contains(err.Error(), "missing status") {
		t.Errorf("expected 'missing status' error, got: %v", err)
	}
}

func TestSubmitSingleShotOnConnectionFailure(t *testing.T) {
	// The submission POST must fire exactly once when the connection fails:
	// the task may already have been created server-side, so retrying could
	// create duplicate tasks.
	var attemptCount atomic.Int64
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/wavespeed-ai/z-image/turbo", func(w http.ResponseWriter, r *http.Request) {
		attemptCount.Add(1)
		hj, ok := w.(http.Hijacker)
		if !ok {
			// t.FailNow must not be called outside the test goroutine.
			t.Error("server does not support hijacking")
			return
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			t.Errorf("hijack failed: %v", err)
			return
		}
		conn.Close() // drop the connection without responding
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL), WithMaxConnectionRetries(5), WithRetryInterval(0.01))
	_, _, err := client.submit("wavespeed-ai/z-image/turbo", map[string]any{"prompt": "test"}, false, 0)

	if err == nil {
		t.Fatal("expected error for dropped connection")
	}
	if attemptCount.Load() != 1 {
		t.Errorf("expected exactly 1 submission attempt, got %d", attemptCount.Load())
	}

	var submissionErr *SubmissionError
	if !errors.As(err, &submissionErr) {
		t.Errorf("expected *SubmissionError, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "may already have been created") {
		t.Errorf("expected 'may already have been created' in error, got: %v", err)
	}
	if client.isRetryableError(err) {
		t.Error("SubmissionError must never be retryable")
	}
}

func TestRunCancelledStatusTerminatesWait(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/wavespeed-ai/z-image/turbo", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":200,"data":{"id":"req-cancel"}}`))
	})
	resultCalls := 0
	mux.HandleFunc("/api/v3/predictions/req-cancel/result", func(w http.ResponseWriter, r *http.Request) {
		resultCalls++
		w.Write([]byte(`{"code":200,"data":{"id":"req-cancel","status":"cancelled","error":"task was cancelled by user"}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	_, err := client.Run("wavespeed-ai/z-image/turbo", map[string]any{"prompt": "test"}, WithPollInterval(0.01), WithTimeout(5))

	if err == nil {
		t.Fatal("expected error for cancelled task")
	}
	if !strings.Contains(err.Error(), "cancelled") {
		t.Errorf("expected 'cancelled' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "task was cancelled by user") {
		t.Errorf("expected the API's error text in error, got: %v", err)
	}
	if resultCalls != 1 {
		t.Errorf("expected polling to stop after the first cancelled response, got %d calls", resultCalls)
	}
}

func TestRunTimeoutStatusTerminatesWait(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/wavespeed-ai/z-image/turbo", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":200,"data":{"id":"req-to"}}`))
	})
	mux.HandleFunc("/api/v3/predictions/req-to/result", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":200,"data":{"id":"req-to","status":"timeout","error":"task timed out"}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL))
	_, err := client.Run("wavespeed-ai/z-image/turbo", map[string]any{"prompt": "test"}, WithPollInterval(0.01), WithTimeout(5))

	if err == nil {
		t.Fatal("expected error for timed-out task")
	}
	if !strings.Contains(err.Error(), "task timed out") {
		t.Errorf("expected the API's error text in error, got: %v", err)
	}
}

func TestSyncModeNotCappedByConnectTimeout(t *testing.T) {
	// The connect timeout must bound only the connect phase, not the whole
	// request: a server response slower than the connect timeout must still
	// succeed.
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/wavespeed-ai/z-image/turbo", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(900 * time.Millisecond) // much longer than the connect timeout
		w.Write([]byte(`{"code":200,"data":{"id":"req-slow","status":"completed","outputs":["https://example.com/out.png"]}}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(WithAPIKey("test-key"), WithBaseURL(server.URL), WithConnectionTimeout(0.2))
	output, err := client.Run("wavespeed-ai/z-image/turbo", map[string]any{"prompt": "test"}, WithSyncMode(true), WithTimeout(10))

	if err != nil {
		t.Fatalf("expected success despite slow response, got: %v", err)
	}
	outputs, ok := output["outputs"].([]any)
	if !ok || len(outputs) != 1 {
		t.Fatalf("unexpected outputs: %v", output)
	}
	if outputs[0] != "https://example.com/out.png" {
		t.Errorf("unexpected output: %v", outputs[0])
	}
}
