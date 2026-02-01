package nextdns

import (
	"errors"
	"testing"
)

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "regular error",
			err:      errors.New("some error"),
			expected: false,
		},
		{
			name:     "wrapped regular error",
			err:      WrapError("operation failed", errors.New("some error")),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFoundError(tt.err); got != tt.expected {
				t.Errorf("IsNotFoundError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "regular error",
			err:      errors.New("unauthorized"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAuthError(tt.err); got != tt.expected {
				t.Errorf("IsAuthError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsDuplicateError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "regular error",
			err:      errors.New("duplicate"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDuplicateError(tt.err); got != tt.expected {
				t.Errorf("IsDuplicateError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	origErr := errors.New("original error")
	wrapped := WrapError("operation failed", origErr)

	if wrapped == nil {
		t.Fatal("WrapError returned nil")
	}

	if !errors.Is(wrapped, origErr) {
		t.Error("wrapped error should contain original error")
	}

	expected := "operation failed: original error"
	if wrapped.Error() != expected {
		t.Errorf("wrapped.Error() = %q, want %q", wrapped.Error(), expected)
	}
}

func TestWrapErrorNil(t *testing.T) {
	wrapped := WrapError("operation failed", nil)
	if wrapped != nil {
		t.Errorf("WrapError(nil) should return nil, got %v", wrapped)
	}
}
