// Package jqyaml provides integration between gojq and go-yamlformat for efficient data querying and formatting.
package jqyaml

import (
	"context"
	"fmt"
	"io"
	"sort"
	"time"

	yamlformat "github.com/apstndb/go-yamlformat"
	"github.com/goccy/go-yaml"
	"github.com/itchyny/gojq"
)

// Pipeline represents a data processing pipeline with jq query support
type Pipeline interface {
	// Execute runs the pipeline with options
	Execute(ctx context.Context, input interface{}, opts ...ExecuteOption) error
}

// Encoder interface for output encoding
type Encoder interface {
	Encode(v interface{}) error
}

// InputMarshaler defines the interface for custom input marshaling
// It converts Go values to gojq-compatible types (nil, bool, int, float64, *big.Int, string, []any, map[string]any)
type InputMarshaler interface {
	Marshal(v interface{}) (interface{}, error)
}

// Format represents the output format (YAML or JSON)
type Format = yamlformat.Format

// Format constants
const (
	FormatYAML = yamlformat.FormatYAML
	FormatJSON = yamlformat.FormatJSON
)

// pipeline implements the Pipeline interface
type pipeline struct {
	query                string
	compiled             *gojq.Code
	defaultEncodeOptions []yaml.EncodeOption
	compilerOptions      []gojq.CompilerOption
	inputMarshaler       InputMarshaler
}

// executeConfig holds execution-specific configuration
type executeConfig struct {
	encoder       Encoder
	callback      func(interface{}) error // For streaming mode
	variables     map[string]interface{}
	timeout       time.Duration
	encodeOptions []yaml.EncodeOption
}

// New creates a new Pipeline with the given options
func New(opts ...Option) (Pipeline, error) {
	p := &pipeline{}
	
	for _, opt := range opts {
		if err := opt(p); err != nil {
			return nil, err
		}
	}
	
	// Validate the query if provided
	if p.query != "" {
		_, err := gojq.Parse(p.query)
		if err != nil {
			return nil, &QueryError{
				Query:   p.query,
				Message: "failed to parse query",
				Err:     err,
			}
		}
		
		// Don't compile yet - we'll compile at execution time with proper variables
	}
	
	return p, nil
}

// Execute runs the pipeline on the input data
func (p *pipeline) Execute(ctx context.Context, input interface{}, opts ...ExecuteOption) error {
	// Configure execution
	cfg := &executeConfig{
		timeout: 30 * time.Second, // default
	}
	
	// Apply options
	for _, opt := range opts {
		opt(cfg)
	}
	
	// Ensure either encoder or callback is set
	if cfg.encoder == nil && cfg.callback == nil {
		return fmt.Errorf("no output method specified: use WithWriter, WithEncoder, or WithCallback")
	}
	if cfg.encoder != nil && cfg.callback != nil {
		return fmt.Errorf("cannot specify both encoder and callback")
	}
	
	// Apply timeout if specified
	if cfg.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.timeout)
		defer cancel()
	}
	
	// Combine encode options (default + execution-specific)
	allEncodeOpts := append(p.defaultEncodeOptions, cfg.encodeOptions...)
	
	// Determine which input marshaler to use
	marshaler := p.inputMarshaler
	if marshaler == nil {
		// Use default marshaler with current encode options
		marshaler = &defaultInputMarshaler{encodeOptions: allEncodeOpts}
	}
	
	// Convert input to jq-compatible format using the input marshaler
	jsonData, err := marshaler.Marshal(input)
	if err != nil {
		return &ConversionError{
			Value: input,
			Type:  "jq-compatible",
			Err:   err,
		}
	}
	
	// Determine callback
	callback := cfg.callback
	if callback == nil && cfg.encoder != nil {
		// Apply encode options if encoder supports them
		if encodeOptsSetter, ok := cfg.encoder.(interface {
			SetOptions(...yaml.EncodeOption)
		}); ok {
			encodeOptsSetter.SetOptions(allEncodeOpts...)
		}
		// Use encoder.Encode as callback
		callback = cfg.encoder.Encode
	}
	
	// Process with streaming (works for both callback and encoder modes)
	return p.streamingProcess(ctx, jsonData, cfg.variables, marshaler, callback, cfg.timeout)
}

// streamingProcess processes data through jq with streaming callback
func (p *pipeline) streamingProcess(ctx context.Context, data interface{}, variables map[string]interface{}, marshaler InputMarshaler, callback func(interface{}) error, timeout time.Duration) error {
	// If no query, stream data as-is
	if p.query == "" {
		return callback(data)
	}
	
	// Convert variables to jq-compatible format using the same marshaler
	convertedVars, err := p.convertVariables(variables, marshaler)
	if err != nil {
		return err
	}
	
	// Run query
	iter := p.runQueryWithVariables(ctx, data, convertedVars)
	
	// Stream results
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			if err == context.DeadlineExceeded {
				return &TimeoutError{Duration: timeout}
			}
			return &QueryError{
				Query:   p.query,
				Message: "execution error",  
				Err:     err,
			}
		}
		if err := callback(v); err != nil {
			return err
		}
	}
	
	return nil
}


// convertVariables converts variables to jq-compatible format
func (p *pipeline) convertVariables(variables map[string]interface{}, marshaler InputMarshaler) (map[string]interface{}, error) {
	if len(variables) == 0 {
		return nil, nil
	}
	
	convertedVars := make(map[string]interface{})
	for k, v := range variables {
		converted, err := marshaler.Marshal(v)
		if err != nil {
			return nil, &ConversionError{
				Value: v,
				Type:  fmt.Sprintf("variable %s", k),
				Err:   err,
			}
		}
		convertedVars[k] = converted
	}
	return convertedVars, nil
}

// runQueryWithVariables runs the compiled query with variables
func (p *pipeline) runQueryWithVariables(ctx context.Context, data interface{}, variables map[string]interface{}) gojq.Iter {
	// Parse the query (already validated in New)
	parsed, _ := gojq.Parse(p.query)
	
	// Prepare variables for gojq
	var varNames []string
	var varValues []interface{}
	if len(variables) > 0 {
		// Collect variable names with $ prefix (as gojq expects)
		for k := range variables {
			varNames = append(varNames, "$" + k)
		}
		sort.Strings(varNames)
		// Collect values in the same order
		for _, varName := range varNames {
			key := varName[1:] // Remove $ to get the key
			varValues = append(varValues, variables[key])
		}
	}
	
	// Compile with variables and user-provided compiler options
	var code *gojq.Code
	var err error
	opts := append([]gojq.CompilerOption{}, p.compilerOptions...)
	if len(varNames) > 0 {
		opts = append(opts, gojq.WithVariables(varNames))
	}
	code, err = gojq.Compile(parsed, opts...)
	if err != nil {
		// Return an iterator that yields the error
		return &errorIter{err: &QueryError{
			Query:   p.query,
			Message: "failed to compile query",
			Err:     err,
		}}
	}
	
	return code.RunWithContext(ctx, data, varValues...)
}

// errorIter is an iterator that yields a single error
type errorIter struct {
	err error
	done bool
}

func (e *errorIter) Next() (interface{}, bool) {
	if e.done {
		return nil, false
	}
	e.done = true
	return e.err, true
}


// convertToJQCompatible converts any Go value to gojq-compatible types
func convertToJQCompatible(v interface{}, opts ...yaml.EncodeOption) (interface{}, error) {
	// Use yamlformat for marshaling to respect CustomMarshaler options
	data, err := yamlformat.MarshalJSON(v, opts...)
	if err != nil {
		return nil, err
	}
	
	// Unmarshal to generic interface
	var result interface{}
	if err := yamlformat.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	
	return result, nil
}

// defaultInputMarshaler implements InputMarshaler using the existing convertToJQCompatible logic
type defaultInputMarshaler struct {
	encodeOptions []yaml.EncodeOption
}

func (d *defaultInputMarshaler) Marshal(v interface{}) (interface{}, error) {
	return convertToJQCompatible(v, d.encodeOptions...)
}

// encoderWrapper wraps yamlformat encoders to support option setting
type encoderWrapper struct {
	writer  io.Writer
	format  Format
	options []yaml.EncodeOption
}

func (e *encoderWrapper) Encode(v interface{}) error {
	encoder := e.format.NewEncoder(e.writer, e.options...)
	return encoder.Encode(v)
}

func (e *encoderWrapper) SetOptions(opts ...yaml.EncodeOption) {
	e.options = append(e.options, opts...)
}