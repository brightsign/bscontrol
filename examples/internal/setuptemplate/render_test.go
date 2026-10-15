package setuptemplate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveVar(t *testing.T) {
	t.Run("flag takes precedence over env", func(t *testing.T) {
		t.Setenv("TEST_VAR", "env-value")
		got := ResolveVar("flag-value", "TEST_VAR")
		if got != "flag-value" {
			t.Errorf("expected flag-value, got %s", got)
		}
	})

	t.Run("falls back to env when flag is empty", func(t *testing.T) {
		t.Setenv("TEST_VAR", "env-value")
		got := ResolveVar("", "TEST_VAR")
		if got != "env-value" {
			t.Errorf("expected env-value, got %s", got)
		}
	})

	t.Run("returns empty when both are empty", func(t *testing.T) {
		got := ResolveVar("", "NONEXISTENT_VAR_12345")
		if got != "" {
			t.Errorf("expected empty string, got %s", got)
		}
	})
}

func TestRender(t *testing.T) {
	// Create a minimal template for testing
	tmplContent := `{
  "version": "3.0.0",
  "bDeploy": {
    "username": "{{.Username}}",
    "networkName": "{{.NetworkName}}",
    "packageName": "{{.PackageName}}"
  },
  "setupType": "lfn",
  "deviceName": "{{.DeviceName}}",
  "deviceDescription": "{{.DeviceDescription}}",
  "bsnGroupName": "{{.GroupName}}"
}`

	t.Run("renders all variables", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmplPath := filepath.Join(tmpDir, "test-template.json")
		if err := os.WriteFile(tmplPath, []byte(tmplContent), 0644); err != nil {
			t.Fatal(err)
		}

		vars := TemplateVars{
			Username:          "user@example.com",
			NetworkName:       "test-network",
			PackageName:       "test-package",
			DeviceName:        "test-device",
			DeviceDescription: "test description",
			GroupName:         "TestGroup",
		}

		record, err := Render(tmplPath, vars)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}

		if record.BDeploy.Username != "user@example.com" {
			t.Errorf("expected username user@example.com, got %s", record.BDeploy.Username)
		}
		if record.BDeploy.NetworkName != "test-network" {
			t.Errorf("expected networkName test-network, got %s", record.BDeploy.NetworkName)
		}
		if record.BDeploy.PackageName != "test-package" {
			t.Errorf("expected packageName test-package, got %s", record.BDeploy.PackageName)
		}
		if record.DeviceName != "test-device" {
			t.Errorf("expected deviceName test-device, got %s", record.DeviceName)
		}
		if record.DeviceDescription != "test description" {
			t.Errorf("expected deviceDescription 'test description', got %s", record.DeviceDescription)
		}
		if record.BSNGroupName != "TestGroup" {
			t.Errorf("expected bsnGroupName TestGroup, got %s", record.BSNGroupName)
		}
		if record.SetupType != "lfn" {
			t.Errorf("expected setupType lfn, got %s", record.SetupType)
		}
	})

	t.Run("renders with empty optional variables", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmplPath := filepath.Join(tmpDir, "test-template.json")
		if err := os.WriteFile(tmplPath, []byte(tmplContent), 0644); err != nil {
			t.Fatal(err)
		}

		vars := TemplateVars{
			Username:    "user@example.com",
			NetworkName: "test-network",
			PackageName: "test-package",
			GroupName:   "Default",
		}

		record, err := Render(tmplPath, vars)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}

		if record.DeviceName != "" {
			t.Errorf("expected empty deviceName, got %s", record.DeviceName)
		}
		if record.DeviceDescription != "" {
			t.Errorf("expected empty deviceDescription, got %s", record.DeviceDescription)
		}
	})

	t.Run("fails on missing template file", func(t *testing.T) {
		_, err := Render("/nonexistent/path.json", TemplateVars{})
		if err == nil {
			t.Fatal("expected error for missing template file")
		}
	})

	t.Run("fails on invalid template syntax", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmplPath := filepath.Join(tmpDir, "bad-template.json")
		if err := os.WriteFile(tmplPath, []byte(`{"bad": "{{.Unclosed"`), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := Render(tmplPath, TemplateVars{})
		if err == nil {
			t.Fatal("expected error for invalid template")
		}
	})

	t.Run("fails on invalid JSON after rendering", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmplPath := filepath.Join(tmpDir, "not-json.json")
		if err := os.WriteFile(tmplPath, []byte(`not json {{.PackageName}}`), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := Render(tmplPath, TemplateVars{PackageName: "test"})
		if err == nil {
			t.Fatal("expected error for invalid JSON output")
		}
	})
}
