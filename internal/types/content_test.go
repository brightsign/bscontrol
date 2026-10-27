package types

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBSNTimeUnmarshal(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    time.Time
		wantErr bool
	}{
		{"rfc3339 zulu", `"2026-08-18T12:34:56Z"`, time.Date(2026, 8, 18, 12, 34, 56, 0, time.UTC), false},
		{"rfc3339 millis", `"2026-08-18T12:34:56.123Z"`, time.Date(2026, 8, 18, 12, 34, 56, 123000000, time.UTC), false},
		{"rfc3339 offset", `"2026-08-18T12:34:56+02:00"`, time.Date(2026, 8, 18, 10, 34, 56, 0, time.UTC), false},
		{"no zone", `"2026-08-18T12:34:56"`, time.Date(2026, 8, 18, 12, 34, 56, 0, time.UTC), false},
		{"space separated with Z", `"0001-01-01 00:00:00Z"`, time.Time{}, false},
		{"space separated no zone", `"2026-08-18 12:34:56"`, time.Date(2026, 8, 18, 12, 34, 56, 0, time.UTC), false},
		{"null", `null`, time.Time{}, false},
		{"empty string", `""`, time.Time{}, false},
		{"garbage", `"yesterday"`, time.Time{}, true},
		{"number", `12345`, time.Time{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got BSNTime
			err := json.Unmarshal([]byte(tc.in), &got)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Unmarshal(%s) error = %v, wantErr %v", tc.in, err, tc.wantErr)
			}
			if err == nil && !got.UTC().Equal(tc.want) {
				t.Errorf("Unmarshal(%s) = %v, want %v", tc.in, got.UTC(), tc.want)
			}
		})
	}
}

func TestBSNTimeMarshal(t *testing.T) {
	zero, err := json.Marshal(BSNTime{})
	if err != nil || string(zero) != `"0001-01-01T00:00:00"` {
		t.Errorf("zero marshalled to %s (err %v), want BSN's null date", zero, err)
	}
	when := time.Date(2026, 8, 18, 12, 34, 56, 0, time.UTC)
	b, err := json.Marshal(BSNTime{Time: when})
	if err != nil || string(b) != `"2026-08-18T12:34:56Z"` {
		t.Errorf("date marshalled to %s (err %v)", b, err)
	}
	var back BSNTime
	if err := json.Unmarshal(b, &back); err != nil || !back.Equal(when) {
		t.Errorf("round trip failed: %v %v", back, err)
	}
}

// The wire key for the hash is shA1Hash, tags is absent, and the two tokens and
// the thumbnail are explicit nulls. All of that is part of what BSN needs.
func TestContentUploadArgumentsWireShape(t *testing.T) {
	args := ContentUploadArguments{
		FileName:             "clip.mp4",
		VirtualPath:          `\videos\`,
		MediaType:            MediaTypeVideo,
		FileSize:             42,
		FileLastModifiedDate: "0001-01-01 00:00:00Z",
		SHA1Hash:             "abc123",
	}
	b, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"sessionToken", "uploadToken", "fileThumb"} {
		v, ok := m[key]
		if !ok || v != nil {
			t.Errorf("%s should be present and null, got %v (present=%v)", key, v, ok)
		}
	}
	if m["shA1Hash"] != "abc123" {
		t.Errorf("shA1Hash = %v, want abc123 (body %s)", m["shA1Hash"], b)
	}
	for _, bad := range []string{"sha1Hash", "sha1hash", "SHA1Hash", "tags"} {
		if _, ok := m[bad]; ok {
			t.Errorf("unexpected key %q in %s", bad, b)
		}
	}
	if !strings.Contains(string(b), `"virtualPath":"\\videos\\"`) {
		t.Errorf("virtualPath not JSON-escaped as expected: %s", b)
	}
}

func TestContentUploadStatusDecodesReferenceShape(t *testing.T) {
	body := `{
	  "uploadToken":"tok-1","contentId":77,"fileName":"clip.mp4","fileSize":1234,
	  "shA1Hash":"deadbeef","state":"Uploading",
	  "startTime":"2026-08-18T12:00:00","endTime":"0001-01-01T00:00:00","lastActivityTime":"2026-08-18T12:00:01Z",
	  "chunks":[{"count":3,"length":524288,"state":"Uploaded"},{"count":1,"length":100,"state":"Queued"}],
	  "matchingFiles":[{"id":9,"fileName":"clip.mp4","virtualPath":"\\videos\\","mediaType":"Video","fileSize":1234,"shA1Hash":"deadbeef","uploadDate":"2026-08-01T00:00:00"}]
	}`
	var st ContentUploadStatus
	if err := json.Unmarshal([]byte(body), &st); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if st.UploadToken != "tok-1" || st.ContentID != 77 || st.State != UploadStateUploading || st.SHA1Hash != "deadbeef" {
		t.Errorf("unexpected status %+v", st)
	}
	if len(st.Chunks) != 2 || st.Chunks[0].State != UploadStateUploaded || st.Chunks[0].Count != 3 {
		t.Errorf("chunks = %+v", st.Chunks)
	}
	if len(st.MatchingFiles) != 1 || st.MatchingFiles[0].ID != 9 || st.MatchingFiles[0].VirtualPath != `\videos\` {
		t.Errorf("matchingFiles = %+v", st.MatchingFiles)
	}
}

// Decoding is case-insensitive, so the PascalCase spelling some responses use
// still fills the struct.
func TestContentUploadStatusDecodesPascalCase(t *testing.T) {
	var st ContentUploadStatus
	if err := json.Unmarshal([]byte(`{"UploadToken":"t","ContentId":5,"SHA1Hash":"ff","State":"Verified"}`), &st); err != nil {
		t.Fatal(err)
	}
	if st.UploadToken != "t" || st.ContentID != 5 || st.SHA1Hash != "ff" || st.State != UploadStateVerified {
		t.Errorf("got %+v", st)
	}
}

func TestContentFileDecode(t *testing.T) {
	body := `{"id":12,"fileName":"a.png","physicalPath":"https://s3/x","virtualPath":"\\img\\","thumbPath":"https://t",
	"mediaType":"Image","fileSize":99,"fileHash":"SHA1:abc","uploadDate":"2026-08-18T12:34:56",
	"lastModifiedDate":"2026-08-18T12:34:56Z","fileLastModifiedDate":"0001-01-01T00:00:00",
	"metadata":{"width":10},"probeData":"ffprobe output"}`
	var f ContentFile
	if err := json.Unmarshal([]byte(body), &f); err != nil {
		t.Fatal(err)
	}
	if f.ID != 12 || f.FileName != "a.png" || f.MediaType != MediaTypeImage || f.FileHash != "SHA1:abc" {
		t.Errorf("got %+v", f)
	}
	if f.UploadDate.IsZero() || !f.FileLastModifiedDate.IsZero() {
		t.Errorf("dates: upload=%v fileLastModified=%v", f.UploadDate, f.FileLastModifiedDate)
	}
	if string(f.Metadata) != `{"width":10}` || string(f.ProbeData) != `"ffprobe output"` {
		t.Errorf("raw fields: %s / %s", f.Metadata, f.ProbeData)
	}
}
