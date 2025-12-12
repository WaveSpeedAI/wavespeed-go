# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

WaveSpeed Go SDK - Official Go SDK for WaveSpeedAI inference platform.

## Commands

### Testing
```bash
# Run all tests
go test -v ./...

# Run tests with coverage
go test -v -cover ./...

# Run specific test
go test -v -run TestClientRun
```

### Building
```bash
# Build the package
go build ./...

# Install locally
go install ./...
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

### Client Structure

Entry point: `NewClient(apiKey string, opts *ClientOptions) (*Client, error)`

The SDK provides a simple client for running models:

```go
client, err := wavespeed.NewClient("your-api-key")
output, err := client.Run("model-id", input)
```

### Key Types

- `Client` - Main client struct with configuration
- `ClientOptions` - Options for client initialization
- `RunOptions` - Options for individual run calls
- `Prediction` - Response type containing outputs

### Features

- **Sync Mode**: Single request that waits for result (`EnableSyncMode`)
- **Retry Logic**: Configurable task-level and connection-level retries
- **Timeout Control**: Per-request and overall timeouts
- **File Upload**: Direct file upload to WaveSpeed storage

### Configuration

Client-level configuration via `ClientOptions`:
- `BaseURL` - API base URL
- `PollIntervalSeconds` - Polling interval
- `TimeoutSeconds` - Overall timeout
- `MaxRetries` - Task-level retries
- `MaxConnectionRetries` - HTTP connection retries
- `RetryInterval` - Base retry delay

Per-request configuration via `RunOptions`:
- `TimeoutSeconds` - Override timeout
- `PollIntervalSeconds` - Override poll interval
- `EnableSyncMode` - Use sync mode
- `MaxRetries` - Override retry count

### Environment Variables

- `WAVESPEED_API_KEY` - API key
- `WAVESPEED_BASE_URL` - Base URL (default: https://api.wavespeed.ai)
- `WAVESPEED_POLL_INTERVAL` - Poll interval in seconds
- `WAVESPEED_TIMEOUT` - Timeout in seconds

## Testing

Tests are located in `wavespeed_test.go` and cover:
- Client initialization
- Run method with different options
- Sync mode
- Retry logic
- Upload functionality

## Release Process

This project uses Git tags for versioning. See VERSIONING.md for details.

To create a release:
1. Tag the version: `git tag v1.0.0`
2. Push the tag: `git push origin v1.0.0`
3. GitHub Actions will automatically create a release
