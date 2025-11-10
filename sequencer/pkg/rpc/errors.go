package rpc

import "fmt"

// ErrInvalidConfig is returned when configuration is invalid
type ErrInvalidConfig struct {
	Field  string
	Reason string
}

func (e ErrInvalidConfig) Error() string {
	return fmt.Sprintf("invalid config: %s - %s", e.Field, e.Reason)
}

// ErrConnectionFailed is returned when connection to RPC endpoint fails
type ErrConnectionFailed struct {
	URL   string
	Cause error
}

func (e ErrConnectionFailed) Error() string {
	return fmt.Sprintf("connection failed to %s: %v", e.URL, e.Cause)
}

func (e ErrConnectionFailed) Unwrap() error {
	return e.Cause
}

// ErrRPCError is returned when RPC call returns an error
type ErrRPCError struct {
	Code    int
	Message string
	Data    interface{}
}

func (e ErrRPCError) Error() string {
	return fmt.Sprintf("RPC error [%d]: %s", e.Code, e.Message)
}

// ErrTimeout is returned when a request times out
type ErrTimeout struct {
	Operation string
	Timeout   string
}

func (e ErrTimeout) Error() string {
	return fmt.Sprintf("timeout waiting for %s (timeout: %s)", e.Operation, e.Timeout)
}

// ErrMaxRetriesExceeded is returned when max retries are exceeded
type ErrMaxRetriesExceeded struct {
	Attempts int
	LastErr  error
}

func (e ErrMaxRetriesExceeded) Error() string {
	return fmt.Sprintf("max retries (%d) exceeded, last error: %v", e.Attempts, e.LastErr)
}

func (e ErrMaxRetriesExceeded) Unwrap() error {
	return e.LastErr
}

