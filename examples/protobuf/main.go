package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/apstndb/go-jq-yamlformat"
	"github.com/apstndb/go-yamlformat"
)

// Example showing how to use WithProtojsonInput for Protocol Buffer messages
// NOTE: This example uses mock types that simulate protobuf messages for demonstration
// In a real application, you would use actual protobuf-generated types

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

// Note: WithProtojsonInput() now handles the protojson marshaling automatically
// You no longer need to implement your own marshaler for basic protobuf support

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
		jqyaml.WithProtojsonInput(), // Now using the built-in function
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
		jqyaml.WithProtojsonInput(),
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
		jqyaml.WithProtojsonInput(),
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
		jqyaml.WithProtojsonInput(),
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