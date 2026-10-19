package errors

import (
	"fmt"
	"net/http"
	"testing"
)

// Service methods wrap the HTTP layer's typed error with context using
// fmt.Errorf("...: %w", err). The predicates must see through that wrapping so
// callers can still classify the underlying failure (status code, kind).
func TestPredicatesSeeThroughWrapping(t *testing.T) {
	t.Run("retryable APIError wrapped", func(t *testing.T) {
		wrapped := fmt.Errorf("get device: %w",
			NewAPIError(http.StatusServiceUnavailable, "unavailable", "Service unavailable", ""))
		if !IsRetryableError(wrapped) {
			t.Error("IsRetryableError = false, want true for wrapped 503")
		}
	})

	t.Run("non-retryable APIError wrapped", func(t *testing.T) {
		wrapped := fmt.Errorf("get device: %w",
			NewAPIError(http.StatusNotFound, "not_found", "Not found", ""))
		if IsRetryableError(wrapped) {
			t.Error("IsRetryableError = true, want false for wrapped 404")
		}
	})

	t.Run("NetworkError wrapped is retryable", func(t *testing.T) {
		wrapped := fmt.Errorf("get device: %w",
			NewNetworkError("GET /devices", fmt.Errorf("connection reset")))
		if !IsRetryableError(wrapped) {
			t.Error("IsRetryableError = false, want true for wrapped NetworkError")
		}
	})

	t.Run("authz APIError wrapped", func(t *testing.T) {
		wrapped := fmt.Errorf("get setup record: %w",
			NewAPIError(http.StatusForbidden, "forbidden", "Insufficient permissions", ""))
		if !IsAuthenticationError(wrapped) {
			t.Error("IsAuthenticationError = false, want true for wrapped 403")
		}
	})

	t.Run("AuthenticationError wrapped", func(t *testing.T) {
		wrapped := fmt.Errorf("get setup record: %w", NewAuthError("token expired", nil))
		if !IsAuthenticationError(wrapped) {
			t.Error("IsAuthenticationError = false, want true for wrapped AuthenticationError")
		}
	})

	t.Run("NetworkError wrapped is network", func(t *testing.T) {
		wrapped := fmt.Errorf("get device: %w",
			NewNetworkError("GET /devices", fmt.Errorf("connection reset")))
		if !IsNetworkError(wrapped) {
			t.Error("IsNetworkError = false, want true for wrapped NetworkError")
		}
	})

	t.Run("ConfigurationError wrapped", func(t *testing.T) {
		wrapped := fmt.Errorf("build client: %w", NewConfigError("ClientID", "required", ""))
		if !IsConfigurationError(wrapped) {
			t.Error("IsConfigurationError = false, want true for wrapped ConfigurationError")
		}
	})
}
