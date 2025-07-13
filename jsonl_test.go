package jqyaml

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONLOutput(t *testing.T) {
	tests := []struct {
		name          string
		input         interface{}
		query         string
		options       []ExecuteOption
		wantLines     []string
		wantLineCount int
	}{
		{
			name: "compact JSON output should produce valid JSONL",
			input: []map[string]interface{}{
				{"id": 1, "name": "Alice"},
				{"id": 2, "name": "Bob"},
				{"id": 3, "name": "Charlie"},
			},
			query:   ".[]",
			options: []ExecuteOption{WithCompactJSONOutput()},
			wantLines: []string{
				`{"id":1,"name":"Alice"}`,
				`{"id":2,"name":"Bob"}`,
				`{"id":3,"name":"Charlie"}`,
			},
			wantLineCount: 3,
		},
		{
			name: "pretty JSON output should have newlines between objects",
			input: []map[string]interface{}{
				{"id": 1, "name": "Alice"},
				{"id": 2, "name": "Bob"},
			},
			query:         ".[]",
			options:       []ExecuteOption{WithPrettyJSONOutput()},
			wantLineCount: 8, // Each object spans multiple lines
		},
		{
			name:    "raw output with strings should produce valid lines",
			input:   []string{"line1", "line2", "line3"},
			query:   ".[]",
			options: []ExecuteOption{WithRawJSONOutput()},
			wantLines: []string{
				"line1",
				"line2",
				"line3",
			},
			wantLineCount: 3,
		},
		{
			name: "raw output with non-strings should fallback to JSON",
			input: []interface{}{
				"string value",
				42,
				map[string]interface{}{"key": "value"},
			},
			query:   ".[]",
			options: []ExecuteOption{WithRawJSONOutput()},
			wantLines: []string{
				"string value",
				"42",
				`{"key":"value"}`,
			},
			wantLineCount: 3,
		},
		{
			name:    "multiple values without query should still produce one line",
			input:   []interface{}{1, 2, 3},
			query:   "",
			options: []ExecuteOption{WithCompactJSONOutput()},
			wantLines: []string{
				"[1,2,3]",
			},
			wantLineCount: 1,
		},
		{
			name:    "empty strings in raw mode should produce empty lines",
			input:   []string{"", "line2", "", "line4"},
			query:   ".[]",
			options: []ExecuteOption{WithRawJSONOutput()},
			wantLines: []string{
				"",
				"line2",
				"",
				"line4",
			},
			wantLineCount: 4,
		},
		{
			name: "mixed compact and raw should produce compact JSON for non-strings",
			input: []interface{}{
				"plain string",
				[]int{1, 2, 3},
				map[string]interface{}{"nested": map[string]int{"value": 42}},
			},
			query:   ".[]",
			options: []ExecuteOption{WithRawJSONOutput(), WithCompactJSONOutput()},
			wantLines: []string{
				"plain string",
				"[1,2,3]",
				`{"nested":{"value":42}}`,
			},
			wantLineCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			p, err := New(WithQuery(tt.query))
			if err != nil {
				t.Fatalf("failed to create pipeline: %v", err)
			}

			opts := append([]ExecuteOption{}, tt.options...)
			opts = append(opts, WithWriter(&buf, FormatJSON))
			err = p.Execute(context.Background(), tt.input, opts...)
			if err != nil {
				t.Fatalf("execute failed: %v", err)
			}

			output := buf.String()

			// Check that output ends with a newline (except for empty output)
			if output != "" && !strings.HasSuffix(output, "\n") {
				t.Errorf("output does not end with newline: %q", output)
			}

			// Split by newlines
			lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

			// Check line count
			if tt.wantLineCount > 0 && len(lines) != tt.wantLineCount {
				t.Errorf("got %d lines, want %d\nOutput:\n%s", len(lines), tt.wantLineCount, output)
			}

			// Check specific line content if provided
			if tt.wantLines != nil {
				if len(lines) != len(tt.wantLines) {
					t.Errorf("got %d lines, want %d\nGot: %v\nWant: %v",
						len(lines), len(tt.wantLines), lines, tt.wantLines)
				}
				for i, wantLine := range tt.wantLines {
					if i < len(lines) && lines[i] != wantLine {
						t.Errorf("line %d:\ngot:  %q\nwant: %q", i, lines[i], wantLine)
					}
				}
			}

			// For compact JSON, verify each line is valid JSON (skip this check for raw output)
			isRawOutput := tt.name == "raw output with strings should produce valid lines" ||
				tt.name == "raw output with non-strings should fallback to JSON"

			if !isRawOutput && tt.name == "compact JSON output should produce valid JSONL" {
				for i, line := range lines {
					if line != "" && !isValidJSON(line) {
						t.Errorf("line %d is not valid JSON: %q", i, line)
					}
				}
			}
		})
	}
}

// Helper function to check if a string is valid JSON
func isValidJSON(s string) bool {
	// Simple check - tries to parse as JSON
	var v interface{}
	d := json.NewDecoder(strings.NewReader(s))
	return d.Decode(&v) == nil
}
