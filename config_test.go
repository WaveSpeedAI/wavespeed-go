package wavespeed

import (
	"testing"
)

func TestAPIConfigHasExpectedAttributes(t *testing.T) {
	if API == nil {
		t.Fatal("API config should not be nil")
	}

	if API.BaseURL != "https://api.wavespeed.ai" {
		t.Errorf("Expected BaseURL to be 'https://api.wavespeed.ai', got '%s'", API.BaseURL)
	}

	if API.ConnectionTimeout != 10.0 {
		t.Errorf("Expected ConnectionTimeout to be 10.0, got %f", API.ConnectionTimeout)
	}

	if API.Timeout != 36000.0 {
		t.Errorf("Expected Timeout to be 36000.0, got %f", API.Timeout)
	}

	if API.MaxRetries != 0 {
		t.Errorf("Expected MaxRetries to be 0, got %d", API.MaxRetries)
	}

	if API.MaxConnectionRetries != 5 {
		t.Errorf("Expected MaxConnectionRetries to be 5, got %d", API.MaxConnectionRetries)
	}

	if API.RetryInterval != 1.0 {
		t.Errorf("Expected RetryInterval to be 1.0, got %f", API.RetryInterval)
	}
}
