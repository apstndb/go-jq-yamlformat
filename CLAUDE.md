# go-jq-yamlformat

## Overview

This library integrates [gojq](https://github.com/itchyny/gojq) and [go-yamlformat](https://github.com/apstndb/go-yamlformat) to provide efficient data querying and formatting capabilities.

## Key Design Decisions

### Variable Naming Convention
- Variables are passed WITHOUT the `$` prefix in the map
- Example: `WithVariables(map[string]interface{}{"threshold": 10})` for query `.[] | select(. > $threshold)`
- The library automatically adds the `$` prefix when compiling with gojq

### Execution Modes
- **Normal mode**: Use `WithWriter` or `WithEncoder` to output formatted data
- **Streaming mode**: Use `WithCallback` to process results one by one
- Cannot use both encoder and callback in the same execution

### JSON Conversion
- Due to gojq limitations (only accepts nil, bool, int, float64, *big.Int, string, []any, map[string]any)
- All Go structs are converted to JSON-compatible format before processing
- Custom marshalers are applied during this conversion

### Query Compilation
- Queries are validated at pipeline creation time
- Actual compilation happens at execution time when variables are known
- This allows pipeline reuse with different variables

### Error Handling
- All error types (`QueryError`, `ConversionError`) implement the `error` interface with `Unwrap()` method
- Supports Go 1.13+ error handling patterns with `errors.Is()` and `errors.As()`
- Timeout errors are returned as `fmt.Errorf` wrapped `context.DeadlineExceeded` with timeout duration in the message

## Testing

```bash
# Run all tests
go test -v ./...

# Run with coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Before Committing

Always run `make all` before committing to ensure:
- Code formatting is correct (uses gofumpt)
- All linting checks pass (golangci-lint)
- All tests pass with race detection enabled

```bash
make all
```

If formatting issues are found, run `make fmt` to fix them automatically.

## Examples

See the `examples/` directory for complete examples:
- `basic/` - Simple filtering and transformation
- `variables/` - Using variables in queries
- `custom-types/` - Custom type marshalers
- `streaming/` - Processing large datasets
- `errors/` - Error handling patterns