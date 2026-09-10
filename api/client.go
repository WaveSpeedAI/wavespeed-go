package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// SubmissionError indicates that a prediction submission POST failed without
// a definitive response. Because the task may already have been created
// server-side, the SDK never retries the submission POST automatically.
// It mirrors the Python SDK's _SubmissionError.
type SubmissionError struct {
	Err error
}

// Error implements the error interface.
func (e *SubmissionError) Error() string {
	return fmt.Sprintf(
		"prediction submission did not return a response; the task may already have been created, so the SDK will not retry the POST automatically: %v",
		e.Err,
	)
}

// Unwrap returns the underlying transport error.
func (e *SubmissionError) Unwrap() error { return e.Err }

// ClientOption is a function that configures a Client.
type ClientOption func(*Client)

// WithAPIKey sets the API key for the client.
func WithAPIKey(apiKey string) ClientOption {
	return func(c *Client) {
		c.apiKey = apiKey
	}
}

// WithBaseURL sets the base URL for the client.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithClientName sets the client name reported in the X-Client-Name header
// for channel attribution. The WAVESPEED_CLIENT_NAME environment variable
// takes precedence over this option.
func WithClientName(clientName string) ClientOption {
	return func(c *Client) {
		c.clientName = clientName
	}
}

// WithConnectionTimeout sets the connection timeout in seconds.
func WithConnectionTimeout(timeout float64) ClientOption {
	return func(c *Client) {
		c.connectionTimeout = timeout
	}
}

// WithClientMaxRetries sets the maximum number of task-level retries.
func WithClientMaxRetries(maxRetries int) ClientOption {
	return func(c *Client) {
		c.maxRetries = maxRetries
	}
}

// WithMaxConnectionRetries sets the maximum number of HTTP connection retries.
func WithMaxConnectionRetries(maxRetries int) ClientOption {
	return func(c *Client) {
		c.maxConnectionRetries = maxRetries
	}
}

// WithRetryInterval sets the base interval between retries in seconds.
func WithRetryInterval(interval float64) ClientOption {
	return func(c *Client) {
		c.retryInterval = interval
	}
}

// RunOption is a function that configures RunOptions.
type RunOption func(*RunOptions)

// RunOptions contains optional parameters for Run.
type RunOptions struct {
	Timeout        float64
	PollInterval   float64
	EnableSyncMode bool
	MaxRetries     int
}

// WithTimeout sets the maximum time to wait for completion.
func WithTimeout(timeout float64) RunOption {
	return func(o *RunOptions) {
		o.Timeout = timeout
	}
}

// WithPollInterval sets the interval between status checks.
func WithPollInterval(interval float64) RunOption {
	return func(o *RunOptions) {
		o.PollInterval = interval
	}
}

// WithSyncMode enables or disables synchronous mode.
func WithSyncMode(enable bool) RunOption {
	return func(o *RunOptions) {
		o.EnableSyncMode = enable
	}
}

// WithMaxRetries sets the maximum number of task-level retries.
func WithMaxRetries(retries int) RunOption {
	return func(o *RunOptions) {
		o.MaxRetries = retries
	}
}

// UploadOption is a function that configures UploadOptions.
type UploadOption func(*UploadOptions)

// UploadOptions contains optional parameters for Upload.
type UploadOptions struct {
	Timeout float64
}

// WithUploadTimeout sets the timeout for file upload.
func WithUploadTimeout(timeout float64) UploadOption {
	return func(o *UploadOptions) {
		o.Timeout = timeout
	}
}

// Client is the WaveSpeed API client.
type Client struct {
	apiKey               string
	baseURL              string
	clientName           string
	connectionTimeout    float64
	maxRetries           int
	maxConnectionRetries int
	retryInterval        float64
}

// ClientOptions configures the client at initialization time.
type ClientOptions struct {
	APIKey               string
	BaseURL              string
	ClientName           string
	ConnectionTimeout    float64
	MaxRetries           int
	MaxConnectionRetries int
	RetryInterval        float64
}

type prediction struct {
	ID        string            `json:"id"`
	Model     string            `json:"model"`
	Status    string            `json:"status"`
	Outputs   []any             `json:"outputs"`
	Error     string            `json:"error"`
	Code      int               `json:"code"`
	CreatedAt string            `json:"created_at"`
	URLs      map[string]string `json:"urls"`
}

type predictionResponse struct {
	Code    int        `json:"code"`
	Message string     `json:"message"`
	Data    prediction `json:"data"`
}

type uploadResponse struct {
	Code    int        `json:"code"`
	Message string     `json:"message"`
	Data    uploadData `json:"data"`
}

type uploadData struct {
	DownloadURL string            `json:"download_url"`
	Upload      uploadInstruction `json:"upload"`
}

type uploadInstruction struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

// NewClient creates a new WaveSpeed API client with optional configuration.
//
// All parameters are optional and can be configured using functional options.
// If not specified, the following defaults are used:
//   - apiKey: from WAVESPEED_API_KEY environment variable
//   - baseURL: "https://api.wavespeed.ai"
//   - connectionTimeout: 10.0 seconds
//   - maxRetries: 0 (no task-level retries)
//   - maxConnectionRetries: 5
//   - retryInterval: 1.0 second
//
// Example:
//
//	// With defaults (API key from environment)
//	client := api.NewClient()
//
//	// With custom API key
//	client := api.NewClient(api.WithAPIKey("your-api-key"))
//
//	// With multiple options
//	client := api.NewClient(
//	    api.WithAPIKey("your-api-key"),
//	    api.WithClientMaxRetries(3),
//	    api.WithRetryInterval(2.0),
//	)
func NewClient(opts ...ClientOption) *Client {
	// Create client with defaults from the global API config
	client := &Client{
		apiKey:               API.APIKey,
		baseURL:              API.BaseURL,
		connectionTimeout:    API.ConnectionTimeout,
		maxRetries:           API.MaxRetries,
		maxConnectionRetries: API.MaxConnectionRetries,
		retryInterval:        API.RetryInterval,
	}
	if client.apiKey == "" {
		client.apiKey = os.Getenv("WAVESPEED_API_KEY")
	}
	if client.baseURL == "" {
		client.baseURL = "https://api.wavespeed.ai"
	}

	// Apply user-provided options
	for _, opt := range opts {
		opt(client)
	}

	// Normalize baseURL
	client.baseURL = strings.TrimRight(client.baseURL, "/")

	return client
}

// defaultClientName is the X-Client-Name value used when neither the
// WAVESPEED_CLIENT_NAME environment variable nor WithClientName is set.
const defaultClientName = "wavespeed-go"

// resolveClientName returns the value for the X-Client-Name header.
// Precedence: WAVESPEED_CLIENT_NAME environment variable > WithClientName option > default.
func (c *Client) resolveClientName() string {
	if name := os.Getenv("WAVESPEED_CLIENT_NAME"); name != "" {
		return name
	}
	if c.clientName != "" {
		return c.clientName
	}
	return defaultClientName
}

func (c *Client) getHeaders() (map[string]string, error) {
	if c.apiKey == "" {
		return nil, errors.New("API key is required. Set WAVESPEED_API_KEY environment variable or pass api_key to Client()")
	}
	return map[string]string{
		"Content-Type":     "application/json",
		"Authorization":    "Bearer " + c.apiKey,
		"X-Client-Name":    c.resolveClientName(),
		"X-Client-Version": Version,
		"X-Client-OS":      runtime.GOOS,
	}, nil
}

func (c *Client) submit(model string, input map[string]any, enableSyncMode bool, timeout float64) (string, map[string]any, error) {
	url := c.baseURL + "/api/v3/" + model
	body := make(map[string]any)
	if input != nil {
		for k, v := range input {
			body[k] = v
		}
	}
	if enableSyncMode {
		body["enable_sync_mode"] = true
	}

	requestTimeout := timeout
	if requestTimeout == 0 {
		requestTimeout = defaultTimeout()
	}

	connectTimeout := c.connectionTimeout
	if requestTimeout > 0 && connectTimeout > requestTimeout {
		connectTimeout = requestTimeout
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", nil, err
	}

	// The submission POST is deliberately single-shot: if it fails without a
	// definitive response, the task may already have been created server-side,
	// so retrying could create duplicate (billed) tasks.
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(requestTimeout*float64(time.Second)))
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", nil, err
	}

	headers, err := c.getHeaders()
	if err != nil {
		return "", nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := newHTTPClient(connectTimeout).Do(req)
	if err != nil {
		return "", nil, &SubmissionError{Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyText, _ := io.ReadAll(resp.Body)
		return "", nil, fmt.Errorf("failed to submit prediction: HTTP %d: %s", resp.StatusCode, string(bodyText))
	}

	var result predictionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", nil, err
	}

	if enableSyncMode {
		return "", map[string]any{
			"data": map[string]any{
				"id":         result.Data.ID,
				"status":     result.Data.Status,
				"error":      result.Data.Error,
				"outputs":    result.Data.Outputs,
				"code":       result.Data.Code,
				"created_at": result.Data.CreatedAt,
				"urls":       result.Data.URLs,
			},
		}, nil
	}

	requestID := result.Data.ID
	if requestID == "" {
		return "", nil, fmt.Errorf("no request ID in response: %v", result)
	}

	return requestID, nil, nil
}

// newHTTPClient returns an HTTP client whose connect phase (dial + TLS
// handshake) is bounded by connectTimeout, while the total request duration is
// governed solely by the request context deadline. This mirrors the Python
// SDK's (connect, read) timeout tuple: the connect timeout must never cap the
// whole request.
func newHTTPClient(connectTimeout float64) *http.Client {
	d := time.Duration(connectTimeout * float64(time.Second))
	if d <= 0 {
		d = 10 * time.Second
	}
	return &http.Client{
		Transport: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			DialContext:         (&net.Dialer{Timeout: d}).DialContext,
			TLSHandshakeTimeout: d,
		},
	}
}

func (c *Client) getResult(requestID string, timeout float64) (map[string]any, error) {
	url := c.baseURL + "/api/v3/predictions/" + requestID + "/result"
	requestTimeout := timeout
	if requestTimeout == 0 {
		requestTimeout = defaultTimeout()
	}

	connectTimeout := c.connectionTimeout
	if requestTimeout > 0 && connectTimeout > requestTimeout {
		connectTimeout = requestTimeout
	}

	var lastErr error
	for retry := 0; retry <= c.maxConnectionRetries; retry++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(requestTimeout*float64(time.Second)))
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}

		headers, err := c.getHeaders()
		if err != nil {
			return nil, err
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := newHTTPClient(connectTimeout).Do(req)
		if err != nil {
			lastErr = err
			if retry < c.maxConnectionRetries {
				delay := c.retryInterval * float64(retry+1)
				fmt.Printf("Connection error getting result on attempt %d/%d:\n", retry+1, c.maxConnectionRetries+1)
				fmt.Printf("%v\n", err)
				fmt.Printf("Retrying in %.1f seconds...\n", delay)
				time.Sleep(time.Duration(delay * float64(time.Second)))
				continue
			}
			return nil, fmt.Errorf("failed to get result for task %s after %d attempts: %w", requestID, c.maxConnectionRetries+1, lastErr)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			bodyText, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("failed to get result for task %s: HTTP %d: %s", requestID, resp.StatusCode, string(bodyText))
		}

		var result map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}

		return result, nil
	}

	return nil, fmt.Errorf("failed to get result for task %s after %d attempts: %w", requestID, c.maxConnectionRetries+1, lastErr)
}

func (c *Client) wait(requestID string, timeout float64, pollInterval float64) (map[string]any, error) {
	startTime := time.Now()

	for {
		if timeout > 0 {
			elapsed := time.Since(startTime).Seconds()
			if elapsed >= timeout {
				return nil, fmt.Errorf("prediction timed out after %.0f seconds (task_id: %s)", timeout, requestID)
			}
		}

		result, err := c.getResult(requestID, timeout)
		if err != nil {
			return nil, err
		}

		data, ok := result["data"].(map[string]any)
		if !ok {
			return nil, errors.New("invalid response format")
		}

		status, ok := data["status"].(string)
		if !ok {
			return nil, errors.New("missing status in response")
		}

		if status == "completed" {
			outputs, ok := data["outputs"]
			if !ok {
				outputs = []any{}
			}
			return map[string]any{"outputs": outputs}, nil
		}

		// "failed", "cancelled", "timeout" and "deleted" are all terminal: the task will
		// never complete, so polling further would loop forever.
		if status == "failed" || status == "cancelled" || status == "timeout" || status == "deleted" {
			errorMsg := "Unknown error"
			if e, ok := data["error"].(string); ok && e != "" {
				errorMsg = e
			}
			return nil, fmt.Errorf("prediction %s (task_id: %s): %s", status, requestID, errorMsg)
		}

		time.Sleep(time.Duration(pollInterval * float64(time.Second)))
	}
}

func (c *Client) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// A failed submission POST must never be retried: the task may already
	// have been created server-side.
	var submissionErr *SubmissionError
	if errors.As(err, &submissionErr) {
		return false
	}

	errStr := strings.ToLower(err.Error())
	if strings.Contains(errStr, "sync mode timed out") {
		return false
	}
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "http 5") ||
		strings.Contains(errStr, "429")
}

func syncResultURL(baseURL string, data map[string]any) string {
	requestID, ok := data["id"].(string)
	if !ok || requestID == "" {
		return ""
	}
	return strings.TrimRight(baseURL, "/") + "/api/v3/predictions/" + requestID + "/result"
}

func syncResultCode(data map[string]any) int {
	switch code := data["code"].(type) {
	case int:
		return code
	case int64:
		return int(code)
	case float64:
		return int(code)
	}
	return 0
}

func isSyncTimeoutData(data map[string]any) bool {
	errorMsg, _ := data["error"].(string)
	status, _ := data["status"].(string)
	return syncResultCode(data) == 5004 ||
		(status == "processing" && strings.Contains(errorMsg, "Sync mode timed out"))
}

func syncModeError(baseURL string, data map[string]any) error {
	errorMsg := "Unknown error"
	if e, ok := data["error"].(string); ok && e != "" {
		errorMsg = e
	}

	requestID := "unknown"
	if id, ok := data["id"].(string); ok && id != "" {
		requestID = id
	}

	if isSyncTimeoutData(data) {
		message := fmt.Sprintf("sync mode timed out (task_id: %s): %s", requestID, errorMsg)
		if resultURL := syncResultURL(baseURL, data); resultURL != "" && !strings.Contains(message, resultURL) {
			message += " Query the result later at: " + resultURL
		}
		return errors.New(message)
	}

	return fmt.Errorf("prediction failed (task_id: %s): %s", requestID, errorMsg)
}

// Run executes a model and waits for the output.
func (c *Client) Run(model string, input map[string]any, opts ...RunOption) (map[string]any, error) {
	// Apply default options
	options := &RunOptions{
		Timeout:        defaultTimeout(),
		PollInterval:   1.0,
		EnableSyncMode: false,
		MaxRetries:     c.maxRetries,
	}

	// Apply user-provided options
	for _, opt := range opts {
		opt(options)
	}

	timeout := options.Timeout
	pollInterval := options.PollInterval
	enableSyncMode := options.EnableSyncMode
	taskRetries := options.MaxRetries

	var lastError error

	for attempt := 0; attempt <= taskRetries; attempt++ {
		requestID, syncResult, err := c.submit(model, input, enableSyncMode, timeout)
		if err == nil {
			if enableSyncMode {
				// In sync mode, extract outputs from the result
				data, ok := syncResult["data"].(map[string]any)
				if !ok {
					return map[string]any{"outputs": []any{}}, nil
				}

				status, _ := data["status"].(string)
				if status != "completed" {
					return nil, syncModeError(c.baseURL, data)
				}

				outputs, ok := data["outputs"]
				if !ok {
					outputs = []any{}
				}
				return map[string]any{"outputs": outputs}, nil
			}

			return c.wait(requestID, timeout, pollInterval)
		}

		lastError = err
		isRetryable := c.isRetryableError(err)

		if !isRetryable || attempt >= taskRetries {
			return nil, err
		}

		delay := c.retryInterval * float64(attempt+1)
		fmt.Printf("Task attempt %d/%d failed: %v\n", attempt+1, taskRetries+1, err)
		fmt.Printf("Retrying in %.1f seconds...\n", delay)
		time.Sleep(time.Duration(delay * float64(time.Second)))
	}

	if lastError != nil {
		return nil, lastError
	}
	return nil, fmt.Errorf("all %d attempts failed", taskRetries+1)
}

// RunDetail contains detailed information about a task execution.
type RunDetail struct {
	TaskID    string `json:"taskId"`
	Status    string `json:"status"`
	Model     string `json:"model"`
	Error     string `json:"error,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
	ResultURL string `json:"resultUrl,omitempty"`
}

// RunNoThrowResult is the result of RunNoThrow method.
type RunNoThrowResult struct {
	Outputs []any     `json:"outputs"`
	Detail  RunDetail `json:"detail"`
}

// RunNoThrow executes a model and waits for the output (no-throw version).
//
// This method is similar to Run() but does not return errors for task failures.
// Instead, it returns a result object with outputs (nil on failure) and detail information.
// The detail object always contains the taskId, which is useful for debugging and tracking.
//
// Example:
//
//	result := client.RunNoThrow("wavespeed-ai/z-image/turbo", map[string]any{"prompt": "Cat"})
//
//	if result.Outputs != nil {
//	    fmt.Println("Success:", result.Outputs)
//	    fmt.Println("Task ID:", result.Detail.TaskID)
//	} else {
//	    fmt.Println("Failed:", result.Detail.Error)
//	    fmt.Println("Task ID:", result.Detail.TaskID)
//	}
func (c *Client) RunNoThrow(model string, input map[string]any, opts ...RunOption) *RunNoThrowResult {
	// Apply default options
	options := &RunOptions{
		Timeout:        defaultTimeout(),
		PollInterval:   1.0,
		EnableSyncMode: false,
		MaxRetries:     c.maxRetries,
	}

	// Apply user-provided options
	for _, opt := range opts {
		opt(options)
	}

	timeout := options.Timeout
	pollInterval := options.PollInterval
	enableSyncMode := options.EnableSyncMode
	taskRetries := options.MaxRetries

	for attempt := 0; attempt <= taskRetries; attempt++ {
		requestID, syncResult, err := c.submit(model, input, enableSyncMode, timeout)
		if err == nil {
			if enableSyncMode {
				// In sync mode, extract outputs from the result
				data, ok := syncResult["data"].(map[string]any)
				if !ok {
					return &RunNoThrowResult{
						Outputs: nil,
						Detail: RunDetail{
							TaskID: "unknown",
							Status: "failed",
							Model:  model,
							Error:  "Invalid response format",
						},
					}
				}

				status, _ := data["status"].(string)
				taskID, _ := data["id"].(string)
				if taskID == "" {
					taskID = "unknown"
				}

				if status != "completed" {
					errorMsg := "Unknown error"
					if e, ok := data["error"].(string); ok && e != "" {
						errorMsg = e
					}
					createdAt, _ := data["created_at"].(string)
					resultURL := syncResultURL(c.baseURL, data)
					detailStatus := "failed"
					if isSyncTimeoutData(data) {
						detailStatus = "processing"
						errorMsg = syncModeError(c.baseURL, data).Error()
					}
					return &RunNoThrowResult{
						Outputs: nil,
						Detail: RunDetail{
							TaskID:    taskID,
							Status:    detailStatus,
							Model:     model,
							Error:     errorMsg,
							CreatedAt: createdAt,
							ResultURL: resultURL,
						},
					}
				}

				outputs, ok := data["outputs"].([]any)
				if !ok {
					outputs = []any{}
				}
				createdAt, _ := data["created_at"].(string)
				return &RunNoThrowResult{
					Outputs: outputs,
					Detail: RunDetail{
						TaskID:    taskID,
						Status:    "completed",
						Model:     model,
						CreatedAt: createdAt,
					},
				}
			}

			// Async mode
			result, err := c.wait(requestID, timeout, pollInterval)
			if err == nil {
				outputs, ok := result["outputs"].([]any)
				if !ok {
					outputs = []any{}
				}
				return &RunNoThrowResult{
					Outputs: outputs,
					Detail: RunDetail{
						TaskID: requestID,
						Status: "completed",
						Model:  model,
					},
				}
			}

			// Wait failed, but we have taskID
			return &RunNoThrowResult{
				Outputs: nil,
				Detail: RunDetail{
					TaskID: requestID,
					Status: "failed",
					Model:  model,
					Error:  err.Error(),
				},
			}
		}

		// Submit failed
		isRetryable := c.isRetryableError(err)

		if !isRetryable || attempt >= taskRetries {
			// Try to extract taskID from error message
			taskID := "unknown"
			errStr := err.Error()
			if idx := strings.Index(errStr, "task_id: "); idx != -1 {
				start := idx + 9
				end := start
				for end < len(errStr) && (errStr[end] != ')' && errStr[end] != ' ' && errStr[end] != '\n') {
					end++
				}
				if end > start {
					taskID = errStr[start:end]
				}
			}

			return &RunNoThrowResult{
				Outputs: nil,
				Detail: RunDetail{
					TaskID: taskID,
					Status: "failed",
					Model:  model,
					Error:  err.Error(),
				},
			}
		}

		delay := c.retryInterval * float64(attempt+1)
		fmt.Printf("Task attempt %d/%d failed: %v\n", attempt+1, taskRetries+1, err)
		fmt.Printf("Retrying in %.1f seconds...\n", delay)
		time.Sleep(time.Duration(delay * float64(time.Second)))
	}

	// Should not reach here
	return &RunNoThrowResult{
		Outputs: nil,
		Detail: RunDetail{
			TaskID: "unknown",
			Status: "failed",
			Model:  model,
			Error:  fmt.Sprintf("All %d attempts failed", taskRetries+1),
		},
	}
}

// Upload uploads a file to WaveSpeed.
func (c *Client) Upload(file string, opts ...UploadOption) (string, error) {
	if c.apiKey == "" {
		return "", errors.New("API key is required. Set WAVESPEED_API_KEY environment variable or pass api_key to Client()")
	}

	// Apply default options
	options := &UploadOptions{
		Timeout: 36000.0,
	}

	// Apply user-provided options
	for _, opt := range opts {
		opt(options)
	}

	url := c.baseURL + "/api/v3/media/uploads"
	requestTimeout := options.Timeout

	fileInfo, err := os.Stat(file)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("file not found: %s", file)
	}
	if err != nil {
		return "", err
	}

	payload := map[string]any{
		"filename": filepath.Base(file),
		"size":     fileInfo.Size(),
	}
	if contentType := mime.TypeByExtension(filepath.Ext(file)); contentType != "" {
		payload["content_type"] = contentType
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(requestTimeout*float64(time.Second)))
	defer cancel()

	ticketReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	ticketHeaders, err := c.getHeaders()
	if err != nil {
		return "", err
	}
	for k, v := range ticketHeaders {
		ticketReq.Header.Set(k, v)
	}

	client := &http.Client{Timeout: time.Duration(requestTimeout * float64(time.Second))}
	resp, err := client.Do(ticketReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyText, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create upload: HTTP %d: %s", resp.StatusCode, string(bodyText))
	}

	var result uploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Code != 200 {
		return "", fmt.Errorf("upload failed: %s", result.Message)
	}
	if result.Data.DownloadURL == "" || result.Data.Upload.URL == "" {
		return "", errors.New("upload failed: no download_url in response")
	}

	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()

	method := result.Data.Upload.Method
	if method == "" {
		method = http.MethodPut
	}
	uploadReq, err := http.NewRequestWithContext(ctx, method, result.Data.Upload.URL, f)
	if err != nil {
		return "", err
	}
	uploadReq.ContentLength = fileInfo.Size()
	for key, value := range result.Data.Upload.Headers {
		uploadReq.Header.Set(key, value)
	}
	uploadResp, err := client.Do(uploadReq)
	if err != nil {
		return "", err
	}
	defer uploadResp.Body.Close()
	if uploadResp.StatusCode < 200 || uploadResp.StatusCode >= 300 {
		bodyText, _ := io.ReadAll(uploadResp.Body)
		return "", fmt.Errorf("failed to upload file: HTTP %d: %s", uploadResp.StatusCode, string(bodyText))
	}

	return result.Data.DownloadURL, nil
}
