package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/apstndb/go-jq-yamlformat"
	"github.com/apstndb/go-yamlformat"
)

func main() {
	// Sample event data
	data := map[string]interface{}{
		"events": []map[string]interface{}{
			{
				"id":        "evt-001",
				"type":      "login",
				"user":      "alice",
				"timestamp": time.Now().Add(-2 * time.Hour),
				"success":   true,
			},
			{
				"id":        "evt-002",
				"type":      "purchase",
				"user":      "bob",
				"timestamp": time.Now().Add(-30 * time.Minute),
				"amount":    49.99,
			},
			{
				"id":        "evt-003",
				"type":      "login",
				"user":      "charlie",
				"timestamp": time.Now().Add(-5 * time.Minute),
				"success":   false,
			},
			{
				"id":        "evt-004",
				"type":      "logout",
				"user":      "alice",
				"timestamp": time.Now().Add(-1 * time.Hour),
			},
		},
	}

	// Example 1: Filter events by time
	log.Println("Example 1: Recent events (last hour)")
	p1, err := jqyaml.New(
		jqyaml.WithQuery(".events[] | select(.timestamp > $since)"),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = p1.Execute(context.Background(), data,
		jqyaml.WithWriter(os.Stdout, yamlformat.FormatYAML),
		jqyaml.WithVariables(map[string]interface{}{
			"since": time.Now().Add(-1 * time.Hour),
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Example 2: Filter by event type
	log.Println("\nExample 2: Login events")
	p2, err := jqyaml.New(
		jqyaml.WithQuery(".events[] | select(.type == $eventType)"),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = p2.Execute(context.Background(), data,
		jqyaml.WithWriter(os.Stdout, yamlformat.FormatJSON),
		jqyaml.WithVariables(map[string]interface{}{
			"eventType": "login",
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Example 3: Complex filtering with multiple variables
	log.Println("\n\nExample 3: Filtered summary")
	p3, err := jqyaml.New(
		jqyaml.WithQuery(`{
			period: {from: $from, to: $to},
			events: [.events[] | select(.timestamp >= $from and .timestamp <= $to)],
			count: [.events[] | select(.timestamp >= $from and .timestamp <= $to)] | length
		}`),
	)
	if err != nil {
		log.Fatal(err)
	}

	now := time.Now()
	err = p3.Execute(context.Background(), data,
		jqyaml.WithWriter(os.Stdout, yamlformat.FormatYAML),
		jqyaml.WithVariables(map[string]interface{}{
			"from": now.Add(-2 * time.Hour),
			"to":   now,
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Example 4: Using Go structs as variables
	log.Println("\nExample 4: Struct as variable")
	type Filter struct {
		Types     []string `json:"types"`
		MinAmount float64  `json:"min_amount"`
	}

	p4, err := jqyaml.New(
		jqyaml.WithQuery(".events[] | select(.type as $t | $filter.types | contains([$t])) | select(.amount == null or .amount >= $filter.min_amount)"),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = p4.Execute(context.Background(), data,
		jqyaml.WithWriter(os.Stdout, yamlformat.FormatYAML),
		jqyaml.WithVariables(map[string]interface{}{
			"filter": Filter{
				Types:     []string{"purchase", "subscription"},
				MinAmount: 25.0,
			},
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
}