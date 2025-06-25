# go-jq-yamlformat

[![Go Reference](https://pkg.go.dev/badge/github.com/apstndb/go-jq-yamlformat.svg)](https://pkg.go.dev/github.com/apstndb/go-jq-yamlformat)
[![Go Report Card](https://goreportcard.com/badge/github.com/apstndb/go-jq-yamlformat)](https://goreportcard.com/report/github.com/apstndb/go-jq-yamlformat)

`go-jq-yamlformat` is a Go library that integrates [gojq](https://github.com/itchyny/gojq) and [go-yamlformat](https://github.com/apstndb/go-yamlformat) to provide efficient data querying and formatting capabilities.

## Features

- **Seamless Integration**: Combines the power of jq queries with flexible YAML/JSON formatting
- **Struct Support**: Accept any Go struct with customizable JSON marshaling via UseJSONMarshaler
- **Flexible API**: Uses functional options pattern for clean, extensible configuration
- **Streaming Support**: Process large datasets efficiently with streaming execution
- **Context-Aware**: Full support for context cancellation and timeouts
- **Custom Encoding**: Support for custom type marshalers via go-yaml options
- **Custom Input Marshaling**: Define custom input conversion logic for special types like Protocol Buffers

## Installation

```bash
go get github.com/apstndb/go-jq-yamlformat
```

## Quick Start

```go
package main

import (
    "context"
    "os"
    
    "github.com/apstndb/go-jq-yamlformat"
)

func main() {
    // Create a pipeline with a jq query
    p, err := jqyaml.New(
        jqyaml.WithQuery(".users[] | select(.active)"),
    )
    if err != nil {
        panic(err)
    }
    
    // Your data
    data := map[string]interface{}{
        "users": []map[string]interface{}{
            {"name": "Alice", "active": true},
            {"name": "Bob", "active": false},
            {"name": "Charlie", "active": true},
        },
    }
    
    // Execute the pipeline
    err = p.Execute(context.Background(), data,
        jqyaml.WithWriter(os.Stdout, jqyaml.FormatYAML),
    )
    if err != nil {
        panic(err)
    }
}
```

## Advanced Usage

### Using Variables in jq Queries

```go
p, _ := jqyaml.New(
    jqyaml.WithQuery(".events[] | select(.timestamp > $since)"),
)

err := p.Execute(ctx, data,
    jqyaml.WithWriter(os.Stdout, jqyaml.FormatJSON),
    jqyaml.WithVariables(map[string]interface{}{
        "since": time.Now().Add(-24 * time.Hour),
    }),
)
```

### Custom Type Marshalers

```go
import (
    "strconv"
    "github.com/goccy/go-yaml"
    "github.com/google/uuid"
)

p, _ := jqyaml.New(
    jqyaml.WithQuery(".items[]"),
    jqyaml.WithDefaultEncodeOptions(
        // Custom marshaler for time.Time
        yaml.CustomMarshaler[time.Time](func(t time.Time) ([]byte, error) {
            return []byte(strconv.Quote(t.Format(time.RFC3339))), nil
        }),
        // Custom marshaler for UUID
        yaml.CustomMarshaler[uuid.UUID](func(u uuid.UUID) ([]byte, error) {
            return []byte(strconv.Quote(u.String())), nil
        }),
    ),
)
```

### Streaming Large Datasets

```go
p, _ := jqyaml.New(
    jqyaml.WithQuery(".items[]"),
)

encoder := yamlformat.NewEncoder(os.Stdout)
err := p.Execute(ctx, largeData,
    jqyaml.WithCallback(func(item interface{}) error {
        // Process each item individually
        return encoder.Encode(item)
    }),
)
```

### Pipeline Reuse

```go
// Create a reusable pipeline with default encoding options
p, _ := jqyaml.New(
    jqyaml.WithQuery(".data[] | select(.status == $status)"),
    jqyaml.WithDefaultEncodeOptions(
        yaml.Indent(2),
        yaml.UseLiteralStyleIfMultiline(true),
    ),
)

// Use the same pipeline with different data and variables
for _, dataset := range datasets {
    err := p.Execute(ctx, dataset,
        jqyaml.WithWriter(os.Stdout, jqyaml.FormatYAML),
        jqyaml.WithVariables(map[string]interface{}{
            "status": "active",
        }),
    )
}
```

### Custom Encoders

```go
// Implement custom encoder
type CustomEncoder struct {
    // your fields
}

func (e *CustomEncoder) Encode(v interface{}) error {
    // your encoding logic
    return nil
}

// Use with pipeline
err := p.Execute(ctx, data,
    jqyaml.WithEncoder(&CustomEncoder{}),
)
```

### Custom Input Marshaling

```go
// Define custom input marshaler for special types (e.g., Protocol Buffers)
type protojsonMarshaler struct{}

func (m *protojsonMarshaler) Marshal(v interface{}) (interface{}, error) {
    if msg, ok := v.(proto.Message); ok {
        // Use protojson for protobuf messages
        b, err := protojson.Marshal(msg)
        if err != nil {
            return nil, err
        }
        var result interface{}
        return result, json.Unmarshal(b, &result)
    }
    // Handle other types with default JSON marshaling
    b, err := json.Marshal(v)
    if err != nil {
        return nil, err
    }
    var result interface{}
    return result, json.Unmarshal(b, &result)
}

// Use with pipeline
p, _ := jqyaml.New(
    jqyaml.WithQuery(".users[] | select(.status == \"ACTIVE\")"),
    jqyaml.WithInputMarshaler(&protojsonMarshaler{}),
)
```

### Format-Specific Output Options

```go
// Compact JSON output (no pretty-printing)
err := p.Execute(ctx, data,
    jqyaml.WithWriter(os.Stdout, jqyaml.FormatJSON),
    jqyaml.WithCompactOutput(), // {"id":1,"name":"Alice"}
)

// Pretty JSON output with indentation
err := p.Execute(ctx, data,
    jqyaml.WithWriter(os.Stdout, jqyaml.FormatJSON),
    jqyaml.WithPrettyOutput(), // {
                               //   "id": 1,
                               //   "name": "Alice"
                               // }
)

// Raw string output (no JSON quotes)
p, _ := jqyaml.New(jqyaml.WithQuery(".message"))
err := p.Execute(ctx, map[string]interface{}{"message": "Hello, World!"},
    jqyaml.WithWriter(os.Stdout, jqyaml.FormatJSON),
    jqyaml.WithRawOutput(), // Hello, World!
)

// Note: These options only apply to JSON format and are ignored for YAML
// By default, JSON output uses the go-yamlformat default (compact)
```

## API Reference

### Pipeline Creation

- `New(opts ...Option) (Pipeline, error)` - Creates a new pipeline with options
- `WithQuery(query string) Option` - Sets the jq query
- `WithDefaultEncodeOptions(opts ...yaml.EncodeOption) Option` - Sets default encoding options
- `WithInputMarshaler(marshaler InputMarshaler) Option` - Sets custom input marshaler for converting data to jq-compatible types
- `WithCompilerOptions(opts ...gojq.CompilerOption) Option` - Sets gojq compiler options

### Execution Options

- `WithWriter(w io.Writer, format yamlformat.Format) ExecuteOption` - Sets output writer and format
- `WithEncoder(encoder Encoder) ExecuteOption` - Sets custom encoder
- `WithVariables(vars map[string]interface{}) ExecuteOption` - Sets jq variables
- `WithTimeout(timeout time.Duration) ExecuteOption` - Sets execution timeout
- `WithEncodeOptions(opts ...yaml.EncodeOption) ExecuteOption` - Sets additional encoding options
- `WithCallback(callback func(interface{}) error) ExecuteOption` - Sets callback for streaming mode
- `WithCompactOutput() ExecuteOption` - Enables compact JSON output (no pretty-printing). **Only applies to JSON format**
- `WithPrettyOutput() ExecuteOption` - Enables pretty JSON output with indentation. **Only applies to JSON format**
- `WithRawOutput() ExecuteOption` - Outputs raw strings without JSON quotes. **Only applies to JSON format**

### Error Types

- `QueryError` - jq query compilation or execution errors
- `ConversionError` - Data conversion errors
- `TimeoutError` - Execution timeout errors

## Examples

See the [examples](examples/) directory for more detailed examples:

- [Basic Usage](examples/basic/main.go)
- [Variables](examples/variables/main.go)
- [Custom Types](examples/custom-types/main.go)
- [Streaming](examples/streaming/main.go)
- [Error Handling](examples/errors/main.go)
- [Protocol Buffers](examples/protobuf/main.go) - Using custom input marshaler for protobuf messages

## Design Principles

1. **Realistic Approach**: Accepts gojq's limitations and uses JSON conversion where necessary
2. **Clean API**: Leverages functional options pattern for flexible configuration
3. **Testability**: Each component is independently testable
4. **Extensibility**: Easy to add new processing steps or custom encoders

## gojq Limitations

gojq only accepts the following types:
- `nil`, `bool`, `int`, `float64`, `*big.Int`, `string`, `[]any`, `map[string]any`

This library handles the necessary conversions transparently while preserving type information through custom marshalers.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [gojq](https://github.com/itchyny/gojq) - Pure Go implementation of jq
- [go-yamlformat](https://github.com/apstndb/go-yamlformat) - YAML/JSON formatting library
- [goccy/go-yaml](https://github.com/goccy/go-yaml) - YAML support for Go