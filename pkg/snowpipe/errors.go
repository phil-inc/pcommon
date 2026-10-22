package snowpipe

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var (
	// ErrTooLarge is a row over MaxRow, or a request Snowflake answered 413.
	ErrTooLarge = errors.New("snowpipe: too large")
	ErrClosed   = errors.New("snowpipe: client closed")
)

// AppendError lists the pipes whose rows an append did not store. It holds no row values.
type AppendError struct{ Pipes []PipeError }

// PipeError is one pipe's rows that may not be stored, and why.
type PipeError struct {
	Pipe string
	Rows int
	Err  error
}

func (e *AppendError) Error() string {
	parts := make([]string, len(e.Pipes))
	for i, p := range e.Pipes {
		parts[i] = fmt.Sprintf("%s: %d rows not stored: %v", p.Pipe, p.Rows, p.Err)
	}
	return strings.Join(parts, "; ")
}

func (e *AppendError) Unwrap() []error {
	errs := make([]error, len(e.Pipes))
	for i, p := range e.Pipes {
		errs[i] = p.Err
	}
	return errs
}

// StatusError is a response other than 2xx. Message is empty for appends, where it can quote
// a row.
type StatusError struct {
	Status  int
	Code    string
	Message string
}

func (e *StatusError) Error() string {
	s := fmt.Sprintf("snowflake: HTTP %d", e.Status)
	if e.Code != "" {
		s += " " + e.Code
	}
	if e.Message != "" {
		s += ": " + e.Message
	}
	return s
}

// Is makes a 413 match ErrTooLarge.
func (e *StatusError) Is(target error) bool {
	return target == ErrTooLarge && e.Status == http.StatusRequestEntityTooLarge
}

// readBody reads and closes a 2xx response, or turns any other into a *StatusError.
func readBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode/100 == 2 {
		return body, nil
	}
	var v struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(body, &v) // a body that is not JSON keeps only the status
	return nil, &StatusError{Status: resp.StatusCode, Code: v.Code, Message: v.Message}
}
