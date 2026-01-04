package api

var defaultClient *Client

func getDefaultClient() *Client {
	if defaultClient == nil {
		defaultClient = NewClient("", "", 0, 0, 0, 0)
	}
	return defaultClient
}

// Run executes a model and waits for the output.
//
// Args:
//   - model: Model identifier (e.g., "wavespeed-ai/flux-dev").
//   - input: Input parameters for the model.
//   - timeout: Maximum time to wait for completion (0 = no timeout).
//   - pollInterval: Interval between status checks in seconds.
//   - enableSyncMode: If true, use synchronous mode (single request).
//   - maxRetries: Maximum retries for this request (overrides default setting).
//
// Returns:
//   - map[string]any containing "outputs" array with model outputs.
//
// Example:
//
//	output := api.Run(
//	    "wavespeed-ai/z-image/turbo",
//	    map[string]any{"prompt": "Cat"},
//	    0, 1.0, false, 0,
//	)
//	fmt.Println(output["outputs"].([]any)[0])
func Run(model string, input map[string]any, timeout float64, pollInterval float64, enableSyncMode bool, maxRetries int) (map[string]any, error) {
	return getDefaultClient().Run(model, input, timeout, pollInterval, enableSyncMode, maxRetries)
}

// Upload uploads a file to WaveSpeed.
//
// Args:
//   - file: File path string to upload.
//   - timeout: Total API call timeout in seconds.
//
// Returns:
//   - URL of the uploaded file.
//
// Example:
//
//	url, err := api.Upload("/path/to/image.png", 0)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(url)
func Upload(file string, timeout float64) (string, error) {
	return getDefaultClient().Upload(file, timeout)
}
