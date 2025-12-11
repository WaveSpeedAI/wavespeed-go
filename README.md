# WaveSpeed Go SDK (Preview)

> Simple, Python-like API for running WaveSpeed models from Go. Serverless features are not included in this preview.

## Installation

```bash
go get github.com/WaveSpeedAI/wavespeed-go
```

## Quick Start

### Run a model

```go
package main

import (
	"fmt"
	"log"

	wavespeed "github.com/WaveSpeedAI/wavespeed-go"
)

func main() {
	// API key is read from WAVESPEED_API_KEY if not provided
	client, err := wavespeed.NewClient("")
	if err != nil {
		log.Fatal(err)
	}

	output, err := client.Run("wavespeed-ai/z-image/turbo", map[string]any{
		"prompt": "Cat",
	}, &wavespeed.RunOptions{
		TimeoutSeconds:    36000, // optional, overall wait timeout (seconds)
		PollIntervalSeconds: 1,   // optional, status poll interval (seconds)
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(output.Outputs[0]) // first output URL
}
```

### Upload a file

```go
url, err := client.Upload("/path/to/image.png")
if err != nil {
	log.Fatal(err)
}
fmt.Println(url)
```

## Options

- `TimeoutSeconds` (optional): overall wait timeout for `Run`, default `36000`.
- `PollIntervalSeconds` (optional): status check interval, default `1`.

## Authentication

Set your API key via environment variable:

```bash
export WAVESPEED_API_KEY="your-api-key"
```

Or pass it when creating the client:

```go
client, err := wavespeed.NewClient("your-api-key")
```

## Environment Variables

- `WAVESPEED_API_KEY` — WaveSpeed API key
- `WAVESPEED_BASE_URL` — API base URL without path (default: `https://api.wavespeed.ai`)
- `WAVESPEED_POLL_INTERVAL` — poll interval seconds for `Run` (default: `1`)
- `WAVESPEED_TIMEOUT` — overall wait timeout seconds for `Run` (default: `36000`)

## License

MIT

