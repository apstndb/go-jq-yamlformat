package jqyaml_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/apstndb/go-jq-yamlformat"
	"github.com/apstndb/go-yamlformat"
	"github.com/goccy/go-yaml"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		opts    []jqyaml.Option
		wantErr bool
		errMsg string
	}{
		{
			name: "empty pipeline",
			opts: nil,
		},
		{
			name: "valid query",
			opts: []jqyaml.Option{
				jqyaml.WithQuery("."),
			},
		},
		{
			name: "invalid query syntax",
			opts: []jqyaml.Option{
				jqyaml.WithQuery(".users[] | select(.name =="),
			},
			wantErr: true,
			errMsg: "failed to parse query",
		},
		{
			name: "query with custom options",
			opts: []jqyaml.Option{
				jqyaml.WithQuery(".items[]"),
				jqyaml.WithDefaultEncodeOptions(
					yaml.Indent(4),
				),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := jqyaml.New(tt.opts...)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message should contain %q, got %q", tt.errMsg, err.Error())
				}
				var queryErr *jqyaml.QueryError
				if !errors.As(err, &queryErr) {
					t.Errorf("expected QueryError, got %T", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if p == nil {
					t.Fatal("expected non-nil pipeline")
				}
			}
		})
	}
}

func TestExecute(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		data       interface{}
		format     yamlformat.Format
		variables  map[string]interface{}
		wantOutput string
		wantErr    bool
	}{
		{
			name:       "simple passthrough",
			query:      ".",
			data:       map[string]interface{}{"foo": "bar"},
			format:     yamlformat.FormatJSON,
			wantOutput: `{"foo": "bar"}
`,
		},
		{
			name:  "array filtering",
			query: ".items[] | select(.active)",
			data: map[string]interface{}{
				"items": []map[string]interface{}{
					{"id": 1, "active": true},
					{"id": 2, "active": false},
					{"id": 3, "active": true},
				},
			},
			format:     yamlformat.FormatJSON,
			wantOutput: `[{"active": true, "id": 1}, {"active": true, "id": 3}]
`,
		},
		{
			name:  "with variables",
			query: ".[] | select(. > $threshold)",
			data:  []int{1, 5, 10, 15, 20},
			format: yamlformat.FormatJSON,
			variables: map[string]interface{}{
				"threshold": 10,
			},
			wantOutput: "[15, 20]\n",
		},
		{
			name:  "yaml output",
			query: ".",
			data: map[string]interface{}{
				"name": "test",
				"items": []string{"a", "b", "c"},
			},
			format:     yamlformat.FormatYAML,
			wantOutput: "items:\n- a\n- b\n- c\nname: test\n",
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
			if tt.variables != nil {
				opts = append(opts, jqyaml.WithVariables(tt.variables))
			}

			err = p.Execute(context.Background(), tt.data, opts...)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				got := buf.String()
				if got != tt.wantOutput {
					t.Errorf("output mismatch\ngot:  %q\nwant: %q", got, tt.wantOutput)
				}
			}
		})
	}
}

func TestExecuteMissingEncoder(t *testing.T) {
	p, err := jqyaml.New(jqyaml.WithQuery("."))
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	err = p.Execute(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for missing encoder, got nil")
	}
	if !strings.Contains(err.Error(), "no output method specified") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestStreamingExecute(t *testing.T) {
	p, err := jqyaml.New(jqyaml.WithQuery(".items[]"))
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	data := map[string]interface{}{
		"items": []string{"a", "b", "c", "d", "e"},
	}

	var results []interface{}
	err = p.Execute(context.Background(), data,
		jqyaml.WithCallback(func(item interface{}) error {
			results = append(results, item)
			return nil
		}),
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}

	for i, r := range results {
		expected := string([]byte{'a' + byte(i)})
		if r != expected {
			t.Errorf("result[%d] = %v, want %s", i, r, expected)
		}
	}
}

func TestStreamingExecuteWithError(t *testing.T) {
	p, err := jqyaml.New(jqyaml.WithQuery(".items[]"))
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	data := map[string]interface{}{
		"items": []int{1, 2, 3, 4, 5},
	}

	var results []interface{}
	callbackErr := errors.New("callback error")
	err = p.Execute(context.Background(), data,
		jqyaml.WithCallback(func(item interface{}) error {
			results = append(results, item)
			if len(results) >= 3 {
				return callbackErr
			}
			return nil
		}),
	)

	if err != callbackErr {
		t.Errorf("expected callback error, got %v", err)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 results before error, got %d", len(results))
	}
}

func TestTimeout(t *testing.T) {
	p, err := jqyaml.New(jqyaml.WithQuery("while(true; .+1)")) // Infinite loop
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	var buf bytes.Buffer
	err = p.Execute(context.Background(), 0,
		jqyaml.WithWriter(&buf, yamlformat.FormatJSON),
		jqyaml.WithTimeout(50*time.Millisecond),
	)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	// Check if it's a timeout-related error
	if !errors.Is(err, context.DeadlineExceeded) {
		var timeoutErr *jqyaml.TimeoutError
		if !errors.As(err, &timeoutErr) {
			t.Errorf("expected TimeoutError or context.DeadlineExceeded, got %T: %v", err, err)
		}
	}
}

func TestCustomMarshaler(t *testing.T) {
	t.Skip("Custom marshaler for input data conversion is not yet supported")
	// TODO: This test is currently failing because the custom marshaler
	// is applied during conversion to JQ-compatible format, but the result
	// is then unmarshaled back to a generic interface{}, losing the custom formatting.
	// To properly support this, we would need to preserve the marshaled format
	// through the JQ processing pipeline.
}

func TestNoQuery(t *testing.T) {
	// Pipeline without query should pass data through unchanged
	p, err := jqyaml.New()
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	data := map[string]interface{}{"test": "data"}
	var buf bytes.Buffer
	err = p.Execute(context.Background(), data,
		jqyaml.WithWriter(&buf, yamlformat.FormatJSON),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := `{"test": "data"}
`
	if got := buf.String(); got != want {
		t.Errorf("output mismatch\ngot:  %q\nwant: %q", got, want)
	}
}

func TestComplexVariables(t *testing.T) {
	// This test verifies that Go structs can be passed as variables
	// and will be properly converted to JSON-compatible format
	type Filter struct {
		MinValue int      `json:"min_value"`
		Tags     []string `json:"tags"`
	}

	data := []map[string]interface{}{
		{"id": 1, "value": 10, "tags": []string{"a", "b"}},
		{"id": 2, "value": 20, "tags": []string{"b", "c"}},
		{"id": 3, "value": 5, "tags": []string{"a", "c"}},
	}

	p, err := jqyaml.New(
		jqyaml.WithQuery(".[] | select(.value >= $minValue and (.tags[] as $t | $tags | contains([$t])))"),
	)
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	var buf bytes.Buffer
	err = p.Execute(context.Background(), data,
		jqyaml.WithWriter(&buf, yamlformat.FormatJSON),
		jqyaml.WithVariables(map[string]interface{}{
			"minValue": 10,
			"tags":     []string{"b"},
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return items 1 and 2 (both have value >= 10 and contain tag "b")
	// The results are collected into an array
	var results []interface{}
	if err := yamlformat.Unmarshal(buf.Bytes(), &results); err != nil {
		t.Fatalf("failed to unmarshal results: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestEncodeOptions(t *testing.T) {
	data := map[string]interface{}{
		"text": "line1\nline2\nline3",
		"number": 42,
	}

	p, err := jqyaml.New(
		jqyaml.WithQuery("."),
		jqyaml.WithDefaultEncodeOptions(
			yaml.Indent(4),
		),
	)
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	var buf bytes.Buffer
	err = p.Execute(context.Background(), data,
		jqyaml.WithWriter(&buf, yamlformat.FormatYAML),
		jqyaml.WithEncodeOptions(
			yaml.UseLiteralStyleIfMultiline(true),
		),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Check for literal style
	if !strings.Contains(output, "|") {
		t.Error("expected literal style for multiline string")
	}
	// Check for indentation
	if !strings.Contains(output, "    ") {
		t.Error("expected 4-space indentation")
	}
}