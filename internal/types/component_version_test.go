package types

import (
	"encoding/json"
	"testing"
)

// Firmware on some players reports the /v1/system version components as JSON
// strings ("9") rather than numbers (9). Before RDWSComponentVersion carried a
// tolerant UnmarshalJSON, a string component failed the whole decode, which the
// HTTP client then retried pointlessly (a ~0.3s call became ~7s). Both encodings
// must now decode to the same value.
func TestRDWSComponentVersionAcceptsStringOrNumber(t *testing.T) {
	cases := []struct {
		name string
		json string
		want RDWSComponentVersion
	}{
		{"all numbers", `{"major":9,"minor":1,"patch":52,"build":3}`, RDWSComponentVersion{9, 1, 52, 3}},
		{"all strings", `{"major":"9","minor":"1","patch":"52","build":"3"}`, RDWSComponentVersion{9, 1, 52, 3}},
		{"mixed", `{"major":"9","minor":1,"patch":"52"}`, RDWSComponentVersion{9, 1, 52, 0}},
		{"empty string build", `{"major":9,"minor":1,"patch":52,"build":""}`, RDWSComponentVersion{9, 1, 52, 0}},
		{"missing build", `{"major":9,"minor":1,"patch":52}`, RDWSComponentVersion{9, 1, 52, 0}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got RDWSComponentVersion
			if err := json.Unmarshal([]byte(tc.json), &got); err != nil {
				t.Fatalf("Unmarshal(%s) errored, want success: %v", tc.json, err)
			}
			if got != tc.want {
				t.Errorf("Unmarshal(%s) = %+v, want %+v", tc.json, got, tc.want)
			}
		})
	}
}

// A component that is neither a number nor an integer-bearing string is a real
// decode error and must still surface as one — the tolerance is string-or-number,
// not "swallow anything".
func TestRDWSComponentVersionRejectsNonInteger(t *testing.T) {
	var got RDWSComponentVersion
	if err := json.Unmarshal([]byte(`{"major":"nine"}`), &got); err == nil {
		t.Error("Unmarshal of a non-integer string component succeeded, want an error")
	}
}

// The tolerant version decodes correctly when nested inside RDWSSystemInfo, the
// real shape GetSystemInfo returns — a string major here is exactly the field
// that drove the retry storm.
func TestRDWSSystemInfoDecodesStringFirmwareVersion(t *testing.T) {
	body := `{"firmware":{"version":{"major":"9","minor":"1","patch":"52","build":"3"}}}`
	var sys RDWSSystemInfo
	if err := json.Unmarshal([]byte(body), &sys); err != nil {
		t.Fatalf("Unmarshal RDWSSystemInfo errored, want success: %v", err)
	}
	if got := sys.Firmware.Version.String(); got != "9.1.52.3" {
		t.Errorf("firmware version = %q, want 9.1.52.3", got)
	}
}
