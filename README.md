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
import wavespeed "github.com/WaveSpeedAI/wavespeed-go"

client, err := wavespeed.NewClient("your-api-key")
if err != nil {
	log.Fatal(err)
}

output, err := client.Run("wavespeed-ai/z-image/turbo", map[string]any{
	"prompt": "Cat",
})
if err != nil {
	log.Fatal(err)
}

fmt.Println(output.Outputs[0])  // Output URL
```

### Authentication

Set your API key via environment variable (You can get your API key from [https://wavespeed.ai/accesskey](https://wavespeed.ai/accesskey)):

```bash
export WAVESPEED_API_KEY="your-api-key"
```

Or pass it directly when creating the client:

```go
client, err := wavespeed.NewClient("your-api-key")
```

### Options

```go
output, err := client.Run("wavespeed-ai/z-image/turbo", map[string]any{
	"prompt": "Cat",
}, &wavespeed.RunOptions{
	TimeoutSeconds:      36000, // Max wait time in seconds (default: 36000)
	PollIntervalSeconds: 1,     // Status check interval (default: 1)
	EnableSyncMode:      false, // Single request mode, no polling (default: false)
})
```

### Sync Mode

Use `EnableSyncMode: true` for a single request that waits for the result (no polling).

> **Note:** Not all models support sync mode. Check the model documentation for availability.

```go
output, err := client.Run("wavespeed-ai/z-image/turbo", map[string]any{
	"prompt": "Cat",
}, &wavespeed.RunOptions{
	EnableSyncMode: true,
})
```

### Retry Configuration

Configure retries at the client level:

```go
client, err := wavespeed.NewClient("your-api-key", &wavespeed.ClientOptions{
	MaxRetries:           0, // Task-level retries (default: 0)
	MaxConnectionRetries: 5, // HTTP connection retries (default: 5)
	RetryInterval:        1, // Base delay between retries in seconds (default: 1)
})
```

### Upload Files

Upload images, videos, or audio files:

```go
url, err := client.Upload("/path/to/image.png")
if err != nil {
	log.Fatal(err)
}
fmt.Println(url)
```

## Environment Variables

### API Client

| Variable | Description |
|----------|-------------|
| `WAVESPEED_API_KEY` | WaveSpeed API key |

## License

MIT
