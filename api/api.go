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
//   - opts: Optional parameters (WithTimeout, WithSyncMode, etc.)
//
// Returns:
//   - map[string]any containing "outputs" array with model outputs.
//
// Example:
//
//	// Simple usage
//	output, _ := api.Run("wavespeed-ai/z-image/turbo", map[string]any{"prompt": "Cat"})
//	fmt.Println(output["outputs"].([]any)[0])
//
//	// With options
//	output, _ := api.Run(
//	    "wavespeed-ai/z-image/turbo",
//	    map[string]any{"prompt": "Cat"},
//	    api.WithSyncMode(true),
//	    api.WithTimeout(60),
//	)
func Run(model string, input map[string]any, opts ...RunOption) (map[string]any, error) {
	return getDefaultClient().Run(model, input, opts...)
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
//	// Simple usage
//	url, err := api.Upload("/path/to/image.png")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(url)
//
//	// With timeout
//	url, err := api.Upload("/path/to/image.png", api.WithUploadTimeout(30))
func Upload(file string, opts ...UploadOption) (string, error) {
	return getDefaultClient().Upload(file, opts...)
}
