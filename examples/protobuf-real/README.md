# Real Protobuf Example

This example demonstrates the use of `WithProtojsonInput()` with actual Protocol Buffer messages and Well-Known Types.

## Prerequisites

You need to have `protoc` and the Go protobuf plugin installed:

```bash
# Install protoc (on macOS)
brew install protobuf

# Install Go protobuf plugin
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

## Generate Protobuf Code

```bash
protoc --go_out=pb --go_opt=paths=source_relative example.proto
```

## Running the Example

```bash
go run main.go
```

## What This Example Shows

1. **Well-Known Types Support**:
   - `google.protobuf.Timestamp` - Serialized as RFC3339 strings
   - `google.protobuf.Duration` - Serialized as duration strings (e.g., "45m0s")
   - `google.protobuf.Struct` - Serialized as JSON objects
   - `google.protobuf.Any` - Serialized with type URL

2. **Real-World Queries**:
   - Filter users by status
   - Time-based filtering with timestamp comparisons
   - Extract nested fields from Struct types
   - Duration calculations in jq

3. **Custom Protojson Options**:
   - Using `WithProtojsonInputOptions()` for custom marshaling behavior
   - Control enum representation (names vs numbers)
   - Control zero value emission

## Key Features

The library automatically handles:
- Correct serialization of all Well-Known Types
- Nested protobuf messages
- Enums (both as strings and numbers)
- Any type with proper type URL handling
- Slices and maps containing protobuf messages

## Benefits Over Manual Implementation

- No need to write custom marshalers
- Automatic handling of all protobuf types
- Consistent with protojson standards
- Support for all protojson marshal options