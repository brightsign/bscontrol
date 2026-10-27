package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/brightsign/bscontrol/internal/errors"
)

var _ ContentService = (*contentService)(nil)

func newContent(env *testEnv) ContentService {
	return NewContentService(env.cfg, env.hc, env.am)
}

const contentPath = "/" + testAPIVersion + "/Content"

// contentOps is every ContentService call with valid arguments, for the shared
// failure-mode tests.
func contentOps() []op {
	return []op{
		{"List", func(ctx context.Context, e *testEnv) error { _, err := newContent(e).List(ctx); return err }},
		{"ListAll", func(ctx context.Context, e *testEnv) error { _, err := newContent(e).ListAll(ctx); return err }},
		{"ListFolder", func(ctx context.Context, e *testEnv) error {
			_, err := newContent(e).ListFolder(ctx, "videos")
			return err
		}},
		{"GetCount", func(ctx context.Context, e *testEnv) error { _, err := newContent(e).GetCount(ctx); return err }},
		{"GetByID", func(ctx context.Context, e *testEnv) error { _, err := newContent(e).GetByID(ctx, 5); return err }},
		{"Delete", func(ctx context.Context, e *testEnv) error {
			_, err := newContent(e).Delete(ctx, "[FileName] IS 'a'")
			return err
		}},
		{"DeleteByID", func(ctx context.Context, e *testEnv) error { return newContent(e).DeleteByID(ctx, 5) }},
		{"Download", func(ctx context.Context, e *testEnv) error { _, err := newContent(e).Download(ctx, 5); return err }},
		{"GetThumbnail", func(ctx context.Context, e *testEnv) error {
			_, err := newContent(e).GetThumbnail(ctx, 5, 0, 0)
			return err
		}},
		{"CreateFolder", func(ctx context.Context, e *testEnv) error {
			_, err := newContent(e).CreateFolder(ctx, "", "f")
			return err
		}},
		{"Move", func(ctx context.Context, e *testEnv) error { return newContent(e).Move(ctx, 5, "/x") }},
	}
}

func TestContentService_CommonBehaviour(t *testing.T) {
	runCommonBehaviour(t, contentOps())
}

func TestContentService_NetworkSelectedOncePerClient(t *testing.T) {
	env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, `{"items":[]}`) })
	svc := newContent(env)
	for i := 0; i < 3; i++ {
		if _, err := svc.List(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if n := env.networkSelections(); n != 1 {
		t.Errorf("network selected %d times, want 1", n)
	}
}

func TestContentService_List(t *testing.T) {
	cases := []struct {
		name      string
		opts      []ListOption
		wantQuery url.Values
	}{
		{"no options", nil, url.Values{}},
		{"all options", []ListOption{WithPageSize(25), WithMarker("m1"), WithFilter(`[FileName] IS 'a b'`), WithSort("fileName")},
			url.Values{"pageSize": {"25"}, "marker": {"m1"}, "filter": {`[FileName] IS 'a b'`}, "sort": {"fileName"}}},
		{"hostile filter stays one parameter", []ListOption{WithFilter(`x&marker=evil#frag`)},
			url.Values{"filter": {`x&marker=evil#frag`}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, `{"items":[{"id":1,"fileName":"a.png","mediaType":"Image","fileSize":10,"uploadDate":"2026-08-18T12:00:00"}],
				"isTruncated":true,"nextMarker":"next-1"}`)
			})
			got, err := newContent(env).List(context.Background(), tc.opts...)
			if err != nil {
				t.Fatal(err)
			}
			req := env.only()
			requireBearer(t, req)
			if req.Method != http.MethodGet || req.Path != contentPath {
				t.Errorf("request = %s %s", req.Method, req.Path)
			}
			q, _ := url.ParseQuery(req.RawQuery)
			if fmt.Sprint(q) != fmt.Sprint(tc.wantQuery) {
				t.Errorf("query = %v, want %v", q, tc.wantQuery)
			}
			if len(got.Items) != 1 || got.Items[0].FileName != "a.png" || !got.IsTruncated || got.NextMarker != "next-1" {
				t.Errorf("result = %+v", got)
			}
		})
	}
}

func TestContentService_List_DecodeFailureIsNotRetryable(t *testing.T) {
	env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, `{"items":"not a list"}`) })
	_, err := newContent(env).List(context.Background())
	if err == nil {
		t.Fatal("expected a decode error")
	}
	if errors.IsNetworkError(err) || errors.IsRetryableError(err) {
		t.Errorf("a decode failure must not look like a network/retryable error: %v", err)
	}
}

func TestContentService_ListAll(t *testing.T) {
	t.Run("follows markers across pages", func(t *testing.T) {
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Query().Get("marker") {
			case "":
				writeJSON(w, `{"items":[{"id":1,"fileName":"a"},{"id":2,"fileName":"b"}],"isTruncated":true,"nextMarker":"p2"}`)
			case "p2":
				writeJSON(w, `{"items":[{"id":3,"fileName":"c"}],"isTruncated":true,"nextMarker":"p3"}`)
			case "p3":
				writeJSON(w, `{"items":[{"id":4,"fileName":"d"}],"isTruncated":false}`)
			default:
				t.Errorf("unexpected marker %q", r.URL.Query().Get("marker"))
			}
		})
		got, err := newContent(env).ListAll(context.Background(), WithPageSize(2))
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 4 || got[0].ID != 1 || got[3].ID != 4 {
			t.Errorf("got %+v", got)
		}
		reqs := env.requests(false)
		if len(reqs) != 3 {
			t.Fatalf("made %d requests, want 3", len(reqs))
		}
		for _, r := range reqs {
			if !strings.Contains(r.RawQuery, "pageSize=2") {
				t.Errorf("pageSize dropped on a later page: %s", r.RawQuery)
			}
		}
	})

	t.Run("empty", func(t *testing.T) {
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, `{"items":[],"isTruncated":false}`) })
		got, err := newContent(env).ListAll(context.Background())
		if err != nil || len(got) != 0 {
			t.Errorf("got %v, %v", got, err)
		}
	})

	t.Run("truncated without a marker is an error", func(t *testing.T) {
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, `{"items":[{"id":1}],"isTruncated":true}`) })
		_, err := newContent(env).ListAll(context.Background())
		var app *errors.AppError
		if err == nil || !asErr(err, &app) || app.Code != "pagination_no_marker" {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("repeated marker is an error", func(t *testing.T) {
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, `{"items":[{"id":1}],"isTruncated":true,"nextMarker":"same"}`)
		})
		_, err := newContent(env).ListAll(context.Background())
		var app *errors.AppError
		if err == nil || !asErr(err, &app) || app.Code != "pagination_loop" {
			t.Errorf("err = %v", err)
		}
		if n := len(env.requests(false)); n != 2 {
			t.Errorf("made %d requests before giving up, want 2", n)
		}
	})

	t.Run("a failing page stops the walk", func(t *testing.T) {
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("marker") == "" {
				writeJSON(w, `{"items":[{"id":1}],"isTruncated":true,"nextMarker":"p2"}`)
				return
			}
			writeStatus(w, 500, `{}`)
		})
		if _, err := newContent(env).ListAll(context.Background()); !errors.IsRetryableError(err) {
			t.Errorf("err = %v", err)
		}
	})
}

func TestContentService_ListFolder(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		wantRaw  string
		wantFail bool
	}{
		{"root", "", contentPath + "/Root/", false},
		{"slash root", "/", contentPath + "/Root/", false},
		{"nested", "videos/promos", contentPath + "/Root/videos/promos", false},
		{"leading slash", "/videos/", contentPath + "/Root/videos", false},
		{"backslash form", `\videos\promos\`, contentPath + "/Root/videos/promos", false},
		{"spaces and specials escaped", "my videos/a?b#c", contentPath + "/Root/my%20videos/a%3Fb%23c", false},
		{"traversal rejected", "videos/../secret", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, `{"items":[{"id":1,"fileName":"x"}]}`) })
			got, err := newContent(env).ListFolder(context.Background(), tc.path, WithPageSize(5))
			if tc.wantFail {
				expectValidation(t, err)
				env.noRequests()
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			req := env.only()
			if req.RawPath != tc.wantRaw {
				t.Errorf("path = %q, want %q", req.RawPath, tc.wantRaw)
			}
			if req.RawQuery != "pageSize=5" {
				t.Errorf("query = %q", req.RawQuery)
			}
			if len(got.Items) != 1 {
				t.Errorf("got %+v", got)
			}
		})
	}
}

func TestContentService_GetCount(t *testing.T) {
	cases := []struct {
		name string
		body string
		want int
	}{
		{"bare number", `42`, 42},
		{"object count", `{"count":7}`, 7},
		{"object totalCount", `{"totalCount":9}`, 9},
		{"zero", `0`, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, tc.body) })
			got, err := newContent(env).GetCount(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if got.Count != tc.want {
				t.Errorf("count = %d, want %d", got.Count, tc.want)
			}
			if req := env.only(); req.Path != contentPath+"/Count" {
				t.Errorf("path = %s", req.Path)
			}
		})
	}
}

func TestContentService_GetByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, `{"id":12,"fileName":"a.png","virtualPath":"\\img\\","mediaType":"Image","fileHash":"SHA1:ab","uploadDate":"2026-08-18T12:00:00"}`)
		})
		got, err := newContent(env).GetByID(context.Background(), 12)
		if err != nil {
			t.Fatal(err)
		}
		req := env.only()
		requireBearer(t, req)
		if req.Path != contentPath+"/12" || req.Method != http.MethodGet {
			t.Errorf("request = %s %s", req.Method, req.Path)
		}
		if got.ID != 12 || got.FileName != "a.png" || got.VirtualPath != `\img\` || got.UploadDate.IsZero() {
			t.Errorf("got %+v", got)
		}
	})
	for _, id := range []int{0, -3} {
		t.Run(fmt.Sprintf("invalid id %d", id), func(t *testing.T) {
			env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, `{}`) })
			_, err := newContent(env).GetByID(context.Background(), id)
			expectValidation(t, err)
			env.noRequests()
		})
	}
}

func TestContentService_Delete(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		status    int
		wantCount int
	}{
		{"count only", `3`, 200, 3},
		{"object", `{"deletedCount":2,"deletedIds":[4,5]}`, 200, 2},
		{"empty body", ``, 204, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
				if tc.body != "" {
					w.Header().Set("Content-Type", "application/json")
				}
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			})
			filter := `[FileName] IS 'it''s & ok' AND [VirtualPath] IS '\\videos\\'`
			got, err := newContent(env).Delete(context.Background(), filter)
			if err != nil {
				t.Fatal(err)
			}
			req := env.only()
			requireBearer(t, req)
			if req.Method != http.MethodDelete || req.Path != contentPath {
				t.Errorf("request = %s %s", req.Method, req.Path)
			}
			q, _ := url.ParseQuery(req.RawQuery)
			if q.Get("filter") != filter || len(q) != 1 {
				t.Errorf("filter did not survive intact: %v", q)
			}
			if got.DeletedCount != tc.wantCount {
				t.Errorf("DeletedCount = %d, want %d", got.DeletedCount, tc.wantCount)
			}
		})
	}

	t.Run("empty filter refused before any request", func(t *testing.T) {
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, `1`) })
		_, err := newContent(env).Delete(context.Background(), "")
		expectValidation(t, err)
		env.noRequests()
	})
}

func TestContentService_DeleteByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
		if err := newContent(env).DeleteByID(context.Background(), 9); err != nil {
			t.Fatal(err)
		}
		req := env.only()
		requireBearer(t, req)
		if req.Method != http.MethodDelete || req.Path != contentPath+"/9" {
			t.Errorf("request = %s %s", req.Method, req.Path)
		}
	})
	for _, id := range []int{0, -1} {
		t.Run(fmt.Sprintf("invalid id %d", id), func(t *testing.T) {
			env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {})
			expectValidation(t, newContent(env).DeleteByID(context.Background(), id))
			env.noRequests()
		})
	}
}

// Download is two steps: read the entity, then fetch the URL it names. These
// cases pin which URL is fetched and whether the bearer token goes with it.
func TestContentService_Download(t *testing.T) {
	payload := bytes.Repeat([]byte("bytes!"), 5000)

	t.Run("fileUrl on the BSN host carries the token", func(t *testing.T) {
		var env *testEnv
		env = newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case contentPath + "/5":
				writeJSON(w, fmt.Sprintf(`{"id":5,"fileName":"a.bin","fileUrl":%q}`, env.srv.URL+"/static/a.bin"))
			case "/static/a.bin":
				_, _ = w.Write(payload)
			default:
				t.Errorf("unexpected path %s", r.URL.Path)
			}
		})
		got, err := newContent(env).Download(context.Background(), 5)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, payload) {
			t.Errorf("payload mismatch: %d bytes", len(got))
		}
		reqs := env.requests(false)
		if len(reqs) != 2 {
			t.Fatalf("%d requests, want 2 (entity, then file)", len(reqs))
		}
		if reqs[0].Path != contentPath+"/5" || reqs[1].Path != "/static/a.bin" {
			t.Errorf("sequence = %s, %s", reqs[0].Path, reqs[1].Path)
		}
		requireBearer(t, reqs[1])
		for _, r := range reqs {
			if strings.HasSuffix(r.Path, "/Download") {
				t.Errorf("the unverified /Download route must not be used: %s", r.Path)
			}
		}
	})

	t.Run("relative fileUrl resolves against the API host", func(t *testing.T) {
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == contentPath+"/5" {
				writeJSON(w, `{"id":5,"fileUrl":"/static/rel.bin"}`)
				return
			}
			_, _ = w.Write([]byte("rel"))
		})
		got, err := newContent(env).Download(context.Background(), 5)
		if err != nil || string(got) != "rel" {
			t.Fatalf("got %q, %v", got, err)
		}
		requireBearer(t, env.requests(false)[1])
	})

	t.Run("off-host URL does not receive the token", func(t *testing.T) {
		var gotAuth string
		other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			_, _ = w.Write(payload)
		}))
		defer other.Close()
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, fmt.Sprintf(`{"id":5,"physicalPath":%q}`, other.URL+"/bucket/key?sig=abc"))
		})
		got, err := newContent(env).Download(context.Background(), 5)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, payload) {
			t.Error("payload mismatch")
		}
		if gotAuth != "" {
			t.Errorf("bearer token leaked to another host: %q", gotAuth)
		}
	})

	t.Run("fileUrl wins over physicalPath", func(t *testing.T) {
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == contentPath+"/5" {
				writeJSON(w, `{"id":5,"fileUrl":"/static/right","physicalPath":"/static/wrong"}`)
				return
			}
			_, _ = w.Write([]byte(r.URL.Path))
		})
		got, err := newContent(env).Download(context.Background(), 5)
		if err != nil || string(got) != "/static/right" {
			t.Fatalf("got %q, %v", got, err)
		}
	})

	badEntities := []struct {
		name string
		body string
		code string
	}{
		{"no url at all", `{"id":5}`, "content_no_download_url"},
		{"unsupported scheme", `{"id":5,"fileUrl":"file:///etc/passwd"}`, "content_bad_download_url"},
		{"ftp scheme", `{"id":5,"fileUrl":"ftp://host/x"}`, "content_bad_download_url"},
		{"unparseable", `{"id":5,"fileUrl":"http://[::1"}`, "content_bad_download_url"},
	}
	for _, tc := range badEntities {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, tc.body) })
			_, err := newContent(env).Download(context.Background(), 5)
			var app *errors.AppError
			if err == nil || !asErr(err, &app) || app.Code != tc.code {
				t.Fatalf("err = %v, want AppError %s", err, tc.code)
			}
			if n := len(env.requests(false)); n != 1 {
				t.Errorf("made %d requests, want only the entity read", n)
			}
		})
	}

	t.Run("file fetch failure is typed", func(t *testing.T) {
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == contentPath+"/5" {
				writeJSON(w, `{"id":5,"fileUrl":"/static/gone"}`)
				return
			}
			writeStatus(w, 404, `{"error":"missing"}`)
		})
		_, err := newContent(env).Download(context.Background(), 5)
		var api *errors.APIError
		if err == nil || !asErr(err, &api) || api.StatusCode != 404 {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("invalid id and nil writer", func(t *testing.T) {
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {})
		_, err := newContent(env).Download(context.Background(), 0)
		expectValidation(t, err)
		_, err = newContent(env).DownloadTo(context.Background(), 5, nil)
		expectValidation(t, err)
		env.noRequests()
	})
}

func TestContentService_DownloadTo_Streams(t *testing.T) {
	payload := bytes.Repeat([]byte("z"), 1<<20)
	env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == contentPath+"/7" {
			writeJSON(w, `{"id":7,"fileUrl":"/f"}`)
			return
		}
		_, _ = w.Write(payload)
	})
	var buf bytes.Buffer
	n, err := newContent(env).DownloadTo(context.Background(), 7, &buf)
	if err != nil || n != int64(len(payload)) || buf.Len() != len(payload) {
		t.Fatalf("n=%d len=%d err=%v", n, buf.Len(), err)
	}
}

func TestContentService_GetThumbnail(t *testing.T) {
	cases := []struct {
		name      string
		w, h      int
		wantQuery string
	}{
		{"defaults", 0, 0, ""},
		{"both", 200, 100, "maxHeight=100&maxWidth=200"},
		{"width only", 64, 0, "maxWidth=64"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "image/png")
				_, _ = w.Write([]byte("\x89PNGdata"))
			})
			got, err := newContent(env).GetThumbnail(context.Background(), 5, tc.w, tc.h)
			if err != nil {
				t.Fatal(err)
			}
			req := env.only()
			requireBearer(t, req)
			if req.Path != contentPath+"/5" || req.RawQuery != tc.wantQuery {
				t.Errorf("request = %s ?%s", req.Path, req.RawQuery)
			}
			if req.Header.Get("Accept") != "image/*" {
				t.Errorf("Accept = %q", req.Header.Get("Accept"))
			}
			if string(got) != "\x89PNGdata" {
				t.Errorf("body = %q", got)
			}
		})
	}
	for name, args := range map[string][3]int{"bad id": {0, 0, 0}, "negative width": {5, -1, 0}, "negative height": {5, 0, -1}} {
		t.Run(name, func(t *testing.T) {
			env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {})
			_, err := newContent(env).GetThumbnail(context.Background(), args[0], args[1], args[2])
			expectValidation(t, err)
			env.noRequests()
		})
	}
}

func TestContentService_CreateFolder(t *testing.T) {
	t.Run("posts name under the parent path", func(t *testing.T) {
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"id":30,"fileName":"New Folder","virtualPath":"\\videos\\","mediaType":"Folder"}`)
		})
		got, err := newContent(env).CreateFolder(context.Background(), "/videos/promo", "New Folder")
		if err != nil {
			t.Fatal(err)
		}
		req := env.only()
		requireBearer(t, req)
		if req.Method != http.MethodPost || req.RawPath != contentPath+"/Root/videos/promo" {
			t.Errorf("request = %s %s", req.Method, req.RawPath)
		}
		var body map[string]string
		if err := json.Unmarshal(req.Body, &body); err != nil || body["name"] != "New Folder" || len(body) != 1 {
			t.Errorf("body = %s (%v)", req.Body, err)
		}
		if got.ID != 30 || got.MediaType != "Folder" {
			t.Errorf("got %+v", got)
		}
	})

	bad := map[string][2]string{
		"empty name":          {"", ""},
		"slash in name":       {"", "a/b"},
		"backslash":           {"", `a\b`},
		"dot dot":             {"", ".."},
		"control char":        {"", "a\nb"},
		"traversal in parent": {"a/../b", "x"},
	}
	for name, args := range bad {
		t.Run(name, func(t *testing.T) {
			env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {})
			_, err := newContent(env).CreateFolder(context.Background(), args[0], args[1])
			expectValidation(t, err)
			env.noRequests()
		})
	}
}

func TestContentService_Move(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"posix", "videos/promos", `\videos\promos\`},
		{"leading slash", "/archive/", `\archive\`},
		{"root", "", `\`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
			if err := newContent(env).Move(context.Background(), 8, tc.in); err != nil {
				t.Fatal(err)
			}
			req := env.only()
			requireBearer(t, req)
			if req.Method != http.MethodPatch || req.Path != contentPath+"/8/" {
				t.Errorf("request = %s %s", req.Method, req.Path)
			}
			if ct := req.Header.Get("Content-Type"); ct != "application/json-patch+json" {
				t.Errorf("Content-Type = %q", ct)
			}
			var ops []map[string]string
			if err := json.Unmarshal(req.Body, &ops); err != nil || len(ops) != 1 {
				t.Fatalf("body = %s (%v)", req.Body, err)
			}
			if ops[0]["op"] != "replace" || ops[0]["path"] != "/virtualPath/" || ops[0]["value"] != tc.want {
				t.Errorf("patch = %v, want value %q", ops[0], tc.want)
			}
		})
	}
	t.Run("validation", func(t *testing.T) {
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {})
		expectValidation(t, newContent(env).Move(context.Background(), 0, "x"))
		expectValidation(t, newContent(env).Move(context.Background(), 1, "a/../b"))
		env.noRequests()
	})
}

func TestToBSNVirtualPath(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"videos/promos", `\videos\promos\`, false},
		{"/videos/", `\videos\`, false},
		{"/", `\`, false},
		{"", `\`, false},
		{`\`, `\`, false},
		{`\videos\promos\`, `\videos\promos\`, false},
		{"videos//promos", `\videos\promos\`, false},
		{"./videos/./a", `\videos\a\`, false},
		{`mixed/slashes\here`, `\mixed\slashes\here\`, false},
		{"with space/ünïcode", `\with space\ünïcode\`, false},
		{"a/../b", "", true},
		{"..", "", true},
		{"a\x00b", "", true},
		{"a\nb", "", true},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%q", tc.in), func(t *testing.T) {
			got, err := ToBSNVirtualPath(tc.in)
			if tc.wantErr {
				expectValidation(t, err)
				return
			}
			if err != nil || got != tc.want {
				t.Errorf("ToBSNVirtualPath(%q) = %q, %v; want %q", tc.in, got, err, tc.want)
			}
			// Idempotent: the output is already in BSN form.
			again, err := ToBSNVirtualPath(got)
			if err != nil || again != got {
				t.Errorf("not idempotent: %q -> %q (%v)", got, again, err)
			}
		})
	}
}

func TestEscapeFilterValue(t *testing.T) {
	cases := map[string]string{
		"plain":      "plain",
		"it's":       "it''s",
		`\videos\`:   `\\videos\\`,
		`a'\b`:       `a''\\b`,
		"":           "",
		"'; DROP --": "''; DROP --",
	}
	for in, want := range cases {
		if got := EscapeFilterValue(in); got != want {
			t.Errorf("EscapeFilterValue(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDetectMediaType(t *testing.T) {
	cases := map[string]string{
		"a.jpg": "Image", "a.JPEG": "Image", "a.png": "Image", "a.bmp": "Image",
		"a.mp4": "Video", "a.MOV": "Video", "a.mkv": "Video", "a.webm": "Video", "a.m2ts": "Video", "a.ts": "Video",
		"a.mp3": "Audio", "a.flac": "Audio", "a.eac3": "Audio",
		"a.txt": "Text", "a.ttf": "Font", "a.otf": "Font", "a.css": "Stylesheet",
		"a.json": "Auxiliary", "a.html": "Auxiliary", "noext": "Auxiliary", "": "Auxiliary", ".hidden": "Auxiliary",
		"archive.tar.mp4": "Video",
	}
	for name, want := range cases {
		if got := DetectMediaType(name); got != want {
			t.Errorf("DetectMediaType(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestFlexCountRejectsGarbage(t *testing.T) {
	var c flexCount
	if err := json.Unmarshal([]byte(`"many"`), &c); err == nil {
		t.Error("a string is not a count")
	}
}
