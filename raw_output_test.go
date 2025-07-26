package jqyaml_test

import (
	"bytes"
	"context"
	"testing"

	jqyaml "github.com/apstndb/go-jq-yamlformat"
	"github.com/goccy/go-yaml"
)

// TestRawJSONOutput tests the writeRawJSON functionality comprehensively
func TestRawJSONOutput(t *testing.T) {
	testCases := []struct {
		name     string
		input    interface{}
		query    string
		expected string
	}{
		{
			name:     "string value",
			input:    map[string]string{"value": "hello world"},
			query:    `.value`,
			expected: "hello world\n",
		},
		{
			name:     "empty string",
			input:    map[string]string{"value": ""},
			query:    `.value`,
			expected: "\n",
		},
		{
			name:     "string with newlines",
			input:    map[string]string{"value": "line1\nline2\nline3"},
			query:    `.value`,
			expected: "line1\nline2\nline3\n",
		},
		{
			name:     "string with special characters",
			input:    map[string]string{"value": "hello\t\"world\""},
			query:    `.value`,
			expected: "hello\t\"world\"\n",
		},
		{
			name:     "number value",
			input:    map[string]int{"value": 42},
			query:    `.value`,
			expected: "42\n",
		},
		{
			name:     "boolean true",
			input:    map[string]bool{"value": true},
			query:    `.value`,
			expected: "true\n",
		},
		{
			name:     "boolean false",
			input:    map[string]bool{"value": false},
			query:    `.value`,
			expected: "false\n",
		},
		{
			name:     "null value",
			input:    map[string]interface{}{"value": nil},
			query:    `.value`,
			expected: "null\n",
		},
		{
			name:     "object value",
			input:    map[string]interface{}{"value": map[string]int{"a": 1, "b": 2}},
			query:    `.value`,
			expected: "{\"a\":1,\"b\":2}\n",
		},
		{
			name:     "array value",
			input:    map[string]interface{}{"value": []int{1, 2, 3}},
			query:    `.value`,
			expected: "[1,2,3]\n",
		},
		{
			name:     "float value",
			input:    map[string]float64{"value": 3.14},
			query:    `.value`,
			expected: "3.14\n",
		},
		{
			name:     "unicode string",
			input:    map[string]string{"value": "Hello 世界"},
			query:    `.value`,
			expected: "Hello 世界\n",
		},
		{
			name:     "string that looks like JSON",
			input:    map[string]string{"value": `{"fake": "json"}`},
			query:    `.value`,
			expected: `{"fake": "json"}` + "\n",
		},
		{
			name:     "string with backslashes",
			input:    map[string]string{"value": `C:\Users\test`},
			query:    `.value`,
			expected: `C:\Users\test` + "\n",
		},
		{
			name:     "very long string",
			input:    map[string]string{"value": string(make([]byte, 1000))},
			query:    `.value | length`,
			expected: "1000\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := jqyaml.New(jqyaml.WithQuery(tc.query))
			if err != nil {
				t.Fatalf("failed to create pipeline: %v", err)
			}

			var buf bytes.Buffer
			err = p.Execute(context.Background(), tc.input,
				jqyaml.WithWriter(&buf, jqyaml.FormatJSON),
				jqyaml.WithRawJSONOutput(),
			)
			if err != nil {
				t.Fatalf("execution failed: %v", err)
			}

			if got := buf.String(); got != tc.expected {
				t.Errorf("output mismatch\ngot:  %q\nwant: %q", got, tc.expected)
			}
		})
	}
}

// TestRawJSONOutputWithStreaming tests raw output in streaming mode
func TestRawJSONOutputWithStreaming(t *testing.T) {
	input := []string{"hello", "world", "test"}

	p, err := jqyaml.New(jqyaml.WithQuery(`.[]`))
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	var buf bytes.Buffer
	err = p.Execute(context.Background(), input,
		jqyaml.WithWriter(&buf, jqyaml.FormatJSON),
		jqyaml.WithRawJSONOutput(),
	)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	expected := "hello\nworld\ntest\n"
	if got := buf.String(); got != expected {
		t.Errorf("output mismatch\ngot:  %q\nwant: %q", got, expected)
	}
}

// TestRawJSONOutputWithCustomTypes tests raw output with custom marshalers
func TestRawJSONOutputWithCustomTypes(t *testing.T) {
	type CustomString struct {
		Value string
	}

	input := map[string]CustomString{
		"message": {Value: "Hello from custom type"},
	}

	p, err := jqyaml.New(
		jqyaml.WithQuery(`.message`),
		jqyaml.WithDefaultEncodeOptions(
			yaml.CustomMarshaler[CustomString](func(cs CustomString) ([]byte, error) {
				// Custom marshaler returns the string value directly
				return []byte(`"` + cs.Value + `"`), nil
			}),
		),
	)
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	var buf bytes.Buffer
	err = p.Execute(context.Background(), input,
		jqyaml.WithWriter(&buf, jqyaml.FormatJSON),
		jqyaml.WithRawJSONOutput(),
	)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	expected := "Hello from custom type\n"
	if got := buf.String(); got != expected {
		t.Errorf("output mismatch\ngot:  %q\nwant: %q", got, expected)
	}
}