<div align="center">
  <a href="https://wavespeed.ai" target="_blank" rel="noopener noreferrer">
    <img src="https://raw.githubusercontent.com/WaveSpeedAI/waverless/main/docs/images/wavespeed-dark-logo.png" alt="WaveSpeedAI logo" width="200"/>
  </a>

  <h1>WaveSpeedAI Go SDK</h1>

  <p>
    <strong>Official Go SDK for the WaveSpeedAI inference platform</strong>
  </p>

  <p>
    <a href="https://wavespeed.ai" target="_blank" rel="noopener noreferrer">🌐 Visit wavespeed.ai</a> •
    <a href="https://wavespeed.ai/docs">📖 Documentation</a> •
    <a href="https://github.com/WaveSpeedAI/wavespeed-go/issues">💬 Issues</a>
  </p>
</div>

---

## Installation

```bash
go get github.com/WaveSpeedAI/wavespeed-go
```

## API Client

Run WaveSpeed AI models with a simple API:

```go
import "github.com/WaveSpeedAI/wavespeed-go"

output, err := wavespeed.Run(
    "wavespeed-ai/z-image/turbo",
    map[string]any{"prompt": "Cat"},
)
if err != nil {
    log.Fatal(err)
}

fmt.Println(output["outputs"].([]any)[0])  // Output URL
```

### Authentication

Set your API key via environment variable (You can get your API key from [https://wavespeed.ai/accesskey](https://wavespeed.ai/accesskey)):

```bash
export WAVESPEED_API_KEY="your-api-key"
```

Or pass it directly:

```go
import "github.com/WaveSpeedAI/wavespeed-go/api"

client := api.NewClient("your-api-key", "", 0, 0, 0, 0)
output, err := client.Run(
    "wavespeed-ai/z-image/turbo",
    map[string]any{"prompt": "Cat"},
)
```

### Options

All parameters are optional. Use option functions to customize behavior:

```go
output, err := wavespeed.Run(
    "wavespeed-ai/z-image/turbo",
    map[string]any{"prompt": "Cat"},
    wavespeed.WithTimeout(60),           // Max wait time in seconds
    wavespeed.WithPollInterval(1.0),     // Status check interval
    wavespeed.WithSyncMode(true),        // Enable synchronous mode
    wavespeed.WithMaxRetries(3),         // Maximum task retries
)
```

### Sync Mode

Use `WithSyncMode(true)` for a single request that waits for the result (no polling).

> **Note:** Not all models support sync mode. Check the model documentation for availability.

```go
output, err := wavespeed.Run(
    "wavespeed-ai/z-image/turbo",
    map[string]any{"prompt": "Cat"},
    wavespeed.WithSyncMode(true),
)
```

### Retry Configuration

Configure retries at the client level:

```go
import "github.com/WaveSpeedAI/wavespeed-go/api"

client := api.NewClient(
    "your-api-key",
    "",       // baseURL (empty = default)
    10.0,     // connectionTimeout in seconds (default: 10.0)
    0,        // maxRetries: Task-level retries (default: 0)
    5,        // maxConnectionRetries: HTTP connection retries (default: 5)
    1.0,      // retryInterval: Base delay between retries in seconds (default: 1.0)
)
```

### Upload Files

Upload images, videos, or audio files:

```go
import "github.com/WaveSpeedAI/wavespeed-go"

// Simple upload
url, err := wavespeed.Upload("/path/to/image.png")
if err != nil {
    log.Fatal(err)
}
fmt.Println(url)

// With timeout
url, err := wavespeed.Upload("/path/to/image.png", wavespeed.WithUploadTimeout(30))
```

## Running Tests

```bash
# Run all tests
go test -v ./...

# Run a single test file
go test -v ./api

# Run a specific test
go test -v -run TestRunSuccess ./api
```

## Environment Variables

### API Client

| Variable | Description |
|----------|-------------|
| `WAVESPEED_API_KEY` | WaveSpeed API key |

## License

MIT
