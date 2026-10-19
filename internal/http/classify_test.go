package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brightsign/bscontrol/internal/errors"
)

// A response that arrived but did not decode into the caller's result type is
// a data-shape problem, not a connectivity problem. It must not be classified
// as IsNetworkError or IsRetryableError: retrying would only re-fetch the same
// unparseable body (see TestDecodeFailureIsNotRetried in retry_test.go, which
// proves the client itself already knows not to retry it). A caller building
// its own retry/error handling on these predicates must see the same thing.
func TestDecodeFailureIsNotClassifiedAsNetworkOrRetryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"major":"notanumber"}`))
	}))
	defer srv.Close()

	var result struct {
		Major int `json:"major"`
	}
	err := testClient(0).Do(context.Background(), &Request{
		Method: http.MethodGet,
		URL:    srv.URL,
		Result: &result,
	})
	if err == nil {
		t.Fatal("Do succeeded, want a decode error")
	}
	if errors.IsNetworkError(err) {
		t.Error("IsNetworkError = true, want false: a decode failure is not a connectivity problem")
	}
	if errors.IsRetryableError(err) {
		t.Error("IsRetryableError = true, want false: retrying re-fetches the same unparseable body")
	}
}

// A genuine transport failure (no response ever arrived) must still be
// classified as a network error and as retryable — the decode-failure
// carve-out must not swallow real connectivity problems.
func TestTransportFailureIsNetworkErrorAndRetryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close() // closed before use: connection refused, no response ever arrives

	err := testClient(0).Do(context.Background(), &Request{
		Method: http.MethodGet,
		URL:    url,
	})
	if err == nil {
		t.Fatal("Do succeeded against a closed server, want a transport error")
	}
	if !errors.IsNetworkError(err) {
		t.Error("IsNetworkError = false, want true for a genuine transport failure")
	}
	if !errors.IsRetryableError(err) {
		t.Error("IsRetryableError = false, want true for a genuine transport failure")
	}
}

// PostForm's Result decode can fail the same way Do's does.
func TestPostFormDecodeFailureIsNotClassifiedAsNetworkOrRetryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"major":"notanumber"}`))
	}))
	defer srv.Close()

	var result struct {
		Major int `json:"major"`
	}
	err := testClient(0).PostForm(context.Background(), srv.URL, map[string]string{"x": "y"}, &result)
	if err == nil {
		t.Fatal("PostForm succeeded, want a decode error")
	}
	if errors.IsNetworkError(err) {
		t.Error("IsNetworkError = true, want false: a decode failure is not a connectivity problem")
	}
	if errors.IsRetryableError(err) {
		t.Error("IsRetryableError = true, want false: retrying re-fetches the same unparseable body")
	}
}

// PostFormWithAuth's Result decode can fail the same way.
func TestPostFormWithAuthDecodeFailureIsNotClassifiedAsNetworkOrRetryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"major":"notanumber"}`))
	}))
	defer srv.Close()

	var result struct {
		Major int `json:"major"`
	}
	err := testClient(0).PostFormWithAuth(context.Background(), "id", "secret", srv.URL, map[string]string{"x": "y"}, &result)
	if err == nil {
		t.Fatal("PostFormWithAuth succeeded, want a decode error")
	}
	if errors.IsNetworkError(err) {
		t.Error("IsNetworkError = true, want false: a decode failure is not a connectivity problem")
	}
	if errors.IsRetryableError(err) {
		t.Error("IsRetryableError = true, want false: retrying re-fetches the same unparseable body")
	}
}
