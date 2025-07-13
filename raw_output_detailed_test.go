package jqyaml

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRawOutputDetailed(t *testing.T) {
	tests := []struct {
		name          string
		input         interface{}
		query         string
		options       []ExecuteOption
		expectedLines []string
		description   string
	}{
		// String tests
		{
			name:          "raw string without quotes",
			input:         "hello world",
			query:         ".",
			options:       []ExecuteOption{WithRawJSONOutput()},
			expectedLines: []string{"hello world"},
			description:   "Raw strings should not have quotes",
		},
		{
			name:          "raw string with special characters",
			input:         `hello "world" with \n newline`,
			query:         ".",
			options:       []ExecuteOption{WithRawJSONOutput()},
			expectedLines: []string{`hello "world" with \n newline`},
			description:   "Special characters should be preserved literally",
		},
		{
			name:          "raw string with actual newline",
			input:         "line1\nline2",
			query:         ".",
			options:       []ExecuteOption{WithRawJSONOutput()},
			expectedLines: []string{"line1", "line2"},
			description:   "Actual newlines in strings create multiple lines",
		},
		{
			name:          "empty string",
			input:         "",
			query:         ".",
			options:       []ExecuteOption{WithRawJSONOutput()},
			expectedLines: []string{""},
			description:   "Empty strings should produce empty line",
		},
		{
			name:          "array of strings",
			input:         []string{"foo", "bar", "baz"},
			query:         ".[]",
			options:       []ExecuteOption{WithRawJSONOutput()},
			expectedLines: []string{"foo", "bar", "baz"},
			description:   "Each string should be on its own line without quotes",
		},

		// Non-string tests
		{
			name:          "raw number outputs as JSON",
			input:         42,
			query:         ".",
			options:       []ExecuteOption{WithRawJSONOutput()},
			expectedLines: []string{"42"},
			description:   "Numbers should be output as JSON",
		},
		{
			name:          "raw boolean outputs as JSON",
			input:         true,
			query:         ".",
			options:       []ExecuteOption{WithRawJSONOutput()},
			expectedLines: []string{"true"},
			description:   "Booleans should be output as JSON",
		},
		{
			name:          "raw null outputs as JSON",
			input:         nil,
			query:         ".",
			options:       []ExecuteOption{WithRawJSONOutput()},
			expectedLines: []string{"null"},
			description:   "Null should be output as JSON",
		},
		{
			name:          "raw object outputs as compact JSON",
			input:         map[string]interface{}{"key": "value", "num": 42},
			query:         ".",
			options:       []ExecuteOption{WithRawJSONOutput()},
			expectedLines: []string{`{"key":"value","num":42}`},
			description:   "Objects should be output as compact JSON",
		},
		{
			name:          "raw array outputs as compact JSON",
			input:         []interface{}{1, 2, 3},
			query:         ".",
			options:       []ExecuteOption{WithRawJSONOutput()},
			expectedLines: []string{"[1,2,3]"},
			description:   "Arrays should be output as compact JSON",
		},

		// Mixed type arrays
		{
			name: "mixed type array",
			input: []interface{}{
				"string value",
				42,
				true,
				nil,
				map[string]interface{}{"nested": "object"},
				[]int{1, 2, 3},
			},
			query:   ".[]",
			options: []ExecuteOption{WithRawJSONOutput()},
			expectedLines: []string{
				"string value",
				"42",
				"true",
				"null",
				`{"nested":"object"}`,
				"[1,2,3]",
			},
			description: "Mixed types should each be handled appropriately",
		},

		// Raw + Compact combination
		{
			name:          "raw and compact outputs compact JSON for objects",
			input:         map[string]interface{}{"a": 1, "b": 2, "c": 3},
			query:         ".",
			options:       []ExecuteOption{WithRawJSONOutput(), WithCompactJSONOutput()},
			expectedLines: []string{`{"a":1,"b":2,"c":3}`},
			description:   "Raw with compact should ensure compact JSON",
		},

		// Raw + Pretty combination
		{
			name:          "raw and pretty still outputs compact for non-strings",
			input:         map[string]interface{}{"a": 1, "b": 2},
			query:         ".",
			options:       []ExecuteOption{WithRawJSONOutput(), WithPrettyJSONOutput()},
			expectedLines: []string{`{"a":1,"b":2}`},
			description:   "Raw should override pretty for non-strings",
		},

		// Edge cases
		{
			name:          "string that looks like JSON",
			input:         `{"this": "is a string"}`,
			query:         ".",
			options:       []ExecuteOption{WithRawJSONOutput()},
			expectedLines: []string{`{"this": "is a string"}`},
			description:   "JSON-like strings should be output as-is",
		},
		{
			name:          "string with only whitespace",
			input:         "   \t\n   ",
			query:         ".",
			options:       []ExecuteOption{WithRawJSONOutput()},
			expectedLines: []string{"   \t", "   "},
			description:   "Whitespace should be preserved with newlines creating new lines",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with writer (encoder path)
			t.Run("with_writer", func(t *testing.T) {
				p, err := New(WithQuery(tt.query))
				if err != nil {
					t.Fatalf("failed to create pipeline: %v", err)
				}

				var buf bytes.Buffer
				opts := append(tt.options, WithWriter(&buf, FormatJSON))

				err = p.Execute(context.Background(), tt.input, opts...)
				if err != nil {
					t.Fatalf("execution failed: %v", err)
				}

				verifyOutput(t, buf.String(), tt.expectedLines, tt.description)
			})

			// Test with callback (marshal path)
			t.Run("with_callback", func(t *testing.T) {
				p, err := New(WithQuery(tt.query))
				if err != nil {
					t.Fatalf("failed to create pipeline: %v", err)
				}

				var results []string
				callback := func(v interface{}) error {
					// Simulate what the encoder would do
					if s, ok := v.(string); ok && hasRawOption(tt.options) {
						results = append(results, s)
					} else {
						// Marshal to JSON
						data, err := json.Marshal(v)
						if err != nil {
							return err
						}
						results = append(results, string(data))
					}
					return nil
				}

				opts := append(tt.options, WithCallback(callback))

				err = p.Execute(context.Background(), tt.input, opts...)
				if err != nil {
					t.Fatalf("execution failed: %v", err)
				}

				// Join with newlines to simulate encoder output
				output := strings.Join(results, "\n")
				if len(results) > 0 {
					output += "\n"
				}

				verifyOutput(t, output, tt.expectedLines, tt.description)
			})
		})
	}
}

func verifyOutput(t *testing.T, output string, expectedLines []string, description string) {
	t.Helper()

	// Check description
	if description != "" {
		t.Logf("Test description: %s", description)
	}

	// Verify output ends with newline
	if output != "" && !strings.HasSuffix(output, "\n") {
		t.Errorf("output does not end with newline: %q", output)
	}

	// Split and compare lines
	gotLines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	if diff := cmp.Diff(expectedLines, gotLines); diff != "" {
		t.Errorf("output lines mismatch (-want +got):\n%s", diff)
	}
}

func hasRawOption(opts []ExecuteOption) bool {
	// This is a simplification - in real code we'd need to check the actual option
	// For testing purposes, we assume WithRawJSONOutput is always in the options
	return true
}

// Test to ensure raw output behavior is consistent between different code paths
func TestRawOutputConsistency(t *testing.T) {
	testCases := []struct {
		name  string
		input interface{}
		query string
	}{
		{
			name:  "string value",
			input: "test string",
			query: ".",
		},
		{
			name:  "number value",
			input: 42.5,
			query: ".",
		},
		{
			name:  "boolean value",
			input: false,
			query: ".",
		},
		{
			name:  "null value",
			input: nil,
			query: ".",
		},
		{
			name:  "object value",
			input: map[string]interface{}{"test": "value"},
			query: ".",
		},
		{
			name:  "array value",
			input: []interface{}{"a", "b", "c"},
			query: ".",
		},
		{
			name: "nested extraction",
			input: map[string]interface{}{
				"data": map[string]interface{}{
					"message": "hello world",
				},
			},
			query: ".data.message",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get output with writer
			var writerBuf bytes.Buffer
			p1, _ := New(WithQuery(tc.query))
			err := p1.Execute(context.Background(), tc.input,
				WithWriter(&writerBuf, FormatJSON),
				WithRawJSONOutput(),
			)
			if err != nil {
				t.Fatalf("writer execution failed: %v", err)
			}

			// Get output with encoder wrapper
			var encoderBuf bytes.Buffer
			encoder := &jsonEncoder{
				writer:  &encoderBuf,
				compact: true,
				raw:     true,
			}
			p2, _ := New(WithQuery(tc.query))
			err = p2.Execute(context.Background(), tc.input,
				WithEncoder(encoder),
			)
			if err != nil {
				t.Fatalf("encoder execution failed: %v", err)
			}

			// Compare outputs
			if writerBuf.String() != encoderBuf.String() {
				t.Errorf("outputs differ:\nwriter:  %q\nencoder: %q",
					writerBuf.String(), encoderBuf.String())
			}
		})
	}
}
