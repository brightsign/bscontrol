package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/brightsign/bscontrol/internal/config"
	gohttp "github.com/brightsign/bscontrol/internal/http"
)

// networkTestServer records every network name PUT to Self/Session/Network and
// issues a fresh, near-expiry token on each token request (so the next call
// always re-authenticates, which is the condition that used to drop the
// selected network).
type networkTestServer struct {
	*httptest.Server
	mu           sync.Mutex
	networkPuts  []string
	tokenCounter int
}

func newNetworkTestServer(t *testing.T) *networkTestServer {
	t.Helper()
	s := &networkTestServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/token"):
			s.mu.Lock()
			s.tokenCounter++
			n := s.tokenCounter
			s.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			// expires_in=1 keeps the token inside the 30s refresh margin, so the
			// next EnsureValid re-authenticates.
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "tok-" + itoa(n),
				"token_type":   "Bearer",
				"expires_in":   1,
			})
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/Self/Session/Network"):
			var body struct {
				Name string `json:"name"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			s.mu.Lock()
			s.networkPuts = append(s.networkPuts, body.Name)
			s.mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(s.Close)
	return s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func (s *networkTestServer) puts() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.networkPuts))
	copy(out, s.networkPuts)
	return out
}

func newManager(srv *networkTestServer) *AuthManager {
	cfg := config.DefaultConfig()
	cfg.ClientID = "id"
	cfg.ClientSecret = "secret"
	cfg.BSNBaseURL = srv.URL
	cfg.TokenEndpoint = srv.URL + "/token"
	return NewAuthManager(cfg, gohttp.NewHTTPClient(cfg))
}

// This is the regression test for the cross-network wire-up hazard. The caller
// selects "Prod". A later token refresh resets the server session's network.
// The SDK must re-apply the network the caller chose ("Prod"), NEVER the
// environment-configured network ("EvilNet"), which used to win after a refresh.
func TestNetworkRestoredAfterReauthIgnoresEnv(t *testing.T) {
	srv := newNetworkTestServer(t)
	am := newManager(srv)
	// Simulate WithEnv having loaded BS_NETWORK=EvilNet into the config: it seeds
	// the desired network, but an explicit SetNetwork must override it.
	am.config.NetworkName = "EvilNet"
	am.desiredNetworkName = "EvilNet"

	ctx := context.Background()
	if err := am.SetNetwork(ctx, "Prod"); err != nil {
		t.Fatalf("SetNetwork: %v", err)
	}

	// A device operation re-validates the token (which is within the refresh
	// margin, so it re-authenticates and clears the session network) and then
	// ensures the network is set again.
	if err := am.EnsureValid(ctx); err != nil {
		t.Fatalf("EnsureValid: %v", err)
	}
	if err := am.EnsureNetworkSet(ctx); err != nil {
		t.Fatalf("EnsureNetworkSet: %v", err)
	}

	puts := srv.puts()
	if len(puts) < 2 {
		t.Fatalf("expected the network to be set at least twice (initial + after refresh), got %v", puts)
	}
	for _, n := range puts {
		if n != "Prod" {
			t.Fatalf("network was set to %q; the client must only ever apply the caller's chosen network. puts=%v", n, puts)
		}
	}
}

// With no network ever selected and nothing configured, EnsureNetworkSet errors
// rather than silently selecting anything.
func TestEnsureNetworkSetErrorsWhenNoneSelected(t *testing.T) {
	srv := newNetworkTestServer(t)
	am := newManager(srv)
	if err := am.EnsureNetworkSet(context.Background()); err == nil {
		t.Fatal("EnsureNetworkSet succeeded with no network selected, want an error")
	}
}

type staticTokenSource struct {
	token  string
	expiry time.Time
	calls  int
}

func (s *staticTokenSource) Token(context.Context) (config.AccessToken, error) {
	s.calls++
	return config.AccessToken{Value: s.token, ExpiresAt: s.expiry}, nil
}

// When a TokenSource is configured, the SDK obtains tokens from it instead of
// running the client-credentials grant, and ExpiresAt reflects the source's
// expiry so callers can cache across invocations.
func TestTokenSourceIsUsed(t *testing.T) {
	cfg := config.DefaultConfig()
	exp := time.Now().Add(time.Hour)
	ts := &staticTokenSource{token: "from-source", expiry: exp}
	cfg.TokenSource = ts
	am := NewAuthManager(cfg, gohttp.NewHTTPClient(cfg))

	if err := am.Authenticate(context.Background()); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	tok, err := am.GetToken()
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	if tok != "from-source" {
		t.Errorf("token = %q, want from-source", tok)
	}
	if !am.ExpiresAt().Equal(exp) {
		t.Errorf("ExpiresAt = %v, want %v", am.ExpiresAt(), exp)
	}
	if ts.calls != 1 {
		t.Errorf("token source called %d times, want 1", ts.calls)
	}
}
