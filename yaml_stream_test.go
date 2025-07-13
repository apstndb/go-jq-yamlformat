package jqyaml

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// TestYAMLStreamDocumentSeparator tests that multiple YAML documents are properly separated with ---
func TestYAMLStreamDocumentSeparator(t *testing.T) {
	testCases := []struct {
		name     string
		query    string
		input    interface{}
		expected string
	}{
		{
			name:  "multiple scalar values",
			query: ".numbers[]",
			input: map[string]interface{}{
				"numbers": []interface{}{1, 2, 3},
			},
			expected: "1\n---\n2\n---\n3\n",
		},
		{
			name:  "multiple objects",
			query: ".items[]",
			input: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"id": 1, "name": "foo"},
					map[string]interface{}{"id": 2, "name": "bar"},
				},
			},
			expected: "id: 1\nname: foo\n---\nid: 2\nname: bar\n",
		},
		{
			name:  "single value (no separator needed)",
			query: ".value",
			input: map[string]interface{}{
				"value": "hello",
			},
			expected: "hello\n",
		},
		{
			name:  "empty result",
			query: ".items[] | select(.id > 10)",
			input: map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{"id": 1},
					map[string]interface{}{"id": 2},
				},
			},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := New(WithQuery(tc.query))
			if err != nil {
				t.Fatalf("failed to create pipeline: %v", err)
			}

			var buf bytes.Buffer
			err = p.Execute(context.Background(), tc.input,
				WithWriter(&buf, FormatYAML),
			)
			if err != nil {
				t.Fatalf("failed to execute pipeline: %v", err)
			}

			got := buf.String()
			if got != tc.expected {
				t.Errorf("unexpected output\ngot:\n%s\nwant:\n%s", got, tc.expected)
				// Also show the difference in a more readable format
				t.Errorf("got lines: %q", strings.Split(got, "\n"))
				t.Errorf("want lines: %q", strings.Split(tc.expected, "\n"))
			}
		})
	}
}

// TestYAMLStreamWithCallback tests that streaming with a callback receives all results from a stream
func TestYAMLStreamWithCallback(t *testing.T) {
	p, err := New(WithQuery(".numbers[]"))
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	input := map[string]interface{}{
		"numbers": []interface{}{1, 2, 3},
	}

	var results []interface{}
	err = p.Execute(context.Background(), input,
		WithCallback(func(v interface{}) error {
			results = append(results, v)
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("failed to execute pipeline: %v", err)
	}

	// Verify we got the expected number of results and they are correct
	expected := []interface{}{1, 2, 3}
	if len(results) != len(expected) {
		t.Fatalf("expected %d results, got %d", len(expected), len(results))
	}

	for i := range results {
		if results[i] != expected[i] {
			t.Errorf("result at index %d mismatch: got %v, want %v", i, results[i], expected[i])
		}
	}
}
