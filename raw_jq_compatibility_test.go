package jqyaml

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// TestRawOutputJqCompatibility tests that raw output behaves like jq's --raw-output
func TestRawOutputJqCompatibility(t *testing.T) {
	tests := []struct {
		name          string
		input         interface{}
		query         string
		expectedLines []string
		description   string
	}{
		// Newline handling - jq preserves newlines in raw output
		{
			name:          "string with embedded newlines",
			input:         "line1\nline2\nline3",
			query:         ".",
			expectedLines: []string{"line1", "line2", "line3"},
			description:   "Embedded newlines should create multiple output lines (jq compatible)",
		},
		{
			name:          "string with CRLF",
			input:         "line1\r\nline2\r\nline3",
			query:         ".",
			expectedLines: []string{"line1\r", "line2\r", "line3"},
			description:   "CRLF should be preserved as-is",
		},
		{
			name:          "string with only newlines",
			input:         "\n\n\n",
			query:         ".",
			expectedLines: []string{"", "", "", ""},
			description:   "Multiple newlines should create empty lines",
		},
		{
			name:          "string with mixed whitespace",
			input:         "  \n\t\n  ",
			query:         ".",
			expectedLines: []string{"  ", "\t", "  "},
			description:   "Whitespace around newlines should be preserved",
		},
		{
			name: "array of strings with newlines",
			input: []string{
				"first\nsecond",
				"third",
				"fourth\nfifth\nsixth",
			},
			query:         ".[]",
			expectedLines: []string{"first", "second", "third", "fourth", "fifth", "sixth"},
			description:   "Each string's newlines should be expanded",
		},
		{
			name:          "string with escaped newline",
			input:         `line1\nline2`,
			query:         ".",
			expectedLines: []string{`line1\nline2`},
			description:   "Escaped newlines (\\n) should remain escaped",
		},
		{
			name:          "empty string",
			input:         "",
			query:         ".",
			expectedLines: []string{""},
			description:   "Empty string should produce single empty line",
		},
		{
			name:          "string ending with newline",
			input:         "line1\nline2\n",
			query:         ".",
			expectedLines: []string{"line1", "line2", ""},
			description:   "Trailing newline should create empty line at end",
		},
		{
			name:          "string starting with newline",
			input:         "\nline1\nline2",
			query:         ".",
			expectedLines: []string{"", "line1", "line2"},
			description:   "Leading newline should create empty line at start",
		},
		
		// Non-string behavior
		{
			name:          "number fallback to JSON",
			input:         42.5,
			query:         ".",
			expectedLines: []string{"42.5"},
			description:   "Numbers should be output as JSON",
		},
		{
			name:          "null fallback to JSON",
			input:         nil,
			query:         ".",
			expectedLines: []string{"null"},
			description:   "Null should be output as JSON",
		},
		{
			name:          "object fallback to compact JSON",
			input:         map[string]interface{}{"key": "value"},
			query:         ".",
			expectedLines: []string{`{"key":"value"}`},
			description:   "Objects should be output as compact JSON",
		},
		
		// Complex queries
		{
			name: "extract string field with newlines",
			input: map[string]interface{}{
				"message": "Hello\nWorld",
				"other": 123,
			},
			query:         ".message",
			expectedLines: []string{"Hello", "World"},
			description:   "Extracted string fields should preserve newlines",
		},
		{
			name: "string concatenation with newlines",
			input: map[string]interface{}{
				"first": "Hello\n",
				"second": "World",
			},
			query:         ".first + .second",
			expectedLines: []string{"Hello", "World"},
			description:   "Concatenated strings should preserve all newlines",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			
			p, err := New(WithQuery(tt.query))
			if err != nil {
				t.Fatalf("failed to create pipeline: %v", err)
			}

			var buf bytes.Buffer
			err = p.Execute(context.Background(), tt.input,
				WithWriter(&buf, FormatJSON),
				WithRawJSONOutput(),
			)
			if err != nil {
				t.Fatalf("execution failed: %v", err)
			}

			output := buf.String()
			
			// For raw output, the final newline is added by our encoder
			// Split by newlines and remove the final empty string if present
			lines := strings.Split(output, "\n")
			if len(lines) > 0 && lines[len(lines)-1] == "" {
				lines = lines[:len(lines)-1]
			}
			
			// Compare line by line
			if len(lines) != len(tt.expectedLines) {
				t.Errorf("Line count mismatch: got %d, want %d", len(lines), len(tt.expectedLines))
				t.Errorf("Got lines: %q", lines)
				t.Errorf("Want lines: %q", tt.expectedLines)
				return
			}
			
			for i, line := range lines {
				if line != tt.expectedLines[i] {
					t.Errorf("Line %d mismatch:\ngot:  %q\nwant: %q", i, line, tt.expectedLines[i])
				}
			}
		})
	}
}

// TestRawOutputStreamingConsistency ensures raw output is consistent in streaming mode
func TestRawOutputStreamingConsistency(t *testing.T) {
	testCases := []struct {
		name  string
		input interface{}
		query string
	}{
		{
			name:  "multiple strings with newlines",
			input: []string{"a\nb", "c\nd\ne", "f"},
			query: ".[]",
		},
		{
			name: "mixed types with string newlines",
			input: []interface{}{
				"string\nwith\nnewlines",
				42,
				map[string]interface{}{"key": "value"},
				"another\nstring",
			},
			query: ".[]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test with writer
			var writerBuf bytes.Buffer
			p1, _ := New(WithQuery(tc.query))
			err := p1.Execute(context.Background(), tc.input,
				WithWriter(&writerBuf, FormatJSON),
				WithRawJSONOutput(),
			)
			if err != nil {
				t.Fatalf("writer execution failed: %v", err)
			}
			
			// Test with callback
			var callbackResults []string
			p2, _ := New(WithQuery(tc.query))
			err = p2.Execute(context.Background(), tc.input,
				WithCallback(func(v interface{}) error {
					// Simulate raw output behavior
					if s, ok := v.(string); ok {
						// For callback, we need to handle newlines ourselves
						lines := strings.Split(s, "\n")
						callbackResults = append(callbackResults, lines...)
					} else {
						// Non-strings get JSON encoded
						data, _ := json.Marshal(v)
						callbackResults = append(callbackResults, string(data))
					}
					return nil
				}),
			)
			if err != nil {
				t.Fatalf("callback execution failed: %v", err)
			}
			
			// Compare outputs
			writerLines := strings.Split(strings.TrimRight(writerBuf.String(), "\n"), "\n")
			
			if len(writerLines) != len(callbackResults) {
				t.Errorf("Line count mismatch: writer=%d, callback=%d", 
					len(writerLines), len(callbackResults))
			}
			
			for i := 0; i < len(writerLines) && i < len(callbackResults); i++ {
				if writerLines[i] != callbackResults[i] {
					t.Errorf("Line %d differs:\nwriter:   %q\ncallback: %q",
						i, writerLines[i], callbackResults[i])
				}
			}
		})
	}
}