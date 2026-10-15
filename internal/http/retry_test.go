package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brightsign/bscontrol/internal/config"
)

func testClient(retryCount int) *HTTPClient {
	cfg := config.DefaultConfig()
	cfg.RetryCount = retryCount
	cfg.Timeout = 10 * time.Second
	return NewHTTPClient(cfg)
}

// A 200 whose body cannot be decoded into the caller's result type is a
// deterministic failure: the retry loop can only re-fetch the same unparseable
// body. Retrying it burned ~7s on GET /v1/system for firmware that reports a
// version component as a string. The request must be made exactly once.
func TestDecodeFailureIsNotRetried(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		// The real /v1/system reply is application/json, which is what makes resty
		// attempt (and here fail) the unmarshal into Result.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// "major" is a string where the result type wants an int -> decode fails.
		_, _ = w.Write([]byte(`{"major":"notanumber"}`))
	}))
	defer srv.Close()

	var result struct {
		Major int `json:"major"`
	}
	err := testClient(3).Do(context.Background(), &Request{
		Method: http.MethodGet,
		URL:    srv.URL,
		Result: &result,
	})
	if err == nil {
		t.Fatal("Do succeeded, want a decode error")
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("server was hit %d times, want 1 (a decode failure must not be retried)", got)
	}
}

// A genuine server error is still retried, so the decode-failure carve-out did
// not disable retries wholesale. With RetryCount=2 the server is hit 3 times.
func TestServerErrorIsStillRetried(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	_ = testClient(2).Do(context.Background(), &Request{
		Method: http.MethodGet,
		URL:    srv.URL,
	})
	if got := atomic.LoadInt32(&hits); got != 3 {
		t.Errorf("server was hit %d times, want 3 (initial + 2 retries)", got)
	}
}

// A non-idempotent method must NOT be retried on a server error: retrying a POST
// can duplicate a side effect (a create, a reboot) when the first attempt
// actually reached the server. With RetryCount=2 the server is hit exactly once.
func TestNonIdempotentNotRetriedOnServerError(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	_ = testClient(2).Do(context.Background(), &Request{
		Method: http.MethodPost,
		URL:    srv.URL,
		Body:   map[string]string{"x": "y"},
	})
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("POST was sent %d times, want 1 (non-idempotent must not be retried)", got)
	}
}

// An idempotent PUT is still retried on a server error, so the carve-out is
// specific to non-idempotent methods.
func TestIdempotentPutRetriedOnServerError(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	_ = testClient(2).Do(context.Background(), &Request{
		Method: http.MethodPut,
		URL:    srv.URL,
		Body:   map[string]string{"x": "y"},
	})
	if got := atomic.LoadInt32(&hits); got != 3 {
		t.Errorf("PUT was sent %d times, want 3 (initial + 2 retries)", got)
	}
}

func TestIsIdempotent(t *testing.T) {
	cases := map[string]bool{
		http.MethodGet:     true,
		http.MethodHead:    true,
		http.MethodPut:     true,
		http.MethodDelete:  true,
		http.MethodOptions: true,
		http.MethodTrace:   true,
		http.MethodPost:    false,
		http.MethodPatch:   false,
	}
	for method, want := range cases {
		if got := isIdempotent(method); got != want {
			t.Errorf("isIdempotent(%q) = %v, want %v", method, got, want)
		}
	}
}
