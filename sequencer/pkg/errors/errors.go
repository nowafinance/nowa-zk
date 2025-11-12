package errors

import "fmt"

// Error represents a sequencer error with context
type Error struct {
	Code    string
	Message string
	Err     error
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.Err
}

// New creates a new error
func New(code, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// Wrap wraps an existing error
func Wrap(code, message string, err error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Error codes
const (
	ErrCodeConfig         = "CONFIG_ERROR"
	ErrCodeRPC            = "RPC_ERROR"
	ErrCodeWebSocket      = "WEBSOCKET_ERROR"
	ErrCodeBatchStore     = "BATCH_STORE_ERROR"
	ErrCodeBatchBuilder   = "BATCH_BUILDER_ERROR"
	ErrCodeAPI            = "API_ERROR"
	ErrCodeBlockProcessing = "BLOCK_PROCESSING_ERROR"
	ErrCodeInvalidInput   = "INVALID_INPUT"
)

// Common error constructors
func ErrConfig(message string, err error) *Error {
	return Wrap(ErrCodeConfig, message, err)
}

func ErrRPC(message string, err error) *Error {
	return Wrap(ErrCodeRPC, message, err)
}

func ErrWebSocket(message string, err error) *Error {
	return Wrap(ErrCodeWebSocket, message, err)
}

func ErrBatchStore(message string, err error) *Error {
	return Wrap(ErrCodeBatchStore, message, err)
}

func ErrBatchBuilder(message string, err error) *Error {
	return Wrap(ErrCodeBatchBuilder, message, err)
}

func ErrAPI(message string, err error) *Error {
	return Wrap(ErrCodeAPI, message, err)
}

func ErrBlockProcessing(message string, err error) *Error {
	return Wrap(ErrCodeBlockProcessing, message, err)
}

func ErrInvalidInput(message string) *Error {
	return New(ErrCodeInvalidInput, message)
}

