package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/apstndb/go-jq-yamlformat"
	"github.com/apstndb/go-yamlformat"
	"github.com/goccy/go-yaml"
)

// Custom types for demonstration
type (
	UserID   string
	Currency struct {
		Amount   float64
		Code     string
	}
	Transaction struct {
		ID        string    `json:"id"`
		UserID    UserID    `json:"user_id"`
		Amount    Currency  `json:"amount"`
		Timestamp time.Time `json:"timestamp"`
		Notes     string    `json:"notes,omitempty"`
	}
)

func main() {
	// Sample data with custom types
	data := map[string]interface{}{
		"transactions": []Transaction{
			{
				ID:        "txn-001",
				UserID:    "user-alice",
				Amount:    Currency{Amount: 150.50, Code: "USD"},
				Timestamp: time.Now().Add(-24 * time.Hour),
				Notes:     "Monthly subscription",
			},
			{
				ID:        "txn-002",
				UserID:    "user-bob",
				Amount:    Currency{Amount: 2500.00, Code: "EUR"},
				Timestamp: time.Now().Add(-12 * time.Hour),
			},
			{
				ID:        "txn-003",
				UserID:    "user-alice",
				Amount:    Currency{Amount: 75.25, Code: "USD"},
				Timestamp: time.Now().Add(-1 * time.Hour),
				Notes:     "Product purchase",
			},
		},
		"stats": map[string]interface{}{
			"total_volume": big.NewInt(1000000),
			"active_users": 42,
		},
	}

	// Create pipeline with custom marshalers
	p, err := jqyaml.New(
		jqyaml.WithQuery(`{
			transactions: .transactions,
			summary: {
				total_transactions: .transactions | length,
				total_volume: .stats.total_volume,
				by_currency: .transactions | group_by(.amount.Code) | map({
					currency: .[0].amount.Code,
					count: length,
					total: map(.amount.Amount) | add
				})
			}
		}`),
		jqyaml.WithDefaultEncodeOptions(
			// Custom marshaler for time.Time
			yaml.CustomMarshaler[time.Time](func(t time.Time) ([]byte, error) {
				return []byte(strconv.Quote(t.Format(time.RFC3339))), nil
			}),
			// Custom marshaler for UserID
			yaml.CustomMarshaler[UserID](func(id UserID) ([]byte, error) {
				return []byte(fmt.Sprintf(`"USER:%s"`, string(id))), nil
			}),
			// Custom marshaler for Currency
			yaml.CustomMarshaler[Currency](func(c Currency) ([]byte, error) {
				return []byte(fmt.Sprintf(`{"amount": %.2f, "code": "%s", "formatted": "%.2f %s"}`, 
					c.Amount, c.Code, c.Amount, c.Code)), nil
			}),
			// Custom marshaler for big.Int
			yaml.CustomMarshaler[*big.Int](func(n *big.Int) ([]byte, error) {
				return []byte(fmt.Sprintf(`"%s"`, n.String())), nil
			}),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Example 1: Output as YAML with custom formatting
	log.Println("Example 1: YAML output with custom types")
	err = p.Execute(context.Background(), data,
		jqyaml.WithWriter(os.Stdout, yamlformat.FormatYAML),
		jqyaml.WithEncodeOptions(
			yaml.Indent(2),
			yaml.UseLiteralStyleIfMultiline(true),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Example 2: Output as JSON
	log.Println("\nExample 2: JSON output with custom types")
	err = p.Execute(context.Background(), data,
		jqyaml.WithWriter(os.Stdout, yamlformat.FormatJSON),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Example 3: Filter with custom type awareness
	log.Println("\nExample 3: Filtering by currency")
	p2, err := jqyaml.New(
		jqyaml.WithQuery(`.transactions[] | select(.amount.Code == "USD")`),
		jqyaml.WithDefaultEncodeOptions(
			yaml.CustomMarshaler[time.Time](func(t time.Time) ([]byte, error) {
				// Different format for this example
				return []byte(strconv.Quote(t.Format("2006-01-02 15:04:05"))), nil
			}),
			yaml.CustomMarshaler[Currency](func(c Currency) ([]byte, error) {
				// Simplified format
				return []byte(fmt.Sprintf(`"$%.2f"`, c.Amount)), nil
			}),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = p2.Execute(context.Background(), data,
		jqyaml.WithWriter(os.Stdout, yamlformat.FormatYAML),
	)
	if err != nil {
		log.Fatal(err)
	}
}