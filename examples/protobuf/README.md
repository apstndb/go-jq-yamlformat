# Protobuf Example

This example demonstrates how to use `WithInputMarshaler` to process Protocol Buffer messages with go-jq-yamlformat.

## Important Note

This example uses mock types to simulate protobuf messages without requiring actual protobuf dependencies. In a real implementation, you would:

1. Import the necessary protobuf packages:
```go
import (
    "google.golang.org/protobuf/encoding/protojson"
    "google.golang.org/protobuf/proto"
)
```

2. Use `protojson.Marshal` instead of `json.Marshal` for proto.Message types
3. Check for `proto.Message` interface instead of our mock `MockProtoMessage`

## Key Features Demonstrated

1. **Custom Input Marshaler**: Shows how to implement `InputMarshaler` interface for protobuf messages
2. **Slice Handling**: Demonstrates processing slices of protobuf messages
3. **Recursive Processing**: Handles nested data structures containing protobuf messages
4. **Mixed Data**: Shows how to work with data that contains both protobuf and regular Go types

## Running the Example

```bash
go run main.go
```

## Real-world Implementation

For actual protobuf usage, implement the marshaler like this:

```go
type protojsonMarshaler struct{}

func (m *protojsonMarshaler) Marshal(v interface{}) (interface{}, error) {
    // Handle proto.Message
    if msg, ok := v.(proto.Message); ok {
        b, err := protojson.Marshal(msg)
        if err != nil {
            return nil, err
        }
        var result interface{}
        if err := json.Unmarshal(b, &result); err != nil {
            return nil, err
        }
        return result, nil
    }
    
    // Handle slices, maps, etc. recursively...
    // (same as in the example)
}
```

Then use it in your pipeline:

```go
p, err := jqyaml.New(
    jqyaml.WithQuery(".users[] | select(.status == \"ACTIVE\")"),
    jqyaml.WithInputMarshaler(&protojsonMarshaler{}),
)
```

## Benefits

- **Correct Serialization**: Ensures protobuf Well-Known Types are serialized correctly
- **Type Safety**: Maintains protobuf type information during processing
- **Flexibility**: Works with any protobuf message type
- **Compatibility**: Seamlessly integrates with existing jq queries