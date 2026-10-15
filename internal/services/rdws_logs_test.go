package services

import (
	"encoding/json"
	"testing"
)

// GET /logs returns dmesg as a plain JSON string, so a string payload is the
// SUCCESS case. The previous implementation reported it as a device error, which
// meant GetLogs could never succeed.
func TestParseLogsResult(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantText  string
		wantFiles int
		wantErr   bool
	}{
		{
			name:     "dmesg string is log text, not an error",
			raw:      `"[    0.000000] Booting Linux\n[   12.345678] JS.Update : Getting up-to-date list of available supervisors from server.\n"`,
			wantText: "[    0.000000] Booting Linux\n[   12.345678] JS.Update : Getting up-to-date list of available supervisors from server.\n",
		},
		{"empty string is valid", `""`, "", 0, false},
		{"structured list still accepted", `{"logs":[{"name":"system.log","size":1024}]}`, "", 1, false},
		{"object without logs key", `{}`, "", 0, false},
		{"malformed payload", `{"logs":`, "", 0, true},
		{"unexpected type", `12345`, "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs, err := parseLogsResult(json.RawMessage(tt.raw))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %+v", logs)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if logs.Text != tt.wantText {
				t.Errorf("Text: got %q, want %q", logs.Text, tt.wantText)
			}
			if len(logs.Files) != tt.wantFiles {
				t.Errorf("Files: got %d, want %d", len(logs.Files), tt.wantFiles)
			}
		})
	}
}
