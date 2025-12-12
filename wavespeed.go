package wavespeed

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
	"strconv"
	"strings"
	"time"
)

// Client provides methods to run models and upload files.
type Client struct {
	apiKey               string
	baseURL              string
	pollInterval         time.Duration
	waitTimeout          time.Duration
	httpClient           *http.Client
	maxRetries           int
	maxConnectionRetries int
	retryInterval        time.Duration
}

// ClientOptions configures the client at construction time.
type ClientOptions struct {
	BaseURL              string   // API base URL without path (default: https://api.wavespeed.ai)
	APIKey               string   // overrides provided apiKey or env
	PollIntervalSeconds  *float64 // default: 1
	TimeoutSeconds       *float64 // overall wait timeout, default: 36000
	HTTPClient           *http.Client
	MaxRetries           *int     // task-level retries (default: 0)
	MaxConnectionRetries *int     // HTTP connection retries (default: 5)
	RetryInterval        *float64 // base delay between retries in seconds (default: 1)
}

// RunOptions applies to a single Run call.
type RunOptions struct {
	TimeoutSeconds      *float64 // overall wait timeout for this call
	PollIntervalSeconds *float64 // poll interval for this call
	EnableSyncMode      *bool    // if true, use synchronous mode (single request)
	MaxRetries          *int     // maximum retries for this request (overrides client default)
}

// Prediction matches the API response data for a prediction.
type Prediction struct {
	ID     string   `json:"id"`
	Model  string   `json:"model"`
	Status string   `json:"status"`
	Input  any      `json:"input"`
	Outputs []string `json:"outputs"`
	Error  string   `json:"error"`
}

type predictionResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    Prediction  `json:"data"`
}

type uploadResponse struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

// NewClient creates a new WaveSpeed client.
// If apiKey is empty, it reads WAVESPEED_API_KEY from the environment.
func NewClient(apiKey string, opts *ClientOptions) (*Client, error) {
	key := apiKey
	if opts != nil && opts.APIKey != "" {
		key = opts.APIKey
	}
	if key == "" {
		key = os.Getenv("WAVESPEED_API_KEY")
	}
	if key == "" {
		return nil, errors.New("API key is required. Set WAVESPEED_API_KEY or pass apiKey")
	}

	base := "https://api.wavespeed.ai"
	if opts != nil && opts.BaseURL != "" {
		base = opts.BaseURL
	} else if env := os.Getenv("WAVESPEED_BASE_URL"); env != "" {
		base = env
	}
	base = strings.TrimRight(base, "/") + "/api/v3"

	poll := time.Second
	if env := os.Getenv("WAVESPEED_POLL_INTERVAL"); env != "" {
		if v, err := parseFloat(env); err == nil {
			poll = time.Duration(v * float64(time.Second))
		}
	}
	if opts != nil && opts.PollIntervalSeconds != nil {
		poll = time.Duration(*opts.PollIntervalSeconds * float64(time.Second))
	}

	wait := 36000.0
	if env := os.Getenv("WAVESPEED_TIMEOUT"); env != "" {
		if v, err := parseFloat(env); err == nil {
			wait = v
		}
	}
	if opts != nil && opts.TimeoutSeconds != nil {
		wait = *opts.TimeoutSeconds
	}

	maxRetries := 0
	if opts != nil && opts.MaxRetries != nil {
		maxRetries = *opts.MaxRetries
	}

	maxConnRetries := 5
	if opts != nil && opts.MaxConnectionRetries != nil {
		maxConnRetries = *opts.MaxConnectionRetries
	}

	retryInt := 1.0
	if opts != nil && opts.RetryInterval != nil {
		retryInt = *opts.RetryInterval
	}

	client := opts.getHTTPClient()

	return &Client{
		apiKey:               key,
		baseURL:              base,
		pollInterval:         poll,
		waitTimeout:          time.Duration(wait * float64(time.Second)),
		httpClient:           client,
		maxRetries:           maxRetries,
		maxConnectionRetries: maxConnRetries,
		retryInterval:        time.Duration(retryInt * float64(time.Second)),
	}, nil
}

func (o *ClientOptions) getHTTPClient() *http.Client {
	if o != nil && o.HTTPClient != nil {
		return o.HTTPClient
	}
	return &http.Client{Timeout: 120 * time.Second}
}

// Run submits a model and waits for completion.
func (c *Client) Run(modelID string, input map[string]any, opts *RunOptions) (*Prediction, error) {
	return c.runWithContext(context.Background(), modelID, input, opts)
}

func (c *Client) runWithContext(ctx context.Context, modelID string, input map[string]any, opts *RunOptions) (*Prediction, error) {
	if modelID == "" {
		return nil, errors.New("modelID is required")
	}
	reqTimeout := c.waitTimeout
	poll := c.pollInterval
	enableSync := false
	taskRetries := c.maxRetries

	if opts != nil {
		if opts.TimeoutSeconds != nil {
			reqTimeout = time.Duration(*opts.TimeoutSeconds * float64(time.Second))
		}
		if opts.PollIntervalSeconds != nil {
			poll = time.Duration(*opts.PollIntervalSeconds * float64(time.Second))
		}
		if opts.EnableSyncMode != nil {
			enableSync = *opts.EnableSyncMode
		}
		if opts.MaxRetries != nil {
			taskRetries = *opts.MaxRetries
		}
	}

	ctx, cancel := context.WithTimeout(ctx, reqTimeout)
	defer cancel()

	var lastErr error
	for attempt := 0; attempt <= taskRetries; attempt++ {
		pred, err := c.runOnce(ctx, modelID, input, enableSync, poll, reqTimeout)
		if err == nil {
			return pred, nil
		}

		lastErr = err
		if !c.isRetryable(err) || attempt >= taskRetries {
			break
		}

		delay := c.retryInterval * time.Duration(attempt+1)
		fmt.Printf("Task attempt %d/%d failed: %v\n", attempt+1, taskRetries+1, err)
		fmt.Printf("Retrying in %v...\n", delay)
		time.Sleep(delay)
	}

	return nil, lastErr
}

func (c *Client) runOnce(ctx context.Context, modelID string, input map[string]any, enableSync bool, poll time.Duration, reqTimeout time.Duration) (*Prediction, error) {
	pred, err := c.submit(ctx, modelID, input, enableSync)
	if err != nil {
		return nil, err
	}

	// In sync mode, the prediction is already complete
	if enableSync {
		return pred, nil
	}

	// Async mode: poll for completion
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("prediction timed out after %.2fs", reqTimeout.Seconds())
		default:
		}

		pred, err = c.getResult(ctx, pred.ID)
		if err != nil {
			return nil, err
		}
		if pred.Status == "completed" || pred.Status == "failed" {
			return pred, nil
		}
		time.Sleep(poll)
	}
}

func (c *Client) isRetryable(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Retry on timeout, connection errors, and 5xx errors
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "HTTP 5") ||
		strings.Contains(errStr, "HTTP 429")
}

// Upload uploads a local file and returns download_url.
func (c *Client) Upload(filePath string) (string, error) {
	if filePath == "" {
		return "", errors.New("filePath is required")
	}
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", err
	}
	if _, err = io.Copy(part, f); err != nil {
		return "", err
	}
	if err = writer.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", c.baseURL+"/media/upload/binary", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload failed: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var ur uploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&ur); err != nil {
		return "", err
	}
	if ur.Code != 200 {
		return "", fmt.Errorf("upload failed: code %d message %s", ur.Code, ur.Message)
	}
	if url, ok := ur.Data["download_url"]; ok {
		return fmt.Sprint(url), nil
	}
	return "", errors.New("upload failed: download_url missing in response")
}

// submit sends a prediction request and returns the prediction (or just ID for async).
func (c *Client) submit(ctx context.Context, modelID string, input map[string]any, enableSync bool) (*Prediction, error) {
	bodyData := input
	if enableSync {
		// Add enable_sync_mode to the input
		bodyData = make(map[string]any)
		for k, v := range input {
			bodyData[k] = v
		}
		bodyData["enable_sync_mode"] = true
	}

	body, err := json.Marshal(bodyData)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for retry := 0; retry <= c.maxConnectionRetries; retry++ {
		req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/"+modelID, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if retry < c.maxConnectionRetries {
				delay := c.retryInterval * time.Duration(retry+1)
				fmt.Printf("Connection error on attempt %d/%d: %v\n", retry+1, c.maxConnectionRetries+1, err)
				fmt.Printf("Retrying in %v...\n", delay)
				time.Sleep(delay)
				continue
			}
			return nil, fmt.Errorf("failed to submit prediction after %d attempts: %w", c.maxConnectionRetries+1, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("submit failed: HTTP %d: %s", resp.StatusCode, string(b))
		}

		var pr predictionResponse
		if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
			return nil, err
		}
		if pr.Code != 200 {
			return nil, fmt.Errorf("submit failed: code %d message %s", pr.Code, pr.Message)
		}

		// In sync mode, the result is returned directly
		if enableSync {
			return &pr.Data, nil
		}

		// In async mode, just return the prediction with ID
		if pr.Data.ID == "" {
			return nil, errors.New("submit failed: missing prediction id")
		}
		return &pr.Data, nil
	}

	return nil, fmt.Errorf("failed to submit prediction after %d attempts: %w", c.maxConnectionRetries+1, lastErr)
}

// getResult fetches prediction status/result by ID.
func (c *Client) getResult(ctx context.Context, predictionID string) (*Prediction, error) {
	var lastErr error
	for retry := 0; retry <= c.maxConnectionRetries; retry++ {
		req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/predictions/"+predictionID+"/result", nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if retry < c.maxConnectionRetries {
				delay := c.retryInterval * time.Duration(retry+1)
				fmt.Printf("Connection error getting result on attempt %d/%d: %v\n", retry+1, c.maxConnectionRetries+1, err)
				fmt.Printf("Retrying in %v...\n", delay)
				time.Sleep(delay)
				continue
			}
			return nil, fmt.Errorf("failed to get result after %d attempts: %w", c.maxConnectionRetries+1, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("getResult failed: HTTP %d: %s", resp.StatusCode, string(b))
		}

		var pr predictionResponse
		if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
			return nil, err
		}
		if pr.Code != 200 {
			return nil, fmt.Errorf("getResult failed: code %d message %s", pr.Code, pr.Message)
		}
		return &pr.Data, nil
	}

	return nil, fmt.Errorf("failed to get result after %d attempts: %w", c.maxConnectionRetries+1, lastErr)
}

func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

