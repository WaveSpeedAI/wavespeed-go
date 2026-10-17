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

## Introduction

**WaveSpeedAI** Go Client — Official Go SDK for **WaveSpeedAI** inference platform. This library provides a clean, type-safe, and high-performance API for your applications.

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

client := api.NewClient(api.WithAPIKey("your-api-key"))
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
    wavespeed.WithTimeout(600),          // Max wait time in seconds (default: no limit)
    wavespeed.WithPollInterval(1.0),     // Status check interval
    wavespeed.WithSyncMode(true),        // Best-effort sync result attempt
    wavespeed.WithMaxRetries(0),         // Task re-runs after a reported failure (default: 0)
)
```

### Timeouts

By default `Run` waits for the task to reach a terminal status with no overall
time limit, so long-running tasks (many hours) are fine. Pass
`WithTimeout(seconds)` with a positive value to bound the total wait; `0` or a
negative value means no limit. Each submit/poll request is separately bounded by
a read timeout by request type (30 seconds for result queries, 60 for the async
submit, 120 for uploads), so one hung request fails alone and result polling
retries it. `WithReadTimeout` or the `WAVESPEED_READ_TIMEOUT` environment
variable replaces these defaults with one value for all three. In sync mode the submit request stays open until
the task finishes, so it has no deadline unless you set `WithTimeout`. An
explicit `WithTimeout` also keeps applying to each individual request, as before.

### Sync Mode

Use `WithSyncMode(true)` to ask the API to wait for the result in the initial
request. If the server-side sync wait times out, the SDK returns an error with
the task ID/result URL; the task continues processing and can be queried later.

> **Note:** Not all models support sync mode. Check the model documentation for availability.

```go
output, err := wavespeed.Run(
    "wavespeed-ai/z-image/turbo",
    map[string]any{"prompt": "Cat"},
    wavespeed.WithSyncMode(true),
)
```

### Retry Configuration

The submission POST is sent exactly once and is never retried, because a failed
submission may still have created (and billed) the task. Result polling
tolerates transient HTTP and network errors. There is no overall timeout
requirement: tasks can run for hours, and a client-side wait timeout does not
mean the task failed.

`WithMaxRetries` / `WithClientMaxRetries` (default: 0) only re-run a task that
itself reported failure with a transient HTTP 5xx/429. Each re-run submits a
**new, separately billed task**. Poll-side HTTP errors and wait timeouts never
re-submit.

Configure retries at the client level:

```go
import "github.com/WaveSpeedAI/wavespeed-go/api"

client := api.NewClient(
    api.WithAPIKey("your-api-key"),
    api.WithClientMaxRetries(0),       // Task re-runs after a reported failure; each is a new paid task (default: 0)
    api.WithMaxConnectionRetries(5),   // Result-query GET retries; the submission POST is never retried (default: 5)
    api.WithRetryInterval(1.0),        // Base delay between retries in seconds (default: 1.0)
    api.WithReadTimeout(45),           // Optional per-request override (defaults: 30 result, 60 submit, 120 upload); not the overall wait
)
```

### Upload Files

Upload images, videos, or audio files. Uploads use the same read timeout as
other requests (default 120 seconds for uploads, or the `WithReadTimeout`
override): it limits only how long
the SDK waits for the server to respond, not how long a large file takes to
send. `WithUploadTimeout(seconds)` instead sets one explicit total deadline for
the whole upload.

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

### Getting Task ID and Debug Information

If you need access to the task ID for logging, tracking, or debugging, use `RunNoThrow()` instead of `Run()`. This method returns detailed information and does not return errors:

```go
result := client.RunNoThrow(model, input)

if result.Outputs != nil {
    fmt.Println("Success:", result.Outputs)
    fmt.Println("Task ID:", result.Detail.TaskID)  // For tracking/debugging
} else {
    fmt.Println("Failed:", result.Detail.Error)
    fmt.Println("Task ID:", result.Detail.TaskID)  // Still available on failure
}
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
| `WAVESPEED_CLIENT_NAME` | Overrides the `X-Client-Name` header used for channel attribution (takes precedence over `WithClientName`; default: `wavespeed-go`) |

## License

MIT

---

**[WaveSpeed AI](https://wavespeed.ai/)** — AI image & video generation platform.
Try it in the browser: **[Image generator](https://wavespeed.ai/image-generator)** · **[Video generator](https://wavespeed.ai/video-generator)**
