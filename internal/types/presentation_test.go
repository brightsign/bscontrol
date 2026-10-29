package types

import (
	"encoding/json"
	"testing"
)

// The service wants the full key set on a create. Nil file refs and empty lists
// must serialise as null / [] rather than being dropped.
func TestPresentationCreateRequestAlwaysWritesPlaceholders(t *testing.T) {
	req := PresentationCreateRequest{
		Name:             "p",
		CreationDate:     PresentationNullDate,
		LastModifiedDate: PresentationNullDate,
		Files:            []PresentationFile{},
		Applications:     []any{},
		Dependencies:     []any{},
		Permissions:      []any{},
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	nulls := []string{"projectFile", "autoplayFile", "resourcesFile", "userDefinedEventsFile", "thumbnailFile", "autorunPlugins", "groups"}
	for _, k := range nulls {
		if string(m[k]) != "null" {
			t.Errorf("%s = %s, want null", k, m[k])
		}
	}
	for _, k := range []string{"files", "applications", "dependencies", "permissions"} {
		if string(m[k]) != "[]" {
			t.Errorf("%s = %s, want []", k, m[k])
		}
	}
	for _, k := range []string{"autorun", "deviceWebPage", "screenSettings", "tags", "deviceModel"} {
		if _, ok := m[k]; ok {
			t.Errorf("%s should be omitted when empty", k)
		}
	}
	if string(m["id"]) != "0" {
		t.Errorf("id = %s, want 0", m["id"])
	}
}

func TestPresentationFileRefWireShape(t *testing.T) {
	ref := PresentationFileRef{Type: "New", Name: "x.bpfx", TransferEncoding: "none", LastModifiedDate: "2026-01-01T00:00:00Z"}
	b, _ := json.Marshal(ref)
	var m map[string]json.RawMessage
	_ = json.Unmarshal(b, &m)
	for _, k := range []string{"type", "name", "body", "transferEncoding", "size", "hash", "creationDate", "lastModifiedDate"} {
		if _, ok := m[k]; !ok {
			t.Errorf("missing key %s in %s", k, b)
		}
	}
	if string(m["body"]) != "null" || string(m["creationDate"]) != "null" {
		t.Errorf("body/creationDate should be null: %s", b)
	}
	for _, k := range []string{"id", "path"} {
		if _, ok := m[k]; ok {
			t.Errorf("%s should be omitted: %s", k, b)
		}
	}
}

func TestPresentationDecode(t *testing.T) {
	body := `{"id":5,"name":"p","type":"Complete","status":"Published","creationDate":"2026-08-18T12:00:00Z",
	"lastModifiedDate":"2026-08-18T12:00:00Z","deviceModel":"XT1144","language":"English",
	"projectFile":{"type":"Project","name":"p.bpfx","id":3,"size":10,"hash":"h","creationDate":"2026-01-01T00:00:00Z","lastModifiedDate":"2026-01-01T00:00:00Z","body":null},
	"files":[{"type":"Media","id":9,"name":"a.png","mediaType":"Image"}],
	"autorun":{"version":"10.0.110","isCustom":false},"screenSettings":{"videoMode":"1920x1080x60p","overscan":"NoOverscan"},
	"groups":null,"permissions":[]}`
	var p Presentation
	if err := json.Unmarshal([]byte(body), &p); err != nil {
		t.Fatal(err)
	}
	if p.ID != 5 || p.Name != "p" || p.Status != "Published" || p.CreationDate.IsZero() {
		t.Errorf("got %+v", p)
	}
	if p.ProjectFile == nil || p.ProjectFile.ID != 3 || p.ProjectFile.CreationDate == nil {
		t.Errorf("projectFile = %+v", p.ProjectFile)
	}
	if len(p.Files) != 1 || p.Files[0].ID != 9 || p.ScreenSettings.Overscan != "NoOverscan" {
		t.Errorf("files/screen = %+v / %+v", p.Files, p.ScreenSettings)
	}
}

func TestScheduledPresentationDaysOfWeekEitherShape(t *testing.T) {
	for name, in := range map[string]string{"string": `"EveryDay"`, "number": `127`} {
		t.Run(name, func(t *testing.T) {
			var s ScheduledPresentation
			if err := json.Unmarshal([]byte(`{"id":1,"presentationId":2,"isRecurrent":true,"startTime":"00:00:00","duration":"23:59:59","daysOfWeek":`+in+`}`), &s); err != nil {
				t.Fatal(err)
			}
			if s.DaysOfWeek == nil || s.ID != 1 || s.PresentationID != 2 {
				t.Errorf("got %+v", s)
			}
		})
	}
}
