package wavespeed

import "github.com/WaveSpeedAI/wavespeed-go/api"

// APIConfig holds API client configuration options.
type APIConfig = api.APIConfig

// API is the global API configuration instance. It is shared with the api
// subpackage: values set here are picked up as defaults by NewClient and by
// the package-level Run/Upload helpers.
var API = api.API
