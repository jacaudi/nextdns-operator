package nextdns

import (
	"fmt"

	"github.com/jacaudi/nextdns-go/nextdns"
)

// IsNotFoundError returns true if the error indicates a resource was not found.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return nextdns.IsNotFound(err)
}

// IsAuthError returns true if the error indicates an authentication failure.
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}
	return nextdns.IsAuthError(err)
}

// IsDuplicateError returns true if the error indicates a duplicate resource.
func IsDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	return nextdns.IsDuplicateError(err)
}

// HasErrorCode returns true if the error contains the specified error code.
func HasErrorCode(err error, code string) bool {
	if err == nil {
		return false
	}
	return nextdns.HasErrorCode(err, code)
}

// WrapError wraps an error with additional context.
// Returns nil if the original error is nil.
func WrapError(message string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}
