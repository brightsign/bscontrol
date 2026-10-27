package services

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/brightsign/bscontrol/internal/auth"
	"github.com/brightsign/bscontrol/internal/config"
	bshttp "github.com/brightsign/bscontrol/internal/http"
)

// The helpers in this file stand up a fake BSN.cloud for the Content,
// Presentation, Schedule and Upload tests: an httptest server that records every
// request, answers the network-selection call the SDK makes before any scoped
// operation, and checks the bearer token. Nothing here touches the real network.

const (
	testAPIVersion = "2022/06/REST"
	testToken      = "test-token"
	testNetwork    = "TestNet"
)

// recordedRequest is one request the fake server saw.
type recordedRequest struct {
	Method   string
	Path     string // decoded path
	RawPath  string // escaped path as sent (RequestURI without the query)
	RawQuery string
	Header   http.Header
	Body     []byte
}

// staticTokenSource hands out a fixed token, so no OAuth endpoint is needed.
type staticTokenSource struct {
	err error
}

func (s staticTokenSource) Token(context.Context) (config.AccessToken, error) {
	if s.err != nil {
		return config.AccessToken{}, s.err
	}
	return config.AccessToken{Value: testToken, ExpiresAt: time.Now().Add(time.Hour)}, nil
}

// testEnv is a fake BSN plus the SDK plumbing pointed at it.
type testEnv struct {
	t   *testing.T
	srv *httptest.Server
	cfg *config.Config
	hc  *bshttp.HTTPClient
	am  *auth.AuthManager

	mu   sync.Mutex
	reqs []recordedRequest
	// failNetworkSet makes the network-selection call fail with 500.
	failNetworkSet bool
}

// newTestEnv starts the fake server. handler answers everything except the
// network-selection call. Retries are off so error cases return immediately.
func newTestEnv(t *testing.T, handler http.HandlerFunc) *testEnv {
	t.Helper()
	env := &testEnv{t: t}
	env.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		env.mu.Lock()
		env.reqs = append(env.reqs, recordedRequest{
			Method:   r.Method,
			Path:     r.URL.Path,
			RawPath:  strings.SplitN(r.RequestURI, "?", 2)[0],
			RawQuery: r.URL.RawQuery,
			Header:   r.Header.Clone(),
			Body:     body,
		})
		fail := env.failNetworkSet
		env.mu.Unlock()

		if r.Method == http.MethodPut && r.URL.Path == "/"+testAPIVersion+"/Self/Session/Network" {
			if fail {
				http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		handler(w, r)
	}))
	t.Cleanup(env.srv.Close)

	env.cfg = config.DefaultConfig()
	env.cfg.BSNBaseURL = env.srv.URL
	env.cfg.APIVersion = testAPIVersion
	env.cfg.RetryCount = 0
	env.cfg.Timeout = 10 * time.Second
	env.cfg.NetworkName = testNetwork
	env.cfg.TokenSource = staticTokenSource{}
	env.hc = bshttp.NewHTTPClient(env.cfg)
	env.am = auth.NewAuthManager(env.cfg, env.hc)
	return env
}

// requests returns a copy of everything recorded, excluding the network-selection
// call unless includeSession is true.
func (e *testEnv) requests(includeSession bool) []recordedRequest {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []recordedRequest
	for _, r := range e.reqs {
		if !includeSession && r.Path == "/"+testAPIVersion+"/Self/Session/Network" {
			continue
		}
		out = append(out, r)
	}
	return out
}

// only asserts exactly one request (besides network selection) was made and
// returns it.
func (e *testEnv) only() recordedRequest {
	e.t.Helper()
	reqs := e.requests(false)
	if len(reqs) != 1 {
		e.t.Fatalf("expected exactly 1 request, got %d: %+v", len(reqs), reqs)
	}
	return reqs[0]
}

// noRequests asserts nothing reached the server (validation short-circuited).
func (e *testEnv) noRequests() {
	e.t.Helper()
	if reqs := e.requests(true); len(reqs) != 0 {
		e.t.Fatalf("expected no requests, got %+v", reqs)
	}
}

// networkSelections counts how many times the session network was selected.
func (e *testEnv) networkSelections() int {
	n := 0
	for _, r := range e.requests(true) {
		if r.Path == "/"+testAPIVersion+"/Self/Session/Network" {
			n++
		}
	}
	return n
}

// writeJSON replies 200 with a JSON body.
func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, body)
}

// writeStatus replies with a status and a JSON error body.
func writeStatus(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

// requireBearer asserts the request carried the test token.
func requireBearer(t *testing.T, r recordedRequest) {
	t.Helper()
	if got := r.Header.Get("Authorization"); got != "Bearer "+testToken {
		t.Errorf("Authorization = %q, want Bearer %s", got, testToken)
	}
}
