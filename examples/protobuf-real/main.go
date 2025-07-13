package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	jqyaml "github.com/apstndb/go-jq-yamlformat"
	"github.com/apstndb/go-jq-yamlformat/examples/protobuf-real/pb"
	"github.com/apstndb/go-yamlformat"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	// Create sample data with Well-Known Types
	now := time.Now()
	userList := &pb.UserList{
		Users: []*pb.User{
			{
				Id:              1,
				Name:            "Alice Johnson",
				Email:           "alice@example.com",
				CreatedAt:       timestamppb.New(now.Add(-365 * 24 * time.Hour)), // 1 year ago
				LastLogin:       timestamppb.New(now.Add(-2 * time.Hour)),
				SessionDuration: durationpb.New(45 * time.Minute),
				Metadata: mustStruct(map[string]interface{}{
					"department": "Engineering",
					"level":      "Senior",
					"skills":     []interface{}{"Go", "Kubernetes", "gRPC"},
				}),
				Status: pb.Status_STATUS_ACTIVE,
			},
			{
				Id:              2,
				Name:            "Bob Smith",
				Email:           "bob@example.com",
				CreatedAt:       timestamppb.New(now.Add(-180 * 24 * time.Hour)), // 6 months ago
				LastLogin:       timestamppb.New(now.Add(-72 * time.Hour)),
				SessionDuration: durationpb.New(30 * time.Minute),
				Metadata: mustStruct(map[string]interface{}{
					"department": "Marketing",
					"level":      "Manager",
					"region":     "EMEA",
				}),
				Status: pb.Status_STATUS_ACTIVE,
			},
			{
				Id:              3,
				Name:            "Charlie Brown",
				Email:           "charlie@example.com",
				CreatedAt:       timestamppb.New(now.Add(-90 * 24 * time.Hour)), // 3 months ago
				LastLogin:       timestamppb.New(now.Add(-720 * time.Hour)),     // 30 days ago
				SessionDuration: durationpb.New(15 * time.Minute),
				Metadata: mustStruct(map[string]interface{}{
					"department": "Sales",
					"level":      "Junior",
					"quota":      1000000,
				}),
				Status: pb.Status_STATUS_INACTIVE,
			},
		},
		FetchedAt: timestamppb.Now(),
	}

	// Example 1: Filter active users with jq
	fmt.Println("Example 1: Active users with their last login time")
	p1, err := jqyaml.New(
		jqyaml.WithQuery(`.users[] | select(.status == "STATUS_ACTIVE") | {name, email, last_login}`),
		jqyaml.WithProtojsonInput(),
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

	// Example 2: Users who logged in within the last 24 hours
	fmt.Println("\nExample 2: Recently active users (last 24 hours)")
	oneDayAgo := timestamppb.New(now.Add(-24 * time.Hour))
	p2, err := jqyaml.New(
		jqyaml.WithQuery(`.users[] | select(.last_login > $cutoff) | {name, email, last_login, session_duration}`),
		jqyaml.WithProtojsonInput(),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = p2.Execute(context.Background(), userList,
		jqyaml.WithVariables(map[string]interface{}{
			"cutoff": oneDayAgo.AsTime().Format(time.RFC3339),
		}),
		jqyaml.WithWriter(os.Stdout, yamlformat.FormatJSON),
		jqyaml.WithPrettyJSONOutput(),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Example 3: Extract metadata fields
	fmt.Println("\nExample 3: User departments from metadata")
	p3, err := jqyaml.New(
		jqyaml.WithQuery(`.users[] | {name, department: .metadata.department}`),
		jqyaml.WithProtojsonInput(),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = p3.Execute(context.Background(), userList,
		jqyaml.WithWriter(os.Stdout, yamlformat.FormatYAML),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Example 4: Complex query with duration calculations
	fmt.Println("\nExample 4: Users with session duration > 30 minutes")
	p4, err := jqyaml.New(
		jqyaml.WithQuery(`.users[] | select(.session_duration | split("s")[0] | tonumber > 1800) | {name, session_duration}`),
		jqyaml.WithProtojsonInput(),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = p4.Execute(context.Background(), userList,
		jqyaml.WithWriter(os.Stdout, yamlformat.FormatJSON),
		jqyaml.WithCompactJSONOutput(),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Example 5: Working with Any type
	fmt.Println("\n\nExample 5: Activity logs with Any type")

	// Create activity logs with different message types wrapped in Any
	activityLog := &pb.ActivityLog{
		Id:        "log-001",
		Timestamp: timestamppb.Now(),
		Details:   mustAny(userList.Users[0]), // Wrap a User message in Any
	}

	p5, err := jqyaml.New(
		jqyaml.WithQuery(`. | {id, timestamp, details_type: .details."@type"}`),
		jqyaml.WithProtojsonInput(),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = p5.Execute(context.Background(), activityLog,
		jqyaml.WithWriter(os.Stdout, yamlformat.FormatYAML),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Example 6: WithProtojsonInputOptions for custom options
	fmt.Println("\nExample 6: Using custom protojson options (emit unpopulated fields)")
	p6, err := jqyaml.New(
		jqyaml.WithQuery(`.users[0] | {id, name, email, status}`),
		jqyaml.WithProtojsonInputOptions(protojson.MarshalOptions{
			UseProtoNames:   true,
			EmitUnpopulated: true, // Show zero values
			UseEnumNumbers:  true, // Show enum as numbers
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = p6.Execute(context.Background(), userList,
		jqyaml.WithWriter(os.Stdout, yamlformat.FormatJSON),
		jqyaml.WithPrettyJSONOutput(),
	)
	if err != nil {
		log.Fatal(err)
	}
}

// Helper functions
func mustStruct(m map[string]interface{}) *structpb.Struct {
	s, err := structpb.NewStruct(m)
	if err != nil {
		panic(err)
	}
	return s
}

func mustAny(msg proto.Message) *anypb.Any {
	a, err := anypb.New(msg)
	if err != nil {
		panic(err)
	}
	return a
}
