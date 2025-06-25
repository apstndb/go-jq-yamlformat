package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"

	"github.com/apstndb/go-jq-yamlformat"
	"github.com/apstndb/go-yamlformat"
)

// Example showing how to use WithInputMarshaler for Protocol Buffer messages
// NOTE: This example demonstrates the pattern without actual protobuf dependencies

// MockProtoMessage simulates a proto.Message interface
type MockProtoMessage interface {
	ProtoReflect() interface{}
}

// Example message types that simulate protobuf messages
type User struct {
	Id    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (u *User) ProtoReflect() interface{} { return nil }

type UserList struct {
	Users []*User `json:"users"`
}

func (u *UserList) ProtoReflect() interface{} { return nil }

// protojsonMarshaler demonstrates how to implement a custom marshaler for protobuf
// In a real implementation, you would import:
//   "google.golang.org/protobuf/encoding/protojson"
//   "google.golang.org/protobuf/proto"
type protojsonMarshaler struct{}

func (m *protojsonMarshaler) Marshal(v interface{}) (interface{}, error) {
	// Check if it's a proto.Message (using our mock interface)
	if msg, ok := v.(MockProtoMessage); ok {
		// In real implementation:
		// b, err := protojson.Marshal(msg)
		// For this example, we'll use regular JSON
		b, err := json.Marshal(msg)
		if err != nil {
			return nil, err
		}
		var result interface{}
		if err := json.Unmarshal(b, &result); err != nil {
			return nil, err
		}
		return result, nil
	}

	// Handle slices that might contain proto.Message
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice {
		result := make([]interface{}, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			elem := rv.Index(i).Interface()
			converted, err := m.Marshal(elem)
			if err != nil {
				return nil, err
			}
			result[i] = converted
		}
		return result, nil
	}

	// Handle maps recursively
	if mapVal, ok := v.(map[string]interface{}); ok {
		result := make(map[string]interface{})
		for k, val := range mapVal {
			converted, err := m.Marshal(val)
			if err != nil {
				return nil, err
			}
			result[k] = converted
		}
		return result, nil
	}

	// Fall back to default JSON marshaling for non-protobuf types
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var result interface{}
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func main() {
	// Sample data - in real usage, this would be actual protobuf messages
	userList := &UserList{
		Users: []*User{
			{Id: 1, Name: "Alice", Email: "alice@example.com"},
			{Id: 2, Name: "Bob", Email: "bob@example.com"},
			{Id: 3, Name: "Charlie", Email: "charlie@example.com"},
		},
	}

	// Example 1: Basic filtering with protobuf messages
	fmt.Println("Example 1: Filter users by name")
	p1, err := jqyaml.New(
		jqyaml.WithQuery(`.users[] | select(.name == "Bob")`),
		jqyaml.WithInputMarshaler(&protojsonMarshaler{}),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = p1.Execute(context.Background(), userList,
		jqyaml.WithWriter(os.Stdout, yamlformat.FormatYAML),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Example 2: Working with arrays of protobuf messages
	fmt.Println("\nExample 2: Direct array of protobuf messages")
	users := []*User{
		{Id: 4, Name: "David", Email: "david@example.com"},
		{Id: 5, Name: "Eve", Email: "eve@example.com"},
	}

	p2, err := jqyaml.New(
		jqyaml.WithQuery(`map({id: .id, display: (.name + " <" + .email + ">")})`),
		jqyaml.WithInputMarshaler(&protojsonMarshaler{}),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = p2.Execute(context.Background(), users,
		jqyaml.WithWriter(os.Stdout, yamlformat.FormatJSON),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Example 3: Mixed data with protobuf messages
	fmt.Println("\nExample 3: Mixed data structures")
	mixedData := map[string]interface{}{
		"metadata": map[string]interface{}{
			"version": "1.0",
			"count":   2,
		},
		"users": []*User{
			{Id: 6, Name: "Frank", Email: "frank@example.com"},
			{Id: 7, Name: "Grace", Email: "grace@example.com"},
		},
		"active_user": &User{Id: 6, Name: "Frank", Email: "frank@example.com"},
	}

	p3, err := jqyaml.New(
		jqyaml.WithQuery(`{
			version: .metadata.version,
			total_users: .metadata.count,
			active: .active_user.name,
			all_emails: .users | map(.email)
		}`),
		jqyaml.WithInputMarshaler(&protojsonMarshaler{}),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = p3.Execute(context.Background(), mixedData,
		jqyaml.WithWriter(os.Stdout, yamlformat.FormatYAML),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Example 4: Using variables with protobuf messages
	fmt.Println("\nExample 4: Using variables with protobuf filtering")
	p4, err := jqyaml.New(
		jqyaml.WithQuery(`.users[] | select(.id == $target_id)`),
		jqyaml.WithInputMarshaler(&protojsonMarshaler{}),
	)
	if err != nil {
		log.Fatal(err)
	}

	variables := map[string]interface{}{
		"target_id": int64(2),
	}

	err = p4.Execute(context.Background(), userList,
		jqyaml.WithVariables(variables),
		jqyaml.WithWriter(os.Stdout, yamlformat.FormatJSON),
	)
	if err != nil {
		log.Fatal(err)
	}
}