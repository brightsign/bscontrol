package services

import (
	"context"
	stderrors "errors"
	"net/http"
	"testing"

	"github.com/brightsign/bscontrol/internal/errors"
)

// op is one service call under test, in a form the shared behaviour tests can
// drive without knowing which service it belongs to.
type op struct {
	name string
	call func(ctx context.Context, env *testEnv) error
}

// runCommonBehaviour checks what every network-scoped call must do the same way:
// map HTTP failures to typed errors, fail before any request when auth or the
// network cannot be established, and stop on a cancelled context.
func runCommonBehaviour(t *testing.T, ops []op) {
	t.Helper()

	statuses := []struct {
		status int
		check  func(error) bool
		desc   string
	}{
		{http.StatusUnauthorized, errors.IsAuthenticationError, "authentication error"},
		{http.StatusForbidden, errors.IsAuthenticationError, "authentication error"},
		{http.StatusNotFound, func(err error) bool {
			var api *errors.APIError
			return stderrors.As(err, &api) && api.StatusCode == http.StatusNotFound && !errors.IsRetryableError(err)
		}, "non-retryable 404 APIError"},
		{http.StatusBadRequest, func(err error) bool {
			var api *errors.APIError
			return stderrors.As(err, &api) && api.StatusCode == http.StatusBadRequest
		}, "400 APIError"},
		{http.StatusInternalServerError, errors.IsRetryableError, "retryable error"},
	}

	for _, o := range ops {
		for _, st := range statuses {
			t.Run(o.name+"/http_"+http.StatusText(st.status), func(t *testing.T) {
				env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
					writeStatus(w, st.status, `{"error":"bsn_error","error_description":"nope"}`)
				})
				err := o.call(context.Background(), env)
				if err == nil {
					t.Fatalf("expected an error for HTTP %d", st.status)
				}
				if !st.check(err) {
					t.Errorf("HTTP %d: error %T %q is not a %s", st.status, err, err, st.desc)
				}
			})
		}

		t.Run(o.name+"/token_source_failure_sends_nothing", func(t *testing.T) {
			env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, `{}`) })
			env.cfg.TokenSource = staticTokenSource{err: stderrors.New("no token for you")}
			err := o.call(context.Background(), env)
			if !errors.IsAuthenticationError(err) {
				t.Fatalf("want an authentication error, got %T: %v", err, err)
			}
			env.noRequests()
		})

		t.Run(o.name+"/network_selection_failure_sends_no_api_call", func(t *testing.T) {
			env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, `{}`) })
			env.failNetworkSet = true
			err := o.call(context.Background(), env)
			if err == nil {
				t.Fatal("expected an error when the network cannot be selected")
			}
			if reqs := env.requests(false); len(reqs) != 0 {
				t.Errorf("API call made without a network: %+v", reqs)
			}
		})

		t.Run(o.name+"/cancelled_context", func(t *testing.T) {
			env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, `{}`) })
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			err := o.call(ctx, env)
			if err == nil {
				t.Fatal("expected an error for a cancelled context")
			}
			if !stderrors.Is(err, context.Canceled) {
				t.Errorf("error does not wrap context.Canceled: %T %v", err, err)
			}
			if reqs := env.requests(false); len(reqs) != 0 {
				t.Errorf("API call made with a cancelled context: %+v", reqs)
			}
		})
	}
}

// asErr is errors.As with the target type inferred.
func asErr[T error](err error, target *T) bool { return stderrors.As(err, target) }

// expectValidation asserts err is a ValidationError.
func expectValidation(t *testing.T, err error) {
	t.Helper()
	var v *errors.ValidationError
	if !stderrors.As(err, &v) {
		t.Fatalf("want *ValidationError, got %T: %v", err, err)
	}
}
