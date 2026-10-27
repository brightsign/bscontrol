package http

import (
	"bytes"
	"context"
	stderrors "errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/brightsign/bscontrol/internal/errors"
)

func TestStream(t *testing.T) {
	payload := bytes.Repeat([]byte("0123456789"), 1000)

	cases := []struct {
		name       string
		token      string
		headers    map[string]string
		status     int
		respBody   []byte
		wantErr    func(error) bool
		wantBody   []byte
		wantAuth   string
		wantAccept string
	}{
		{name: "authed success", token: "tok", status: 200, respBody: payload, wantBody: payload, wantAuth: "Bearer tok"},
		{name: "no token sends no auth", token: "", status: 200, respBody: payload, wantBody: payload, wantAuth: ""},
		{name: "custom accept", token: "tok", headers: map[string]string{"Accept": "image/*"}, status: 200, respBody: []byte("png"), wantBody: []byte("png"), wantAuth: "Bearer tok", wantAccept: "image/*"},
		{name: "401 is auth error", token: "tok", status: 401, respBody: []byte(`{"error":"unauthorized"}`),
			wantErr: errors.IsAuthenticationError, wantAuth: "Bearer tok"},
		{name: "403 is auth error", token: "tok", status: 403, respBody: []byte("nope"), wantErr: errors.IsAuthenticationError, wantAuth: "Bearer tok"},
		{name: "404 is api error", token: "tok", status: 404, respBody: []byte(`{"error":"not_found","error_description":"gone"}`),
			wantErr: func(err error) bool {
				var api *errors.APIError
				return stderrors.As(err, &api) && api.StatusCode == 404 && api.Code == "not_found"
			}, wantAuth: "Bearer tok"},
		{name: "500 is retryable", token: "tok", status: 500, respBody: []byte("boom"), wantErr: errors.IsRetryableError, wantAuth: "Bearer tok"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotAuth, gotAccept string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAuth = r.Header.Get("Authorization")
				gotAccept = r.Header.Get("Accept")
				w.WriteHeader(tc.status)
				_, _ = w.Write(tc.respBody)
			}))
			defer srv.Close()

			var buf bytes.Buffer
			n, err := testClient(0).Stream(context.Background(), tc.token, srv.URL, tc.headers, &buf)
			if tc.wantErr != nil {
				if err == nil || !tc.wantErr(err) {
					t.Fatalf("err = %v, did not match expectation", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if n != int64(len(tc.wantBody)) || !bytes.Equal(buf.Bytes(), tc.wantBody) {
					t.Errorf("body mismatch: n=%d len=%d", n, buf.Len())
				}
			}
			if gotAuth != tc.wantAuth {
				t.Errorf("Authorization = %q, want %q", gotAuth, tc.wantAuth)
			}
			if tc.wantAccept != "" && gotAccept != tc.wantAccept {
				t.Errorf("Accept = %q, want %q", gotAccept, tc.wantAccept)
			}
		})
	}
}

func TestStreamContextCancel(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	defer close(release)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := testClient(0).Stream(ctx, "tok", srv.URL, nil, io.Discard)
	if err == nil {
		t.Fatal("expected an error after the context expired")
	}
	if !errors.IsNetworkError(err) {
		t.Errorf("a cancelled stream should surface as a network error, got %T: %v", err, err)
	}
}

func TestStreamTransportFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close()
	_, err := testClient(0).Stream(context.Background(), "tok", url, nil, io.Discard)
	if !errors.IsNetworkError(err) {
		t.Errorf("expected a network error, got %v", err)
	}
}

// Do keeps defaulting to JSON, but a caller-chosen Content-Type wins. The
// Upload API's vendor media types and JSON-Patch both depend on this.
func TestDoRespectsCallerContentType(t *testing.T) {
	cases := []struct {
		name    string
		headers map[string]string
		want    string
	}{
		{"default json", nil, "application/json"},
		{"vendor type", map[string]string{"Content-Type": "application/vnd.bsn.start.content.upload.arguments.201903+json"}, "application/vnd.bsn.start.content.upload.arguments.201903+json"},
		{"json patch", map[string]string{"Content-Type": "application/json-patch+json"}, "application/json-patch+json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotCT string
			var gotBody []byte
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotCT = r.Header.Get("Content-Type")
				gotBody, _ = io.ReadAll(r.Body)
				w.WriteHeader(200)
			}))
			defer srv.Close()

			err := testClient(0).Do(context.Background(), &Request{
				Method:  http.MethodPatch,
				URL:     srv.URL,
				Body:    []byte(`[{"op":"replace"}]`),
				Headers: tc.headers,
			})
			if err != nil {
				t.Fatal(err)
			}
			if gotCT != tc.want {
				t.Errorf("Content-Type = %q, want %q", gotCT, tc.want)
			}
			if string(gotBody) != `[{"op":"replace"}]` {
				t.Errorf("body = %s", gotBody)
			}
		})
	}
}
