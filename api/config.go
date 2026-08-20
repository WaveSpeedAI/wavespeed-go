package api

import "os"

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

	// Total API call timeout in seconds
	Timeout float64

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
	Timeout:              36000.0,
	MaxRetries:           0,
	MaxConnectionRetries: 5,
	RetryInterval:        1.0,
}

// defaultTimeout returns the global total-timeout default, falling back to
// 36000 seconds if the global config was zeroed out.
func defaultTimeout() float64 {
	if API != nil && API.Timeout > 0 {
		return API.Timeout
	}
	return 36000.0
}
