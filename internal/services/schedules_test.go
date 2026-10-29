package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/brightsign/bscontrol/internal/types"
)

var _ ScheduleService = (*scheduleService)(nil)

func newSched(env *testEnv) ScheduleService {
	return NewScheduleService(env.cfg, env.hc, env.am)
}

const groupsPath = "/" + testAPIVersion + "/Groups/Regular"

func schedOps() []op {
	sched := func() *types.ScheduledPresentation {
		return &types.ScheduledPresentation{PresentationID: 4, StartTime: "00:00:00", Duration: "23:59:59"}
	}
	return []op{
		{"GetGroupSchedule", func(ctx context.Context, e *testEnv) error {
			_, err := newSched(e).GetGroupSchedule(ctx, "g")
			return err
		}},
		{"AddScheduledPresentation", func(ctx context.Context, e *testEnv) error {
			_, err := newSched(e).AddScheduledPresentation(ctx, "g", sched())
			return err
		}},
		{"DeleteScheduledPresentation", func(ctx context.Context, e *testEnv) error {
			return newSched(e).DeleteScheduledPresentation(ctx, "g", 9)
		}},
	}
}

func TestScheduleService_CommonBehaviour(t *testing.T) {
	runCommonBehaviour(t, schedOps())
}

func TestScheduleService_GetGroupSchedule(t *testing.T) {
	groups := []struct{ name, wantRaw string }{
		{"Canary Group", groupsPath + "/Canary%20Group/schedule/"},
		{"a/b", groupsPath + "/a%2Fb/schedule/"},
		{"x?y#z", groupsPath + "/x%3Fy%23z/schedule/"},
	}
	for _, tc := range groups {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, `{"items":[
				 {"id":11,"presentationId":4,"presentationName":"p","isRecurrent":true,"startTime":"00:00:00","duration":"23:59:59","daysOfWeek":"EveryDay","recurrenceStartDate":"2026-01-01T00:00:00Z"},
				 {"id":12,"presentationId":5,"isRecurrent":false,"eventDate":"2026-02-03T04:05:06Z","startTime":"10:00:00","duration":"01:00:00","daysOfWeek":127}
				],"isTruncated":false}`)
			})
			got, err := newSched(env).GetGroupSchedule(context.Background(), tc.name)
			if err != nil {
				t.Fatal(err)
			}
			r := env.only()
			requireBearer(t, r)
			if r.Method != http.MethodGet || r.RawPath != tc.wantRaw {
				t.Errorf("request = %s %q, want %q", r.Method, r.RawPath, tc.wantRaw)
			}
			if len(got.Items) != 2 || got.Items[0].ID != 11 || got.Items[0].DaysOfWeek != "EveryDay" || got.Items[1].DaysOfWeek != float64(127) {
				t.Errorf("got %+v", got.Items)
			}
			if got.Items[1].EventDate == nil || !got.Items[1].EventDate.Equal(time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)) {
				t.Errorf("eventDate = %v", got.Items[1].EventDate)
			}
		})
	}
	t.Run("empty group name", func(t *testing.T) {
		env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {})
		_, err := newSched(env).GetGroupSchedule(context.Background(), "")
		expectValidation(t, err)
		env.noRequests()
	})
}

func TestScheduleService_AddScheduledPresentation(t *testing.T) {
	env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, `{"id":31,"presentationId":4,"isRecurrent":true,"startTime":"00:00:00","duration":"23:59:59","daysOfWeek":"EveryDay"}`)
	})
	rs := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	in := &types.ScheduledPresentation{
		PresentationID: 4, PresentationName: "p", IsRecurrent: true, StartTime: "00:00:00", Duration: "23:59:59",
		DaysOfWeek: types.DaysOfWeekEveryDay, RecurrenceStartDate: &rs,
	}
	got, err := newSched(env).AddScheduledPresentation(context.Background(), "Canary Group", in)
	if err != nil {
		t.Fatal(err)
	}
	r := env.only()
	requireBearer(t, r)
	if r.Method != http.MethodPost || r.RawPath != groupsPath+"/Canary%20Group/schedule/" {
		t.Errorf("request = %s %s", r.Method, r.RawPath)
	}
	var body map[string]any
	if err := json.Unmarshal(r.Body, &body); err != nil {
		t.Fatal(err)
	}
	if body["presentationId"] != float64(4) || body["daysOfWeek"] != "EveryDay" || body["isRecurrent"] != true ||
		body["startTime"] != "00:00:00" || body["duration"] != "23:59:59" || body["recurrenceStartDate"] != "2026-01-01T00:00:00Z" {
		t.Errorf("body = %v", body)
	}
	if _, ok := body["id"]; ok {
		t.Errorf("a new schedule must not send an id: %v", body)
	}
	if got.ID != 31 {
		t.Errorf("got %+v", got)
	}
}

func TestScheduleService_AddScheduledPresentation_Validation(t *testing.T) {
	cases := []struct {
		name  string
		group string
		sched *types.ScheduledPresentation
	}{
		{"empty group", "", &types.ScheduledPresentation{PresentationID: 1}},
		{"nil schedule", "g", nil},
		{"zero presentation", "g", &types.ScheduledPresentation{}},
		{"negative presentation", "g", &types.ScheduledPresentation{PresentationID: -2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {})
			_, err := newSched(env).AddScheduledPresentation(context.Background(), tc.group, tc.sched)
			expectValidation(t, err)
			env.noRequests()
		})
	}
}

func TestScheduleService_DeleteScheduledPresentation(t *testing.T) {
	env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	if err := newSched(env).DeleteScheduledPresentation(context.Background(), "Canary Group", 31); err != nil {
		t.Fatal(err)
	}
	r := env.only()
	requireBearer(t, r)
	if r.Method != http.MethodDelete || r.RawPath != groupsPath+"/Canary%20Group/schedule/31/" {
		t.Errorf("request = %s %s", r.Method, r.RawPath)
	}

	cases := []struct {
		group string
		id    int
	}{{"", 1}, {"g", 0}, {"g", -4}}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%q/%d", tc.group, tc.id), func(t *testing.T) {
			env := newTestEnv(t, func(w http.ResponseWriter, r *http.Request) {})
			expectValidation(t, newSched(env).DeleteScheduledPresentation(context.Background(), tc.group, tc.id))
			env.noRequests()
		})
	}
}
