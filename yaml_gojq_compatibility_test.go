package jqyaml

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

// TestYAMLOutputGojqCompatibility ensures our YAML output matches gojq's --yaml-output
func TestYAMLOutputGojqCompatibility(t *testing.T) {
	// Check if gojq is available
	if _, err := exec.LookPath("gojq"); err != nil {
		t.Skip("gojq not found in PATH, skipping compatibility test")
	}

	testCases := []struct {
		name  string
		query string
		input string
	}{
		{
			name:  "scalar values stream",
			query: "1,2,3",
			input: "{}",
		},
		{
			name:  "array elements",
			query: ".[]",
			input: `[{"name": "Alice"}, {"name": "Bob"}]`,
		},
		{
			name:  "filtered results",
			query: `.[] | select(.active)`,
			input: `[{"name": "Alice", "active": true}, {"name": "Bob", "active": false}, {"name": "Charlie", "active": true}]`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get gojq output
			cmd := exec.Command("gojq", "--yaml-output", tc.query)
			cmd.Stdin = strings.NewReader(tc.input)
			gojqOutput, err := cmd.Output()
			if err != nil {
				t.Fatalf("failed to run gojq: %v", err)
			}

			// Get our output
			var inputData interface{}
			if err := unmarshalJSON([]byte(tc.input), &inputData); err != nil {
				t.Fatalf("failed to unmarshal input: %v", err)
			}

			p, err := New(WithQuery(tc.query))
			if err != nil {
				t.Fatalf("failed to create pipeline: %v", err)
			}

			var buf bytes.Buffer
			err = p.Execute(context.Background(), inputData,
				WithWriter(&buf, FormatYAML),
			)
			if err != nil {
				t.Fatalf("failed to execute pipeline: %v", err)
			}

			ourOutput := buf.Bytes()

			// Compare outputs
			if !bytes.Equal(gojqOutput, ourOutput) {
				t.Errorf("output mismatch with gojq\ngojq output:\n%s\nour output:\n%s", gojqOutput, ourOutput)
				// Show hex dump for detailed comparison
				t.Logf("gojq hex: %x", gojqOutput)
				t.Logf("our hex: %x", ourOutput)
			}
		})
	}
}

// Helper function to unmarshal JSON
func unmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
