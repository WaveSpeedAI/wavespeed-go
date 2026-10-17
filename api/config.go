package api

import (
	"os"
	"strconv"
)

// APIConfig holds global API client configuration options. Fields set here
// are used as defaults by NewClient and by the package-level Run/Upload
// helpers, mirroring the global config honored by the Python and JS SDKs.
type APIConfig struct {
	// Authentication
	APIKey string

	// API base URL
	BaseURL string

	// Connection timeout in seconds
	ConnectionTimeout float64

	// Overall time to wait for a task, in seconds. 0 (the default) means no
	// limit: Run waits until the task reaches a terminal status. A positive
	// value bounds the wait and each request, like WithTimeout.
	Timeout float64

	// Per-request timeout in seconds for the async submit POST and each result
	// GET (default 120; env WAVESPEED_READ_TIMEOUT). It never limits the
	// overall wait for a task.
	ReadTimeout float64

	// Maximum number of retries for the entire operation (task-level retries)
	MaxRetries int

	// Maximum number of retries for individual HTTP requests (connection errors, timeouts)
	MaxConnectionRetries int

	// Base interval between retries in seconds (actual delay = RetryInterval * attempt)
	RetryInterval float64
}

// API is the global API configuration instance. Changes take effect for
// clients created afterwards via NewClient (and for the package-level
// Run/Upload helpers before their first use).
var API = &APIConfig{
	APIKey:               os.Getenv("WAVESPEED_API_KEY"),
	BaseURL:              "https://api.wavespeed.ai",
	ConnectionTimeout:    10.0,
	Timeout:              0,
	ReadTimeout:          envFloat("WAVESPEED_READ_TIMEOUT", defaultReadTimeout),
	MaxRetries:           0,
	MaxConnectionRetries: 5,
	RetryInterval:        1.0,
}

// defaultReadTimeout is the per-request timeout (seconds) for async submit and
// result requests when none is configured.
const defaultReadTimeout = 120.0

// defaultUploadTimeout is the upload timeout (seconds) when WithUploadTimeout
// is not used. Uploads are not task waits and keep a bounded default.
const defaultUploadTimeout = 36000.0

// envFloat parses a positive float from the environment, else returns def.
func envFloat(name string, def float64) float64 {
	if v, err := strconv.ParseFloat(os.Getenv(name), 64); err == nil && v > 0 {
		return v
	}
	return def
}

// defaultTimeout returns the global overall-wait default in seconds; 0 means
// no limit.
func defaultTimeout() float64 {
	if API != nil && API.Timeout > 0 {
		return API.Timeout
	}
	return 0
}
