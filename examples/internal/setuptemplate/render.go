package setuptemplate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"text/template"

	"github.com/brightsign/bscontrol"
)

const (
	// EnvPackageName is the environment variable for the setup package name.
	EnvPackageName = "BS_PACKAGE_NAME"

	// EnvDeviceName is the environment variable for the device name.
	EnvDeviceName = "BS_DEVICE_NAME"

	// EnvDeviceDescription is the environment variable for the device description.
	EnvDeviceDescription = "BS_DEVICE_DESCRIPTION"

	// DefaultTemplate is the default path to the setup template file.
	DefaultTemplate = "setups/setup-template.json"
)

// TemplateVars contains all placeholder values for the setup template.
type TemplateVars struct {
	Username          string
	NetworkName       string
	PackageName       string
	SetupType         string
	RegistrationToken string
	TokenValidFrom    string
	TokenValidTo      string
	DeviceName        string
	DeviceDescription string
	GroupName         string
}

// ResolveVar returns the flag value if non-empty, otherwise falls back to the
// named environment variable.
func ResolveVar(flagValue string, envVar string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv(envVar)
}

// Render reads the template file, executes it with the provided variables, and
// unmarshals the result into a BDeploySetupRecord.
func Render(templatePath string, vars TemplateVars) (*bscontrol.BDeploySetupRecord, error) {
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %w", err)
	}

	tmpl, err := template.New("setup").Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	var record bscontrol.BDeploySetupRecord
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rendered template: %w", err)
	}

	return &record, nil
}

// RenderRaw reads the template file, executes it with the provided variables,
// and returns the rendered JSON string without unmarshalling into a struct.
// This preserves all fields exactly as they appear in the template.
func RenderRaw(templatePath string, vars TemplateVars) (string, error) {
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file: %w", err)
	}

	tmpl, err := template.New("setup").Parse(string(data))
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	// Validate that the result is valid JSON
	if !json.Valid(buf.Bytes()) {
		return "", fmt.Errorf("rendered template is not valid JSON")
	}

	return buf.String(), nil
}
