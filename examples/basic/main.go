package main

import (
	"context"
	"log"
	"os"

	jqyaml "github.com/apstndb/go-jq-yamlformat"
	"github.com/apstndb/go-yamlformat"
)

func main() {
	// Sample data
	data := map[string]interface{}{
		"users": []map[string]interface{}{
			{"id": 1, "name": "Alice", "active": true, "score": 95.5},
			{"id": 2, "name": "Bob", "active": false, "score": 87.3},
			{"id": 3, "name": "Charlie", "active": true, "score": 92.1},
		},
		"metadata": map[string]interface{}{
			"version":   "1.0",
			"generated": "2024-01-15",
		},
	}

	// Example 1: Simple filtering
	log.Println("Example 1: Active users")
	p1, err := jqyaml.New(
		jqyaml.WithQuery(".users[] | select(.active)"),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = p1.Execute(context.Background(), data,
		jqyaml.WithWriter(os.Stdout, yamlformat.FormatYAML),
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("\nExample 2: Top scorer")
	p2, err := jqyaml.New(
		jqyaml.WithQuery(".users | max_by(.score)"),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = p2.Execute(context.Background(), data,
		jqyaml.WithWriter(os.Stdout, yamlformat.FormatJSON),
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("\n\nExample 3: Transform data")
	p3, err := jqyaml.New(
		jqyaml.WithQuery(`{
			active_users: [.users[] | select(.active) | .name],
			average_score: (.users | map(.score) | add / length),
			metadata
		}`),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = p3.Execute(context.Background(), data,
		jqyaml.WithWriter(os.Stdout, yamlformat.FormatYAML),
	)
	if err != nil {
		log.Fatal(err)
	}
}
