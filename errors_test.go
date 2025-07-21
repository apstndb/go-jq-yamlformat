package jqyaml_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	jqyaml "github.com/apstndb/go-jq-yamlformat"
)

func TestQueryError(t *testing.T) {
	err := &jqyaml.QueryError{
		Query:   ".users[] | select(.name",
		Message: "syntax error",
		Err:     errors.New("unexpected EOF"),
	}

	want := "jq query error in '.users[] | select(.name': syntax error"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}

	if unwrapped := err.Unwrap(); unwrapped == nil || unwrapped.Error() != "unexpected EOF" {
		t.Errorf("Unwrap() = %v, want 'unexpected EOF'", unwrapped)
	}
}

func TestConversionError(t *testing.T) {
	err := &jqyaml.ConversionError{
		Value: make(chan int),
		Type:  "jq-compatible",
		Err:   errors.New("unsupported type: chan"),
	}

	if !strings.Contains(err.Error(), "chan int") {
		t.Errorf("Error() should mention the value type, got: %s", err.Error())
	}

	if !strings.Contains(err.Error(), "jq-compatible") {
		t.Errorf("Error() should mention the target type, got: %s", err.Error())
	}

	if unwrapped := err.Unwrap(); unwrapped == nil || !strings.Contains(unwrapped.Error(), "unsupported type") {
		t.Errorf("Unwrap() = %v, want error containing 'unsupported type'", unwrapped)
	}
}

func TestTimeoutError(t *testing.T) {
	t.Run("without wrapped error", func(t *testing.T) {
		err := &jqyaml.TimeoutError{
			Duration: 5 * time.Second,
		}

		want := "execution timeout after 5s"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("with wrapped error", func(t *testing.T) {
		err := &jqyaml.TimeoutError{
			Duration: 5 * time.Second,
			Err:      context.DeadlineExceeded,
		}

		want := "execution timeout after 5s: context deadline exceeded"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}

		// Test Unwrap
		if unwrapped := err.Unwrap(); unwrapped != context.DeadlineExceeded {
			t.Errorf("Unwrap() = %v, want %v", unwrapped, context.DeadlineExceeded)
		}

		// Test errors.Is
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("errors.Is(err, context.DeadlineExceeded) = false, want true")
		}

		// Test errors.As - verify the error type and contents are correctly preserved
		var timeoutErr *jqyaml.TimeoutError
		if !errors.As(err, &timeoutErr) {
			t.Errorf("errors.As(err, &timeoutErr) = false, want true")
		} else if timeoutErr.Duration != 5*time.Second {
			t.Errorf("timeoutErr.Duration = %v, want %v", timeoutErr.Duration, 5*time.Second)
		}
	})
}

func TestErrorTypes(t *testing.T) {
	// Ensure error types implement error interface
	var _ error = (*jqyaml.QueryError)(nil)
	var _ error = (*jqyaml.ConversionError)(nil)
	var _ error = (*jqyaml.TimeoutError)(nil)

	// Ensure error types implement unwrap
	var _ interface{ Unwrap() error } = (*jqyaml.QueryError)(nil)
	var _ interface{ Unwrap() error } = (*jqyaml.ConversionError)(nil)
	var _ interface{ Unwrap() error } = (*jqyaml.TimeoutError)(nil)
}
