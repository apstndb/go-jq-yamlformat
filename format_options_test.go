package jqyaml_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/apstndb/go-jq-yamlformat"
	"github.com/google/go-cmp/cmp"
)

// Helper to create a pointer to bool
func ptrBool(b bool) *bool {
	return &b
}

func TestCompactOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		query    string
		format   jqyaml.Format
		compact  *bool // nil means no option, true means WithCompactOutput, false means WithPrettyOutput
		expected string
	}{
		{
			name: "compact JSON output",
			input: map[string]interface{}{
				"users": []map[string]interface{}{
					{"id": 1, "name": "Alice"},
					{"id": 2, "name": "Bob"},
				},
			},
			query:    ".users[]",
			format:   jqyaml.FormatJSON,
			compact:  ptrBool(true),
			expected: `{"id":1,"name":"Alice"}` + "\n" + `{"id":2,"name":"Bob"}` + "\n",
		},
		{
			name: "pretty JSON output",
			input: map[string]interface{}{
				"id":   1,
				"name": "Alice",
			},
			query:    ".",
			format:   jqyaml.FormatJSON,
			compact:  ptrBool(false),
			expected: `{
  "id": 1,
  "name": "Alice"
}
`,
		},
		{
			name: "default JSON output (go-yamlformat default)",
			input: map[string]interface{}{
				"id":   1,
				"name": "Alice",
			},
			query:    ".",
			format:   jqyaml.FormatJSON,
			compact:  nil, // No explicit option, use default
			expected: `{"id": 1, "name": "Alice"}` + "\n",
		},
		{
			name: "compact option ignored for YAML",
			input: map[string]interface{}{
				"id":   1,
				"name": "Alice",
			},
			query:    ".",
			format:   jqyaml.FormatYAML,
			compact:  ptrBool(true),
			expected: "id: 1\nname: Alice\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := jqyaml.New(jqyaml.WithQuery(tt.query))
			if err != nil {
				t.Fatalf("failed to create pipeline: %v", err)
			}

			var buf bytes.Buffer
			opts := []jqyaml.ExecuteOption{
				jqyaml.WithWriter(&buf, tt.format),
			}
			if tt.compact != nil {
				if *tt.compact {
					opts = append(opts, jqyaml.WithCompactOutput())
				} else {
					opts = append(opts, jqyaml.WithPrettyOutput())
				}
			}

			err = p.Execute(context.Background(), tt.input, opts...)
			if err != nil {
				t.Fatalf("execution failed: %v", err)
			}

			got := buf.String()
			if diff := cmp.Diff(tt.expected, got); diff != "" {
				t.Errorf("output mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRawOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		query    string
		format   jqyaml.Format
		raw      bool
		expected string
	}{
		{
			name:     "raw string output",
			input:    map[string]interface{}{"message": "Hello, World!"},
			query:    ".message",
			format:   jqyaml.FormatJSON,
			raw:      true,
			expected: "Hello, World!",
		},
		{
			name:     "normal string output",
			input:    map[string]interface{}{"message": "Hello, World!"},
			query:    ".message",
			format:   jqyaml.FormatJSON,
			raw:      false,
			expected: "\"Hello, World!\"\n",
		},
		{
			name: "raw output with multiple strings",
			input: map[string]interface{}{
				"items": []string{"apple", "banana", "cherry"},
			},
			query:    ".items[]",
			format:   jqyaml.FormatJSON,
			raw:      true,
			expected: "apple\nbanana\ncherry",
		},
		{
			name: "raw output with non-string falls back to JSON",
			input: map[string]interface{}{
				"number": 42,
				"string": "test",
			},
			query:    ".number, .string",
			format:   jqyaml.FormatJSON,
			raw:      true,
			expected: "42\ntest",
		},
		{
			name:     "raw option ignored for YAML",
			input:    map[string]interface{}{"message": "Hello, World!"},
			query:    ".message",
			format:   jqyaml.FormatYAML,
			raw:      true,
			expected: "Hello, World!\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := jqyaml.New(jqyaml.WithQuery(tt.query))
			if err != nil {
				t.Fatalf("failed to create pipeline: %v", err)
			}

			var buf bytes.Buffer
			opts := []jqyaml.ExecuteOption{
				jqyaml.WithWriter(&buf, tt.format),
			}
			if tt.raw {
				opts = append(opts, jqyaml.WithRawOutput())
			}

			err = p.Execute(context.Background(), tt.input, opts...)
			if err != nil {
				t.Fatalf("execution failed: %v", err)
			}

			got := buf.String()
			if diff := cmp.Diff(tt.expected, got); diff != "" {
				t.Errorf("output mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCombinedCompactAndRawOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		query    string
		compact  bool
		raw      bool
		expected string
	}{
		{
			name: "compact and raw with objects",
			input: map[string]interface{}{
				"items": []map[string]interface{}{
					{"type": "fruit", "name": "apple"},
					{"type": "vegetable", "name": "carrot"},
				},
			},
			query:    ".items[]",
			compact:  true,
			raw:      false,
			expected: `{"name":"apple","type":"fruit"}` + "\n" + `{"name":"carrot","type":"vegetable"}` + "\n",
		},
		{
			name: "compact and raw with strings",
			input: map[string]interface{}{
				"lines": []string{"line 1", "line 2", "line 3"},
			},
			query:    ".lines[]",
			compact:  true,
			raw:      true,
			expected: "line 1\nline 2\nline 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := jqyaml.New(jqyaml.WithQuery(tt.query))
			if err != nil {
				t.Fatalf("failed to create pipeline: %v", err)
			}

			var buf bytes.Buffer
			opts := []jqyaml.ExecuteOption{
				jqyaml.WithWriter(&buf, jqyaml.FormatJSON),
			}
			if tt.compact {
				opts = append(opts, jqyaml.WithCompactOutput())
			}
			if tt.raw {
				opts = append(opts, jqyaml.WithRawOutput())
			}

			err = p.Execute(context.Background(), tt.input, opts...)
			if err != nil {
				t.Fatalf("execution failed: %v", err)
			}

			got := buf.String()
			if diff := cmp.Diff(tt.expected, got); diff != "" {
				t.Errorf("output mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatOptionsWithCallback(t *testing.T) {
	// Test that compact/raw options work with callback mode
	p, err := jqyaml.New(jqyaml.WithQuery(".items[]"))
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	input := map[string]interface{}{
		"items": []string{"a", "b", "c"},
	}

	var results []string
	err = p.Execute(context.Background(), input,
		jqyaml.WithCallback(func(v interface{}) error {
			// In callback mode, the raw/compact options don't apply
			// The callback receives the raw Go values
			if s, ok := v.(string); ok {
				results = append(results, s)
			}
			return nil
		}),
		jqyaml.WithCompactOutput(), // Should be ignored in callback mode
		jqyaml.WithRawOutput(),     // Should be ignored in callback mode
	)

	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	expected := []string{"a", "b", "c"}
	if diff := cmp.Diff(expected, results); diff != "" {
		t.Errorf("results mismatch (-want +got):\n%s", diff)
	}
}

func TestFormatOptionsDocumentation(t *testing.T) {
	// This test verifies the documented behavior of format options
	
	t.Run("WithCompactOutput documentation", func(t *testing.T) {
		// Verify that WithCompactOutput only affects JSON output
		p, err := jqyaml.New(jqyaml.WithQuery("."))
		if err != nil {
			t.Fatal(err)
		}
		
		data := map[string]interface{}{"test": true}
		
		// JSON with compact
		var jsonBuf bytes.Buffer
		err = p.Execute(context.Background(), data,
			jqyaml.WithWriter(&jsonBuf, jqyaml.FormatJSON),
			jqyaml.WithCompactOutput(),
		)
		if err != nil {
			t.Fatal(err)
		}
		
		jsonOutput := jsonBuf.String()
		if strings.Contains(jsonOutput, "\n  ") {
			t.Error("JSON output should be compact (no indentation)")
		}
		
		// YAML with compact (should be ignored)
		var yamlBuf bytes.Buffer
		err = p.Execute(context.Background(), data,
			jqyaml.WithWriter(&yamlBuf, jqyaml.FormatYAML),
			jqyaml.WithCompactOutput(),
		)
		if err != nil {
			t.Fatal(err)
		}
		
		// YAML output should be the same regardless of compact option
		if yamlBuf.String() != "test: true\n" {
			t.Error("YAML output should not be affected by WithCompactOutput")
		}
	})
	
	t.Run("WithRawOutput documentation", func(t *testing.T) {
		// Verify that WithRawOutput only affects JSON string output
		p, err := jqyaml.New(jqyaml.WithQuery("."))
		if err != nil {
			t.Fatal(err)
		}
		
		data := "hello"
		
		// JSON with raw
		var jsonBuf bytes.Buffer
		err = p.Execute(context.Background(), data,
			jqyaml.WithWriter(&jsonBuf, jqyaml.FormatJSON),
			jqyaml.WithRawOutput(),
		)
		if err != nil {
			t.Fatal(err)
		}
		
		if jsonBuf.String() != "hello" {
			t.Error("JSON string output should be raw (no quotes)")
		}
		
		// Non-string data with raw should still be JSON encoded
		var numBuf bytes.Buffer
		err = p.Execute(context.Background(), 42,
			jqyaml.WithWriter(&numBuf, jqyaml.FormatJSON),
			jqyaml.WithRawOutput(),
		)
		if err != nil {
			t.Fatal(err)
		}
		
		if numBuf.String() != "42\n" {
			t.Errorf("JSON number output should be standard JSON, got: %q", numBuf.String())
		}
	})
}