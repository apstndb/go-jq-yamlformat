package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/apstndb/go-jq-yamlformat"
	"github.com/apstndb/go-yamlformat"
)

func main() {
	data := map[string]interface{}{
		"users": []map[string]interface{}{
			{"id": 1, "name": "Alice"},
			{"id": 2, "name": "Bob"},
		},
	}

	// Example 1: Invalid query syntax
	log.Println("Example 1: Invalid query syntax")
	_, err := jqyaml.New(
		jqyaml.WithQuery(".users[] | select(.name =="), // Missing closing quote
	)
	if err != nil {
		var queryErr *jqyaml.QueryError
		if errors.As(err, &queryErr) {
			log.Printf("Query Error: %v\n", queryErr)
			log.Printf("  Query: %s\n", queryErr.Query)
			log.Printf("  Message: %s\n", queryErr.Message)
			log.Printf("  Underlying: %v\n", queryErr.Err)
		}
	}

	// Example 2: Missing encoder
	log.Println("\nExample 2: Missing encoder")
	p, err := jqyaml.New(
		jqyaml.WithQuery(".users[]"),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = p.Execute(context.Background(), data)
	// No WithWriter or WithEncoder specified
	if err != nil {
		log.Printf("Execution Error: %v\n", err)
	}

	// Example 3: Query execution error
	log.Println("\nExample 3: Query execution error")
	p2, err := jqyaml.New(
		jqyaml.WithQuery(".users | .nonexistent.field"),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = p2.Execute(context.Background(), data,
		jqyaml.WithWriter(os.Stdout, yamlformat.FormatJSON),
	)
	if err != nil {
		log.Printf("Query Execution Error: %v\n", err)
	}

	// Example 4: Type conversion error
	log.Println("\nExample 4: Type conversion error")
	// Create a type that will fail to marshal
	type BadType struct {
		Chan chan int `json:"chan"` // Channels cannot be marshaled
	}

	badData := map[string]interface{}{
		"bad": BadType{Chan: make(chan int)},
	}

	p3, err := jqyaml.New(
		jqyaml.WithQuery(".bad"),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = p3.Execute(context.Background(), badData,
		jqyaml.WithWriter(os.Stdout, yamlformat.FormatJSON),
	)
	if err != nil {
		var convErr *jqyaml.ConversionError
		if errors.As(err, &convErr) {
			log.Printf("Conversion Error: %v\n", convErr)
			log.Printf("  Value Type: %T\n", convErr.Value)
			log.Printf("  Target: %s\n", convErr.Type)
		}
	}

	// Example 5: Timeout error
	log.Println("\nExample 5: Timeout error")
	p4, err := jqyaml.New(
		jqyaml.WithQuery("while(true; .+1)"), // Infinite loop
	)
	if err != nil {
		log.Fatal(err)
	}

	err = p4.Execute(context.Background(), 0,
		jqyaml.WithWriter(os.Stdout, yamlformat.FormatJSON),
		jqyaml.WithTimeout(100*time.Millisecond),
	)
	if err != nil {
		var timeoutErr *jqyaml.TimeoutError
		if errors.As(err, &timeoutErr) {
			log.Printf("Timeout Error: %v\n", timeoutErr)
			log.Printf("  Duration: %v\n", timeoutErr.Duration)
		}
	}

	// Example 6: Proper error handling in production
	log.Println("\nExample 6: Production error handling")
	if err := processData(data); err != nil {
		handleError(err)
	}
}

func processData(data interface{}) error {
	p, err := jqyaml.New(
		jqyaml.WithQuery(".users[] | {name, id}"),
	)
	if err != nil {
		return fmt.Errorf("failed to create pipeline: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = p.Execute(ctx, data,
		jqyaml.WithWriter(os.Stdout, yamlformat.FormatYAML),
	)
	if err != nil {
		return fmt.Errorf("failed to execute pipeline: %w", err)
	}

	return nil
}

func handleError(err error) {
	log.Printf("Error occurred: %v\n", err)

	// Check for specific error types
	var queryErr *jqyaml.QueryError
	var convErr *jqyaml.ConversionError
	var timeoutErr *jqyaml.TimeoutError

	switch {
	case errors.As(err, &queryErr):
		log.Println("This is a query-related error. Check your jq syntax.")
	case errors.As(err, &convErr):
		log.Println("This is a data conversion error. Check your data types.")
	case errors.As(err, &timeoutErr):
		log.Println("The operation timed out. Consider increasing the timeout or optimizing the query.")
	case errors.Is(err, context.DeadlineExceeded):
		log.Println("Context deadline exceeded.")
	case errors.Is(err, context.Canceled):
		log.Println("Context was canceled.")
	default:
		log.Println("An unexpected error occurred.")
	}
}