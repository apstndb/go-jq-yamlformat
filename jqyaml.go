// Package jqyaml provides integration between gojq and go-yamlformat for efficient data querying and formatting.
package jqyaml

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"sort"
	"time"

	yamlformat "github.com/apstndb/go-yamlformat"
	"github.com/goccy/go-yaml"
	"github.com/itchyny/gojq"
	"google.golang.org/protobuf/proto"
)

// isProtoMessage checks if v implements proto.Message
func isProtoMessage(v interface{}) bool {
	_, ok := v.(proto.Message)
	return ok
}

// --- Public API ---

// Pipeline represents a data processing pipeline with jq query support.
type Pipeline interface {
	Execute(ctx context.Context, input interface{}, opts ...ExecuteOption) error
}

// Encoder is the interface for custom output encoders provided by the user via WithEncoder.
type Encoder interface {
	Encode(v interface{}) error
}

// InputMarshaler defines the interface for custom input data conversion.
type InputMarshaler interface {
	Marshal(v interface{}) (interface{}, error)
}

// Format represents the output format (YAML or JSON).
type Format = yamlformat.Format

const (
	FormatYAML = yamlformat.FormatYAML
	FormatJSON = yamlformat.FormatJSON
)

// --- Internal Implementation ---

// pipeline implements the Pipeline interface.
type pipeline struct {
	query                string
	defaultEncodeOptions []yaml.EncodeOption
	compilerOptions      []gojq.CompilerOption
	inputMarshaler       InputMarshaler
}

// executeConfig holds all configuration for a single execution.
type executeConfig struct {
	encoder       Encoder // Can be user-provided
	writer        io.Writer
	format        Format
	callback      func(interface{}) error
	variables     map[string]interface{}
	timeout       time.Duration
	encodeOptions []yaml.EncodeOption
	// Flags for jq-compatible JSON output
	compactOutput bool
	rawOutput     bool
}

// New creates a new processing pipeline.
func New(opts ...Option) (Pipeline, error) {
	p := &pipeline{}
	for _, opt := range opts {
		if err := opt(p); err != nil {
			return nil, err
		}
	}
	if p.query != "" {
		if _, err := gojq.Parse(p.query); err != nil {
			return nil, &QueryError{Query: p.query, Message: "failed to parse query", Err: err}
		}
	}
	return p, nil
}

// Execute runs the pipeline.
func (p *pipeline) Execute(ctx context.Context, input interface{}, opts ...ExecuteOption) error {
	cfg := &executeConfig{
		timeout:       30 * time.Second,
		compactOutput: true, // Default to compact for jq compatibility
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// --- Output Multiplexer ---
	// The user can specify one of three mutually exclusive output methods.
	if cfg.callback != nil {
		if cfg.encoder != nil || cfg.writer != nil {
			return errors.New("cannot use WithCallback with WithEncoder or WithWriter")
		}
	} else if cfg.encoder == nil {
		if cfg.writer == nil {
			return errors.New("no output method specified: use WithWriter, WithEncoder, or WithCallback")
		}
		// This is the main path for WithWriter. Create the internal pipelineEncoder.
		cfg.encoder = &pipelineEncoder{
			writer:        cfg.writer,
			format:        cfg.format,
			raw:           cfg.rawOutput,
			compact:       cfg.compactOutput,
			encodeOptions: append(p.defaultEncodeOptions, cfg.encodeOptions...),
		}
	}
	// --- End Output Multiplexer ---

	if cfg.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.timeout)
		defer cancel()
	}

	// Use custom input marshaler if provided, otherwise use default with encode options
	marshaler := p.inputMarshaler
	if marshaler == nil {
		// Pass the combined encode options to defaultInputMarshaler
		// This ensures custom marshalers work for input data conversion
		allEncodeOpts := append([]yaml.EncodeOption{}, p.defaultEncodeOptions...)
		allEncodeOpts = append(allEncodeOpts, cfg.encodeOptions...)
		marshaler = &defaultInputMarshaler{
			encodeOptions: allEncodeOpts,
		}
	}

	jsonData, err := marshaler.Marshal(input)
	if err != nil {
		return &ConversionError{Value: input, Type: "jq-compatible", Err: err}
	}

	finalCallback := cfg.callback
	if finalCallback == nil {
		finalCallback = cfg.encoder.Encode
	}

	return p.streamingProcess(ctx, jsonData, cfg.variables, marshaler, finalCallback, cfg.timeout)
}

// pipelineEncoder is the heart of the new architecture.
// Its Encode method implements the conditional "Chain of Responsibility".
type pipelineEncoder struct {
	writer           io.Writer
	format           Format
	raw              bool
	compact          bool
	encodeOptions    []yaml.EncodeOption
	documentsWritten int
}

func (e *pipelineEncoder) Encode(v interface{}) error {
	// Stage 1: Serialization
	b, err := e.serialize(v)
	if err != nil {
		return err
	}

	// Stage 2: jq Compact Compatibility Round-trip (if needed)
	if e.needsCompactRoundtrip() {
		b, err = e.compactRoundtrip(b)
		if err != nil {
			return err
		}
	}

	// Stage 3: Final Formatting
	return e.writeOutput(b)
}

// serialize handles Stage 1: choosing between goccy/go-yaml or standard library
func (e *pipelineEncoder) serialize(v interface{}) ([]byte, error) {
	if len(e.encodeOptions) > 0 {
		return e.serializeWithGoccy(v)
	}
	return e.serializeWithStdlib(v)
}

// serializeWithGoccy uses goccy/go-yaml for advanced formatting options
func (e *pipelineEncoder) serializeWithGoccy(v interface{}) ([]byte, error) {
	opts := append([]yaml.EncodeOption{}, e.encodeOptions...)
	if e.format == FormatJSON {
		opts = append(opts, yaml.JSON())
		if !e.compact { // Pretty print if not compact
			opts = append(opts, yaml.Indent(2))
		}
	}
	b, err := yaml.MarshalWithOptions(v, opts...)
	if err != nil {
		return nil, err
	}
	return bytes.TrimSpace(b), nil
}

// serializeWithStdlib uses the standard library for perfect jq compatibility
func (e *pipelineEncoder) serializeWithStdlib(v interface{}) ([]byte, error) {
	var b []byte
	var err error

	if e.format == FormatYAML {
		b, err = yamlformat.Marshal(v)
	} else {
		// JSON output
		if !e.compact && !e.raw {
			// Pretty print JSON
			var buf bytes.Buffer
			enc := json.NewEncoder(&buf)
			enc.SetIndent("", "  ")
			err = enc.Encode(v)
			b = buf.Bytes()
		} else {
			// Compact output (default for jq compatibility)
			b, err = json.Marshal(v)
		}
	}
	if err != nil {
		return nil, err
	}
	return bytes.TrimSpace(b), nil
}

// needsCompactRoundtrip determines if Stage 2 is needed
func (e *pipelineEncoder) needsCompactRoundtrip() bool {
	return len(e.encodeOptions) > 0 && e.compact && e.format == FormatJSON
}

// compactRoundtrip performs Stage 2: ensuring compact JSON compatibility
func (e *pipelineEncoder) compactRoundtrip(b []byte) ([]byte, error) {
	var temp interface{}
	if err := json.Unmarshal(b, &temp); err != nil {
		return nil, err
	}
	return json.Marshal(temp)
}

// writeOutput handles Stage 3: final formatting and output
func (e *pipelineEncoder) writeOutput(b []byte) error {
	if e.format == FormatYAML {
		if e.documentsWritten > 0 {
			if _, err := e.writer.Write([]byte("---\n")); err != nil {
				return err
			}
		}
		e.documentsWritten++
		return writeDefault(e.writer, b)
	}

	if e.raw && e.format == FormatJSON {
		return writeRawJSON(e.writer, b)
	}
	return writeDefault(e.writer, b)
}

// writeDefault writes the bytes as is, with a trailing newline.
func writeDefault(w io.Writer, marshaledBytes []byte) error {
	if _, err := w.Write(marshaledBytes); err != nil {
		return err
	}
	_, err := w.Write([]byte("\n"))
	return err
}

// writeRawJSON correctly handles `jq --raw-output` behavior.
func writeRawJSON(w io.Writer, marshaledBytes []byte) error {
	// Check if this is a JSON string (but not null)
	if len(marshaledBytes) > 0 && marshaledBytes[0] == '"' {
		var s string
		if err := json.Unmarshal(marshaledBytes, &s); err == nil {
			// Successfully unmarshaled as string - write raw content
			if _, err_write := io.WriteString(w, s); err_write != nil {
				return err_write
			}
		} else {
			// Should not happen with valid JSON
			if _, err_write := w.Write(marshaledBytes); err_write != nil {
				return err_write
			}
		}
	} else {
		// Not a string (number, bool, null, object, array) - write the JSON representation
		if _, err_write := w.Write(marshaledBytes); err_write != nil {
			return err_write
		}
	}
	_, err := w.Write([]byte("\n"))
	return err
}

// --- Utility Functions and Types ---

func (p *pipeline) streamingProcess(ctx context.Context, data interface{}, variables map[string]interface{}, marshaler InputMarshaler, callback func(interface{}) error, timeout time.Duration) error {
	if p.query == "" {
		return callback(data)
	}
	convertedVars, err := p.convertVariables(variables, marshaler)
	if err != nil {
		return err
	}
	iter := p.runQueryWithVariables(ctx, data, convertedVars)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			if errors.Is(err, context.DeadlineExceeded) {
				return fmt.Errorf("execution timeout after %s: %w", timeout, err)
			}
			if errors.Is(err, context.Canceled) {
				return fmt.Errorf("execution canceled: %w", err)
			}
			return &QueryError{Query: p.query, Message: "execution error", Err: err}
		}
		if err := callback(v); err != nil {
			return err
		}
	}
	return nil
}

func (p *pipeline) convertVariables(variables map[string]interface{}, marshaler InputMarshaler) (map[string]interface{}, error) {
	if len(variables) == 0 {
		return nil, nil
	}
	convertedVars := make(map[string]interface{})
	for k, v := range variables {
		converted, err := marshaler.Marshal(v)
		if err != nil {
			return nil, &ConversionError{Value: v, Type: fmt.Sprintf("variable %s", k), Err: err}
		}
		convertedVars[k] = converted
	}
	return convertedVars, nil
}

func (p *pipeline) runQueryWithVariables(ctx context.Context, data interface{}, variables map[string]interface{}) gojq.Iter {
	parsed, _ := gojq.Parse(p.query)
	var varNames []string
	var varValues []interface{}
	if len(variables) > 0 {
		for k := range variables {
			varNames = append(varNames, "$"+k)
		}
		sort.Strings(varNames)
		for _, varName := range varNames {
			key := varName[1:]
			varValues = append(varValues, variables[key])
		}
	}
	var code *gojq.Code
	var err error
	opts := append([]gojq.CompilerOption{}, p.compilerOptions...)
	if len(varNames) > 0 {
		opts = append(opts, gojq.WithVariables(varNames))
	}
	code, err = gojq.Compile(parsed, opts...)
	if err != nil {
		return &errorIter{err: &QueryError{Query: p.query, Message: "failed to compile query", Err: err}}
	}
	return code.RunWithContext(ctx, data, varValues...)
}

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

// convertToJQCompatible converts any Go value to gojq-compatible types.
// gojq only accepts: nil, bool, int, float64, *big.Int, string, []any, map[string]any
//
// This function is used for INPUT data conversion before passing to gojq.
// It respects custom marshalers via yaml.EncodeOption, allowing users to control
// how their custom types are converted before being processed by jq queries.
//
// For example, a user might want to:
// - Convert time.Time to a specific string format before querying
// - Transform custom types into a queryable structure
// - Handle special numeric types in a specific way
//
// The opts parameter allows passing yaml.CustomMarshaler options that will be
// applied during the conversion process.
func convertToJQCompatible(v interface{}, opts ...yaml.EncodeOption) (interface{}, error) {
	switch v := v.(type) {
	case nil, bool, string, int, float64, *big.Int:
		// These types are already gojq-compatible
		return v, nil
	case []interface{}:
		// Recursively convert array elements
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
		// Recursively convert map values
		result := make(map[string]interface{}, len(v))
		for k, val := range v {
			converted, err := convertToJQCompatible(val, opts...)
			if err != nil {
				return nil, err
			}
			result[k] = converted
		}
		return result, nil
	}

	// For complex types, use yamlformat for marshaling to respect CustomMarshaler options
	// This allows users to define how their custom types should be converted to
	// gojq-compatible format before query execution
	data, err := yamlformat.MarshalJSON(v, opts...)
	if err != nil {
		return nil, err
	}

	// Unmarshal to generic interface to get gojq-compatible types
	var result interface{}
	if err := yamlformat.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// defaultInputMarshaler implements InputMarshaler using the existing convertToJQCompatible logic.
// It supports custom marshalers for input data transformation via encodeOptions.
//
// Design rationale:
// - Input data often contains custom types that need specific conversion before jq processing
// - Users should be able to control this conversion via yaml.CustomMarshaler
// - The same encode options from WithDefaultEncodeOptions are passed here
// - This ensures consistent type handling throughout the pipeline
type defaultInputMarshaler struct {
	encodeOptions      []yaml.EncodeOption
	protojsonMarshaler InputMarshaler
}

func (d *defaultInputMarshaler) ensureProtojsonMarshaler() {
	if d.protojsonMarshaler == nil {
		d.protojsonMarshaler = createProtojsonMarshaler()
	}
}

func (d *defaultInputMarshaler) Marshal(v interface{}) (interface{}, error) {
	if isProtoMessage(v) {
		d.ensureProtojsonMarshaler()
		return d.protojsonMarshaler.Marshal(v)
	}
	if slice := reflect.ValueOf(v); slice.Kind() == reflect.Slice {
		var useProtoMarshaler bool
		for i := 0; i < slice.Len(); i++ {
			elem := slice.Index(i).Interface()
			if elem != nil {
				if isProtoMessage(elem) {
					useProtoMarshaler = true
				}
				break
			}
		}
		if useProtoMarshaler {
			d.ensureProtojsonMarshaler()
			return d.protojsonMarshaler.Marshal(v)
		}
	}
	return convertToJQCompatible(v, d.encodeOptions...)
}
