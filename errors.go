package jqyaml

import (
	"fmt"
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
