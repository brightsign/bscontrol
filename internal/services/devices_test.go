package services

import (
	"context"
	"testing"
	"time"

	"github.com/brightsign/bscontrol/internal/auth"
	"github.com/brightsign/bscontrol/internal/config"
	"github.com/brightsign/bscontrol/internal/http"
	"github.com/brightsign/bscontrol/internal/types"
)

func TestDeviceService_List(t *testing.T) {
	// Create test client
	cfg := config.DefaultConfig()
	cfg.ClientID = "test-id"
	cfg.ClientSecret = "test-secret"

	httpClient := http.NewHTTPClient(cfg)
	authManager := auth.NewAuthManager(cfg, httpClient)

	deviceService := NewDeviceService(cfg, httpClient, authManager)

	ctx := context.Background()

	// Test without authentication should fail
	_, err := deviceService.List(ctx)
	if err == nil {
		t.Error("Expected error when listing devices without authentication")
	}
}

func TestDeviceService_Get(t *testing.T) {
	// Create test client
	cfg := config.DefaultConfig()
	cfg.ClientID = "test-id"
	cfg.ClientSecret = "test-secret"

	httpClient := http.NewHTTPClient(cfg)
	authManager := auth.NewAuthManager(cfg, httpClient)

	deviceService := NewDeviceService(cfg, httpClient, authManager)

	ctx := context.Background()

	// Test with empty serial
	_, err := deviceService.Get(ctx, "")
	if err == nil {
		t.Error("Expected error when getting device with empty serial")
	}

	// Test without authentication should fail
	_, err = deviceService.Get(ctx, "test-serial")
	if err == nil {
		t.Error("Expected error when getting device without authentication")
	}
}

func TestDeviceService_GetByID(t *testing.T) {
	// Create test client
	cfg := config.DefaultConfig()
	cfg.ClientID = "test-id"
	cfg.ClientSecret = "test-secret"

	httpClient := http.NewHTTPClient(cfg)
	authManager := auth.NewAuthManager(cfg, httpClient)

	deviceService := NewDeviceService(cfg, httpClient, authManager)

	ctx := context.Background()

	// Test with invalid ID
	_, err := deviceService.GetByID(ctx, 0)
	if err == nil {
		t.Error("Expected error when getting device with invalid ID")
	}

	_, err = deviceService.GetByID(ctx, -1)
	if err == nil {
		t.Error("Expected error when getting device with negative ID")
	}

	// Test without authentication should fail
	_, err = deviceService.GetByID(ctx, 123)
	if err == nil {
		t.Error("Expected error when getting device without authentication")
	}
}

func TestListOptions(t *testing.T) {
	config := &listConfig{}

	// Test WithPageSize
	opt := WithPageSize(50)
	opt.apply(config)
	if config.pageSize != 50 {
		t.Errorf("Expected pageSize 50, got %d", config.pageSize)
	}

	// Test WithMarker
	opt = WithMarker("test-marker")
	opt.apply(config)
	if config.marker != "test-marker" {
		t.Errorf("Expected marker 'test-marker', got '%s'", config.marker)
	}

	// Test WithFilter
	opt = WithFilter("model=XD1033")
	opt.apply(config)
	if config.filter != "model=XD1033" {
		t.Errorf("Expected filter 'model=XD1033', got '%s'", config.filter)
	}

	// Test WithSort
	opt = WithSort("registrationDate")
	opt.apply(config)
	if config.sort != "registrationDate" {
		t.Errorf("Expected sort 'registrationDate', got '%s'", config.sort)
	}
}

func TestMultipleListOptions(t *testing.T) {
	config := &listConfig{}

	// Apply multiple options
	opts := []ListOption{
		WithPageSize(25),
		WithFilter("family=4K"),
		WithSort("model"),
		WithMarker("next-page"),
	}

	for _, opt := range opts {
		opt.apply(config)
	}

	if config.pageSize != 25 {
		t.Errorf("Expected pageSize 25, got %d", config.pageSize)
	}
	if config.filter != "family=4K" {
		t.Errorf("Expected filter 'family=4K', got '%s'", config.filter)
	}
	if config.sort != "model" {
		t.Errorf("Expected sort 'model', got '%s'", config.sort)
	}
	if config.marker != "next-page" {
		t.Errorf("Expected marker 'next-page', got '%s'", config.marker)
	}
}

func TestDeviceServiceInterface(t *testing.T) {
	// Test that our implementation satisfies the interface
	var _ DeviceService = (*deviceService)(nil)
}

// Mock types for testing data structures
func TestDeviceStructure(t *testing.T) {
	// Test that Device structure can hold expected data
	device := types.Device{
		ID:               123,
		Serial:           "ABC123DEF456",
		Model:            "XD1033",
		Family:           "4K",
		RegistrationDate: time.Now(),
		LastModifiedDate: time.Now(),
		Settings: &types.DeviceSettings{
			Name:        "Conference Room Display",
			Description: "Main display for conference room",
			SetupType:   "Simple",
			Timezone:    "America/New_York",
			Group: &types.Group{
				ID:   1,
				Name: "Conference Room Devices",
			},
		},
	}

	if device.ID != 123 {
		t.Errorf("Expected ID 123, got %d", device.ID)
	}
	if device.Serial != "ABC123DEF456" {
		t.Errorf("Expected serial 'ABC123DEF456', got '%s'", device.Serial)
	}
	if device.Settings == nil {
		t.Error("Expected device settings to be present")
	}
	if device.Settings != nil && device.Settings.Group == nil {
		t.Error("Expected device group to be present")
	}
}

func TestDeviceListStructure(t *testing.T) {
	// Test DeviceList structure
	deviceList := types.DeviceList{
		Items: []types.Device{
			{
				ID:     1,
				Serial: "DEV001",
				Model:  "XD1033",
			},
			{
				ID:     2,
				Serial: "DEV002",
				Model:  "HD1023",
			},
		},
		IsTruncated: true,
		NextMarker:  "marker123",
		TotalCount:  100,
	}

	if len(deviceList.Items) != 2 {
		t.Errorf("Expected 2 devices, got %d", len(deviceList.Items))
	}
	if !deviceList.IsTruncated {
		t.Error("Expected list to be truncated")
	}
	if deviceList.NextMarker != "marker123" {
		t.Errorf("Expected NextMarker 'marker123', got '%s'", deviceList.NextMarker)
	}
	if deviceList.TotalCount != 100 {
		t.Errorf("Expected TotalCount 100, got %d", deviceList.TotalCount)
	}
}

func TestDeviceService_GetStatus(t *testing.T) {
	// Create test client
	cfg := config.DefaultConfig()
	cfg.ClientID = "test-id"
	cfg.ClientSecret = "test-secret"

	httpClient := http.NewHTTPClient(cfg)
	authManager := auth.NewAuthManager(cfg, httpClient)

	deviceService := NewDeviceService(cfg, httpClient, authManager)

	ctx := context.Background()

	// Test with invalid ID
	_, err := deviceService.GetStatus(ctx, 0)
	if err == nil {
		t.Error("Expected error when getting status with invalid ID")
	}

	_, err = deviceService.GetStatus(ctx, -1)
	if err == nil {
		t.Error("Expected error when getting status with negative ID")
	}

	// Test without authentication should fail
	_, err = deviceService.GetStatus(ctx, 123)
	if err == nil {
		t.Error("Expected error when getting status without authentication")
	}
}

func TestDeviceService_GetStatusBySerial(t *testing.T) {
	// Create test client
	cfg := config.DefaultConfig()
	cfg.ClientID = "test-id"
	cfg.ClientSecret = "test-secret"

	httpClient := http.NewHTTPClient(cfg)
	authManager := auth.NewAuthManager(cfg, httpClient)

	deviceService := NewDeviceService(cfg, httpClient, authManager)

	ctx := context.Background()

	// Test with empty serial
	_, err := deviceService.GetStatusBySerial(ctx, "")
	if err == nil {
		t.Error("Expected error when getting status with empty serial")
	}

	// Test without authentication should fail (this will fail at device lookup)
	_, err = deviceService.GetStatusBySerial(ctx, "test-serial")
	if err == nil {
		t.Error("Expected error when getting status without authentication")
	}
}

func TestDeviceService_GetErrors(t *testing.T) {
	// Create test client
	cfg := config.DefaultConfig()
	cfg.ClientID = "test-id"
	cfg.ClientSecret = "test-secret"

	httpClient := http.NewHTTPClient(cfg)
	authManager := auth.NewAuthManager(cfg, httpClient)

	deviceService := NewDeviceService(cfg, httpClient, authManager)

	ctx := context.Background()

	// Test with invalid ID
	_, err := deviceService.GetErrors(ctx, 0)
	if err == nil {
		t.Error("Expected error when getting errors with invalid ID")
	}

	_, err = deviceService.GetErrors(ctx, -1)
	if err == nil {
		t.Error("Expected error when getting errors with negative ID")
	}

	// Test without authentication should fail
	_, err = deviceService.GetErrors(ctx, 123)
	if err == nil {
		t.Error("Expected error when getting errors without authentication")
	}

	// Test with options
	_, err = deviceService.GetErrors(ctx, 123,
		WithPageSize(10),
		WithFilter("severity=critical"),
		WithSort("timestamp"),
	)
	if err == nil {
		t.Error("Expected error when getting errors without authentication")
	}
}

func TestDeviceService_GetErrorsBySerial(t *testing.T) {
	// Create test client
	cfg := config.DefaultConfig()
	cfg.ClientID = "test-id"
	cfg.ClientSecret = "test-secret"

	httpClient := http.NewHTTPClient(cfg)
	authManager := auth.NewAuthManager(cfg, httpClient)

	deviceService := NewDeviceService(cfg, httpClient, authManager)

	ctx := context.Background()

	// Test with empty serial
	_, err := deviceService.GetErrorsBySerial(ctx, "")
	if err == nil {
		t.Error("Expected error when getting errors with empty serial")
	}

	// Test without authentication should fail (this will fail at device lookup)
	_, err = deviceService.GetErrorsBySerial(ctx, "test-serial")
	if err == nil {
		t.Error("Expected error when getting errors without authentication")
	}
}

func TestDeviceStatusStructure(t *testing.T) {
	// Test DeviceStatus structure
	status := types.DeviceStatus{
		DeviceID:        "dev123",
		Serial:          "ABC123DEF456",
		Model:           "XD1033",
		FirmwareVersion: "8.5.25",
		IsOnline:        true,
		LastSeen:        time.Now(),
		Status:          "Running",
		Uptime:          3600,
		UptimeDisplay:   "1 hour",
		HealthStatus:    "Healthy",
		LastHealthCheck: time.Now(),
		IPAddress:       "192.168.1.100",
		ConnectionType:  "ethernet",
		SignalStrength:  0,
	}

	if status.DeviceID != "dev123" {
		t.Errorf("Expected DeviceID 'dev123', got '%s'", status.DeviceID)
	}
	if status.Serial != "ABC123DEF456" {
		t.Errorf("Expected Serial 'ABC123DEF456', got '%s'", status.Serial)
	}
	if !status.IsOnline {
		t.Error("Expected device to be online")
	}
	if status.Uptime != 3600 {
		t.Errorf("Expected Uptime 3600, got %d", status.Uptime)
	}
	if status.ConnectionType != "ethernet" {
		t.Errorf("Expected ConnectionType 'ethernet', got '%s'", status.ConnectionType)
	}
}

func TestDeviceErrorStructure(t *testing.T) {
	// Test DeviceError structure
	resolvedTime := time.Now()
	deviceError := types.DeviceError{
		ID:         1,
		DeviceID:   "dev123",
		Serial:     "ABC123DEF456",
		ErrorCode:  "E001",
		ErrorType:  "system",
		Severity:   "critical",
		Message:    "System overheating",
		Details:    "CPU temperature exceeded 85°C",
		Timestamp:  time.Now(),
		Source:     "thermal_monitor",
		Resolved:   true,
		ResolvedAt: &resolvedTime,
	}

	if deviceError.ID != 1 {
		t.Errorf("Expected ID 1, got %d", deviceError.ID)
	}
	if deviceError.ErrorType != "system" {
		t.Errorf("Expected ErrorType 'system', got '%s'", deviceError.ErrorType)
	}
	if deviceError.Severity != "critical" {
		t.Errorf("Expected Severity 'critical', got '%s'", deviceError.Severity)
	}
	if !deviceError.Resolved {
		t.Error("Expected error to be resolved")
	}
	if deviceError.ResolvedAt == nil {
		t.Error("Expected ResolvedAt to be set")
	}
}

func TestDeviceErrorListStructure(t *testing.T) {
	// Test DeviceErrorList structure
	errorList := types.DeviceErrorList{
		Items: []types.DeviceError{
			{
				ID:        1,
				ErrorCode: "E001",
				Message:   "Error 1",
				Severity:  "warning",
			},
			{
				ID:        2,
				ErrorCode: "E002",
				Message:   "Error 2",
				Severity:  "critical",
			},
		},
		IsTruncated: false,
		NextMarker:  "",
		TotalCount:  2,
	}

	if len(errorList.Items) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errorList.Items))
	}
	if errorList.IsTruncated {
		t.Error("Expected list not to be truncated")
	}
	if errorList.TotalCount != 2 {
		t.Errorf("Expected TotalCount 2, got %d", errorList.TotalCount)
	}
}
