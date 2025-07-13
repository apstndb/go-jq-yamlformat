package jqyaml_test

import (
	"bytes"
	"context"
	"testing"

	jqyaml "github.com/apstndb/go-jq-yamlformat"
)

// TestAliases verifies that Format aliases work without importing yamlformat
func TestAliases(t *testing.T) {
	p, err := jqyaml.New(jqyaml.WithQuery("."))
	if err != nil {
		t.Fatal(err)
	}

	data := map[string]interface{}{"test": "value"}

	// Test FormatJSON alias
	var jsonBuf bytes.Buffer
	err = p.Execute(context.Background(), data,
		jqyaml.WithWriter(&jsonBuf, jqyaml.FormatJSON),
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := jsonBuf.String(); got != `{"test":"value"}`+"\n" {
		t.Errorf("FormatJSON: expected %q, got %q", `{"test":"value"}`+"\n", got)
	}

	// Test FormatYAML alias
	var yamlBuf bytes.Buffer
	err = p.Execute(context.Background(), data,
		jqyaml.WithWriter(&yamlBuf, jqyaml.FormatYAML),
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := yamlBuf.String(); got != "test: value\n" {
		t.Errorf("FormatYAML: expected %q, got %q", "test: value\n", got)
	}
}
