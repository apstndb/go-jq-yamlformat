package jqyaml_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	jqyaml "github.com/apstndb/go-jq-yamlformat"
	"github.com/google/go-cmp/cmp"
)

// mockEncoder is a helper for TestWithEncoder to verify that a custom encoder is called.
type mockEncoder struct {
	buf         *bytes.Buffer
	encodeCalls int
}

func (m *mockEncoder) Encode(v interface{}) error {
	m.encodeCalls++
	// Simulate a custom encoding format.
	str := fmt.Sprintf("MOCK_ENCODED<%T>: %v", v, v)
	_, err := m.buf.WriteString(str)
	return err
}

// TestWithEncoder verifies that a user-provided custom encoder via WithEncoder
// is correctly used by the pipeline, bypassing the internal pipelineEncoder.
func TestWithEncoder(t *testing.T) {
	var buf bytes.Buffer
	mock := &mockEncoder{buf: &buf}

	p, err := jqyaml.New(jqyaml.WithQuery(".message"))
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}

	input := map[string]string{"message": "hello"}
	err = p.Execute(context.Background(), input, jqyaml.WithEncoder(mock))
	if err != nil {
		t.Fatalf("execute with custom encoder failed: %v", err)
	}

	if mock.encodeCalls != 1 {
		t.Errorf("expected mock encoder's Encode method to be called once, but was called %d times", mock.encodeCalls)
	}

	want := "MOCK_ENCODED<string>: hello"
	if got := buf.String(); got != want {
		t.Errorf("output mismatch\ngot:  %q\nwant: %q", got, want)
	}
}

// TestWithCallback verifies that a user-provided callback via WithCallback
// receives the raw, unprocessed Go types from the gojq engine.
func TestWithCallback(t *testing.T) {
	testCases := []struct {
		name          string
		input         interface{}
		query         string
		expected      []interface{}
		wantErr       bool
		expectedError error
	}{
		{
			name:     "simple object",
			input:    map[string]interface{}{"a": 1, "b": 2},
			query:    ".",
			expected: []interface{}{map[string]interface{}{"a": 1, "b": 2}},
		},
		{
			name:     "stream array elements",
			input:    []string{"foo", "bar", "baz"},
			query:    ".[]",
			expected: []interface{}{"foo", "bar", "baz"},
		},
		{
			name:          "error in callback",
			input:         []int{1, 2, 3},
			query:         ".[]",
			expected:      []interface{}{1, 2},
			wantErr:       true,
			expectedError: errors.New("stop!"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := jqyaml.New(jqyaml.WithQuery(tc.query))
			if err != nil {
				t.Fatalf("failed to create pipeline: %v", err)
			}

			var got []interface{}
			callback := func(v interface{}) error {
				if tc.wantErr && len(got) == 2 {
					return tc.expectedError
				}
				got = append(got, v)
				return nil
			}

			err = p.Execute(context.Background(), tc.input, jqyaml.WithCallback(callback))

			if tc.wantErr {
				if !errors.Is(err, tc.expectedError) {
					t.Errorf("expected error %v, got %v", tc.expectedError, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			if !reflect.DeepEqual(tc.expected, got) {
				t.Errorf("callback results mismatch: %s", cmp.Diff(tc.expected, got))
			}
		})
	}
}
