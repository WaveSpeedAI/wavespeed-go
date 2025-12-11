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
	apiKey        string
	baseURL       string
	pollInterval  time.Duration
	waitTimeout   time.Duration
	httpClient    *http.Client
}

// ClientOptions configures the client at construction time.
type ClientOptions struct {
	BaseURL             string   // API base URL without path (default: https://api.wavespeed.ai)
	APIKey              string   // overrides provided apiKey or env
	PollIntervalSeconds *float64 // default: 1
	TimeoutSeconds      *float64 // overall wait timeout, default: 36000
	HTTPClient          *http.Client
}

// RunOptions applies to a single Run call.
type RunOptions struct {
	TimeoutSeconds      *float64 // overall wait timeout for this call
	PollIntervalSeconds *float64 // poll interval for this call
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

	client := opts.getHTTPClient()

	return &Client{
		apiKey:       key,
		baseURL:      base,
		pollInterval: poll,
		waitTimeout:  time.Duration(wait * float64(time.Second)),
		httpClient:   client,
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
	if opts != nil {
		if opts.TimeoutSeconds != nil {
			reqTimeout = time.Duration(*opts.TimeoutSeconds * float64(time.Second))
		}
		if opts.PollIntervalSeconds != nil {
			poll = time.Duration(*opts.PollIntervalSeconds * float64(time.Second))
		}
	}

	ctx, cancel := context.WithTimeout(ctx, reqTimeout)
	defer cancel()

	id, err := c.submit(ctx, modelID, input)
	if err != nil {
		return nil, err
	}

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("prediction timed out after %.2fs", reqTimeout.Seconds())
		default:
		}

		pred, err := c.getResult(ctx, id)
		if err != nil {
			return nil, err
		}
		if pred.Status == "completed" || pred.Status == "failed" {
			return pred, nil
		}
		time.Sleep(poll)
	}
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

// submit sends a prediction request and returns the prediction ID.
func (c *Client) submit(ctx context.Context, modelID string, input map[string]any) (string, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/"+modelID, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("submit failed: HTTP %d: %s", resp.StatusCode, string(b))
	}

	var pr predictionResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return "", err
	}
	if pr.Code != 200 {
		return "", fmt.Errorf("submit failed: code %d message %s", pr.Code, pr.Message)
	}
	if pr.Data.ID == "" {
		return "", errors.New("submit failed: missing prediction id")
	}
	return pr.Data.ID, nil
}

// getResult fetches prediction status/result by ID.
func (c *Client) getResult(ctx context.Context, predictionID string) (*Prediction, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/predictions/"+predictionID+"/result", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
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

func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

