package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Error types for different categories of API errors

// APIError represents an error response from the BSN.cloud API.
type APIError struct {
	StatusCode int    `json:"status_code"`
	Code       string `json:"error"`
	Message    string `json:"error_description"`
	Details    string `json:"details,omitempty"`
}

func (e *APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("API error %d (%s): %s - %s", e.StatusCode, e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("API error %d (%s): %s", e.StatusCode, e.Code, e.Message)
}

// AuthenticationError indicates authentication-related errors.
type AuthenticationError struct {
	Reason string
	Err    error
}

func (e *AuthenticationError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("authentication failed: %s: %v", e.Reason, e.Err)
	}
	return fmt.Sprintf("authentication failed: %s", e.Reason)
}

func (e *AuthenticationError) Unwrap() error {
	return e.Err
}

// NetworkError indicates network-related errors.
type NetworkError struct {
	Operation string
	Err       error
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("network error during %s: %v", e.Operation, e.Err)
}

func (e *NetworkError) Unwrap() error {
	return e.Err
}

// DecodeError indicates a response was received but its body could not be
// decoded into the caller's expected type. This is deliberately NOT a
// NetworkError: a response did arrive, so there is no connectivity problem,
// and retrying would only re-fetch the same unparseable body. It
// intentionally matches neither IsNetworkError nor IsRetryableError.
type DecodeError struct {
	Operation string
	Err       error
}

func (e *DecodeError) Error() string {
	return fmt.Sprintf("failed to decode response for %s: %v", e.Operation, e.Err)
}

func (e *DecodeError) Unwrap() error {
	return e.Err
}

// AppError indicates an application-level failure reported inside an
// otherwise successful (2xx) HTTP response body — for example a B-Deploy
// JSON envelope carrying its own "error" field, or a response missing a
// field the caller required. It deliberately has no StatusCode: unlike
// APIError, there is no real HTTP status to report, and claiming one (such as
// 0) would misrepresent this as a malformed or unknown-status API error.
type AppError struct {
	Code    string
	Message string
	Details string
}

func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("application error (%s): %s - %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("application error (%s): %s", e.Code, e.Message)
}

// ConfigurationError indicates configuration-related errors.
type ConfigurationError struct {
	Field   string
	Reason  string
	Suggest string
}

func (e *ConfigurationError) Error() string {
	msg := fmt.Sprintf("configuration error: %s - %s", e.Field, e.Reason)
	if e.Suggest != "" {
		msg += fmt.Sprintf(" (suggestion: %s)", e.Suggest)
	}
	return msg
}

// ValidationError indicates data validation errors.
type ValidationError struct {
	Field  string
	Value  interface{}
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s=%v - %s", e.Field, e.Value, e.Reason)
}

// Helper functions for creating common errors

// NewAPIError creates an APIError from an HTTP response.
func NewAPIError(statusCode int, code, message, details string) *APIError {
	return &APIError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
		Details:    details,
	}
}

// NewAuthError creates an AuthenticationError.
func NewAuthError(reason string, err error) *AuthenticationError {
	return &AuthenticationError{
		Reason: reason,
		Err:    err,
	}
}

// NewNetworkError creates a NetworkError.
func NewNetworkError(operation string, err error) *NetworkError {
	return &NetworkError{
		Operation: operation,
		Err:       err,
	}
}

// NewDecodeError creates a DecodeError.
func NewDecodeError(operation string, err error) *DecodeError {
	return &DecodeError{
		Operation: operation,
		Err:       err,
	}
}

// NewAppError creates an AppError.
func NewAppError(code, message, details string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// NewConfigError creates a ConfigurationError.
func NewConfigError(field, reason, suggest string) *ConfigurationError {
	return &ConfigurationError{
		Field:   field,
		Reason:  reason,
		Suggest: suggest,
	}
}

// NewValidationError creates a ValidationError.
func NewValidationError(field string, value interface{}, reason string) *ValidationError {
	return &ValidationError{
		Field:  field,
		Value:  value,
		Reason: reason,
	}
}

// IsAuthenticationError checks if an error is authentication-related.
//
// It unwraps the error chain (errors.As) so a service-layer error that wraps
// the HTTP layer's typed error is still classified correctly.
func IsAuthenticationError(err error) bool {
	var authErr *AuthenticationError
	if errors.As(err, &authErr) {
		return true
	}

	// Also check for API errors with authentication status codes
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden
	}

	return false
}

// IsNetworkError checks if an error is network-related.
func IsNetworkError(err error) bool {
	var netErr *NetworkError
	return errors.As(err, &netErr)
}

// IsConfigurationError checks if an error is configuration-related.
func IsConfigurationError(err error) bool {
	var configErr *ConfigurationError
	return errors.As(err, &configErr)
}

// IsRetryableError checks if an error might succeed on retry.
func IsRetryableError(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		// Retry on server errors and rate limiting
		return apiErr.StatusCode >= 500 || apiErr.StatusCode == http.StatusTooManyRequests
	}

	var netErr *NetworkError
	if errors.As(err, &netErr) {
		// Retry network errors (connection failures, etc.)
		return true
	}

	return false
}
