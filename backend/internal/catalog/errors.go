package catalog

import "errors"

// ErrInvalidInput marks a rejected caller value. Handlers answer it with 400;
// every other error is a catalog failure the caller did not cause.
var ErrInvalidInput = errors.New("invalid input")
