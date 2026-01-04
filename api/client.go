package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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
	connectionTimeout    float64
	maxRetries           int
	maxConnectionRetries int
	retryInterval        float64
}

// ClientOptions configures the client at initialization time.
type ClientOptions struct {
	APIKey               string
	BaseURL              string
	ConnectionTimeout    float64
	MaxRetries           int
	MaxConnectionRetries int
	RetryInterval        float64
}

type prediction struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Status  string `json:"status"`
	Input   any    `json:"input"`
	Outputs []any  `json:"outputs"`
	Error   string `json:"error"`
}

type predictionResponse struct {
	Code    int        `json:"code"`
	Message string     `json:"message"`
	Data    prediction `json:"data"`
}

type uploadResponse struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
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
	// Create client with default values
	client := &Client{
		apiKey:               os.Getenv("WAVESPEED_API_KEY"),
		baseURL:              "https://api.wavespeed.ai",
		connectionTimeout:    10.0,
		maxRetries:           0,
		maxConnectionRetries: 5,
		retryInterval:        1.0,
	}

	// Apply user-provided options
	for _, opt := range opts {
		opt(client)
	}

	// Normalize baseURL
	client.baseURL = strings.TrimRight(client.baseURL, "/")

	return client
}

func (c *Client) getHeaders() (map[string]string, error) {
	if c.apiKey == "" {
		return nil, errors.New("API key is required. Set WAVESPEED_API_KEY environment variable or pass api_key to Client()")
	}
	return map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + c.apiKey,
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
		requestTimeout = 36000.0
	}

	connectTimeout := c.connectionTimeout
	if requestTimeout > 0 && connectTimeout > requestTimeout {
		connectTimeout = requestTimeout
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", nil, err
	}

	var lastErr error
	for retry := 0; retry <= c.maxConnectionRetries; retry++ {
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

		client := &http.Client{
			Timeout: time.Duration(connectTimeout * float64(time.Second)),
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			if retry < c.maxConnectionRetries {
				delay := c.retryInterval * float64(retry+1)
				fmt.Printf("Connection error on attempt %d/%d:\n", retry+1, c.maxConnectionRetries+1)
				fmt.Printf("%v\n", err)
				fmt.Printf("Retrying in %.1f seconds...\n", delay)
				time.Sleep(time.Duration(delay * float64(time.Second)))
				continue
			}
			return "", nil, fmt.Errorf("failed to submit prediction after %d attempts: %w", c.maxConnectionRetries+1, lastErr)
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
					"id":      result.Data.ID,
					"status":  result.Data.Status,
					"error":   result.Data.Error,
					"outputs": result.Data.Outputs,
				},
			}, nil
		}

		requestID := result.Data.ID
		if requestID == "" {
			return "", nil, fmt.Errorf("no request ID in response: %v", result)
		}

		return requestID, nil, nil
	}

	return "", nil, fmt.Errorf("failed to submit prediction after %d attempts: %w", c.maxConnectionRetries+1, lastErr)
}

func (c *Client) getResult(requestID string, timeout float64) (map[string]any, error) {
	url := c.baseURL + "/api/v3/predictions/" + requestID + "/result"
	requestTimeout := timeout
	if requestTimeout == 0 {
		requestTimeout = 36000.0
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

		client := &http.Client{
			Timeout: time.Duration(connectTimeout * float64(time.Second)),
		}
		resp, err := client.Do(req)
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

		if status == "failed" {
			errorMsg := "Unknown error"
			if e, ok := data["error"].(string); ok && e != "" {
				errorMsg = e
			}
			return nil, fmt.Errorf("prediction failed (task_id: %s): %s", requestID, errorMsg)
		}

		time.Sleep(time.Duration(pollInterval * float64(time.Second)))
	}
}

func (c *Client) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "http 5") ||
		strings.Contains(errStr, "429")
}

// Run executes a model and waits for the output.
func (c *Client) Run(model string, input map[string]any, opts ...RunOption) (map[string]any, error) {
	// Apply default options
	options := &RunOptions{
		Timeout:        36000.0,
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
					errorMsg := "Unknown error"
					if e, ok := data["error"].(string); ok && e != "" {
						errorMsg = e
					}
					requestIDStr := "unknown"
					if id, ok := data["id"].(string); ok && id != "" {
						requestIDStr = id
					}
					return nil, fmt.Errorf("prediction failed (task_id: %s): %s", requestIDStr, errorMsg)
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

	url := c.baseURL + "/api/v3/media/upload/binary"
	headers := map[string]string{
		"Authorization": "Bearer " + c.apiKey,
	}
	requestTimeout := options.Timeout

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found: %s", file)
	}

	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filepath.Base(file))
	if err != nil {
		return "", err
	}
	if _, err = io.Copy(part, f); err != nil {
		return "", err
	}
	if err = writer.Close(); err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(requestTimeout*float64(time.Second)))
	defer cancel()

	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return "", err
	}
	req = req.WithContext(ctx)

	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{
		Timeout: time.Duration(requestTimeout * float64(time.Second)),
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyText, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to upload file: HTTP %d: %s", resp.StatusCode, string(bodyText))
	}

	var result uploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Code != 200 {
		return "", fmt.Errorf("upload failed: %s", result.Message)
	}

	downloadURL, ok := result.Data["download_url"]
	if !ok {
		return "", errors.New("upload failed: no download_url in response")
	}

	return fmt.Sprint(downloadURL), nil
}
