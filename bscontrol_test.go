package bscontrol

import (
	"context"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	// Test with valid credentials
	client, err := New(
		WithCredentials("test-id", "test-secret"),
		WithNetwork("test-network"),
		WithTimeout(60*time.Second),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client == nil {
		t.Error("Expected client to be created, got nil")
	}

	// Test configuration
	config := client.Config()
	if config.ClientID != "test-id" {
		t.Errorf("Expected client ID 'test-id', got '%s'", config.ClientID)
	}

	if config.ClientSecret != "test-secret" {
		t.Errorf("Expected client secret 'test-secret', got '%s'", config.ClientSecret)
	}

	if config.NetworkName != "test-network" {
		t.Errorf("Expected network name 'test-network', got '%s'", config.NetworkName)
	}

	if config.Timeout != 60*time.Second {
		t.Errorf("Expected timeout 60s, got %v", config.Timeout)
	}
}

func TestNewWithoutCredentials(t *testing.T) {
	// Isolate from any credentials the developer's shell exports (e.g. via .envrc),
	// which New would otherwise pick up through LoadFromEnv. t.Setenv restores the
	// originals when the test finishes; LoadFromEnv skips empty values.
	t.Setenv("BS_CLIENT_ID", "")
	t.Setenv("BS_SECRET", "")
	t.Setenv("BS_NETWORK", "")

	// Test without credentials should fail validation
	_, err := New()
	if err == nil {
		t.Error("Expected error when creating client without credentials")
	}

	if !IsConfigurationError(err) {
		t.Errorf("Expected configuration error, got: %v", err)
	}
}

func TestClientMethods(t *testing.T) {
	client, err := New(
		WithCredentials("test-id", "test-secret"),
		WithNetwork("test-network"),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test initial state
	if client.IsAuthenticated() {
		t.Error("Expected client to not be authenticated initially")
	}

	if client.IsNetworkSet() {
		t.Error("Expected network to not be set initially")
	}

	// Test that methods requiring authentication fail appropriately
	ctx := context.Background()

	_, err = client.GetNetworks(ctx)
	if err == nil {
		t.Error("Expected error when getting networks without authentication")
	}

	if !IsAuthenticationError(err) {
		t.Errorf("Expected authentication error, got: %v", err)
	}
}

func TestErrorTypes(t *testing.T) {
	// Test that error checking functions are available
	testErr := &testError{"test error"}

	// These should all return false for a generic error
	if IsAuthenticationError(testErr) {
		t.Error("IsAuthenticationError should return false for generic error")
	}

	if IsNetworkError(testErr) {
		t.Error("IsNetworkError should return false for generic error")
	}

	if IsConfigurationError(testErr) {
		t.Error("IsConfigurationError should return false for generic error")
	}

	if IsRetryableError(testErr) {
		t.Error("IsRetryableError should return false for generic error")
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestConvenienceMethods(t *testing.T) {
	client, err := New(
		WithCredentials("test-id", "test-secret"),
		WithNetwork("test-network"),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Test WithAuthentication - should fail since we don't have real creds
	err = client.WithAuthentication(ctx, func() error {
		return nil
	})
	if err == nil {
		t.Error("Expected error in WithAuthentication without real credentials")
	}

	// Test WithNetworkContext - should fail since we don't have real creds
	err = client.WithNetworkContext(ctx, func() error {
		return nil
	})
	if err == nil {
		t.Error("Expected error in WithNetworkContext without real credentials")
	}
}

func TestDeviceService(t *testing.T) {
	client, err := New(
		WithCredentials("test-id", "test-secret"),
		WithNetwork("test-network"),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test that device service is available
	if client.Devices == nil {
		t.Error("Expected Devices service to be available")
	}

	ctx := context.Background()

	// Test device listing without auth should fail
	_, err = client.Devices.List(ctx)
	if err == nil {
		t.Error("Expected error when listing devices without authentication")
	}

	// Test device get without auth should fail
	_, err = client.Devices.Get(ctx, "test-serial")
	if err == nil {
		t.Error("Expected error when getting device without authentication")
	}

	// Test device get by ID without auth should fail
	_, err = client.Devices.GetByID(ctx, 123)
	if err == nil {
		t.Error("Expected error when getting device by ID without authentication")
	}
}

func TestListOptions(t *testing.T) {
	// Test that list options are available at the top level
	opts := []ListOption{
		WithPageSize(10),
		WithMarker("test-marker"),
		WithFilter("model=XD1033"),
		WithSort("registrationDate"),
	}

	if len(opts) != 4 {
		t.Errorf("Expected 4 list options, got %d", len(opts))
	}
}

func TestDeviceStatusAndErrorMethods(t *testing.T) {
	client, err := New(
		WithCredentials("test-id", "test-secret"),
		WithNetwork("test-network"),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Test device status methods without auth should fail
	_, err = client.Devices.GetStatus(ctx, 123)
	if err == nil {
		t.Error("Expected error when getting device status without authentication")
	}

	_, err = client.Devices.GetStatusBySerial(ctx, "test-serial")
	if err == nil {
		t.Error("Expected error when getting device status by serial without authentication")
	}

	// Test device error methods without auth should fail
	_, err = client.Devices.GetErrors(ctx, 123)
	if err == nil {
		t.Error("Expected error when getting device errors without authentication")
	}

	_, err = client.Devices.GetErrorsBySerial(ctx, "test-serial")
	if err == nil {
		t.Error("Expected error when getting device errors by serial without authentication")
	}
}

func TestPublicTypeExports(t *testing.T) {
	// Test that all public types are available
	var _ *DeviceStatus
	var _ *DeviceError
	var _ *DeviceErrorList

	// Test that we can create instances
	status := &DeviceStatus{
		DeviceID: "test",
		Serial:   "test-serial",
	}
	if status.DeviceID != "test" {
		t.Error("DeviceStatus type export failed")
	}

	deviceErr := &DeviceError{
		ID:       1,
		Message:  "test error",
		Severity: "warning",
	}
	if deviceErr.Message != "test error" {
		t.Error("DeviceError type export failed")
	}

	errorList := &DeviceErrorList{
		Items: []DeviceError{*deviceErr},
	}
	if len(errorList.Items) != 1 {
		t.Error("DeviceErrorList type export failed")
	}

	// Test Device Status nested types
	t.Run("DeviceStatusEmbed", func(t *testing.T) {
		status := &DeviceStatusEmbed{
			Uptime: "12:34:56",
			Health: "Healthy",
			Firmware: &FirmwareInfo{
				Version: "8.5.42",
			},
			Script: &ScriptInfo{
				Type:    "BrightScript",
				Version: "1.0",
			},
			Synchronization: &SyncSettings{
				Status: SyncPeriod{Period: "5m"},
			},
			Storage: []StorageInfo{
				{Total: 1000000, Used: 500000},
			},
			Network: &NetworkInfo{
				ExternalIP: "192.168.1.100",
				Interfaces: []NetworkInterfaceBSN{
					{Name: "eth0", Type: "ethernet"},
				},
			},
		}
		if status.Firmware.Version != "8.5.42" {
			t.Errorf("DeviceStatusEmbed firmware access failed")
		}
	})

	// Test Snapshot types
	t.Run("SnapshotTypes", func(t *testing.T) {
		req := &SnapshotRequest{
			Format:  "png",
			Quality: 90,
			Region:  &Region{X: 0, Y: 0, Width: 1920, Height: 1080},
		}
		resp := &SnapshotResponse{
			Filename:  "snapshot.png",
			Timestamp: "2024-01-01T12:00:00Z",
			Width:     1920,
			Height:    1080,
		}
		if req.Format != "png" || resp.Width != 1920 {
			t.Errorf("Snapshot types access failed")
		}
	})

	// Test RDWS nested type
	t.Run("RDWSInfoSubResult", func(t *testing.T) {
		info := &RDWSInfo{
			Serial: "ABC123",
			Power:  &RDWSInfoSubResult{Result: map[string]interface{}{"voltage": "12V"}},
		}
		if info.Power == nil {
			t.Errorf("RDWSInfoSubResult access failed")
		}
	})

	// Test B-Deploy nested type
	t.Run("IdleScreenColor", func(t *testing.T) {
		setup := &BDeploySetupRecord{
			IdleScreenColor: &IdleScreenColor{R: 255, G: 0, B: 0, A: 1},
		}
		if setup.IdleScreenColor.R != 255 {
			t.Errorf("IdleScreenColor access failed")
		}
	})
}
