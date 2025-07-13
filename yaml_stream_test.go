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

// TestYAMLStreamWithCallback tests that streaming with callback also produces proper separators
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

	// Verify we got the expected number of results
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	// Now test that when we manually encode these with YAML, we would need separators
	var buf bytes.Buffer
	for i, result := range results {
		if i > 0 {
			// This is what we need to add automatically in the encoder
			buf.WriteString("---\n")
		}
		// Simulate encoding each result
		if err := FormatYAML.NewEncoder(&buf).Encode(result); err != nil {
			t.Fatalf("failed to encode result: %v", err)
		}
	}

	expected := "1\n---\n2\n---\n3\n"
	if buf.String() != expected {
		t.Errorf("manual encoding shows we need separators\ngot:\n%s\nwant:\n%s", buf.String(), expected)
	}
}
