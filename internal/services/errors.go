package services

import "errors"

// Sentinel errors handlers map to HTTP status codes. Wrapping with
// fmt.Errorf("...: %w", ErrNotFound) preserves errors.Is compatibility.
var (
	ErrNotFound     = errors.New("resource not found")
	ErrConflict     = errors.New("resource conflict")
	ErrValidation   = errors.New("validation failed")
	ErrUnauthorized = errors.New("not authorized")
)
