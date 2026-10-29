package bscontrol_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/brightsign/bscontrol"
)

type fixedToken struct{}

func (fixedToken) Token(context.Context) (bscontrol.AccessToken, error) {
	return bscontrol.AccessToken{Value: "t", ExpiresAt: time.Now().Add(time.Hour)}, nil
}

// Everything the canary-content workflow does, through the public client and
// only the re-exported names: upload, create a presentation, schedule it on a
// group, then clean up in reverse. A fake BSN checks each call lands where the
// workflow expects.
func TestClientContentWorkflowThroughPublicAPI(t *testing.T) {
	var mu sync.Mutex
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		calls = append(calls, r.Method+" "+r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/Self/Session/Network"):
		case r.Method == http.MethodPost && r.URL.Path == "/Upload/2019/03/REST/Sessions/None/Uploads/":
			_, _ = w.Write([]byte(`{"uploadToken":"u1","state":"Started"}`))
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/Chunks/"):
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/Upload/"):
			_, _ = w.Write([]byte(`{"state":"Uploaded"}`))
		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/Upload/"):
			_, _ = w.Write([]byte(`{"state":"Verified","contentId":42}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/Presentations"):
			var req bscontrol.PresentationCreateRequest
			if err := json.Unmarshal(body, &req); err != nil || len(req.Files) != 1 || req.Files[0].ID != 42 {
				t.Errorf("create body = %s (%v)", body, err)
			}
			_, _ = w.Write([]byte(`{"id":7,"name":"p"}`))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/Groups/Regular/"):
			_, _ = w.Write([]byte(`{"id":1,"name":"g","devices":[]}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/schedule/"):
			_, _ = w.Write([]byte(`{"id":9,"presentationId":7,"isRecurrent":true,"startTime":"00:00:00","duration":"23:59:59"}`))
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client, err := bscontrol.New(
		bscontrol.WithTokenSource(fixedToken{}),
		bscontrol.WithEndpoints(srv.URL, srv.URL),
		bscontrol.WithNetwork("Net"),
		bscontrol.WithRetryCount(0),
	)
	if err != nil {
		t.Fatal(err)
	}
	if client.Content == nil || client.Upload == nil || client.Presentations == nil || client.Schedules == nil {
		t.Fatal("content services are not wired on the client")
	}

	ctx := context.Background()
	contentID, err := client.Upload.UploadBytes(ctx, "pattern.png", []byte("png!"), bscontrol.WithUploadMediaType(bscontrol.MediaTypeImage))
	if err != nil || contentID != 42 {
		t.Fatalf("upload: %d, %v", contentID, err)
	}
	pres, err := client.Presentations.Create(ctx, &bscontrol.PresentationCreateRequest{
		Name:  "p",
		Files: []bscontrol.PresentationFile{{Type: "Media", ID: contentID, Name: "pattern.png", MediaType: bscontrol.MediaTypeImage}},
	})
	if err != nil || pres.ID != 7 {
		t.Fatalf("create: %+v, %v", pres, err)
	}
	group, err := client.Devices.GetGroupByName(ctx, "g")
	if err != nil || group.Name != "g" {
		t.Fatalf("group: %+v, %v", group, err)
	}
	sched, err := client.Schedules.AddScheduledPresentation(ctx, group.Name, &bscontrol.ScheduledPresentation{
		PresentationID: pres.ID, IsRecurrent: true, StartTime: "00:00:00", Duration: "23:59:59", DaysOfWeek: bscontrol.DaysOfWeekEveryDay,
	})
	if err != nil || sched.ID != 9 {
		t.Fatalf("schedule: %+v, %v", sched, err)
	}
	if err := client.Schedules.DeleteScheduledPresentation(ctx, group.Name, sched.ID); err != nil {
		t.Fatal(err)
	}
	if err := client.Presentations.DeleteByName(ctx, pres.Name); err != nil {
		t.Fatal(err)
	}
	if err := client.Content.DeleteByID(ctx, contentID); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	last := calls[len(calls)-3:]
	want := []string{
		"DELETE /2022/06/REST/Groups/Regular/g/schedule/9/",
		"DELETE /2022/06/REST/Presentations/p",
		"DELETE /2022/06/REST/Content/42",
	}
	for i := range want {
		if last[i] != want[i] {
			t.Errorf("cleanup call %d = %q, want %q", i, last[i], want[i])
		}
	}
}

func TestHelpersAreReExported(t *testing.T) {
	if got, err := bscontrol.ToBSNVirtualPath("videos/promos"); err != nil || got != `\videos\promos\` {
		t.Errorf("ToBSNVirtualPath = %q, %v", got, err)
	}
	if bscontrol.EscapeFilterValue("it's") != "it''s" {
		t.Error("EscapeFilterValue not wired")
	}
	if bscontrol.DetectMediaType("a.mp4") != bscontrol.MediaTypeVideo {
		t.Error("DetectMediaType not wired")
	}
}
