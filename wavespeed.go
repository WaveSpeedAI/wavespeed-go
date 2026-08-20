package wavespeed

import (
	"github.com/WaveSpeedAI/wavespeed-go/api"
)

// Client is the WaveSpeed API client.
type Client = api.Client

// ClientOption configures a Client created with NewClient.
type ClientOption = api.ClientOption

// SubmissionError indicates that a prediction submission POST failed without
// a definitive response. The task may or may not have been created, so the
// SDK never retries the submission POST automatically.
type SubmissionError = api.SubmissionError

// RunDetail contains detailed information about a task execution.
type RunDetail = api.RunDetail

// RunNoThrowResult is the result of the Client.RunNoThrow method.
type RunNoThrowResult = api.RunNoThrowResult

// RunOption configures optional parameters for Run.
type RunOption = api.RunOption

// UploadOption configures optional parameters for Upload.
type UploadOption = api.UploadOption

// NewClient creates a new WaveSpeed API client with optional configuration.
// It is re-exported from the api subpackage so users never need to import it:
//
//	client := wavespeed.NewClient(wavespeed.WithAPIKey("your-api-key"))
var NewClient = api.NewClient

// Client option constructors
var (
	// WithAPIKey sets the API key for the client.
	WithAPIKey = api.WithAPIKey
	// WithBaseURL sets the base URL for the client.
	WithBaseURL = api.WithBaseURL
	// WithClientName sets the client name reported in the X-Client-Name header.
	WithClientName = api.WithClientName
	// WithConnectionTimeout sets the connection timeout in seconds.
	WithConnectionTimeout = api.WithConnectionTimeout
	// WithClientMaxRetries sets the maximum number of task-level retries.
	WithClientMaxRetries = api.WithClientMaxRetries
	// WithMaxConnectionRetries sets the maximum number of HTTP connection retries.
	WithMaxConnectionRetries = api.WithMaxConnectionRetries
	// WithRetryInterval sets the base interval between retries in seconds.
	WithRetryInterval = api.WithRetryInterval
)

// Run option constructors
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
