package jqyaml_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	jqyaml "github.com/apstndb/go-jq-yamlformat"
	"github.com/google/go-cmp/cmp"
)

// Define a custom type for testing
type CustomTime struct {
	Year  int
	Month int
	Day   int
}

// Custom marshaler that converts CustomTime to a string
type customTimeMarshaler struct{}

func (m *customTimeMarshaler) Marshal(v interface{}) (interface{}, error) {
	switch val := v.(type) {
	case CustomTime:
		// Convert to ISO date string
		return fmt.Sprintf("%04d-%02d-%02d", val.Year, val.Month, val.Day), nil
	case *CustomTime:
		if val == nil {
			return nil, nil
		}
		return fmt.Sprintf("%04d-%02d-%02d", val.Year, val.Month, val.Day), nil
	case map[string]interface{}:
		// Recursively handle maps
		result := make(map[string]interface{})
		for k, v := range val {
			converted, err := m.Marshal(v)
			if err != nil {
				return nil, err
			}
			result[k] = converted
		}
		return result, nil
	case []interface{}:
		// Recursively handle slices
		result := make([]interface{}, len(val))
		for i, v := range val {
			converted, err := m.Marshal(v)
			if err != nil {
				return nil, err
			}
			result[i] = converted
		}
		return result, nil
	default:
		// For everything else, use JSON round-trip
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
}

// TestInputMarshaler tests the custom input marshaler functionality
func TestInputMarshaler(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		query    string
		expected string
	}{
		{
			name: "custom type conversion",
			input: map[string]interface{}{
				"date": CustomTime{Year: 2023, Month: 12, Day: 25},
			},
			query:    ".date",
			expected: `"2023-12-25"`,
		},
		{
			name: "nested custom type",
			input: map[string]interface{}{
				"events": []interface{}{
					map[string]interface{}{
						"name": "Christmas",
						"date": CustomTime{Year: 2023, Month: 12, Day: 25},
					},
					map[string]interface{}{
						"name": "New Year",
						"date": CustomTime{Year: 2024, Month: 1, Day: 1},
					},
				},
			},
			query:    ".events | map(.date)",
			expected: `["2023-12-25","2024-01-01"]`,
		},
		{
			name: "filter with custom type",
			input: map[string]interface{}{
				"events": []interface{}{
					map[string]interface{}{
						"name": "Event 1",
						"date": CustomTime{Year: 2023, Month: 12, Day: 25},
					},
					map[string]interface{}{
						"name": "Event 2",
						"date": CustomTime{Year: 2024, Month: 1, Day: 1},
					},
				},
			},
			query:    `.events[] | select(.date == "2024-01-01") | .name`,
			expected: `"Event 2"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create pipeline with custom marshaler
			p, err := jqyaml.New(
				jqyaml.WithQuery(tt.query),
				jqyaml.WithInputMarshaler(&customTimeMarshaler{}),
			)
			if err != nil {
				t.Fatalf("failed to create pipeline: %v", err)
			}

			// Execute
			var buf bytes.Buffer
			err = p.Execute(context.Background(), tt.input,
				jqyaml.WithWriter(&buf, jqyaml.FormatJSON),
			)
			if err != nil {
				t.Fatalf("execution failed: %v", err)
			}

			// Compare results
			got := strings.TrimSpace(buf.String())
			want := strings.TrimSpace(tt.expected)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("output mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// Custom marshaler that prefixes strings with "PREFIX:"
type prefixMarshaler struct{}

func (m *prefixMarshaler) Marshal(v interface{}) (interface{}, error) {
	switch val := v.(type) {
	case string:
		return "PREFIX:" + val, nil
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, v := range val {
			converted, err := m.Marshal(v)
			if err != nil {
				return nil, err
			}
			result[k] = converted
		}
		return result, nil
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, v := range val {
			converted, err := m.Marshal(v)
			if err != nil {
				return nil, err
			}
			result[i] = converted
		}
		return result, nil
	default:
		// Use JSON round-trip for other types
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
}

// TestInputMarshalerWithVariables tests custom marshaler with variables
func TestInputMarshalerWithVariables(t *testing.T) {
	// Create pipeline
	p, err := jqyaml.New(
		jqyaml.WithQuery(`. as $data | $filter | map(select(.name == $data.target))[]`),
		jqyaml.WithInputMarshaler(&prefixMarshaler{}),
	)
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	// Test data
	input := map[string]interface{}{
		"target": "item1",
	}

	variables := map[string]interface{}{
		"filter": []interface{}{
			map[string]interface{}{"name": "item1", "value": 10},
			map[string]interface{}{"name": "item2", "value": 20},
		},
	}

	// Execute
	var buf bytes.Buffer
	err = p.Execute(context.Background(), input,
		jqyaml.WithVariables(variables),
		jqyaml.WithWriter(&buf, jqyaml.FormatJSON),
	)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	// Check that the marshaler was applied to both input and variables
	got := strings.TrimSpace(buf.String())
	expected := `{"name":"PREFIX:item1","value":10}`
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("output mismatch (-want +got):\n%s", diff)
	}
}

// Marshaler that returns an error for specific values
type errorMarshaler struct{}

func (m *errorMarshaler) Marshal(v interface{}) (interface{}, error) {
	if s, ok := v.(string); ok && s == "error" {
		return nil, fmt.Errorf("marshaling error for value: %s", s)
	}

	// Handle maps recursively to find error strings
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

	// Handle slices recursively
	if sliceVal, ok := v.([]interface{}); ok {
		result := make([]interface{}, len(sliceVal))
		for i, val := range sliceVal {
			converted, err := m.Marshal(val)
			if err != nil {
				return nil, err
			}
			result[i] = converted
		}
		return result, nil
	}

	// Default behavior
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

// TestInputMarshalerError tests error handling in custom marshaler
func TestInputMarshalerError(t *testing.T) {
	p, err := jqyaml.New(
		jqyaml.WithQuery("."),
		jqyaml.WithInputMarshaler(&errorMarshaler{}),
	)
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	// Test with error-triggering input
	input := map[string]interface{}{
		"field": "error",
	}

	var buf bytes.Buffer
	err = p.Execute(context.Background(), input,
		jqyaml.WithWriter(&buf, jqyaml.FormatJSON),
	)

	// Should get a ConversionError
	if err == nil {
		t.Error("expected error, got nil")
	}
	var convErr *jqyaml.ConversionError
	if !errors.As(err, &convErr) {
		t.Errorf("expected ConversionError, got %T: %v", err, err)
	}
}

// TestWithInputMarshalerValidation tests WithInputMarshaler validation
func TestWithInputMarshalerValidation(t *testing.T) {
	_, err := jqyaml.New(
		jqyaml.WithInputMarshaler(nil),
	)
	if err == nil {
		t.Error("expected error for nil marshaler, got nil")
	}
	if !strings.Contains(err.Error(), "input marshaler cannot be nil") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestWithProtojsonInput tests the protojson input marshaler
func TestWithProtojsonInput(t *testing.T) {
	// Note: This test uses the mock types from the protobuf example
	// In a real scenario, you would use actual protobuf messages

	t.Run("basic functionality", func(t *testing.T) {
		p, err := jqyaml.New(
			jqyaml.WithQuery("."),
			jqyaml.WithProtojsonInput(),
		)
		if err != nil {
			t.Fatalf("failed to create pipeline with WithProtojsonInput: %v", err)
		}

		// Test with a simple map (non-protobuf data should still work)
		data := map[string]interface{}{
			"test":   "value",
			"number": 42,
		}

		var buf bytes.Buffer
		err = p.Execute(context.Background(), data,
			jqyaml.WithWriter(&buf, jqyaml.FormatJSON),
		)
		if err != nil {
			t.Fatalf("execution failed: %v", err)
		}

		expected := `{"number":42,"test":"value"}`
		got := strings.TrimSpace(buf.String())
		if diff := cmp.Diff(expected, got); diff != "" {
			t.Errorf("output mismatch (-want +got):\n%s", diff)
		}
	})
}
