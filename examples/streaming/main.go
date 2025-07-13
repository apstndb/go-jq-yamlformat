package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	jqyaml "github.com/apstndb/go-jq-yamlformat"
	"github.com/apstndb/go-yamlformat"
	"github.com/goccy/go-yaml"
)

func main() {
	// Generate large dataset
	var items []map[string]interface{}
	for i := 0; i < 1000; i++ {
		items = append(items, map[string]interface{}{
			"id":        fmt.Sprintf("item-%04d", i),
			"value":     i * 10,
			"category":  []string{"A", "B", "C", "D"}[i%4],
			"active":    i%2 == 0,
			"timestamp": time.Now().Add(-time.Duration(i) * time.Minute),
		})
	}

	data := map[string]interface{}{
		"items": items,
		"metadata": map[string]interface{}{
			"total_count": len(items),
			"generated":   time.Now(),
		},
	}

	// Example 1: Stream all items
	log.Println("Example 1: Streaming all items (first 5 shown)")
	p1, err := jqyaml.New(
		jqyaml.WithQuery(".items[]"),
		jqyaml.WithDefaultEncodeOptions(
			yaml.CustomMarshaler[time.Time](func(t time.Time) ([]byte, error) {
				return []byte(strconv.Quote(t.Format(time.RFC3339))), nil
			}),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	count := 0
	encoder := yamlformat.NewEncoder(os.Stdout,
		yaml.Indent(2),
	)

	err = p1.Execute(context.Background(), data,
		jqyaml.WithCallback(func(item interface{}) error {
			if count < 5 { // Only show first 5 items
				if count > 0 {
					fmt.Println("---") // YAML document separator
				}
				if err := encoder.Encode(item); err != nil {
					return err
				}
			}
			count++
			return nil
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Total items processed: %d\n", count)

	// Example 2: Stream with filtering and transformation
	log.Println("\nExample 2: Stream active items from category A or B")
	p2, err := jqyaml.New(
		jqyaml.WithQuery(`.items[] | select(.active and (.category == "A" or .category == "B")) | {id, value, category}`),
	)
	if err != nil {
		log.Fatal(err)
	}

	activeCount := 0
	totalValue := 0
	err = p2.Execute(context.Background(), data,
		jqyaml.WithCallback(func(item interface{}) error {
			if m, ok := item.(map[string]interface{}); ok {
				activeCount++
				if v, ok := m["value"].(float64); ok {
					totalValue += int(v)
				}
			}
			return nil
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Active items in categories A/B: %d, Total value: %d\n", activeCount, totalValue)

	// Example 3: Stream with timeout
	log.Println("\nExample 3: Stream with timeout (simulated slow processing)")
	p3, err := jqyaml.New(
		jqyaml.WithQuery(".items[]"),
	)
	if err != nil {
		log.Fatal(err)
	}

	processed := 0
	err = p3.Execute(context.Background(), data,
		jqyaml.WithCallback(func(item interface{}) error {
			// Simulate slow processing
			time.Sleep(10 * time.Millisecond)
			processed++
			return nil
		}),
		jqyaml.WithTimeout(100*time.Millisecond),
	)
	if err != nil {
		if _, ok := err.(*jqyaml.TimeoutError); ok {
			log.Printf("Processing timed out after processing %d items\n", processed)
		} else {
			log.Fatal(err)
		}
	}

	// Example 4: Stream to file
	log.Println("\nExample 4: Stream to file")
	file, err := os.Create("output.jsonl")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	p4, err := jqyaml.New(
		jqyaml.WithQuery(`.items[] | select(.value > 5000)`),
	)
	if err != nil {
		log.Fatal(err)
	}

	jsonEncoder := yamlformat.NewJSONEncoder(file)
	fileCount := 0
	err = p4.Execute(context.Background(), data,
		jqyaml.WithCallback(func(item interface{}) error {
			fileCount++
			return jsonEncoder.Encode(item)
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Wrote %d items to output.jsonl\n", fileCount)
}
