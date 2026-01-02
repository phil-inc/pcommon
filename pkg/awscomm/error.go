package awscomm

import (
	"errors"
	"fmt"
)

// Error represents a custom error for comm package
type Error struct {
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

// NewError creates a new Error
func NewError(message string) *Error {
	return &Error{Message: message}
}

// WrapError wraps an existing error with a message
func WrapError(err error, message string) *Error {
	return &Error{Message: message, Err: err}
}

// IsCommError checks if the given error is from the comm client SDK
func IsCommError(err error) bool {
	var commErr *Error
	return errors.As(err, &commErr)
}
