package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/brightsign/bscontrol/internal/errors"
	"github.com/brightsign/bscontrol/internal/types"
)

var _ PresentationService = (*presentationService)(nil)

func newPres(env *testEnv) PresentationService {
	return NewPresentationService(env.cfg, env.hc, env.am)
}

const presPath = "/" + testAPIVersion + "/Presentations"

func presOps() []op {
	req := func() *types.PresentationCreateRequest { return &types.PresentationCreateRequest{Name: "p"} }
	return []op{
		{"List", func(ctx context.Context, e *testEnv) error { _, err := newPres(e).List(ctx); return err }},
		{"ListAll", func(ctx context.Context, e *testEnv) error { _, err := newPres(e).ListAll(ctx); return err }},
		{"GetCount", func(ctx context.Context, e *testEnv) error { _, err := newPres(e).GetCount(ctx); return err }},
		{"GetByID", func(ctx context.Context, e *testEnv) error { _, err := newPres(e).GetByID(ctx, 3); return err }},
		{"GetByName", func(ctx context.Context, e *testEnv) error { _, err := newPres(e).GetByName(ctx, "p"); return err }},
		{"Create", func(ctx context.Context, e *testEnv) error { _, err := newPres(e).Create(ctx, req()); return err }},
		{"Update", func(ctx context.Context, e *testEnv) error { _, err := newPres(e).Update(ctx, 3, req()); return err }},
		{"DeleteByID", func(ctx context.Context, e *testEnv) error { return newPres(e).DeleteByID(ctx, 3) }},
		{"DeleteByName", func(ctx context.Context, e *testEnv) error { return newPres(e).DeleteByName(ctx, "p") }},
		{"DeleteByFilter", func(ctx context.Context, e *testEnv) error {
			_, err := newPres(e).DeleteByFilter(ctx, "[Name] IS 'p'")
			return err
		}},
	}
}

func TestPresentationService_CommonBehaviour(t *testing.T) {
	runCommonBehaviour(t, presOps())
}

func TestPresentationService_List(t *testing.T) {
	env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"items":[{"id":1,"name":"a","creationDate":"2026-08-18T12:00:00Z","lastModifiedDate":"2026-08-18T12:00:00Z"}],"isTruncated":true,"nextMarker":"n"}`)
	})
	got, err := newPres(env).List(context.Background(), WithPageSize(10), WithMarker("m"), WithFilter("[Name] STARTS WITH 'content-test-'"), WithSort("name"))
	if err != nil {
		t.Fatal(err)
	}
	req := env.only()
	requireBearer(t, req)
	if req.Method != http.MethodGet || req.Path != presPath {
		t.Errorf("request = %s %s", req.Method, req.Path)
	}
	q, _ := url.ParseQuery(req.RawQuery)
	if q.Get("pageSize") != "10" || q.Get("marker") != "m" || q.Get("filter") != "[Name] STARTS WITH 'content-test-'" || q.Get("sort") != "name" {
		t.Errorf("query = %v", q)
	}
	if len(got.Items) != 1 || got.Items[0].Name != "a" || !got.IsTruncated || got.NextMarker != "n" {
		t.Errorf("got %+v", got)
	}
}

func TestPresentationService_ListAll_Paginates(t *testing.T) {
	env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("marker") {
		case "":
			writeJSON(w, `{"items":[{"id":1,"name":"a"}],"isTruncated":true,"nextMarker":"2"}`)
		case "2":
			writeJSON(w, `{"items":[{"id":2,"name":"b"}],"isTruncated":false}`)
		}
	})
	got, err := newPres(env).ListAll(context.Background())
	if err != nil || len(got) != 2 || got[1].Name != "b" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestPresentationService_GetCount(t *testing.T) {
	for name, tc := range map[string]struct {
		body string
		want int
	}{"bare number": {`12`, 12}, "object": {`{"count":3}`, 3}} {
		t.Run(name, func(t *testing.T) {
			env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, tc.body) })
			got, err := newPres(env).GetCount(context.Background())
			if err != nil || got.Count != tc.want {
				t.Fatalf("got %+v, %v", got, err)
			}
			if p := env.only().Path; p != presPath+"/Count" {
				t.Errorf("path = %s", p)
			}
		})
	}
}

func TestPresentationService_GetByID(t *testing.T) {
	env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"id":3,"name":"p","status":"Published","type":"Complete","creationDate":"2026-08-18T12:00:00Z","lastModifiedDate":"2026-08-18T12:00:00Z"}`)
	})
	got, err := newPres(env).GetByID(context.Background(), 3)
	if err != nil {
		t.Fatal(err)
	}
	if req := env.only(); req.Path != presPath+"/3" || req.Method != http.MethodGet {
		t.Errorf("request = %s %s", req.Method, req.Path)
	}
	if got.ID != 3 || got.Status != "Published" {
		t.Errorf("got %+v", got)
	}
	for _, id := range []int{0, -1} {
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {})
		_, err := newPres(env).GetByID(context.Background(), id)
		expectValidation(t, err)
		env.noRequests()
	}
}

func TestPresentationService_NameEscaping(t *testing.T) {
	names := []struct{ in, wantRaw string }{
		{"plain", presPath + "/plain"},
		{"with space", presPath + "/with%20space"},
		{"slash/inside", presPath + "/slash%2Finside"},
		{"q?x=1&y#z", presPath + "/q%3Fx=1&y%23z"},
		{"../escape", presPath + "/..%2Fescape"},
	}
	for _, tc := range names {
		t.Run(tc.in, func(t *testing.T) {
			env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, `{"id":1,"name":"x"}`) })
			if _, err := newPres(env).GetByName(context.Background(), tc.in); err != nil {
				t.Fatal(err)
			}
			if got := env.only().RawPath; got != tc.wantRaw {
				t.Errorf("GetByName path = %q, want %q", got, tc.wantRaw)
			}

			env2 := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
			if err := newPres(env2).DeleteByName(context.Background(), tc.in); err != nil {
				t.Fatal(err)
			}
			r := env2.only()
			if r.Method != http.MethodDelete || r.RawPath != tc.wantRaw {
				t.Errorf("DeleteByName = %s %q, want %q", r.Method, r.RawPath, tc.wantRaw)
			}
		})
	}
	t.Run("empty name", func(t *testing.T) {
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {})
		_, err := newPres(env).GetByName(context.Background(), "")
		expectValidation(t, err)
		expectValidation(t, newPres(env).DeleteByName(context.Background(), ""))
		env.noRequests()
	})
}

// A minimal request is enough: dates and the empty lists the service insists on
// are filled in, and the caller's own struct is not modified.
func TestPresentationService_Create_FillsDefaults(t *testing.T) {
	env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":77,"name":"canary","status":"Draft"}`))
	})
	req := &types.PresentationCreateRequest{Name: "canary", DeviceModel: "XT1144"}
	got, err := newPres(env).Create(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 77 {
		t.Errorf("got %+v", got)
	}
	r := env.only()
	requireBearer(t, r)
	if r.Method != http.MethodPost || r.Path != presPath {
		t.Errorf("request = %s %s", r.Method, r.Path)
	}
	if ct := r.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(r.Body, &body); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"id": "0", "name": `"canary"`, "creationDate": `"0001-01-01T00:00:00"`, "lastModifiedDate": `"0001-01-01T00:00:00"`,
		"projectFile": "null", "autoplayFile": "null", "resourcesFile": "null", "thumbnailFile": "null", "userDefinedEventsFile": "null",
		"files": "[]", "applications": "[]", "dependencies": "[]", "permissions": "[]", "autorunPlugins": "null", "groups": "null",
		"deviceModel": `"XT1144"`,
	}
	for k, v := range want {
		if string(body[k]) != v {
			t.Errorf("%s = %s, want %s", k, body[k], v)
		}
	}
	if req.CreationDate != "" || req.Files != nil {
		t.Errorf("caller's request was modified: %+v", req)
	}
}

// What canary-content sends: file refs with bodies, a media file and settings.
func TestPresentationService_Create_FullRequestSurvivesIntact(t *testing.T) {
	env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, `{"id":1,"name":"p"}`) })
	now := "2026-10-27T12:00:00Z"
	req := &types.PresentationCreateRequest{
		Name: "p", DeviceModel: "XT1144", Language: "English", Status: "Published",
		ScreenSettings: &types.PresentationScreenSettings{VideoMode: "1920x1080x60p", Orientation: "Landscape", Connector: "HDMI", BackgroundColor: "RGB:000000", Overscan: "NoOverscan"},
		ProjectFile:    &types.PresentationFileRef{Type: "New", Name: "p.bpfx", Body: map[string]any{"meta": map[string]any{"v": 1}}, TransferEncoding: "none", LastModifiedDate: now},
		Autorun:        &types.PresentationAutorun{Version: "10.0.110"},
		Files:          []types.PresentationFile{{Type: "Media", ID: 42, Name: "pat.png", MediaType: "Image"}},
	}
	if _, err := newPres(env).Create(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	var body struct {
		ProjectFile struct {
			Type, Name, TransferEncoding string
			Body                         map[string]map[string]int
			CreationDate                 *string
			Size                         int
			Hash                         string
		}
		Files                   []map[string]any
		Autorun, ScreenSettings map[string]any
	}
	if err := json.Unmarshal(env.only().Body, &body); err != nil {
		t.Fatal(err)
	}
	if body.ProjectFile.Type != "New" || body.ProjectFile.Name != "p.bpfx" || body.ProjectFile.Body["meta"]["v"] != 1 || body.ProjectFile.CreationDate != nil {
		t.Errorf("projectFile = %+v", body.ProjectFile)
	}
	if len(body.Files) != 1 || body.Files[0]["id"] != float64(42) || body.Files[0]["mediaType"] != "Image" || body.Files[0]["type"] != "Media" {
		t.Errorf("files = %v", body.Files)
	}
	if body.Autorun["version"] != "10.0.110" || body.ScreenSettings["overscan"] != "NoOverscan" {
		t.Errorf("autorun/screen = %v / %v", body.Autorun, body.ScreenSettings)
	}
}

func TestPresentationService_Create_Validation(t *testing.T) {
	cases := map[string]*types.PresentationCreateRequest{
		"nil":         nil,
		"empty name":  {},
		"existing id": {ID: 5, Name: "x"},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {})
			_, err := newPres(env).Create(context.Background(), req)
			expectValidation(t, err)
			env.noRequests()
		})
	}
}

func TestPresentationService_Update(t *testing.T) {
	t.Run("puts to the id and stamps it into the body", func(t *testing.T) {
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, `{"id":8,"name":"renamed"}`) })
		got, err := newPres(env).Update(context.Background(), 8, &types.PresentationCreateRequest{Name: "renamed"})
		if err != nil {
			t.Fatal(err)
		}
		r := env.only()
		requireBearer(t, r)
		if r.Method != http.MethodPut || r.Path != presPath+"/8" {
			t.Errorf("request = %s %s", r.Method, r.Path)
		}
		var body struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		_ = json.Unmarshal(r.Body, &body)
		if body.ID != 8 || body.Name != "renamed" || got.Name != "renamed" {
			t.Errorf("body %s, got %+v", r.Body, got)
		}
	})
	t.Run("matching body id is fine", func(t *testing.T) {
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, `{"id":8,"name":"x"}`) })
		if _, err := newPres(env).Update(context.Background(), 8, &types.PresentationCreateRequest{ID: 8, Name: "x"}); err != nil {
			t.Fatal(err)
		}
	})
	bad := map[string]struct {
		id  int
		req *types.PresentationCreateRequest
	}{
		"zero id":       {0, &types.PresentationCreateRequest{Name: "x"}},
		"nil":           {1, nil},
		"empty name":    {1, &types.PresentationCreateRequest{}},
		"mismatched id": {1, &types.PresentationCreateRequest{ID: 2, Name: "x"}},
	}
	for name, tc := range bad {
		t.Run(name, func(t *testing.T) {
			env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {})
			_, err := newPres(env).Update(context.Background(), tc.id, tc.req)
			expectValidation(t, err)
			env.noRequests()
		})
	}
}

func TestPresentationService_Deletes(t *testing.T) {
	t.Run("by id", func(t *testing.T) {
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
		if err := newPres(env).DeleteByID(context.Background(), 4); err != nil {
			t.Fatal(err)
		}
		r := env.only()
		requireBearer(t, r)
		if r.Method != http.MethodDelete || r.Path != presPath+"/4" {
			t.Errorf("request = %s %s", r.Method, r.Path)
		}
	})
	for _, id := range []int{0, -1} {
		t.Run(fmt.Sprintf("invalid id %d", id), func(t *testing.T) {
			env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {})
			expectValidation(t, newPres(env).DeleteByID(context.Background(), id))
			env.noRequests()
		})
	}
	t.Run("by filter", func(t *testing.T) {
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, `{"deletedCount":2,"deletedIds":[1,2]}`) })
		filter := `[Name] STARTS WITH 'content-test-' AND [Name] IS NOT 'x&y'`
		got, err := newPres(env).DeleteByFilter(context.Background(), filter)
		if err != nil {
			t.Fatal(err)
		}
		r := env.only()
		q, _ := url.ParseQuery(r.RawQuery)
		if r.Method != http.MethodDelete || q.Get("filter") != filter || len(q) != 1 {
			t.Errorf("request = %s ?%s", r.Method, r.RawQuery)
		}
		if got.DeletedCount != 2 || len(got.DeletedIDs) != 2 {
			t.Errorf("got %+v", got)
		}
	})
	t.Run("by filter, bare count", func(t *testing.T) {
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, `5`) })
		got, err := newPres(env).DeleteByFilter(context.Background(), "x")
		if err != nil || got.DeletedCount != 5 {
			t.Errorf("got %+v, %v", got, err)
		}
	})
	t.Run("empty filter", func(t *testing.T) {
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {})
		_, err := newPres(env).DeleteByFilter(context.Background(), "")
		expectValidation(t, err)
		env.noRequests()
	})
}

// Typed errors survive the wrapping so callers can branch on them.
func TestPresentationService_ErrorsKeepTheirType(t *testing.T) {
	env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
		writeStatus(w, 404, `{"error":"not_found","error_description":"no such presentation"}`)
	})
	_, err := newPres(env).GetByID(context.Background(), 12345)
	var api *errors.APIError
	if !asErr(err, &api) || api.StatusCode != 404 || api.Code != "not_found" {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(err.Error(), "12345") {
		t.Errorf("error does not say which presentation: %v", err)
	}
}
