package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/brightsign/bscontrol/internal/config"
	"github.com/brightsign/bscontrol/internal/types"
)

// A registered tracer must fire exactly once for a successful request and carry
// the method, status, and a non-zero total time measured by the transport.
func TestRequestTracerReportsSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	var traces []types.RequestTrace
	cfg := config.DefaultConfig()
	cfg.Timeout = 10 * time.Second
	cfg.RequestTracer = func(tr types.RequestTrace) { traces = append(traces, tr) }

	var result struct {
		OK bool `json:"ok"`
	}
	err := NewHTTPClient(cfg).Do(context.Background(), &Request{
		Method: http.MethodGet,
		URL:    srv.URL,
		Result: &result,
	})
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}

	if len(traces) != 1 {
		t.Fatalf("tracer fired %d times, want 1", len(traces))
	}
	tr := traces[0]
	if tr.Method != http.MethodGet {
		t.Errorf("Method = %q, want GET", tr.Method)
	}
	if tr.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", tr.StatusCode)
	}
	if tr.TotalTime <= 0 {
		t.Errorf("TotalTime = %v, want > 0", tr.TotalTime)
	}
	if tr.Err != nil {
		t.Errorf("Err = %v, want nil", tr.Err)
	}
}

// A request that fails must still produce exactly one trace, with a non-nil Err,
// so a tracer sees failures as well as successes.
func TestRequestTracerReportsFailure(t *testing.T) {
	// A closed server's address refuses the connection: a transport-level failure
	// with no response, so OnError (not OnAfterResponse) drives the trace.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()

	var traces []types.RequestTrace
	cfg := config.DefaultConfig()
	cfg.Timeout = 2 * time.Second
	cfg.RetryCount = 0
	cfg.RequestTracer = func(tr types.RequestTrace) { traces = append(traces, tr) }

	err := NewHTTPClient(cfg).Do(context.Background(), &Request{
		Method: http.MethodGet,
		URL:    url,
	})
	if err == nil {
		t.Fatal("Do succeeded, want a connection error")
	}

	if len(traces) != 1 {
		t.Fatalf("tracer fired %d times, want 1", len(traces))
	}
	if traces[0].Err == nil {
		t.Error("trace Err is nil, want the connection error")
	}
}

// With no tracer registered, the client must behave exactly as before and make
// requests normally.
func TestNoTracerLeavesRequestsUnaffected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := config.DefaultConfig()
	cfg.Timeout = 10 * time.Second
	if cfg.RequestTracer != nil {
		t.Fatal("default config has a tracer, want nil")
	}

	err := NewHTTPClient(cfg).Do(context.Background(), &Request{
		Method: http.MethodGet,
		URL:    srv.URL,
	})
	if err != nil {
		t.Fatalf("Do failed with no tracer: %v", err)
	}
}
