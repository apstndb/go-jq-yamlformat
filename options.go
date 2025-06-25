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

// WithProtojsonInput is a helper that uses protojson for marshaling Protocol Buffer messages
// This option requires google.golang.org/protobuf to be imported in your project
//
// Usage:
//   p, err := jqyaml.New(
//       jqyaml.WithQuery(".field | select(.value > 10)"),
//       jqyaml.WithProtojsonInput(),
//   )
//
// Note: To use this option, you need to implement a custom InputMarshaler using protojson:
//   import "google.golang.org/protobuf/encoding/protojson"
//   import "google.golang.org/protobuf/proto"
//   import "reflect"
//
//   type protojsonMarshaler struct{}
//   
//   func (m *protojsonMarshaler) Marshal(v interface{}) (interface{}, error) {
//       // Handle proto.Message
//       if msg, ok := v.(proto.Message); ok {
//           b, err := protojson.Marshal(msg)
//           if err != nil {
//               return nil, err
//           }
//           var result interface{}
//           if err := json.Unmarshal(b, &result); err != nil {
//               return nil, err
//           }
//           return result, nil
//       }
//       
//       // Handle slices that might contain proto.Message
//       rv := reflect.ValueOf(v)
//       if rv.Kind() == reflect.Slice {
//           result := make([]interface{}, rv.Len())
//           for i := 0; i < rv.Len(); i++ {
//               elem := rv.Index(i).Interface()
//               converted, err := m.Marshal(elem)
//               if err != nil {
//                   return nil, err
//               }
//               result[i] = converted
//           }
//           return result, nil
//       }
//       
//       // Handle maps recursively
//       if m, ok := v.(map[string]interface{}); ok {
//           result := make(map[string]interface{})
//           for k, val := range m {
//               converted, err := m.Marshal(val)
//               if err != nil {
//                   return nil, err
//               }
//               result[k] = converted
//           }
//           return result, nil
//       }
//       
//       // Fall back to default JSON marshaling for non-protobuf types
//       return (&defaultInputMarshaler{}).Marshal(v)
//   }
func WithProtojsonInput() Option {
	// This is a documentation-only function that shows how to use protojson
	// Users need to implement their own protojson marshaler
	return func(p *pipeline) error {
		return fmt.Errorf("WithProtojsonInput requires a custom implementation. " +
			"Please create your own InputMarshaler using protojson as shown in the documentation")
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