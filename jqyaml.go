// Package jqyaml provides integration between gojq and go-yamlformat for efficient data querying and formatting.
package jqyaml

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"sort"
	"time"

	yamlformat "github.com/apstndb/go-yamlformat"
	"github.com/goccy/go-yaml"
	"github.com/itchyny/gojq"
)

// isProtoMessage checks if v implements proto.Message using reflection
func isProtoMessage(v interface{}) bool {
	if v == nil {
		return false
	}
	t := reflect.TypeOf(v)
	// Check for the three required methods of proto.Message
	_, hasProtoReflect := t.MethodByName("ProtoReflect")
	_, hasReset := t.MethodByName("Reset")
	_, hasString := t.MethodByName("String")
	_, hasProtoMessage := t.MethodByName("ProtoMessage")

	return hasProtoReflect && hasReset && hasString && hasProtoMessage
}

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

// JSONStyle represents JSON output style options
type JSONStyle int

// JSONStyle constants
const (
	JSONStyleCompact JSONStyle = 0
	JSONStylePretty  JSONStyle = 1 << iota
	JSONStyleRaw
)

// pipeline implements the Pipeline interface
type pipeline struct {
	query                string
	defaultEncodeOptions []yaml.EncodeOption
	compilerOptions      []gojq.CompilerOption
	inputMarshaler       InputMarshaler
	defaultJSONStyle     JSONStyle
}

// executeConfig holds execution-specific configuration
type executeConfig struct {
	encoder          Encoder
	writer           io.Writer
	format           Format
	callback         func(interface{}) error // For streaming mode
	variables        map[string]interface{}
	timeout          time.Duration
	encodeOptions    []yaml.EncodeOption
	compactOutputSet bool // Whether compactOutput was explicitly set
	compactOutput    bool // For JSON output only
	rawOutput        bool // For JSON output only
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

	// Apply pipeline defaults for JSON if not explicitly set by execution options
	if p.defaultJSONStyle != 0 && !cfg.compactOutputSet && !cfg.rawOutput {
		// Apply defaults - user options can override these later
		cfg.rawOutput = (p.defaultJSONStyle & JSONStyleRaw) != 0
		cfg.compactOutput = (p.defaultJSONStyle & JSONStylePretty) == 0
		cfg.compactOutputSet = true
	}

	// Apply options (these can override defaults)
	for _, opt := range opts {
		opt(cfg)
	}

	// Handle WithWriter case - create appropriate encoder
	if cfg.writer != nil && cfg.encoder == nil {
		if cfg.format == FormatJSON {
			// Always use custom JSON encoder for JSON output (encoding/json based)
			pretty := !cfg.compactOutput && cfg.compactOutputSet
			cfg.encoder = newJSONEncoder(cfg.writer, pretty, cfg.rawOutput)
		} else {
			// Use YAML encoder wrapper for YAML
			cfg.encoder = &yamlEncoderWrapper{
				writer:  cfg.writer,
				options: []yaml.EncodeOption{},
			}
		}
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
	allEncodeOpts := append([]yaml.EncodeOption{}, p.defaultEncodeOptions...)
	allEncodeOpts = append(allEncodeOpts, cfg.encodeOptions...)

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
			varNames = append(varNames, "$"+k)
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
	err  error
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
	// Fast path for already compatible types
	switch v := v.(type) {
	case nil, bool, string:
		return v, nil
	case int:
		// gojq accepts int directly
		return v, nil
	case float64:
		return v, nil
	case *big.Int:
		// gojq accepts *big.Int directly
		return v, nil
	case []interface{}:
		// Recursively convert elements
		result := make([]interface{}, len(v))
		for i, elem := range v {
			converted, err := convertToJQCompatible(elem, opts...)
			if err != nil {
				return nil, err
			}
			result[i] = converted
		}
		return result, nil
	case map[string]interface{}:
		// Recursively convert values
		result := make(map[string]interface{}, len(v))
		for k, val := range v {
			converted, err := convertToJQCompatible(val, opts...)
			if err != nil {
				return nil, err
			}
			result[k] = converted
		}
		return result, nil
	// Convert other numeric types to float64 (gojq's number type)
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case float32:
		return float64(v), nil
	}

	// For complex types, use yamlformat for marshaling to respect CustomMarshaler options
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
// with automatic proto.Message detection
type defaultInputMarshaler struct {
	encodeOptions      []yaml.EncodeOption
	protojsonMarshaler InputMarshaler
}

func (d *defaultInputMarshaler) Marshal(v interface{}) (interface{}, error) {
	// Check if v implements proto.Message
	if isProtoMessage(v) {
		if d.protojsonMarshaler == nil {
			// Lazy initialization to avoid import if not needed
			d.protojsonMarshaler = createProtojsonMarshaler()
		}
		return d.protojsonMarshaler.Marshal(v)
	}

	// Check if v is a slice of proto.Message
	if slice := reflect.ValueOf(v); slice.Kind() == reflect.Slice {
		if slice.Len() > 0 {
			// Check the first element
			if isProtoMessage(slice.Index(0).Interface()) {
				if d.protojsonMarshaler == nil {
					d.protojsonMarshaler = createProtojsonMarshaler()
				}
				return d.protojsonMarshaler.Marshal(v)
			}
		}
	}

	// Fallback to default conversion
	return convertToJQCompatible(v, d.encodeOptions...)
}

// yamlEncoderWrapper wraps yamlformat YAML encoder to support option setting
type yamlEncoderWrapper struct {
	writer        io.Writer
	options       []yaml.EncodeOption
	documentCount int
}

func (e *yamlEncoderWrapper) Encode(v interface{}) error {
	// Add YAML document separator for subsequent documents
	if e.documentCount > 0 {
		if _, err := e.writer.Write([]byte("---\n")); err != nil {
			return err
		}
	}
	e.documentCount++

	encoder := FormatYAML.NewEncoder(e.writer, e.options...)
	return encoder.Encode(v)
}

func (e *yamlEncoderWrapper) SetOptions(opts ...yaml.EncodeOption) {
	e.options = append(e.options, opts...)
}

// jsonEncoder implements custom JSON encoding with pretty and raw output support
type jsonEncoder struct {
	writer      io.Writer
	pretty      bool
	raw         bool
	needNewline bool
}

func newJSONEncoder(w io.Writer, pretty, raw bool) *jsonEncoder {
	return &jsonEncoder{
		writer:      w,
		pretty:      pretty,
		raw:         raw,
		needNewline: false,
	}
}

func (e *jsonEncoder) Encode(v interface{}) error {
	// Add newline before next item if needed (for raw output)
	if e.needNewline {
		if _, err := e.writer.Write([]byte("\n")); err != nil {
			return err
		}
	}

	// Handle raw output for strings
	if e.raw {
		if s, ok := v.(string); ok {
			if _, err := io.WriteString(e.writer, s); err != nil {
				return err
			}
			// Add newline after the string
			if _, err := e.writer.Write([]byte("\n")); err != nil {
				return err
			}
			e.needNewline = false
			return nil
		}
	}

	// Use standard JSON encoder
	encoder := json.NewEncoder(e.writer)
	// By default, json.Encoder produces compact output
	// Only set indent for pretty output (and not raw mode for non-strings)
	if e.pretty && !e.raw {
		encoder.SetIndent("", "  ")
	}

	err := encoder.Encode(v)
	e.needNewline = false // json.Encoder already adds newline
	return err
}
