package jqyaml_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	jqyaml "github.com/apstndb/go-jq-yamlformat"
	"github.com/goccy/go-yaml"
)

// TestInputCustomMarshaler verifies that custom marshalers work for input data conversion
func TestInputCustomMarshaler(t *testing.T) {
	// Custom type that needs special conversion for jq processing
	type CustomTime struct {
		time.Time
	}

	// Create test data
	testTime, _ := time.Parse(time.RFC3339, "2024-01-15T10:30:00Z")
	data := map[string]CustomTime{
		"created": {testTime},
		"updated": {testTime.Add(24 * time.Hour)}, // Next day
	}

	// Create pipeline with custom marshaler for CustomTime
	// This converts CustomTime to Unix timestamp for jq processing
	p, err := jqyaml.New(
		jqyaml.WithQuery(`. | to_entries | .[] | select(.value < 1705400000) | .key`),
		jqyaml.WithDefaultEncodeOptions(
			yaml.CustomMarshaler[CustomTime](func(ct CustomTime) ([]byte, error) {
				// Convert to Unix timestamp for easier numeric comparison
				return []byte(fmt.Sprintf("%d", ct.Unix())), nil
			}),
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err = p.Execute(context.Background(), data, jqyaml.WithWriter(&buf, jqyaml.FormatJSON))
	if err != nil {
		t.Fatal(err)
	}

	// The query should only select "created" (before 2024-01-16)
	want := `"created"` + "\n"
	if got := buf.String(); got != want {
		t.Errorf("output mismatch\ngot:  %q\nwant: %q", got, want)
	}
}

// TestInputAndOutputCustomMarshalers verifies that input and output marshalers can be different
func TestInputAndOutputCustomMarshalers(t *testing.T) {
	type Money struct {
		Amount   int
		Currency string
	}

	data := []Money{
		{Amount: 100, Currency: "USD"},
		{Amount: 200, Currency: "EUR"},
	}

	// Pipeline with different marshalers for input and output
	p, err := jqyaml.New(
		// Query filters by amount field (which is converted to "amount_cents" by input marshaler)
		jqyaml.WithQuery(`.[] | select(.amount_cents > 150)`),
		jqyaml.WithDefaultEncodeOptions(
			// Input marshaler: convert Money to a different structure for querying
			yaml.CustomMarshaler[Money](func(m Money) ([]byte, error) {
				// Convert to a structure that's easier to query
				return []byte(fmt.Sprintf(`{"amount_cents": %d, "currency": "%s"}`, m.Amount, m.Currency)), nil
			}),
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Execute with additional output marshaler
	var buf bytes.Buffer
	err = p.Execute(context.Background(), data,
		jqyaml.WithWriter(&buf, jqyaml.FormatJSON),
		jqyaml.WithEncodeOptions(
			// Output marshaler: format the result differently
			yaml.CustomMarshaler[map[string]interface{}](func(m map[string]interface{}) ([]byte, error) {
				// Format as a string representation in a type-safe way
				// Note: amount_cents could be int or float64 depending on the JSON unmarshaling path
				amount, okA := m["amount_cents"]
				currency, okC := m["currency"].(string)
				if !okA || !okC {
					return nil, fmt.Errorf("marshaler expected keys 'amount_cents' and 'currency' (string), but got: %+v", m)
				}
				
				// Handle both int and float64 cases
				var amountInt int
				switch v := amount.(type) {
				case int:
					amountInt = v
				case float64:
					amountInt = int(v)
				default:
					return nil, fmt.Errorf("amount_cents has unexpected type %T", v)
				}
				
				return []byte(fmt.Sprintf(`"%d %s"`, amountInt, currency)), nil
			}),
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	want := `"200 EUR"` + "\n"
	if got := buf.String(); got != want {
		t.Errorf("output mismatch\ngot:  %q\nwant: %q", got, want)
	}
}
