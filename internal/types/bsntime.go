package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

// BSNTime is a timestamp as BSN.cloud writes it. The service is not consistent:
// a date can arrive as RFC 3339 ("2026-08-18T12:34:56Z"), as ISO without a zone
// ("2026-08-18T12:34:56"), or in the space-separated form the upload API uses
// ("0001-01-01 00:00:00Z"). A plain time.Time rejects the zone-less forms and
// would turn a whole list response into a decode error, so BSNTime accepts all of
// them. Null and the empty string decode to the zero time.
//
// Zone-less values are read as UTC.
type BSNTime struct {
	time.Time
}

// bsnTimeLayouts are tried in order. The first that parses wins.
var bsnTimeLayouts = []string{
	time.RFC3339Nano,
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05Z07:00",
	"2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05Z07:00",
}

// bsnZeroTime is how BSN spells "no date" when we have to write one back.
const bsnZeroTime = "0001-01-01T00:00:00"

// UnmarshalJSON implements json.Unmarshaler.
func (t *BSNTime) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if bytes.Equal(b, []byte("null")) {
		t.Time = time.Time{}
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("BSN time must be a string: %w", err)
	}
	if s == "" {
		t.Time = time.Time{}
		return nil
	}
	for _, layout := range bsnTimeLayouts {
		if parsed, err := time.Parse(layout, s); err == nil {
			t.Time = parsed
			return nil
		}
	}
	return fmt.Errorf("unrecognised BSN time %q", s)
}

// MarshalJSON implements json.Marshaler. The zero time is written as BSN's own
// "no date" value; anything else as RFC 3339 in UTC.
func (t BSNTime) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return json.Marshal(bsnZeroTime)
	}
	return json.Marshal(t.UTC().Format(time.RFC3339Nano))
}
