# Protobuf Example

This example demonstrates how to use `WithProtojsonInput` to process Protocol Buffer messages with go-jq-yamlformat.

## Important Note

This example uses mock types to simulate protobuf messages for demonstration purposes. The library now includes built-in protobuf support via `WithProtojsonInput()`.

In a real application, you would:
1. Use protobuf-generated types from your `.proto` files
2. Call `WithProtojsonInput()` to automatically handle protobuf marshaling

## Key Features Demonstrated

1. **Built-in Protobuf Support**: Shows how to use `WithProtojsonInput()` for automatic protobuf handling
2. **Slice Handling**: Demonstrates processing slices of protobuf messages
3. **Recursive Processing**: Handles nested data structures containing protobuf messages
4. **Mixed Data**: Shows how to work with data that contains both protobuf and regular Go types

## Running the Example

```bash
go run main.go
```

## Real-world Usage

For actual protobuf usage, simply use the built-in support:

```go
import (
    "github.com/apstndb/go-jq-yamlformat"
    "google.golang.org/protobuf/types/known/timestamppb"
    "google.golang.org/protobuf/types/known/durationpb"
)

// Your protobuf-generated types
type User struct {
    Id        int64                  `protobuf:"varint,1,opt,name=id,proto3"`
    Name      string                 `protobuf:"bytes,2,opt,name=name,proto3"`
    CreatedAt *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=created_at,json=createdAt,proto3"`
    Duration  *durationpb.Duration   `protobuf:"bytes,4,opt,name=duration,proto3"`
}

// Use WithProtojsonInput to handle all protobuf types including Well-Known Types
p, err := jqyaml.New(
    jqyaml.WithQuery(".users[] | select(.created_at > \"2024-01-01T00:00:00Z\")"),
    jqyaml.WithProtojsonInput(),
)
```

The library automatically handles:
- All protobuf message types
- Well-Known Types (Timestamp, Duration, Struct, etc.)
- Nested protobuf messages
- Slices and maps containing protobuf messages

## Benefits

- **Correct Serialization**: Ensures protobuf Well-Known Types are serialized correctly
- **Type Safety**: Maintains protobuf type information during processing
- **Flexibility**: Works with any protobuf message type
- **Compatibility**: Seamlessly integrates with existing jq queries