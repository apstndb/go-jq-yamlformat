package jqyaml

import (
	"testing"
)

// TestRawOutputBehaviorSummary documents the behavior of raw output
func TestRawOutputBehaviorSummary(t *testing.T) {
	t.Log(`
Raw Output Behavior Summary:

1. STRING VALUES:
   - Output without quotes (literal string content)
   - Newlines in strings create multiple output lines (jq compatible)
   - Each output value gets a trailing newline

2. NON-STRING VALUES:
   - Fall back to compact JSON encoding
   - Objects and arrays are always compact, even with WithPrettyJSONOutput()
   - Numbers, booleans, and null are encoded as JSON

3. OPTION COMBINATIONS:
   - WithRawJSONOutput() + WithCompactJSONOutput():
     * Strings: raw (no quotes)
     * Non-strings: compact JSON (redundant but harmless)
   
   - WithRawJSONOutput() + WithPrettyJSONOutput():
     * Strings: raw (no quotes)
     * Non-strings: compact JSON (raw overrides pretty)

4. CONSISTENCY:
   - Behavior is identical whether using WithWriter() or WithCallback()
   - Option order doesn't matter (raw always takes precedence)

5. JQ COMPATIBILITY:
   - Mimics jq's --raw-output flag behavior
   - Preserves newlines in strings as separate output lines
   - Non-strings fall back to JSON encoding
`)
}

// TestRawOutputExamples provides concrete examples of raw output behavior
func TestRawOutputExamples(t *testing.T) {
	examples := []struct {
		description string
		input       string
		output      string
	}{
		{
			description: "Simple string",
			input:       `"hello"`,
			output:      "hello\n",
		},
		{
			description: "String with newline",
			input:       `"line1\nline2"`,
			output:      "line1\nline2\n",
		},
		{
			description: "Number",
			input:       `42`,
			output:      "42\n",
		},
		{
			description: "Object",
			input:       `{"a":1}`,
			output:      `{"a":1}` + "\n",
		},
		{
			description: "Array",
			input:       `[1,2,3]`,
			output:      "[1,2,3]\n",
		},
	}

	for _, ex := range examples {
		t.Logf("%-25s: Input: %-20s → Output: %q", ex.description, ex.input, ex.output)
	}
}
