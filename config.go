package wavespeed

import "os"

// APIConfig holds API client configuration options.
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

// API is the global API configuration instance.
var API = &APIConfig{
	APIKey:               os.Getenv("WAVESPEED_API_KEY"),
	BaseURL:              "https://api.wavespeed.ai",
	ConnectionTimeout:    10.0,
	Timeout:              36000.0,
	MaxRetries:           0,
	MaxConnectionRetries: 5,
	RetryInterval:        1.0,
}
