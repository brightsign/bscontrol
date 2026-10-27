package services

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/brightsign/bscontrol/internal/errors"
	"github.com/brightsign/bscontrol/internal/types"
)

var _ UploadService = (*uploadService)(nil)

const uploadBase = "/Upload/2019/03/REST"

// newUpload returns an upload service tuned for tests: tiny chunks, no waiting.
func newUpload(env *testEnv, chunk int) *uploadService {
	svc := NewUploadService(env.cfg, env.hc, env.am).(*uploadService)
	svc.chunkSize = chunk
	svc.pollInterval = time.Millisecond
	svc.pollTimeout = 50 * time.Millisecond
	svc.retryDelay = time.Millisecond
	return svc
}

func sha1Hex(b []byte) string {
	sum := sha1.Sum(b)
	return hex.EncodeToString(sum[:])
}

// fakeUpload is a scriptable Upload API: it assembles chunks at their offsets
// like the real backend and reports whatever statuses the test queues.
type fakeUpload struct {
	t *testing.T

	mu sync.Mutex
	// token is what start returns; "" means start returns no token.
	token string
	// matching is returned in the start response.
	matching []types.UploadMatchingFile
	// polls is the queue of statuses GET returns; the last one repeats.
	polls []types.ContentUploadStatus
	// completeStatus is the PUT reply.
	completeStatus types.ContentUploadStatus
	// chunkFailures maps offset -> number of 500s to return before succeeding;
	// -1 means always.
	chunkFailures map[int64]int
	chunkStatus   int // status for failing chunk attempts (default 500)
	// onChunk runs after a chunk is stored (for cancel tests).
	onChunk func(offset int64)

	assembled   map[int64][]byte
	startArgs   []map[string]any
	startBody   []byte
	completeArg map[string]any
	pollCount   int
	order       []string // "start", "chunk@N", "poll", "complete", "cancel"
	chunkTries  map[int64]int
}

func newFakeUpload(t *testing.T) *fakeUpload {
	return &fakeUpload{
		t:             t,
		token:         "tok-1",
		assembled:     map[int64][]byte{},
		chunkFailures: map[int64]int{},
		chunkTries:    map[int64]int{},
		polls:         []types.ContentUploadStatus{{State: types.UploadStateUploaded}},
		completeStatus: types.ContentUploadStatus{
			State: types.UploadStateVerified, ContentID: 99,
			EndTime: types.BSNTime{Time: time.Date(2026, 8, 18, 1, 2, 3, 0, time.UTC)},
		},
	}
}

func (f *fakeUpload) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		tokPath := uploadBase + "/Sessions/None/Uploads/" + f.token + "/"
		switch {
		case r.Method == http.MethodPost && r.URL.Path == uploadBase+"/Sessions/None/Uploads/":
			body := readAll(r)
			f.startBody = body
			var args map[string]any
			if err := json.Unmarshal(body, &args); err != nil {
				f.t.Errorf("start body is not JSON: %v", err)
			}
			f.startArgs = append(f.startArgs, args)
			f.order = append(f.order, "start")
			st := types.ContentUploadStatus{UploadToken: f.token, State: types.UploadStateStarted, MatchingFiles: f.matching}
			writeVendorJSON(w, st)

		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, tokPath+"Chunks"):
			off, err := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64)
			if err != nil {
				f.t.Errorf("chunk without a numeric offset: %q", r.URL.RawQuery)
			}
			f.chunkTries[off]++
			if left := f.chunkFailures[off]; left != 0 {
				if left > 0 {
					f.chunkFailures[off]--
				}
				status := f.chunkStatus
				if status == 0 {
					status = 500
				}
				writeStatus(w, status, `{"message":"chunk failed"}`)
				return
			}
			f.assembled[off] = readAll(r)
			f.order = append(f.order, fmt.Sprintf("chunk@%d", off))
			if f.onChunk != nil {
				f.onChunk(off)
			}
			w.WriteHeader(http.StatusOK)

		case r.Method == http.MethodGet && r.URL.Path == tokPath:
			f.pollCount++
			f.order = append(f.order, "poll")
			idx := f.pollCount - 1
			if idx >= len(f.polls) {
				idx = len(f.polls) - 1
			}
			st := f.polls[idx]
			st.UploadToken = f.token
			writeVendorJSON(w, st)

		case r.Method == http.MethodPut && r.URL.Path == tokPath:
			f.order = append(f.order, "complete")
			if err := json.Unmarshal(readAll(r), &f.completeArg); err != nil {
				f.t.Errorf("complete body is not JSON: %v", err)
			}
			writeVendorJSON(w, f.completeStatus)

		case r.Method == http.MethodDelete && r.URL.Path == tokPath:
			f.order = append(f.order, "cancel")
			w.WriteHeader(http.StatusNoContent)

		default:
			f.t.Errorf("unexpected request %s %s", r.Method, r.URL.RequestURI())
			http.NotFound(w, r)
		}
	}
}

func readAll(r *http.Request) []byte {
	b, _ := io.ReadAll(r.Body)
	return b
}

func writeVendorJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/vnd.bsn.content.upload.status.201903+json")
	_ = json.NewEncoder(w).Encode(v)
}

// file reassembles what the server received, in offset order, and returns the
// sorted offsets with it.
func (f *fakeUpload) file() ([]byte, []int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var offsets []int64
	for off := range f.assembled {
		offsets = append(offsets, off)
	}
	for i := range offsets {
		for j := i + 1; j < len(offsets); j++ {
			if offsets[j] < offsets[i] {
				offsets[i], offsets[j] = offsets[j], offsets[i]
			}
		}
	}
	var out []byte
	for _, off := range offsets {
		if int64(len(out)) != off {
			f.t.Errorf("gap in chunks: have %d bytes, next chunk at %d", len(out), off)
		}
		out = append(out, f.assembled[off]...)
	}
	return out, offsets
}

func (f *fakeUpload) hasOrder(want ...string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return fmt.Sprint(f.order) == fmt.Sprint(want)
}

// ---------------------------------------------------------------------------

func TestUpload_HappyPath_WireDetails(t *testing.T) {
	data := []byte("hello brightsign!") // 17 bytes -> chunks of 8: 8, 8, 1
	fake := newFakeUpload(t)
	env := newTestEnv(t, fake.handler())
	svc := newUpload(env, 8)

	var progress []types.UploadProgress
	got, err := svc.UploadReader(context.Background(), "Clip.MP4", int64(len(data)), bytes.NewReader(data), "videos/promos",
		WithUploadProgress(func(p types.UploadProgress) { progress = append(progress, p) }),
		WithUploadModTime(time.Date(2026, 8, 18, 12, 34, 56, 0, time.UTC)))
	if err != nil {
		t.Fatal(err)
	}

	// Result.
	want := types.UploadResponse{ContentID: 99, FileName: "Clip.MP4", FileSize: 17, VirtualPath: `\videos\promos\`,
		MediaType: "Video", FileHash: sha1Hex(data), UploadComplete: true}
	got.UploadDate = types.BSNTime{}
	if *got != want {
		t.Errorf("result = %+v\nwant     %+v", *got, want)
	}

	// Chunk offsets and content.
	body, offsets := fake.file()
	if !bytes.Equal(body, data) {
		t.Errorf("server assembled %q, want %q", body, data)
	}
	if fmt.Sprint(offsets) != "[0 8 16]" {
		t.Errorf("chunk offsets = %v, want [0 8 16]", offsets)
	}
	if len(progress) != 3 || progress[2].Sent != 17 || progress[2].Total != 17 || progress[0].Sent != 8 || progress[0].FileName != "Clip.MP4" {
		t.Errorf("progress = %+v", progress)
	}

	// Order: start, every chunk, then poll before complete.
	if !fake.hasOrder("start", "chunk@0", "chunk@8", "chunk@16", "poll", "complete") {
		t.Errorf("call order = %v", fake.order)
	}

	// Start request.
	reqs := env.requests(false)
	start := reqs[0]
	requireBearer(t, start)
	if start.Path != uploadBase+"/Sessions/None/Uploads/" || start.Method != http.MethodPost {
		t.Errorf("start = %s %s", start.Method, start.Path)
	}
	if ct := start.Header.Get("Content-Type"); ct != "application/vnd.bsn.start.content.upload.arguments.201903+json" {
		t.Errorf("start Content-Type = %q", ct)
	}
	if acc := start.Header.Get("Accept"); acc != "application/vnd.bsn.content.upload.status.201903+json, application/vnd.bsn.content.upload.negotiation.status.201903+json, application/vnd.bsn.error+json" {
		t.Errorf("start Accept = %q", acc)
	}
	args := fake.startArgs[0]
	for k, v := range map[string]any{
		"fileName":             "Clip.MP4",
		"virtualPath":          `\videos\promos\`,
		"mediaType":            "Video",
		"fileSize":             float64(17),
		"fileLastModifiedDate": "0001-01-01 00:00:00Z",
		"shA1Hash":             sha1Hex(data),
	} {
		if args[k] != v {
			t.Errorf("start %s = %v, want %v", k, args[k], v)
		}
	}
	for _, k := range []string{"sessionToken", "uploadToken", "fileThumb"} {
		if v, ok := args[k]; !ok || v != nil {
			t.Errorf("start %s should be an explicit null, got %v (present %v)", k, v, ok)
		}
	}
	for _, bad := range []string{"tags", "sha1Hash", "SHA1Hash"} {
		if _, ok := args[bad]; ok {
			t.Errorf("start body must not contain %q: %s", bad, fake.startBody)
		}
	}
	if h := args["shA1Hash"].(string); h != strings.ToLower(h) || len(h) != 40 {
		t.Errorf("hash %q is not 40 lowercase hex chars", h)
	}

	// Chunk requests.
	for _, r := range reqs[1:4] {
		requireBearer(t, r)
		if r.Header.Get("Content-Type") != "application/octet-stream" {
			t.Errorf("chunk Content-Type = %q", r.Header.Get("Content-Type"))
		}
		if r.Path != uploadBase+"/Sessions/None/Uploads/tok-1/Chunks/" {
			t.Errorf("chunk path = %s", r.Path)
		}
	}

	// Poll request.
	poll := reqs[4]
	if poll.Method != http.MethodGet || !strings.Contains(poll.Header.Get("Accept"), "application/vnd.bsn.content.upload.status.201903+json") {
		t.Errorf("poll = %s Accept %q", poll.Method, poll.Header.Get("Accept"))
	}

	// Complete request.
	put := reqs[5]
	requireBearer(t, put)
	if put.Method != http.MethodPut || put.Path != uploadBase+"/Sessions/None/Uploads/tok-1/" {
		t.Errorf("complete = %s %s", put.Method, put.Path)
	}
	if ct := put.Header.Get("Content-Type"); ct != "application/vnd.bsn.complete.content.upload.arguments.201903+json" {
		t.Errorf("complete Content-Type = %q", ct)
	}
	c := fake.completeArg
	if c["uploadToken"] != "tok-1" {
		t.Errorf("complete uploadToken = %v, want tok-1", c["uploadToken"])
	}
	if c["sessionToken"] != nil {
		t.Errorf("complete sessionToken = %v, want null", c["sessionToken"])
	}
	if c["fileLastModifiedDate"] != "2026-08-18T12:34:56.000Z" {
		t.Errorf("complete fileLastModifiedDate = %v, want a real T-ISO date", c["fileLastModifiedDate"])
	}
	if c["shA1Hash"] != sha1Hex(data) || c["fileName"] != "Clip.MP4" || c["virtualPath"] != `\videos\promos\` {
		t.Errorf("complete args = %v", c)
	}
	if _, ok := c["tags"]; ok {
		t.Error("complete must not carry tags")
	}
}

func TestUpload_DefaultChunkSizeAndOffsets(t *testing.T) {
	data := bytes.Repeat([]byte{0xAB}, 11*1024*1024) // 5 MiB chunks: 5, 5, 1 MiB
	fake := newFakeUpload(t)
	env := newTestEnv(t, fake.handler())
	svc := NewUploadService(env.cfg, env.hc, env.am).(*uploadService)
	svc.pollInterval, svc.pollTimeout = time.Millisecond, 50*time.Millisecond

	if _, err := svc.UploadReader(context.Background(), "big.bin", int64(len(data)), bytes.NewReader(data), ""); err != nil {
		t.Fatal(err)
	}
	body, offsets := fake.file()
	if fmt.Sprint(offsets) != "[0 5242880 10485760]" {
		t.Errorf("offsets = %v", offsets)
	}
	if !bytes.Equal(body, data) {
		t.Error("reassembled file differs")
	}
	if fake.startArgs[0]["mediaType"] != "Auxiliary" || fake.startArgs[0]["virtualPath"] != `\` {
		t.Errorf("unknown extension / root path: %v", fake.startArgs[0])
	}
}

func TestUpload_ExactChunkBoundary(t *testing.T) {
	data := []byte("12345678") // exactly two chunks of 4
	fake := newFakeUpload(t)
	env := newTestEnv(t, fake.handler())
	if _, err := newUpload(env, 4).UploadReader(context.Background(), "a.txt", 8, bytes.NewReader(data), ""); err != nil {
		t.Fatal(err)
	}
	if _, offsets := fake.file(); fmt.Sprint(offsets) != "[0 4]" {
		t.Errorf("offsets = %v, want [0 4] (no empty trailing chunk)", offsets)
	}
}

func TestUpload_PollsUntilUploaded(t *testing.T) {
	cases := []struct {
		name      string
		polls     []types.ContentUploadStatus
		wantPolls int
	}{
		{"already uploaded", []types.ContentUploadStatus{{State: types.UploadStateUploaded}}, 1},
		{"uploading then uploaded", []types.ContentUploadStatus{{State: types.UploadStateUploading}, {State: types.UploadStateUploading}, {State: types.UploadStateUploaded}}, 3},
		{"started then verified", []types.ContentUploadStatus{{State: types.UploadStateStarted}, {State: types.UploadStateVerified}}, 2},
		{"queued and unknown keep polling", []types.ContentUploadStatus{{State: types.UploadStateQueued}, {State: types.UploadStateUnknown}, {State: types.UploadStateStopped}, {State: types.UploadStateUploaded}}, 4},
		{"chunk runs lagging then all uploaded", []types.ContentUploadStatus{
			{State: types.UploadStateUploading, Chunks: []types.UploadChunkStatus{{Count: 1, State: types.UploadStateUploaded}, {Count: 2, State: types.UploadStateQueued}}},
			{State: types.UploadStateUploading, Chunks: []types.UploadChunkStatus{{Count: 3, State: types.UploadStateUploaded}}},
		}, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := newFakeUpload(t)
			fake.polls = tc.polls
			env := newTestEnv(t, fake.handler())
			if _, err := newUpload(env, 4).UploadBytes2(t, "a.png", []byte("abcdefgh")); err != nil {
				t.Fatal(err)
			}
			if fake.pollCount != tc.wantPolls {
				t.Errorf("polled %d times, want %d", fake.pollCount, tc.wantPolls)
			}
			// complete must come after the final poll, never before.
			if fake.order[len(fake.order)-1] != "complete" || fake.order[len(fake.order)-2] != "poll" {
				t.Errorf("order = %v", fake.order)
			}
		})
	}
}

// UploadBytes2 is a test shorthand for uploading bytes and returning the response.
func (s *uploadService) UploadBytes2(t *testing.T, name string, data []byte) (*types.UploadResponse, error) {
	t.Helper()
	return s.UploadReader(context.Background(), name, int64(len(data)), bytes.NewReader(data), "")
}

func TestUpload_TerminalAndTimeoutStates(t *testing.T) {
	t.Run("poll sees a failure state", func(t *testing.T) {
		for _, st := range []types.UploadState{types.UploadStateCorrupted, types.UploadStateCancelled, types.UploadStateTerminated, types.UploadStateCollected} {
			t.Run(string(st), func(t *testing.T) {
				fake := newFakeUpload(t)
				fake.polls = []types.ContentUploadStatus{{State: types.UploadStateUploading}, {State: st}}
				env := newTestEnv(t, fake.handler())
				_, err := newUpload(env, 4).UploadBytes2(t, "a.png", []byte("abcdefgh"))
				var app *errors.AppError
				if !asErr(err, &app) || app.Code != "upload_failed" || !strings.Contains(app.Message, string(st)) {
					t.Fatalf("err = %v", err)
				}
				if fake.completeArg != nil {
					t.Error("complete must not be sent after a failure state")
				}
			})
		}
	})

	t.Run("never reaches Uploaded: timeout names the last state", func(t *testing.T) {
		fake := newFakeUpload(t)
		fake.polls = []types.ContentUploadStatus{{State: types.UploadStateUploading}}
		env := newTestEnv(t, fake.handler())
		_, err := newUpload(env, 4).UploadBytes2(t, "a.png", []byte("abcdefgh"))
		var app *errors.AppError
		if !asErr(err, &app) || app.Code != "upload_timeout" || !strings.Contains(app.Details, "Uploading") {
			t.Fatalf("err = %v", err)
		}
		if fake.completeArg != nil {
			t.Error("complete must not be sent before chunks are Uploaded")
		}
		if fake.pollCount < 2 {
			t.Errorf("only polled %d times", fake.pollCount)
		}
	})

	t.Run("status check error is typed", func(t *testing.T) {
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				if strings.HasSuffix(r.URL.Path, "/Uploads/") {
					writeVendorJSON(w, types.ContentUploadStatus{UploadToken: "tok-1"})
					return
				}
				w.WriteHeader(200)
			case http.MethodGet:
				writeStatus(w, 500, `{"message":"down"}`)
			}
		})
		_, err := newUpload(env, 4).UploadBytes2(t, "a.png", []byte("abcdefgh"))
		if !errors.IsRetryableError(err) {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("complete reports Corrupted (hash mismatch)", func(t *testing.T) {
		fake := newFakeUpload(t)
		fake.completeStatus = types.ContentUploadStatus{State: types.UploadStateCorrupted}
		env := newTestEnv(t, fake.handler())
		_, err := newUpload(env, 4).UploadBytes2(t, "a.png", []byte("abcdefgh"))
		var app *errors.AppError
		if !asErr(err, &app) || app.Code != "upload_corrupted" {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("complete reports another failure state", func(t *testing.T) {
		fake := newFakeUpload(t)
		fake.completeStatus = types.ContentUploadStatus{State: types.UploadStateTerminated}
		env := newTestEnv(t, fake.handler())
		_, err := newUpload(env, 4).UploadBytes2(t, "a.png", []byte("abcdefgh"))
		var app *errors.AppError
		if !asErr(err, &app) || app.Code != "upload_failed" {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("complete still verifying: waits for the final state", func(t *testing.T) {
		fake := newFakeUpload(t)
		fake.completeStatus = types.ContentUploadStatus{State: types.UploadStateUploaded}
		// poll 1 gates completion; polls 2.. follow the PUT.
		fake.polls = []types.ContentUploadStatus{
			{State: types.UploadStateUploaded},
			{State: types.UploadStateUploaded},
			{State: types.UploadStateVerified, ContentID: 55},
		}
		env := newTestEnv(t, fake.handler())
		got, err := newUpload(env, 4).UploadBytes2(t, "a.png", []byte("abcdefgh"))
		if err != nil {
			t.Fatal(err)
		}
		if got.ContentID != 55 || !got.UploadComplete {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("complete still verifying and never finishes", func(t *testing.T) {
		fake := newFakeUpload(t)
		fake.completeStatus = types.ContentUploadStatus{State: types.UploadStateUploaded}
		env := newTestEnv(t, fake.handler())
		_, err := newUpload(env, 4).UploadBytes2(t, "a.png", []byte("abcdefgh"))
		var app *errors.AppError
		if !asErr(err, &app) || app.Code != "upload_timeout" {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("finished without a content id", func(t *testing.T) {
		fake := newFakeUpload(t)
		fake.completeStatus = types.ContentUploadStatus{State: types.UploadStateCompleted}
		env := newTestEnv(t, fake.handler())
		_, err := newUpload(env, 4).UploadBytes2(t, "a.png", []byte("abcdefgh"))
		var app *errors.AppError
		if !asErr(err, &app) || app.Code != "upload_no_content_id" {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestUpload_Dedup(t *testing.T) {
	data := []byte("duplicate me")
	existing := func(name, vpath string, id int) types.UploadMatchingFile {
		return types.UploadMatchingFile{ID: id, FileName: name, VirtualPath: vpath, MediaType: "Video", FileSize: int64(len(data)),
			UploadDate: types.BSNTime{Time: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}}
	}
	cases := []struct {
		name       string
		matching   []types.UploadMatchingFile
		virtual    string
		wantDedup  bool
		wantID     int
		wantCancel bool
	}{
		{"same path", []types.UploadMatchingFile{existing("a.mp4", `\videos\`, 41)}, "videos", true, 41, true},
		{"server spells path without trailing slash", []types.UploadMatchingFile{existing("a.mp4", `\videos`, 42)}, "/videos/", true, 42, true},
		{"second of several matches is the duplicate", []types.UploadMatchingFile{existing("a.mp4", `\other\`, 1), existing("a.mp4", `\videos\`, 43)}, "videos", true, 43, true},
		{"different folder is not a duplicate", []types.UploadMatchingFile{existing("a.mp4", `\other\`, 44)}, "videos", false, 99, false},
		{"different name is not a duplicate", []types.UploadMatchingFile{existing("b.mp4", `\videos\`, 45)}, "videos", false, 99, false},
		{"root vs folder", []types.UploadMatchingFile{existing("a.mp4", `\`, 46)}, "videos", false, 99, false},
		{"unparseable server path is ignored", []types.UploadMatchingFile{existing("a.mp4", `\a\..\b\`, 47)}, "videos", false, 99, false},
		{"no id is ignored", []types.UploadMatchingFile{existing("a.mp4", `\videos\`, 0)}, "videos", false, 99, false},
		{"no matches", nil, "videos", false, 99, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := newFakeUpload(t)
			fake.matching = tc.matching
			env := newTestEnv(t, fake.handler())
			got, err := newUpload(env, 4).UploadReader(context.Background(), "a.mp4", int64(len(data)), bytes.NewReader(data), tc.virtual)
			if err != nil {
				t.Fatal(err)
			}
			if got.Deduplicated != tc.wantDedup || got.ContentID != tc.wantID || !got.UploadComplete {
				t.Errorf("got %+v", got)
			}
			_, offsets := fake.file()
			if tc.wantDedup {
				if len(offsets) != 0 || fake.pollCount != 0 || fake.completeArg != nil {
					t.Errorf("a duplicate must send no bytes, poll or complete: offsets=%v polls=%d", offsets, fake.pollCount)
				}
				if !fake.hasOrder("start", "cancel") {
					t.Errorf("order = %v, want start then cancel", fake.order)
				}
				if got.FileHash != sha1Hex(data) || got.VirtualPath == "" {
					t.Errorf("dedup response lost hash/path: %+v", got)
				}
			} else if len(offsets) == 0 || fake.completeArg == nil {
				t.Errorf("a non-duplicate must upload: offsets=%v", offsets)
			}
		})
	}

	t.Run("a failing cancel does not fail the upload", func(t *testing.T) {
		fake := newFakeUpload(t)
		fake.matching = []types.UploadMatchingFile{existing("a.mp4", `\`, 7)}
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				writeStatus(w, 500, `{}`)
				return
			}
			fake.handler()(w, r)
		})
		got, err := newUpload(env, 4).UploadReader(context.Background(), "a.mp4", int64(len(data)), bytes.NewReader(data), "")
		if err != nil || !got.Deduplicated || got.ContentID != 7 {
			t.Errorf("got %+v, %v", got, err)
		}
	})
}

func TestUpload_StartFailures(t *testing.T) {
	t.Run("no upload token", func(t *testing.T) {
		fake := newFakeUpload(t)
		fake.token = ""
		env := newTestEnv(t, fake.handler())
		_, err := newUpload(env, 4).UploadBytes2(t, "a.png", []byte("abc"))
		var app *errors.AppError
		if !asErr(err, &app) || app.Code != "upload_no_token" {
			t.Fatalf("err = %v", err)
		}
	})

	// Typed HTTP mapping and the shared auth/network/cancel behaviour for the
	// most-used entry point.
	runCommonBehaviour(t, []op{
		{"UploadBytes", func(ctx context.Context, e *testEnv) error {
			_, err := newUpload(e, 4).UploadBytes(ctx, "a.png", []byte("abcdefgh"))
			return err
		}},
		{"UploadReader", func(ctx context.Context, e *testEnv) error {
			_, err := newUpload(e, 4).UploadReader(ctx, "a.png", 3, strings.NewReader("abc"), "x")
			return err
		}},
	})
}

func TestUpload_ChunkRetries(t *testing.T) {
	cases := []struct {
		name      string
		failures  int
		status    int
		wantErr   bool
		wantTries int
	}{
		{"one 500 then ok", 1, 500, false, 2},
		{"two 500s then ok", 2, 500, false, 3},
		{"500 every time gives up after three", -1, 500, true, 3},
		{"400 is not retried", -1, 400, true, 1},
		{"503 is retried", 1, 503, false, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := newFakeUpload(t)
			fake.chunkFailures[4] = tc.failures
			fake.chunkStatus = tc.status
			env := newTestEnv(t, fake.handler())
			_, err := newUpload(env, 4).UploadBytes2(t, "a.png", []byte("abcdefghij"))
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if fake.chunkTries[4] != tc.wantTries {
				t.Errorf("chunk@4 attempted %d times, want %d", fake.chunkTries[4], tc.wantTries)
			}
			if tc.wantErr {
				var api *errors.APIError
				if !asErr(err, &api) || api.StatusCode != tc.status {
					t.Errorf("err = %v, want APIError %d", err, tc.status)
				}
				if !strings.Contains(err.Error(), "offset 4") {
					t.Errorf("error does not say which offset failed: %v", err)
				}
				if fake.completeArg != nil {
					t.Error("complete sent after a failed chunk")
				}
			} else {
				body, _ := fake.file()
				if string(body) != "abcdefghij" {
					t.Errorf("assembled %q after retries", body)
				}
			}
		})
	}
}

func TestUpload_Validation(t *testing.T) {
	cases := []struct {
		name   string
		file   string
		size   int64
		reader io.Reader
		vpath  string
		opts   []UploadOption
	}{
		{"empty name", "", 3, strings.NewReader("abc"), "", nil},
		{"slash in name", "a/b.png", 3, strings.NewReader("abc"), "", nil},
		{"backslash in name", `a\b.png`, 3, strings.NewReader("abc"), "", nil},
		{"zero size", "a.png", 0, strings.NewReader(""), "", nil},
		{"negative size", "a.png", -1, strings.NewReader("abc"), "", nil},
		{"nil reader", "a.png", 3, nil, "", nil},
		{"traversing virtual path", "a.png", 3, strings.NewReader("abc"), "../x", nil},
		{"unknown media type", "a.png", 3, strings.NewReader("abc"), "", []UploadOption{WithUploadMediaType("Hologram")}},
		{"seekable reader too short", "a.png", 10, strings.NewReader("abc"), "", nil},
		{"seekable reader too long", "a.png", 2, strings.NewReader("abc"), "", nil},
		{"plain reader too short", "a.png", 10, io.MultiReader(strings.NewReader("abc")), "", nil},
		{"plain reader too long", "a.png", 2, io.MultiReader(strings.NewReader("abc")), "", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := newFakeUpload(t)
			env := newTestEnv(t, fake.handler())
			_, err := newUpload(env, 4).UploadReader(context.Background(), tc.file, tc.size, tc.reader, tc.vpath, tc.opts...)
			expectValidation(t, err)
			if reqs := env.requests(false); len(reqs) != 0 {
				t.Errorf("validation failure still hit the API: %+v", reqs)
			}
		})
	}
}

func TestUpload_Readers(t *testing.T) {
	t.Run("non-seekable reader is buffered and sent", func(t *testing.T) {
		fake := newFakeUpload(t)
		env := newTestEnv(t, fake.handler())
		data := []byte("streamed data")
		got, err := newUpload(env, 5).UploadReader(context.Background(), "a.txt", int64(len(data)), io.MultiReader(bytes.NewReader(data)), "")
		if err != nil {
			t.Fatal(err)
		}
		body, _ := fake.file()
		if !bytes.Equal(body, data) || got.FileHash != sha1Hex(data) {
			t.Errorf("body=%q hash=%s", body, got.FileHash)
		}
	})

	t.Run("seekable reader starts at its current position", func(t *testing.T) {
		fake := newFakeUpload(t)
		env := newTestEnv(t, fake.handler())
		r := bytes.NewReader([]byte("HEADERpayload"))
		if _, err := r.Seek(6, io.SeekStart); err != nil {
			t.Fatal(err)
		}
		got, err := newUpload(env, 4).UploadReader(context.Background(), "a.txt", 7, r, "")
		if err != nil {
			t.Fatal(err)
		}
		body, _ := fake.file()
		if string(body) != "payload" || got.FileHash != sha1Hex([]byte("payload")) {
			t.Errorf("body=%q hash=%s", body, got.FileHash)
		}
	})
}

// shrinkAfterHash is a seekable source whose file is truncated between the hash
// pass and the send pass, as when a file is rewritten mid-upload.
type shrinkAfterHash struct {
	data []byte
	pos  int64
	cut  bool
}

func (s *shrinkAfterHash) Read(p []byte) (int, error) {
	if s.pos >= int64(len(s.data)) {
		return 0, io.EOF
	}
	n := copy(p, s.data[s.pos:])
	s.pos += int64(n)
	return n, nil
}

func (s *shrinkAfterHash) Seek(off int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		s.pos = off
		if off > 0 || s.cut {
			s.cut = true
			s.data = s.data[:3]
		}
	case io.SeekCurrent:
		s.pos += off
	case io.SeekEnd:
		s.pos = int64(len(s.data)) + off
	}
	return s.pos, nil
}

func TestUpload_SourceChangesDuringUpload(t *testing.T) {
	fake := newFakeUpload(t)
	env := newTestEnv(t, fake.handler())
	r := &shrinkAfterHash{data: []byte("0123456789")}
	_, err := newUpload(env, 4).UploadReader(context.Background(), "a.txt", 10, r, "")
	if err == nil || !strings.Contains(err.Error(), "failed to read") {
		t.Fatalf("err = %v, want a read failure", err)
	}
	if fake.completeArg != nil {
		t.Error("complete sent for a file that could not be read")
	}
}

func TestUpload_RetryWaitHonoursContext(t *testing.T) {
	fake := newFakeUpload(t)
	fake.chunkFailures[0] = -1
	ctx, cancel := context.WithCancel(context.Background())
	env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/Chunks/") {
			cancel() // cancel while the chunk fails, before the retry delay elapses
		}
		fake.handler()(w, r)
	})
	svc := newUpload(env, 4)
	svc.retryDelay = time.Hour
	_, err := svc.UploadReader(ctx, "a.txt", 4, strings.NewReader("abcd"), "")
	if !stderrors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	// The handler may still be finishing the cancelled request, so read under its lock.
	fake.mu.Lock()
	tries := fake.chunkTries[0]
	fake.mu.Unlock()
	if tries > 1 {
		t.Errorf("retried %d times despite cancellation", tries)
	}
}

func TestUpload_OptionsAndMediaType(t *testing.T) {
	cases := []struct {
		name string
		file string
		opts []UploadOption
		want string
	}{
		{"detected image", "pic.PNG", nil, "Image"},
		{"detected video", "movie.mkv", nil, "Video"},
		{"detected audio", "song.flac", nil, "Audio"},
		{"detected font", "f.ttf", nil, "Font"},
		{"detected css", "s.css", nil, "Stylesheet"},
		{"detected text", "t.txt", nil, "Text"},
		{"unknown is Auxiliary, never Auto", "data.bin", nil, "Auxiliary"},
		{"override wins", "pic.png", []UploadOption{WithUploadMediaType(types.MediaTypeAuto)}, "Auto"},
		{"override to webpage", "x.html", []UploadOption{WithUploadMediaType(types.MediaTypeWebPage)}, "WebPage"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := newFakeUpload(t)
			env := newTestEnv(t, fake.handler())
			got, err := newUpload(env, 4).UploadReader(context.Background(), tc.file, 3, strings.NewReader("abc"), "", tc.opts...)
			if err != nil {
				t.Fatal(err)
			}
			if fake.startArgs[0]["mediaType"] != tc.want || got.MediaType != tc.want {
				t.Errorf("mediaType sent %v, returned %q, want %q", fake.startArgs[0]["mediaType"], got.MediaType, tc.want)
			}
		})
	}

	t.Run("default modification date is now", func(t *testing.T) {
		fake := newFakeUpload(t)
		env := newTestEnv(t, fake.handler())
		svc := newUpload(env, 4)
		svc.now = func() time.Time { return time.Date(2030, 5, 6, 7, 8, 9, 0, time.UTC) }
		if _, err := svc.UploadBytes2(t, "a.txt", []byte("abc")); err != nil {
			t.Fatal(err)
		}
		if fake.completeArg["fileLastModifiedDate"] != "2030-05-06T07:08:09.000Z" {
			t.Errorf("date = %v", fake.completeArg["fileLastModifiedDate"])
		}
	})

	t.Run("real date has T-ISO shape", func(t *testing.T) {
		fake := newFakeUpload(t)
		env := newTestEnv(t, fake.handler())
		if _, err := newUpload(env, 4).UploadBytes2(t, "a.txt", []byte("abc")); err != nil {
			t.Fatal(err)
		}
		d, _ := fake.completeArg["fileLastModifiedDate"].(string)
		if !regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$`).MatchString(d) {
			t.Errorf("complete date %q is not T-ISO", d)
		}
		if fake.startArgs[0]["fileLastModifiedDate"] != "0001-01-01 00:00:00Z" {
			t.Errorf("start must carry the space-separated sentinel, got %v", fake.startArgs[0]["fileLastModifiedDate"])
		}
	})
}

func TestUpload_FromFile(t *testing.T) {
	dir := t.TempDir()
	data := []byte("file on disk")
	path := filepath.Join(dir, "poster.jpg")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	mtime := time.Date(2025, 3, 4, 5, 6, 7, 0, time.UTC)
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}

	t.Run("uses base name, mtime and extension", func(t *testing.T) {
		fake := newFakeUpload(t)
		env := newTestEnv(t, fake.handler())
		got, err := newUpload(env, 5).Upload(context.Background(), path, "/images/")
		if err != nil {
			t.Fatal(err)
		}
		if got.FileName != "poster.jpg" || got.MediaType != "Image" || got.VirtualPath != `\images\` || got.ContentID != 99 {
			t.Errorf("got %+v", got)
		}
		body, _ := fake.file()
		if !bytes.Equal(body, data) {
			t.Errorf("body = %q", body)
		}
		if fake.completeArg["fileLastModifiedDate"] != "2025-03-04T05:06:07.000Z" {
			t.Errorf("mtime not used: %v", fake.completeArg["fileLastModifiedDate"])
		}
		if strings.Contains(string(fake.startBody), dir) {
			t.Error("local directory leaked into the request")
		}
	})

	t.Run("option overrides mtime", func(t *testing.T) {
		fake := newFakeUpload(t)
		env := newTestEnv(t, fake.handler())
		_, err := newUpload(env, 5).Upload(context.Background(), path, "", WithUploadModTime(time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)))
		if err != nil {
			t.Fatal(err)
		}
		if fake.completeArg["fileLastModifiedDate"] != "2001-01-01T00:00:00.000Z" {
			t.Errorf("date = %v", fake.completeArg["fileLastModifiedDate"])
		}
	})

	t.Run("bad paths", func(t *testing.T) {
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {})
		svc := newUpload(env, 5)
		_, err := svc.Upload(context.Background(), "", "")
		expectValidation(t, err)
		_, err = svc.Upload(context.Background(), dir, "")
		expectValidation(t, err)
		_, err = svc.Upload(context.Background(), filepath.Join(dir, "missing.png"), "")
		if !stderrors.Is(err, os.ErrNotExist) {
			t.Errorf("missing file err = %v", err)
		}
		empty := filepath.Join(dir, "empty.png")
		if err := os.WriteFile(empty, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		_, err = svc.Upload(context.Background(), empty, "")
		expectValidation(t, err)
		env.noRequests()
	})
}

func TestUpload_CancelMidUpload(t *testing.T) {
	fake := newFakeUpload(t)
	ctx, cancel := context.WithCancel(context.Background())
	fake.onChunk = func(off int64) {
		if off == 4 {
			cancel()
		}
	}
	env := newTestEnv(t, fake.handler())
	_, err := newUpload(env, 4).UploadReader(ctx, "a.png", 16, bytes.NewReader(bytes.Repeat([]byte("x"), 16)), "")
	if !stderrors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if _, offsets := fake.file(); len(offsets) > 3 {
		t.Errorf("kept sending after cancel: %v", offsets)
	}
	if fake.completeArg != nil {
		t.Error("complete sent after cancel")
	}
}

func TestUpload_CancelDuringHash(t *testing.T) {
	env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {})
	ctx, cancel := context.WithCancel(context.Background())
	// Cancelled after auth but before hashing completes: the reader notices.
	r := &cancelOnRead{cancel: cancel, r: bytes.NewReader(bytes.Repeat([]byte("x"), 1<<16))}
	_, err := newUpload(env, 4).UploadReader(ctx, "a.png", 1<<16, r, "")
	if !stderrors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	if reqs := env.requests(false); len(reqs) != 0 {
		t.Errorf("API calls after cancel: %+v", reqs)
	}
}

// cancelOnRead cancels its context on first read, then keeps serving bytes.
type cancelOnRead struct {
	cancel context.CancelFunc
	r      *bytes.Reader
}

func (c *cancelOnRead) Read(p []byte) (int, error) {
	c.cancel()
	return c.r.Read(p[:1])
}
func (c *cancelOnRead) Seek(o int64, w int) (int64, error) { return c.r.Seek(o, w) }

func TestUpload_TokenIsEscapedInURLs(t *testing.T) {
	fake := newFakeUpload(t)
	fake.token = "a/b c?d"
	env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
		// Route on the escaped form: the decoded path would be ambiguous.
		switch {
		case r.Method == http.MethodPost && r.URL.Path == uploadBase+"/Sessions/None/Uploads/":
			writeVendorJSON(w, types.ContentUploadStatus{UploadToken: "a/b c?d", State: types.UploadStateStarted})
		case r.Method == http.MethodPost:
			w.WriteHeader(200)
		case r.Method == http.MethodGet:
			writeVendorJSON(w, types.ContentUploadStatus{State: types.UploadStateUploaded})
		case r.Method == http.MethodPut:
			writeVendorJSON(w, types.ContentUploadStatus{State: types.UploadStateVerified, ContentID: 3})
		}
	})
	if _, err := newUpload(env, 4).UploadBytes2(t, "a.png", []byte("abcdef")); err != nil {
		t.Fatal(err)
	}
	for _, r := range env.requests(false)[1:] {
		if !strings.Contains(r.RawPath, "/Uploads/a%2Fb%20c%3Fd/") {
			t.Errorf("token not escaped in %s", r.RawPath)
		}
		if r.Method == http.MethodPost && r.RawQuery != "offset=0" && r.RawQuery != "offset=4" {
			t.Errorf("query = %q", r.RawQuery)
		}
	}
}

func TestUpload_NetworkSelectedBeforeStart(t *testing.T) {
	fake := newFakeUpload(t)
	env := newTestEnv(t, fake.handler())
	if _, err := newUpload(env, 4).UploadBytes2(t, "a.png", []byte("abcdefgh")); err != nil {
		t.Fatal(err)
	}
	all := env.requests(true)
	if all[0].Method != http.MethodPut || !strings.HasSuffix(all[0].Path, "/Self/Session/Network") {
		t.Errorf("first request = %s %s, want network selection", all[0].Method, all[0].Path)
	}
	if n := env.networkSelections(); n != 1 {
		t.Errorf("network selected %d times, want 1", n)
	}
}

// UploadBytes uploads to the library root and returns just the content ID.
func TestUpload_UploadBytes(t *testing.T) {
	fake := newFakeUpload(t)
	env := newTestEnv(t, fake.handler())
	id, err := newUpload(env, 4).UploadBytes(context.Background(), "pattern.png", []byte("pngbytes"), WithUploadMediaType(types.MediaTypeImage))
	if err != nil || id != 99 {
		t.Fatalf("id=%d err=%v", id, err)
	}
	if fake.startArgs[0]["virtualPath"] != `\` || fake.startArgs[0]["mediaType"] != "Image" {
		t.Errorf("start = %v", fake.startArgs[0])
	}
	if _, err := newUpload(env, 4).UploadBytes(context.Background(), "x.png", nil); err == nil {
		t.Error("empty data should be refused")
	}
}
