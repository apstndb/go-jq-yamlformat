package jqyaml

import (
	"fmt"
	"time"
)

// QueryError represents a jq query compilation or execution error
type QueryError struct {
	Query   string
	Message string
	Err     error
}

func (e *QueryError) Error() string {
	return fmt.Sprintf("jq query error in '%s': %s", e.Query, e.Message)
}

func (e *QueryError) Unwrap() error {
	return e.Err
}

// ConversionError represents data conversion error
type ConversionError struct {
	Value interface{}
	Type  string
	Err   error
}

func (e *ConversionError) Error() string {
	return fmt.Sprintf("failed to convert %T to %s: %v", e.Value, e.Type, e.Err)
}

func (e *ConversionError) Unwrap() error {
	return e.Err
}

// TimeoutError represents execution timeout
type TimeoutError struct {
	// Duration is the timeout duration configured for the pipeline via WithTimeout.
	// Note: This reflects the configured timeout, not necessarily the actual elapsed time
	// before the timeout occurred. If the context deadline comes from a parent context
	// with a shorter deadline, this duration may be longer than the actual time elapsed.
	Duration time.Duration
	// Err is the underlying error (typically context.DeadlineExceeded)
	Err error
}

func (e *TimeoutError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("execution timeout after %s: %v", e.Duration, e.Err)
	}
	return fmt.Sprintf("execution timeout after %s", e.Duration)
}

func (e *TimeoutError) Unwrap() error {
	return e.Err
}
