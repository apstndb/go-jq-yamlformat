package jqyaml

import (
	"fmt"
	"io"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/itchyny/gojq"
)

// Option configures a Pipeline
type Option func(*pipeline) error

// WithQuery sets the jq query
func WithQuery(query string) Option {
	return func(p *pipeline) error {
		p.query = query
		return nil
	}
}

// WithDefaultEncodeOptions sets default encoding options
// These options apply to both jq conversion and output formatting
func WithDefaultEncodeOptions(opts ...yaml.EncodeOption) Option {
	return func(p *pipeline) error {
		p.defaultEncodeOptions = append(p.defaultEncodeOptions, opts...)
		return nil
	}
}

// WithCompilerOptions sets gojq compiler options
func WithCompilerOptions(opts ...gojq.CompilerOption) Option {
	return func(p *pipeline) error {
		p.compilerOptions = append(p.compilerOptions, opts...)
		return nil
	}
}

// WithInputMarshaler sets a custom input marshaler for converting Go values to gojq-compatible types
// The marshaler is responsible for converting input data and variables before they are processed by jq
func WithInputMarshaler(marshaler InputMarshaler) Option {
	return func(p *pipeline) error {
		if marshaler == nil {
			return fmt.Errorf("input marshaler cannot be nil")
		}
		p.inputMarshaler = marshaler
		return nil
	}
}

// ExecuteOption configures the execution
type ExecuteOption func(*executeConfig)

// WithEncoder sets a custom encoder
func WithEncoder(encoder Encoder) ExecuteOption {
	return func(c *executeConfig) {
		c.encoder = encoder
	}
}

// WithWriter sets the output writer and format
func WithWriter(w io.Writer, format Format) ExecuteOption {
	return func(c *executeConfig) {
		// Store the writer and format for later processing in Execute
		c.writer = w
		c.format = format
	}
}

// WithVariables sets jq variables (accepts any Go object, including structs with json tags)
func WithVariables(vars map[string]interface{}) ExecuteOption {
	return func(c *executeConfig) {
		c.variables = vars
	}
}

// WithTimeout sets execution timeout
func WithTimeout(timeout time.Duration) ExecuteOption {
	return func(c *executeConfig) {
		c.timeout = timeout
	}
}

// WithEncodeOptions sets encoding options for both jq conversion and output formatting
func WithEncodeOptions(opts ...yaml.EncodeOption) ExecuteOption {
	return func(c *executeConfig) {
		c.encodeOptions = append(c.encodeOptions, opts...)
	}
}

// WithCallback sets a callback for streaming mode
func WithCallback(callback func(interface{}) error) ExecuteOption {
	return func(c *executeConfig) {
		c.callback = callback
	}
}

// WithCompactJSONOutput enables compact JSON output (no pretty-printing)
// This option only applies to JSON output format and is ignored for YAML
func WithCompactJSONOutput() ExecuteOption {
	return func(c *executeConfig) {
		c.compactOutputSet = true
		c.compactOutput = true
	}
}

// WithPrettyJSONOutput enables pretty JSON output with indentation
// This option only applies to JSON output format and is ignored for YAML
func WithPrettyJSONOutput() ExecuteOption {
	return func(c *executeConfig) {
		c.compactOutputSet = true
		c.compactOutput = false
	}
}

// WithRawJSONOutput enables raw output for string values (no JSON quotes)
// This option only applies to JSON output format and is ignored for YAML
// When enabled, string values are written directly without JSON encoding
func WithRawJSONOutput() ExecuteOption {
	return func(c *executeConfig) {
		c.rawOutput = true
	}
}
