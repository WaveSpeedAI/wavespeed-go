package wavespeed

import (
	"github.com/WaveSpeedAI/wavespeed-go/api"
)

// Client is the WaveSpeed API client.
type Client = api.Client

// Run executes a model and waits for the output.
//
// Args:
//   - model: Model identifier (e.g., "wavespeed-ai/z-image/turbo").
//   - input: Input parameters for the model.
//   - timeout: Maximum time to wait for completion (0 = default).
//   - pollInterval: Interval between status checks in seconds (0 = default).
//   - enableSyncMode: If true, use synchronous mode (single request).
//   - maxRetries: Maximum retries for this request (0 = default).
//
// Returns:
//   - map[string]any containing "outputs" array with model outputs.
//
// Example:
//
//	import "github.com/WaveSpeedAI/wavespeed-go"
//
//	output, err := wavespeed.Run(
//	    "wavespeed-ai/z-image/turbo",
//	    map[string]any{"prompt": "Cat"},
//	    0, 0, false, 0,
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(output["outputs"].([]any)[0])
func Run(model string, input map[string]any, timeout float64, pollInterval float64, enableSyncMode bool, maxRetries int) (map[string]any, error) {
	return api.Run(model, input, timeout, pollInterval, enableSyncMode, maxRetries)
}

// Upload uploads a file to WaveSpeed.
//
// Args:
//   - file: File path string to upload.
//   - timeout: Total API call timeout in seconds (0 = default).
//
// Returns:
//   - URL of the uploaded file.
//
// Example:
//
//	import "github.com/WaveSpeedAI/wavespeed-go"
//
//	url, err := wavespeed.Upload("/path/to/image.png", 0)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(url)
func Upload(file string, timeout float64) (string, error) {
	return api.Upload(file, timeout)
}
