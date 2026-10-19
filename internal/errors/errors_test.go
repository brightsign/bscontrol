package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestAPIError(t *testing.T) {
	err := NewAPIError(400, "invalid_request", "Invalid request format", "Missing required field")

	expected := "API error 400 (invalid_request): Invalid request format - Missing required field"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}

	// Test without details
	err = NewAPIError(404, "not_found", "Resource not found", "")
	expected = "API error 404 (not_found): Resource not found"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}

func TestAppError(t *testing.T) {
	err := NewAppError("bdeploy_api_error", "B-Deploy API returned an error", "some detail")

	expected := "application error (bdeploy_api_error): B-Deploy API returned an error - some detail"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}

	// Test without details
	err = NewAppError("invalid_token_response", "API returned empty token", "")
	expected = "application error (invalid_token_response): API returned empty token"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}

// AppError represents an application-level failure embedded in an otherwise
// successful HTTP response. It must not be confused with APIError, which
// always carries a real HTTP status code; AppError has none to give.
func TestAppErrorIsNotAnAPIError(t *testing.T) {
	err := NewAppError("bdeploy_api_error", "B-Deploy API returned an error", "")

	var apiErr *APIError
	if errors.As(error(err), &apiErr) {
		t.Error("AppError should not match errors.As(*APIError) — it carries no real HTTP status")
	}
}

func TestAuthenticationError(t *testing.T) {
	// Test with wrapped error
	innerErr := errors.New("connection failed")
	err := NewAuthError("token expired", innerErr)

	expected := "authentication failed: token expired: connection failed"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}

	// Test unwrapping
	if unwrapped := err.Unwrap(); unwrapped != innerErr {
		t.Errorf("Expected unwrapped error to be %v, got %v", innerErr, unwrapped)
	}

	// Test without wrapped error
	err = NewAuthError("invalid credentials", nil)
	expected = "authentication failed: invalid credentials"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}

func TestNetworkError(t *testing.T) {
	innerErr := errors.New("connection timeout")
	err := NewNetworkError("GET /devices", innerErr)

	expected := "network error during GET /devices: connection timeout"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}

	// Test unwrapping
	if unwrapped := err.Unwrap(); unwrapped != innerErr {
		t.Errorf("Expected unwrapped error to be %v, got %v", innerErr, unwrapped)
	}
}

func TestConfigurationError(t *testing.T) {
	err := NewConfigError("ClientID", "field is required", "set BS_CLIENT_ID environment variable")

	expected := "configuration error: ClientID - field is required (suggestion: set BS_CLIENT_ID environment variable)"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}

	// Test without suggestion
	err = NewConfigError("Timeout", "must be positive", "")
	expected = "configuration error: Timeout - must be positive"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}

func TestValidationError(t *testing.T) {
	err := NewValidationError("networkName", "", "cannot be empty")

	expected := "validation error: networkName= - cannot be empty"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}

func TestIsAuthenticationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "AuthenticationError",
			err:      NewAuthError("test", nil),
			expected: true,
		},
		{
			name:     "APIError 401",
			err:      NewAPIError(http.StatusUnauthorized, "unauthorized", "Invalid token", ""),
			expected: true,
		},
		{
			name:     "APIError 403",
			err:      NewAPIError(http.StatusForbidden, "forbidden", "Insufficient permissions", ""),
			expected: true,
		},
		{
			name:     "APIError 400",
			err:      NewAPIError(http.StatusBadRequest, "bad_request", "Invalid request", ""),
			expected: false,
		},
		{
			name:     "NetworkError",
			err:      NewNetworkError("test", errors.New("connection failed")),
			expected: false,
		},
		{
			name:     "Generic error",
			err:      errors.New("some error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAuthenticationError(tt.err)
			if result != tt.expected {
				t.Errorf("IsAuthenticationError() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "APIError 500",
			err:      NewAPIError(http.StatusInternalServerError, "internal_error", "Server error", ""),
			expected: true,
		},
		{
			name:     "APIError 502",
			err:      NewAPIError(http.StatusBadGateway, "bad_gateway", "Bad gateway", ""),
			expected: true,
		},
		{
			name:     "APIError 429",
			err:      NewAPIError(http.StatusTooManyRequests, "rate_limit", "Rate limit exceeded", ""),
			expected: true,
		},
		{
			name:     "APIError 400",
			err:      NewAPIError(http.StatusBadRequest, "bad_request", "Invalid request", ""),
			expected: false,
		},
		{
			name:     "APIError 404",
			err:      NewAPIError(http.StatusNotFound, "not_found", "Not found", ""),
			expected: false,
		},
		{
			name:     "NetworkError",
			err:      NewNetworkError("test", errors.New("connection failed")),
			expected: true,
		},
		{
			name:     "AuthenticationError",
			err:      NewAuthError("test", nil),
			expected: false,
		},
		{
			name:     "Generic error",
			err:      errors.New("some error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetryableError() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
