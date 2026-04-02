package registry

import "fmt"

var (
	ErrNotRegistered   error = fmt.Errorf("Agent not registered")
	ErrNoEndpoint      error = fmt.Errorf("Endpoint not found")
	ErrNoService       error = fmt.Errorf("Service not found")
	ErrBadEndpointInfo error = fmt.Errorf("Invalid endpoint information")
)
