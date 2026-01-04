# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

WaveSpeed Go SDK - Official Go SDK for WaveSpeedAI inference platform.

## Commands

### Testing
```bash
# Run all tests
go test -v ./...

# Run a single test file
go test -v ./api

# Run a specific test
go test -v -run TestRunSuccess ./api
```

### Building
```bash
# Build the package
go build ./...
```

### Development
```bash
# Format code
go fmt ./...

# Run linter
go vet ./...

# Update dependencies
go mod tidy
```

## Architecture

### API Client (`api/`)

Entry point: `import "github.com/WaveSpeedAI/wavespeed-go"`

The SDK provides a simple client for running models:

```go
output, err := wavespeed.Run("model-id", map[string]any{"input": "data"}, 0, 0, false, 0)
outputs := output["outputs"].([]any)
```

Key modules in `api/`:
- `client.go` - Client implementation
- `api.go` - Global Run and Upload functions

### Key Types

- `Client` - Main client struct with configuration
- `api.NewClient(apiKey, baseURL, connectionTimeout, maxRetries, maxConnectionRetries, retryInterval)` - Create new client

### Features

- **Sync Mode**: Single request that waits for result
- **Retry Logic**: Configurable task-level and connection-level retries
- **Timeout Control**: Per-request and overall timeouts
- **File Upload**: Direct file upload to WaveSpeed storage

### Configuration

Client-level configuration via `api.NewClient`:
- `apiKey` - API key (or from `WAVESPEED_API_KEY`)
- `baseURL` - API base URL (default: https://api.wavespeed.ai)
- `connectionTimeout` - Connection timeout in seconds (default: 10.0)
- `maxRetries` - Task-level retries (default: 0)
- `maxConnectionRetries` - HTTP connection retries (default: 5)
- `retryInterval` - Base retry delay in seconds (default: 1.0)

Per-request configuration via Run parameters:
- `timeout` - Override timeout
- `pollInterval` - Override poll interval
- `enableSyncMode` - Use sync mode
- `maxRetries` - Override retry count

### Environment Variables

- `WAVESPEED_API_KEY` - API key

## Testing

Tests are located in:
- `config_test.go` - Configuration tests
- `api/client_test.go` - API client tests

Coverage:
- Client initialization
- Run method with different options
- Sync mode
- Retry logic
- Upload functionality
