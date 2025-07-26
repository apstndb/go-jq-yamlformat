package jqyaml_test

import (
	"bytes"
	"context"
	"math/big"
	"strconv"
	"testing"

	jqyaml "github.com/apstndb/go-jq-yamlformat"
	"github.com/goccy/go-yaml"
)

// TestCustomMarshaler verifies that custom type marshalers work for both JSON and YAML.
func TestCustomMarshaler(t *testing.T) {
	largeNumber, _ := new(big.Int).SetString("12345678901234567890", 10)
	customMarshalerOpt := jqyaml.WithDefaultEncodeOptions(
		yaml.CustomMarshaler[*big.Int](func(i *big.Int) ([]byte, error) {
			return []byte(strconv.Quote("big:" + i.String())), nil
		}),
	)
	p, err := jqyaml.New(jqyaml.WithQuery("."), customMarshalerOpt)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("json_output", func(t *testing.T) {
		var buf bytes.Buffer
		err := p.Execute(context.Background(), largeNumber, jqyaml.WithWriter(&buf, jqyaml.FormatJSON))
		if err != nil {
			t.Fatal(err)
		}
		want := `"big:12345678901234567890"` + "\n"
		if got := buf.String(); got != want {
			t.Errorf("output mismatch\ngot:  %q\nwant: %q", got, want)
		}
	})

	t.Run("yaml_output", func(t *testing.T) {
		var buf bytes.Buffer
		err := p.Execute(context.Background(), largeNumber, jqyaml.WithWriter(&buf, jqyaml.FormatYAML))
		if err != nil {
			t.Fatal(err)
		}
		// The custom marshaler returns a quoted string, so YAML will preserve those quotes
		want := `"big:12345678901234567890"` + "\n"
		if got := buf.String(); got != want {
			t.Errorf("output mismatch\ngot:  %q\nwant: %q", got, want)
		}
	})
}

// TestCompactWithCustomMarshaler verifies the round-trip for perfect compact output.
func TestCompactWithCustomMarshaler(t *testing.T) {
	data := map[string]interface{}{"key": "value"}
	// This option adds a custom marshaler that will introduce whitespace.
	customMarshalerOpt := jqyaml.WithDefaultEncodeOptions(
		yaml.CustomMarshaler[string](func(s string) ([]byte, error) {
			// A silly marshaler that adds spaces around the string.
			return []byte(`"  ` + s + `  "`), nil
		}),
	)

	p, err := jqyaml.New(jqyaml.WithQuery("."), customMarshalerOpt)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	// Enable compact output, which should trigger the round-trip.
	err = p.Execute(context.Background(), data, jqyaml.WithWriter(&buf, jqyaml.FormatJSON), jqyaml.WithCompactJSONOutput())
	if err != nil {
		t.Fatal(err)
	}

	// The round-trip ensures the output is compact, but preserves spaces within the string value.
	want := `{"key":"  value  "}` + "\n"
	if got := buf.String(); got != want {
		t.Errorf("output mismatch\ngot:  %q\nwant: %q", got, want)
	}
}
