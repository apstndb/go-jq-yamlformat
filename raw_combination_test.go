package jqyaml

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

const (
	helloWorldWithNewline = "hello world\n"
)

// TestRawOutputCombinations tests the behavior of raw output with compact and pretty options
func TestRawOutputCombinations(t *testing.T) {
	tests := []struct {
		name        string
		input       interface{}
		query       string
		options     []ExecuteOption
		description string
		checkFunc   func(t *testing.T, output string)
	}{
		// String tests - raw should always output strings without quotes
		{
			name:        "raw only with string",
			input:       "hello world",
			query:       ".",
			options:     []ExecuteOption{WithRawJSONOutput()},
			description: "Raw output for strings: no quotes, literal output",
			checkFunc: func(t *testing.T, output string) {
				if output != helloWorldWithNewline {
					t.Errorf("Expected 'hello world\\n', got %q", output)
				}
			},
		},
		{
			name:        "raw + compact with string",
			input:       "hello world",
			query:       ".",
			options:     []ExecuteOption{WithRawJSONOutput(), WithCompactJSONOutput()},
			description: "Raw + Compact for strings: still no quotes (raw takes precedence)",
			checkFunc: func(t *testing.T, output string) {
				if output != helloWorldWithNewline {
					t.Errorf("Expected 'hello world\\n', got %q", output)
				}
			},
		},
		{
			name:        "raw + pretty with string",
			input:       "hello world",
			query:       ".",
			options:     []ExecuteOption{WithRawJSONOutput(), WithPrettyJSONOutput()},
			description: "Raw + Pretty for strings: still no quotes (raw takes precedence)",
			checkFunc: func(t *testing.T, output string) {
				if output != helloWorldWithNewline {
					t.Errorf("Expected 'hello world\\n', got %q", output)
				}
			},
		},

		// Object tests - raw should force compact for non-strings
		{
			name:        "raw only with object",
			input:       map[string]interface{}{"a": 1, "b": 2},
			query:       ".",
			options:     []ExecuteOption{WithRawJSONOutput()},
			description: "Raw output for objects: compact JSON",
			checkFunc: func(t *testing.T, output string) {
				// Should be compact, single line
				lines := strings.Split(strings.TrimSpace(output), "\n")
				if len(lines) != 1 {
					t.Errorf("Expected single line, got %d lines: %v", len(lines), lines)
				}
				if !strings.Contains(output, `{"a":1,"b":2}`) && !strings.Contains(output, `{"b":2,"a":1}`) {
					t.Errorf("Expected compact JSON, got %q", output)
				}
			},
		},
		{
			name:        "raw + compact with object",
			input:       map[string]interface{}{"a": 1, "b": 2},
			query:       ".",
			options:     []ExecuteOption{WithRawJSONOutput(), WithCompactJSONOutput()},
			description: "Raw + Compact for objects: compact JSON (redundant but compatible)",
			checkFunc: func(t *testing.T, output string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")
				if len(lines) != 1 {
					t.Errorf("Expected single line, got %d lines", len(lines))
				}
			},
		},
		{
			name:        "raw + pretty with object",
			input:       map[string]interface{}{"a": 1, "b": 2},
			query:       ".",
			options:     []ExecuteOption{WithRawJSONOutput(), WithPrettyJSONOutput()},
			description: "Raw + Pretty for objects: compact JSON (raw overrides pretty)",
			checkFunc: func(t *testing.T, output string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")
				if len(lines) != 1 {
					t.Errorf("Expected single line (raw overrides pretty), got %d lines: %v", len(lines), lines)
				}
			},
		},

		// Array tests
		{
			name:        "raw + pretty with array",
			input:       []int{1, 2, 3},
			query:       ".",
			options:     []ExecuteOption{WithRawJSONOutput(), WithPrettyJSONOutput()},
			description: "Raw + Pretty for arrays: compact JSON (raw overrides pretty)",
			checkFunc: func(t *testing.T, output string) {
				if output != "[1,2,3]\n" {
					t.Errorf("Expected '[1,2,3]\\n', got %q", output)
				}
			},
		},

		// Mixed type streaming
		{
			name: "raw + pretty with mixed types",
			input: []interface{}{
				"plain string",
				map[string]interface{}{"key": "value"},
				42,
			},
			query:       ".[]",
			options:     []ExecuteOption{WithRawJSONOutput(), WithPrettyJSONOutput()},
			description: "Raw + Pretty for mixed types: strings raw, others compact",
			checkFunc: func(t *testing.T, output string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")
				if len(lines) != 3 {
					t.Errorf("Expected 3 lines, got %d: %v", len(lines), lines)
				}
				if lines[0] != "plain string" {
					t.Errorf("First line should be raw string, got %q", lines[0])
				}
				if lines[1] != `{"key":"value"}` {
					t.Errorf("Second line should be compact JSON, got %q", lines[1])
				}
				if lines[2] != "42" {
					t.Errorf("Third line should be number as JSON, got %q", lines[2])
				}
			},
		},

		// Comparison: pretty without raw
		{
			name:        "pretty only with object (for comparison)",
			input:       map[string]interface{}{"a": 1, "b": 2},
			query:       ".",
			options:     []ExecuteOption{WithPrettyJSONOutput()},
			description: "Pretty without raw: indented JSON",
			checkFunc: func(t *testing.T, output string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")
				if len(lines) <= 1 {
					t.Errorf("Expected multiple lines for pretty JSON, got %d", len(lines))
				}
				if !strings.Contains(output, "  ") {
					t.Errorf("Expected indentation in pretty JSON")
				}
			},
		},

		// Nested objects
		{
			name: "raw + pretty with nested object",
			input: map[string]interface{}{
				"outer": map[string]interface{}{
					"inner": map[string]interface{}{
						"value": 42,
					},
				},
			},
			query:       ".",
			options:     []ExecuteOption{WithRawJSONOutput(), WithPrettyJSONOutput()},
			description: "Raw + Pretty for nested objects: still compact (raw wins)",
			checkFunc: func(t *testing.T, output string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")
				if len(lines) != 1 {
					t.Errorf("Expected single line, got %d lines", len(lines))
				}
				if !strings.Contains(output, `{"outer":{"inner":{"value":42}}}`) {
					t.Errorf("Expected compact nested JSON, got %q", output)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Description: %s", tt.description)

			p, err := New(WithQuery(tt.query))
			if err != nil {
				t.Fatalf("failed to create pipeline: %v", err)
			}

			var buf bytes.Buffer
			opts := append([]ExecuteOption{}, tt.options...)
			opts = append(opts, WithWriter(&buf, FormatJSON))

			err = p.Execute(context.Background(), tt.input, opts...)
			if err != nil {
				t.Fatalf("execution failed: %v", err)
			}

			output := buf.String()
			tt.checkFunc(t, output)
		})
	}
}

// TestRawOutputPriority ensures raw output behavior is consistent regardless of option order
func TestRawOutputPriority(t *testing.T) {
	input := map[string]interface{}{"test": "value", "num": 123}
	query := "."

	optionSets := [][]ExecuteOption{
		// Different orderings of the same options
		{WithRawJSONOutput(), WithPrettyJSONOutput()},
		{WithPrettyJSONOutput(), WithRawJSONOutput()},
		{WithCompactJSONOutput(), WithRawJSONOutput()},
		{WithRawJSONOutput(), WithCompactJSONOutput()},
		{WithRawJSONOutput(), WithPrettyJSONOutput(), WithCompactJSONOutput()},
		{WithCompactJSONOutput(), WithPrettyJSONOutput(), WithRawJSONOutput()},
	}

	outputs := make([]string, 0, len(optionSets))

	for i, opts := range optionSets {
		p, err := New(WithQuery(query))
		if err != nil {
			t.Fatalf("failed to create pipeline: %v", err)
		}

		var buf bytes.Buffer
		execOpts := append([]ExecuteOption{}, opts...)
		execOpts = append(execOpts, WithWriter(&buf, FormatJSON))

		err = p.Execute(context.Background(), input, execOpts...)
		if err != nil {
			t.Fatalf("execution %d failed: %v", i, err)
		}

		outputs = append(outputs, buf.String())
	}

	// All outputs should be identical (compact JSON) regardless of option order
	for i := 1; i < len(outputs); i++ {
		if outputs[i] != outputs[0] {
			t.Errorf("Output %d differs from output 0:\nOutput 0: %q\nOutput %d: %q",
				i, outputs[0], i, outputs[i])
		}
	}

	// Verify it's actually compact
	if strings.Contains(outputs[0], "\n  ") || strings.Contains(outputs[0], "{\n") {
		t.Errorf("Expected compact output, but got: %q", outputs[0])
	}
}
