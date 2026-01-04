package wavespeed

import (
	"github.com/WaveSpeedAI/wavespeed-go/api"
)

// Client is the WaveSpeed API client.
type Client = api.Client

// RunOption configures optional parameters for Run.
type RunOption = api.RunOption

// UploadOption configures optional parameters for Upload.
type UploadOption = api.UploadOption

// Option constructors
var (
	// WithTimeout sets the maximum time to wait for completion.
	WithTimeout = api.WithTimeout
	// WithPollInterval sets the interval between status checks.
	WithPollInterval = api.WithPollInterval
	// WithSyncMode enables or disables synchronous mode.
	WithSyncMode = api.WithSyncMode
	// WithMaxRetries sets the maximum number of task-level retries.
	WithMaxRetries = api.WithMaxRetries
	// WithUploadTimeout sets the timeout for file upload.
	WithUploadTimeout = api.WithUploadTimeout
)

// Run executes a model and waits for the output.
//
// Args:
//   - model: Model identifier (e.g., "wavespeed-ai/z-image/turbo").
//   - input: Input parameters for the model.
//   - opts: Optional parameters (WithTimeout, WithSyncMode, etc.)
//
// Returns:
//   - map[string]any containing "outputs" array with model outputs.
//
// Example:
//
//	import "github.com/WaveSpeedAI/wavespeed-go"
//
//	// Simple usage
//	output, err := wavespeed.Run(
//	    "wavespeed-ai/z-image/turbo",
//	    map[string]any{"prompt": "Cat"},
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(output["outputs"].([]any)[0])
//
//	// With options
//	output, err := wavespeed.Run(
//	    "wavespeed-ai/z-image/turbo",
//	    map[string]any{"prompt": "Cat"},
//	    wavespeed.WithSyncMode(true),
//	    wavespeed.WithTimeout(60),
//	)
func Run(model string, input map[string]any, opts ...RunOption) (map[string]any, error) {
	return api.Run(model, input, opts...)
}

// Upload uploads a file to WaveSpeed.
//
// Args:
//   - file: File path string to upload.
//   - opts: Optional upload options (WithUploadTimeout, etc.)
//
// Returns:
//   - URL of the uploaded file.
//
// Example:
//
//	import "github.com/WaveSpeedAI/wavespeed-go"
//
//	// Simple usage
//	url, err := wavespeed.Upload("/path/to/image.png")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(url)
//
//	// With timeout
//	url, err := wavespeed.Upload("/path/to/image.png", wavespeed.WithUploadTimeout(30))
func Upload(file string, opts ...UploadOption) (string, error) {
	return api.Upload(file, opts...)
}
