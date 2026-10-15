package services

import (
	"context"
	"testing"

	"github.com/brightsign/bscontrol/internal/auth"
	"github.com/brightsign/bscontrol/internal/config"
	"github.com/brightsign/bscontrol/internal/http"
	"github.com/brightsign/bscontrol/internal/types"
)

// TestRDWSServiceInterface verifies that our implementation satisfies the interface
func TestRDWSServiceInterface(t *testing.T) {
	var _ RDWSService = (*rdwsService)(nil)
}

// Helper function to create a test RDWS service
func createTestRDWSService() RDWSService {
	cfg := config.DefaultConfig()
	cfg.ClientID = "test-id"
	cfg.ClientSecret = "test-secret"

	httpClient := http.NewHTTPClient(cfg)
	authManager := auth.NewAuthManager(cfg, httpClient)

	return NewRDWSService(cfg, httpClient, authManager)
}

// ============================================================================
// Information and Status Tests
// ============================================================================

func TestRDWSService_GetInfo(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.GetInfo(ctx, "")
	if err == nil {
		t.Error("Expected error when getting info with empty serial")
	}

	// Test without authentication should fail
	_, err = service.GetInfo(ctx, "ABC123DEF456")
	if err == nil {
		t.Error("Expected error when getting info without authentication")
	}
}

func TestRDWSService_GetTime(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.GetTime(ctx, "")
	if err == nil {
		t.Error("Expected error when getting time with empty serial")
	}

	// Test without authentication should fail
	_, err = service.GetTime(ctx, "ABC123DEF456")
	if err == nil {
		t.Error("Expected error when getting time without authentication")
	}
}

func TestRDWSService_SetTime(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	request := &types.RDWSTimeSetRequest{
		Time:          "12:30:45",
		Date:          "2025-11-11",
		ApplyTimezone: true,
	}

	// Test with empty serial
	_, err := service.SetTime(ctx, "", request)
	if err == nil {
		t.Error("Expected error when setting time with empty serial")
	}

	// Test with nil request
	_, err = service.SetTime(ctx, "ABC123DEF456", nil)
	if err == nil {
		t.Error("Expected error when setting time with nil request")
	}

	// Test without authentication should fail
	_, err = service.SetTime(ctx, "ABC123DEF456", request)
	if err == nil {
		t.Error("Expected error when setting time without authentication")
	}
}

func TestRDWSService_GetHealth(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.GetHealth(ctx, "")
	if err == nil {
		t.Error("Expected error when getting health with empty serial")
	}

	// Test without authentication should fail
	_, err = service.GetHealth(ctx, "ABC123DEF456")
	if err == nil {
		t.Error("Expected error when getting health without authentication")
	}
}

// ============================================================================
// File Management Tests
// ============================================================================

func TestRDWSService_ListFiles(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.ListFiles(ctx, "", "/storage/sd")
	if err == nil {
		t.Error("Expected error when listing files with empty serial")
	}

	// Test with empty path
	_, err = service.ListFiles(ctx, "ABC123DEF456", "")
	if err == nil {
		t.Error("Expected error when listing files with empty path")
	}

	// Test without authentication should fail
	_, err = service.ListFiles(ctx, "ABC123DEF456", "/storage/sd")
	if err == nil {
		t.Error("Expected error when listing files without authentication")
	}
}

func TestRDWSService_UploadFile(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.UploadFile(ctx, "", "/storage/sd", "test.txt", "content", "text/plain")
	if err == nil {
		t.Error("Expected error when uploading file with empty serial")
	}

	// Test with empty path
	_, err = service.UploadFile(ctx, "ABC123DEF456", "", "test.txt", "content", "text/plain")
	if err == nil {
		t.Error("Expected error when uploading file with empty path")
	}

	// Test with empty filename
	_, err = service.UploadFile(ctx, "ABC123DEF456", "/storage/sd", "", "content", "text/plain")
	if err == nil {
		t.Error("Expected error when uploading file with empty filename")
	}

	// Test without authentication should fail
	_, err = service.UploadFile(ctx, "ABC123DEF456", "/storage/sd", "test.txt", "content", "text/plain")
	if err == nil {
		t.Error("Expected error when uploading file without authentication")
	}
}

func TestRDWSService_CreateFolder(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.CreateFolder(ctx, "", "/storage/sd/newfolder")
	if err == nil {
		t.Error("Expected error when creating folder with empty serial")
	}

	// Test with empty path
	_, err = service.CreateFolder(ctx, "ABC123DEF456", "")
	if err == nil {
		t.Error("Expected error when creating folder with empty path")
	}

	// Test without authentication should fail
	_, err = service.CreateFolder(ctx, "ABC123DEF456", "/storage/sd/newfolder")
	if err == nil {
		t.Error("Expected error when creating folder without authentication")
	}
}

func TestRDWSService_RenameFile(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.RenameFile(ctx, "", "/storage/sd/oldfile.txt", "newfile.txt")
	if err == nil {
		t.Error("Expected error when renaming file with empty serial")
	}

	// Test with empty path
	_, err = service.RenameFile(ctx, "ABC123DEF456", "", "newfile.txt")
	if err == nil {
		t.Error("Expected error when renaming file with empty path")
	}

	// Test with empty new name
	_, err = service.RenameFile(ctx, "ABC123DEF456", "/storage/sd/oldfile.txt", "")
	if err == nil {
		t.Error("Expected error when renaming file with empty new name")
	}

	// Test without authentication should fail
	_, err = service.RenameFile(ctx, "ABC123DEF456", "/storage/sd/oldfile.txt", "newfile.txt")
	if err == nil {
		t.Error("Expected error when renaming file without authentication")
	}
}

func TestRDWSService_DeleteFile(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.DeleteFile(ctx, "", "/storage/sd/file.txt")
	if err == nil {
		t.Error("Expected error when deleting file with empty serial")
	}

	// Test with empty path
	_, err = service.DeleteFile(ctx, "ABC123DEF456", "")
	if err == nil {
		t.Error("Expected error when deleting file with empty path")
	}

	// Test without authentication should fail
	_, err = service.DeleteFile(ctx, "ABC123DEF456", "/storage/sd/file.txt")
	if err == nil {
		t.Error("Expected error when deleting file without authentication")
	}
}

// ============================================================================
// Control Tests
// ============================================================================

func TestRDWSService_GetLocalDWS(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.GetLocalDWS(ctx, "")
	if err == nil {
		t.Error("Expected error when getting local DWS with empty serial")
	}

	// Test without authentication should fail
	_, err = service.GetLocalDWS(ctx, "ABC123DEF456")
	if err == nil {
		t.Error("Expected error when getting local DWS without authentication")
	}
}

func TestRDWSService_SetLocalDWS(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.SetLocalDWS(ctx, "", true)
	if err == nil {
		t.Error("Expected error when setting local DWS with empty serial")
	}

	// Test without authentication should fail
	_, err = service.SetLocalDWS(ctx, "ABC123DEF456", true)
	if err == nil {
		t.Error("Expected error when setting local DWS without authentication")
	}
}

// ============================================================================
// Diagnostics Tests
// ============================================================================

func TestRDWSService_GetDiagnostics(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.GetDiagnostics(ctx, "")
	if err == nil {
		t.Error("Expected error when getting diagnostics with empty serial")
	}

	// Test without authentication should fail
	_, err = service.GetDiagnostics(ctx, "ABC123DEF456")
	if err == nil {
		t.Error("Expected error when getting diagnostics without authentication")
	}
}

func TestRDWSService_DNSLookup(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.DNSLookup(ctx, "", "google.com")
	if err == nil {
		t.Error("Expected error when performing DNS lookup with empty serial")
	}

	// Test with empty domain
	_, err = service.DNSLookup(ctx, "ABC123DEF456", "")
	if err == nil {
		t.Error("Expected error when performing DNS lookup with empty domain")
	}

	// Test without authentication should fail
	_, err = service.DNSLookup(ctx, "ABC123DEF456", "google.com")
	if err == nil {
		t.Error("Expected error when performing DNS lookup without authentication")
	}
}

func TestRDWSService_Ping(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.Ping(ctx, "", "8.8.8.8")
	if err == nil {
		t.Error("Expected error when pinging with empty serial")
	}

	// Test with empty host
	_, err = service.Ping(ctx, "ABC123DEF456", "")
	if err == nil {
		t.Error("Expected error when pinging with empty host")
	}

	// Test without authentication should fail
	_, err = service.Ping(ctx, "ABC123DEF456", "8.8.8.8")
	if err == nil {
		t.Error("Expected error when pinging without authentication")
	}
}

func TestRDWSService_TraceRoute(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.TraceRoute(ctx, "", "8.8.8.8")
	if err == nil {
		t.Error("Expected error when tracing route with empty serial")
	}

	// Test with empty host
	_, err = service.TraceRoute(ctx, "ABC123DEF456", "")
	if err == nil {
		t.Error("Expected error when tracing route with empty host")
	}

	// Test without authentication should fail
	_, err = service.TraceRoute(ctx, "ABC123DEF456", "8.8.8.8")
	if err == nil {
		t.Error("Expected error when tracing route without authentication")
	}
}

func TestRDWSService_GetNetworkConfig(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.GetNetworkConfig(ctx, "", "eth0")
	if err == nil {
		t.Error("Expected error when getting network config with empty serial")
	}

	// Test with empty interface
	_, err = service.GetNetworkConfig(ctx, "ABC123DEF456", "")
	if err == nil {
		t.Error("Expected error when getting network config with empty interface")
	}

	// Test without authentication should fail
	_, err = service.GetNetworkConfig(ctx, "ABC123DEF456", "eth0")
	if err == nil {
		t.Error("Expected error when getting network config without authentication")
	}
}

func TestRDWSService_SetNetworkConfig(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	request := &types.RDWSNetworkConfigSetRequest{}
	request.Data.Type = "dhcp"

	// Test with empty serial
	_, err := service.SetNetworkConfig(ctx, "", "eth0", request)
	if err == nil {
		t.Error("Expected error when setting network config with empty serial")
	}

	// Test with empty interface
	_, err = service.SetNetworkConfig(ctx, "ABC123DEF456", "", request)
	if err == nil {
		t.Error("Expected error when setting network config with empty interface")
	}

	// Test with nil request
	_, err = service.SetNetworkConfig(ctx, "ABC123DEF456", "eth0", nil)
	if err == nil {
		t.Error("Expected error when setting network config with nil request")
	}

	// Test without authentication should fail
	_, err = service.SetNetworkConfig(ctx, "ABC123DEF456", "eth0", request)
	if err == nil {
		t.Error("Expected error when setting network config without authentication")
	}
}

func TestRDWSService_GetNetworkNeighborhood(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.GetNetworkNeighborhood(ctx, "")
	if err == nil {
		t.Error("Expected error when getting network neighborhood with empty serial")
	}

	// Test without authentication should fail
	_, err = service.GetNetworkNeighborhood(ctx, "ABC123DEF456")
	if err == nil {
		t.Error("Expected error when getting network neighborhood without authentication")
	}
}

func TestRDWSService_GetPacketCaptureStatus(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.GetPacketCaptureStatus(ctx, "")
	if err == nil {
		t.Error("Expected error when getting packet capture status with empty serial")
	}

	// Test without authentication should fail
	_, err = service.GetPacketCaptureStatus(ctx, "ABC123DEF456")
	if err == nil {
		t.Error("Expected error when getting packet capture status without authentication")
	}
}

func TestRDWSService_StartPacketCapture(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	request := &types.RDWSPacketCaptureStartRequest{}
	request.Data.Interface = "eth0"
	request.Data.Duration = 60

	// Test with empty serial
	_, err := service.StartPacketCapture(ctx, "", request)
	if err == nil {
		t.Error("Expected error when starting packet capture with empty serial")
	}

	// Test with nil request
	_, err = service.StartPacketCapture(ctx, "ABC123DEF456", nil)
	if err == nil {
		t.Error("Expected error when starting packet capture with nil request")
	}

	// Test without authentication should fail
	_, err = service.StartPacketCapture(ctx, "ABC123DEF456", request)
	if err == nil {
		t.Error("Expected error when starting packet capture without authentication")
	}
}

func TestRDWSService_StopPacketCapture(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.StopPacketCapture(ctx, "")
	if err == nil {
		t.Error("Expected error when stopping packet capture with empty serial")
	}

	// Test without authentication should fail
	_, err = service.StopPacketCapture(ctx, "ABC123DEF456")
	if err == nil {
		t.Error("Expected error when stopping packet capture without authentication")
	}
}

func TestRDWSService_GetTelnetStatus(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.GetTelnetStatus(ctx, "")
	if err == nil {
		t.Error("Expected error when getting telnet status with empty serial")
	}

	// Test without authentication should fail
	_, err = service.GetTelnetStatus(ctx, "ABC123DEF456")
	if err == nil {
		t.Error("Expected error when getting telnet status without authentication")
	}
}

func TestRDWSService_SetTelnetStatus(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.SetTelnetStatus(ctx, "", true, 23)
	if err == nil {
		t.Error("Expected error when setting telnet status with empty serial")
	}

	// Test without authentication should fail
	_, err = service.SetTelnetStatus(ctx, "ABC123DEF456", true, 23)
	if err == nil {
		t.Error("Expected error when setting telnet status without authentication")
	}
}

func TestRDWSService_GetSSHStatus(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.GetSSHStatus(ctx, "")
	if err == nil {
		t.Error("Expected error when getting SSH status with empty serial")
	}

	// Test without authentication should fail
	_, err = service.GetSSHStatus(ctx, "ABC123DEF456")
	if err == nil {
		t.Error("Expected error when getting SSH status without authentication")
	}
}

func TestRDWSService_SetSSHStatus(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.SetSSHStatus(ctx, "", true, 22, "")
	if err == nil {
		t.Error("Expected error when setting SSH status with empty serial")
	}

	// Test without authentication should fail
	_, err = service.SetSSHStatus(ctx, "ABC123DEF456", true, 22, "")
	if err == nil {
		t.Error("Expected error when setting SSH status without authentication")
	}
}

// ============================================================================
// Type Structure Tests
// ============================================================================

func TestRDWSInfoStructure(t *testing.T) {
	info := types.RDWSInfo{
		Serial:         "ABC123DEF456",
		Model:          "XD1033",
		FWVersion:      "8.5.25",
		BootVersion:    "1.2.3",
		Family:         "4K",
		IsPlayer:       true,
		UpTime:         "2 days 5 hours",
		UpTimeSeconds:  187200,
		ConnectionType: "ethernet",
		BSNCE:          true,
	}

	if info.Serial != "ABC123DEF456" {
		t.Errorf("Expected Serial 'ABC123DEF456', got '%s'", info.Serial)
	}
	if info.Model != "XD1033" {
		t.Errorf("Expected Model 'XD1033', got '%s'", info.Model)
	}
	if !info.IsPlayer {
		t.Error("Expected IsPlayer to be true")
	}
	if info.UpTimeSeconds != 187200 {
		t.Errorf("Expected UpTimeSeconds 187200, got %d", info.UpTimeSeconds)
	}
}

func TestRDWSTimeInfoStructure(t *testing.T) {
	timezoneMins := -300
	timeInfo := types.RDWSTimeInfo{
		Time:         "14:30:45",
		TimezoneMin:  &timezoneMins,
		TimezoneName: "America/New_York",
		TimezoneAbbr: "EST",
		Year:         2025,
		Month:        11,
		Date:         11,
		Hour:         14,
		Minute:       30,
		Second:       45,
		Millisecond:  123,
	}

	if timeInfo.Time != "14:30:45" {
		t.Errorf("Expected Time '14:30:45', got '%s'", timeInfo.Time)
	}
	if timeInfo.Year != 2025 {
		t.Errorf("Expected Year 2025, got %d", timeInfo.Year)
	}
	if *timeInfo.TimezoneMin != -300 {
		t.Errorf("Expected TimezoneMin -300, got %d", *timeInfo.TimezoneMin)
	}
}

func TestRDWSHealthInfoStructure(t *testing.T) {
	healthInfo := types.RDWSHealthInfo{
		Status:     "active",
		StatusTime: "2025-11-11 14:30:45 EST",
	}

	if healthInfo.Status != "active" {
		t.Errorf("Expected Status 'active', got '%s'", healthInfo.Status)
	}
}

func TestRDWSDiagnosticsInfoStructure(t *testing.T) {
	diagnostics := types.RDWSDiagnosticsInfo{
		Gateway:             "192.168.1.1",
		DNS:                 []string{"8.8.8.8", "8.8.4.4"},
		ConnectedToRouter:   true,
		ConnectedToInternet: true,
		ExternalIPAddress:   "203.0.113.42",
	}

	if diagnostics.Gateway != "192.168.1.1" {
		t.Errorf("Expected Gateway '192.168.1.1', got '%s'", diagnostics.Gateway)
	}
	if len(diagnostics.DNS) != 2 {
		t.Errorf("Expected 2 DNS servers, got %d", len(diagnostics.DNS))
	}
	if !diagnostics.ConnectedToInternet {
		t.Error("Expected ConnectedToInternet to be true")
	}
}

func TestRDWSPingResultStructure(t *testing.T) {
	pingResult := types.RDWSPingResult{
		Success:    true,
		Host:       "8.8.8.8",
		PacketLoss: 0.0,
		MinRTT:     12.5,
		MaxRTT:     15.8,
		AvgRTT:     14.2,
	}

	if !pingResult.Success {
		t.Error("Expected Success to be true")
	}
	if pingResult.Host != "8.8.8.8" {
		t.Errorf("Expected Host '8.8.8.8', got '%s'", pingResult.Host)
	}
	if pingResult.AvgRTT != 14.2 {
		t.Errorf("Expected AvgRTT 14.2, got %f", pingResult.AvgRTT)
	}
}

func TestRDWSNetworkConfigStructure(t *testing.T) {
	networkConfig := types.RDWSNetworkConfig{
		Interface:  "eth0",
		Type:       "static",
		IPAddress:  "192.168.1.100",
		Netmask:    "255.255.255.0",
		Gateway:    "192.168.1.1",
		DNS:        []string{"8.8.8.8"},
		MACAddress: "00:11:22:33:44:55",
		LinkStatus: "up",
	}

	if networkConfig.Interface != "eth0" {
		t.Errorf("Expected Interface 'eth0', got '%s'", networkConfig.Interface)
	}
	if networkConfig.Type != "static" {
		t.Errorf("Expected Type 'static', got '%s'", networkConfig.Type)
	}
	if networkConfig.IPAddress != "192.168.1.100" {
		t.Errorf("Expected IPAddress '192.168.1.100', got '%s'", networkConfig.IPAddress)
	}
}

func TestRDWSPacketCaptureStatusStructure(t *testing.T) {
	captureStatus := types.RDWSPacketCaptureStatus{
		Running:     true,
		Interface:   "eth0",
		Duration:    60,
		FilePath:    "/storage/sd/capture.pcap",
		StartTime:   "2025-11-11 14:30:00",
		ElapsedTime: 30,
	}

	if !captureStatus.Running {
		t.Error("Expected Running to be true")
	}
	if captureStatus.Interface != "eth0" {
		t.Errorf("Expected Interface 'eth0', got '%s'", captureStatus.Interface)
	}
	if captureStatus.Duration != 60 {
		t.Errorf("Expected Duration 60, got %d", captureStatus.Duration)
	}
}

func TestRDWSTelnetInfoStructure(t *testing.T) {
	telnetInfo := types.RDWSTelnetInfo{
		Enabled: true,
		Port:    23,
	}

	if !telnetInfo.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if telnetInfo.Port != 23 {
		t.Errorf("Expected Port 23, got %d", telnetInfo.Port)
	}
}

func TestRDWSSSHInfoStructure(t *testing.T) {
	sshInfo := types.RDWSSSHInfo{
		Enabled: true,
		Port:    22,
	}

	if !sshInfo.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if sshInfo.Port != 22 {
		t.Errorf("Expected Port 22, got %d", sshInfo.Port)
	}
}

// ============================================================================
// Storage Management Tests
// ============================================================================

func TestRDWSService_ReformatStorage(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.ReformatStorage(ctx, "", "sd")
	if err == nil {
		t.Error("Expected error when reformatting storage with empty serial")
	}

	// Test with empty device name
	_, err = service.ReformatStorage(ctx, "ABC123DEF456", "")
	if err == nil {
		t.Error("Expected error when reformatting storage with empty device name")
	}

	// Test without authentication should fail
	_, err = service.ReformatStorage(ctx, "ABC123DEF456", "sd")
	if err == nil {
		t.Error("Expected error when reformatting storage without authentication")
	}
}

func TestRDWSStorageReformatResponseStructure(t *testing.T) {
	response := types.RDWSStorageReformatResponse{
		Route:  "/storage/sd",
		Method: "DELETE",
	}
	response.Data.Result.Success = true
	response.Data.Result.Message = "Storage device reformatted successfully"

	if response.Route != "/storage/sd" {
		t.Errorf("Expected Route '/storage/sd', got '%s'", response.Route)
	}
	if response.Method != "DELETE" {
		t.Errorf("Expected Method 'DELETE', got '%s'", response.Method)
	}
	if !response.Data.Result.Success {
		t.Error("Expected Success to be true")
	}
	if response.Data.Result.Message != "Storage device reformatted successfully" {
		t.Errorf("Expected success message, got '%s'", response.Data.Result.Message)
	}
}

// ============================================================================
// Custom Commands Tests
// ============================================================================

func TestRDWSService_SendCustomData(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.SendCustomData(ctx, "", "test data")
	if err == nil {
		t.Error("Expected error when sending custom data with empty serial")
	}

	// Test with empty data
	_, err = service.SendCustomData(ctx, "ABC123DEF456", "")
	if err == nil {
		t.Error("Expected error when sending custom data with empty data")
	}

	// Test without authentication should fail
	_, err = service.SendCustomData(ctx, "ABC123DEF456", "test data")
	if err == nil {
		t.Error("Expected error when sending custom data without authentication")
	}
}

func TestRDWSCustomDataRequestStructure(t *testing.T) {
	request := types.RDWSCustomDataRequest{}
	request.Data.Data = "custom command string"

	if request.Data.Data != "custom command string" {
		t.Errorf("Expected Data 'custom command string', got '%s'", request.Data.Data)
	}
}

func TestRDWSCustomDataResponseStructure(t *testing.T) {
	response := types.RDWSCustomDataResponse{
		Route:  "/custom",
		Method: "PUT",
	}
	response.Data.Result.Success = true
	response.Data.Result.Message = "Custom data sent successfully"

	if response.Route != "/custom" {
		t.Errorf("Expected Route '/custom', got '%s'", response.Route)
	}
	if response.Method != "PUT" {
		t.Errorf("Expected Method 'PUT', got '%s'", response.Method)
	}
	if !response.Data.Result.Success {
		t.Error("Expected Success to be true")
	}
	if response.Data.Result.Message != "Custom data sent successfully" {
		t.Errorf("Expected success message, got '%s'", response.Data.Result.Message)
	}
}

// ============================================================================
// Firmware Management Tests
// ============================================================================

func TestRDWSService_DownloadFirmware(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	firmwareURL := "https://example.com/firmware/brightsign-8.5.25.bsfw"

	// Test with empty serial
	_, err := service.DownloadFirmware(ctx, "", firmwareURL, nil)
	if err == nil {
		t.Error("Expected error when downloading firmware with empty serial")
	}

	// Test with empty firmware URL
	_, err = service.DownloadFirmware(ctx, "ABC123DEF456", "", nil)
	if err == nil {
		t.Error("Expected error when downloading firmware with empty URL")
	}

	// Test without authentication should fail
	_, err = service.DownloadFirmware(ctx, "ABC123DEF456", firmwareURL, nil)
	if err == nil {
		t.Error("Expected error when downloading firmware without authentication")
	}
}

func TestRDWSFirmwareDownloadRequestStructure(t *testing.T) {
	request := types.RDWSFirmwareDownloadRequest{}
	request.Data.URL = "https://example.com/firmware/brightsign-8.5.25.bsfw"

	if request.Data.URL != "https://example.com/firmware/brightsign-8.5.25.bsfw" {
		t.Errorf("Expected URL 'https://example.com/firmware/brightsign-8.5.25.bsfw', got '%s'", request.Data.URL)
	}
}

func TestRDWSFirmwareDownloadResponseStructure(t *testing.T) {
	response := types.RDWSFirmwareDownloadResponse{
		Route:  "/download-firmware",
		Method: "GET",
	}
	response.Data.Result.Success = true
	response.Data.Result.Message = "Firmware download initiated"

	if response.Route != "/download-firmware" {
		t.Errorf("Expected Route '/download-firmware', got '%s'", response.Route)
	}
	if response.Method != "GET" {
		t.Errorf("Expected Method 'GET', got '%s'", response.Method)
	}
	if !response.Data.Result.Success {
		t.Error("Expected Success to be true")
	}
	if response.Data.Result.Message != "Firmware download initiated" {
		t.Errorf("Expected success message, got '%s'", response.Data.Result.Message)
	}
}

// ============================================================================
// Registry Management Tests
// ============================================================================

func TestRDWSService_GetRegistry(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.GetRegistry(ctx, "")
	if err == nil {
		t.Error("Expected error when getting registry with empty serial")
	}

	// Test without authentication should fail
	_, err = service.GetRegistry(ctx, "ABC123DEF456")
	if err == nil {
		t.Error("Expected error when getting registry without authentication")
	}
}

func TestRDWSService_GetRegistryValue(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.GetRegistryValue(ctx, "", "networking", "hostname")
	if err == nil {
		t.Error("Expected error when getting registry value with empty serial")
	}

	// Test with empty section
	_, err = service.GetRegistryValue(ctx, "ABC123DEF456", "", "hostname")
	if err == nil {
		t.Error("Expected error when getting registry value with empty section")
	}

	// Test with empty key
	_, err = service.GetRegistryValue(ctx, "ABC123DEF456", "networking", "")
	if err == nil {
		t.Error("Expected error when getting registry value with empty key")
	}

	// Test without authentication should fail
	_, err = service.GetRegistryValue(ctx, "ABC123DEF456", "networking", "hostname")
	if err == nil {
		t.Error("Expected error when getting registry value without authentication")
	}
}

func TestRDWSService_SetRegistryValue(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.SetRegistryValue(ctx, "", "networking", "hostname", "player1")
	if err == nil {
		t.Error("Expected error when setting registry value with empty serial")
	}

	// Test with empty section
	_, err = service.SetRegistryValue(ctx, "ABC123DEF456", "", "hostname", "player1")
	if err == nil {
		t.Error("Expected error when setting registry value with empty section")
	}

	// Test with empty key
	_, err = service.SetRegistryValue(ctx, "ABC123DEF456", "networking", "", "player1")
	if err == nil {
		t.Error("Expected error when setting registry value with empty key")
	}

	// Test without authentication should fail
	_, err = service.SetRegistryValue(ctx, "ABC123DEF456", "networking", "hostname", "player1")
	if err == nil {
		t.Error("Expected error when setting registry value without authentication")
	}
}

func TestRDWSService_DeleteRegistryValue(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.DeleteRegistryValue(ctx, "", "networking", "hostname")
	if err == nil {
		t.Error("Expected error when deleting registry value with empty serial")
	}

	// Test with empty section
	_, err = service.DeleteRegistryValue(ctx, "ABC123DEF456", "", "hostname")
	if err == nil {
		t.Error("Expected error when deleting registry value with empty section")
	}

	// Test with empty key
	_, err = service.DeleteRegistryValue(ctx, "ABC123DEF456", "networking", "")
	if err == nil {
		t.Error("Expected error when deleting registry value with empty key")
	}

	// Test without authentication should fail
	_, err = service.DeleteRegistryValue(ctx, "ABC123DEF456", "networking", "hostname")
	if err == nil {
		t.Error("Expected error when deleting registry value without authentication")
	}
}

func TestRDWSService_FlushRegistry(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.FlushRegistry(ctx, "")
	if err == nil {
		t.Error("Expected error when flushing registry with empty serial")
	}

	// Test without authentication should fail
	_, err = service.FlushRegistry(ctx, "ABC123DEF456")
	if err == nil {
		t.Error("Expected error when flushing registry without authentication")
	}
}

func TestRDWSService_GetRecoveryURL(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.GetRecoveryURL(ctx, "")
	if err == nil {
		t.Error("Expected error when getting recovery URL with empty serial")
	}

	// Test without authentication should fail
	_, err = service.GetRecoveryURL(ctx, "ABC123DEF456")
	if err == nil {
		t.Error("Expected error when getting recovery URL without authentication")
	}
}

func TestRDWSService_SetRecoveryURL(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	recoveryURL := "https://example.com/recovery"

	// Test with empty serial
	_, err := service.SetRecoveryURL(ctx, "", recoveryURL)
	if err == nil {
		t.Error("Expected error when setting recovery URL with empty serial")
	}

	// Test with empty recovery URL
	_, err = service.SetRecoveryURL(ctx, "ABC123DEF456", "")
	if err == nil {
		t.Error("Expected error when setting recovery URL with empty URL")
	}

	// Test without authentication should fail
	_, err = service.SetRecoveryURL(ctx, "ABC123DEF456", recoveryURL)
	if err == nil {
		t.Error("Expected error when setting recovery URL without authentication")
	}
}

func TestRDWSRegistryStructure(t *testing.T) {
	registry := types.RDWSRegistry{
		Sections: map[string]map[string]string{
			"networking": {
				"hostname": "player1",
				"domain":   "example.com",
			},
			"system": {
				"timezone": "America/New_York",
			},
		},
	}

	if len(registry.Sections) != 2 {
		t.Errorf("Expected 2 sections, got %d", len(registry.Sections))
	}

	if registry.Sections["networking"]["hostname"] != "player1" {
		t.Errorf("Expected hostname 'player1', got '%s'", registry.Sections["networking"]["hostname"])
	}
}

func TestRDWSRegistryValueStructure(t *testing.T) {
	regValue := types.RDWSRegistryValue{
		Section: "networking",
		Key:     "hostname",
		Value:   "player1",
	}

	if regValue.Section != "networking" {
		t.Errorf("Expected Section 'networking', got '%s'", regValue.Section)
	}
	if regValue.Key != "hostname" {
		t.Errorf("Expected Key 'hostname', got '%s'", regValue.Key)
	}
	if regValue.Value != "player1" {
		t.Errorf("Expected Value 'player1', got '%s'", regValue.Value)
	}
}

func TestRDWSRecoveryURLStructure(t *testing.T) {
	recoveryURL := types.RDWSRecoveryURL{
		URL: "https://example.com/recovery",
	}

	if recoveryURL.URL != "https://example.com/recovery" {
		t.Errorf("Expected URL 'https://example.com/recovery', got '%s'", recoveryURL.URL)
	}
}

// Tests for logs operations
func TestRDWSService_GetLogs(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.GetLogs(ctx, "")
	if err == nil {
		t.Error("Expected error when getting logs with empty serial")
	}

	// Test without authentication should fail
	_, err = service.GetLogs(ctx, "ABC123DEF456")
	if err == nil {
		t.Error("Expected error when getting logs without authentication")
	}
}

func TestRDWSLogsStructure(t *testing.T) {
	logs := types.RDWSLogs{
		Files: []types.RDWSLogFile{
			{
				Name:    "system.log",
				Size:    1024,
				Content: "Log content here",
			},
			{
				Name:    "debug.log",
				Size:    2048,
				Content: "Debug information",
			},
		},
	}

	if len(logs.Files) != 2 {
		t.Errorf("Expected 2 log files, got %d", len(logs.Files))
	}

	if logs.Files[0].Name != "system.log" {
		t.Errorf("Expected first log name 'system.log', got '%s'", logs.Files[0].Name)
	}

	if logs.Files[0].Size != 1024 {
		t.Errorf("Expected first log size 1024, got %d", logs.Files[0].Size)
	}

	if logs.Files[1].Name != "debug.log" {
		t.Errorf("Expected second log name 'debug.log', got '%s'", logs.Files[1].Name)
	}
}

// Tests for crash dump operations
func TestRDWSService_GetCrashDump(t *testing.T) {
	service := createTestRDWSService()
	ctx := context.Background()

	// Test with empty serial
	_, err := service.GetCrashDump(ctx, "")
	if err == nil {
		t.Error("Expected error when getting crash dump with empty serial")
	}

	// Test without authentication should fail
	_, err = service.GetCrashDump(ctx, "ABC123DEF456")
	if err == nil {
		t.Error("Expected error when getting crash dump without authentication")
	}
}

func TestRDWSCrashDumpStructure(t *testing.T) {
	crashDump := types.RDWSCrashDump{
		Files: []types.RDWSCrashDumpFile{
			{
				Name:      "crash_2024-01-15.dmp",
				Timestamp: "2024-01-15T10:30:00Z",
				Size:      5120,
				Content:   "Crash dump data",
			},
			{
				Name:      "crash_2024-01-16.dmp",
				Timestamp: "2024-01-16T14:45:00Z",
				Size:      3072,
				Content:   "Another crash dump",
			},
		},
	}

	if len(crashDump.Files) != 2 {
		t.Errorf("Expected 2 crash dump files, got %d", len(crashDump.Files))
	}

	if crashDump.Files[0].Name != "crash_2024-01-15.dmp" {
		t.Errorf("Expected first crash dump name 'crash_2024-01-15.dmp', got '%s'", crashDump.Files[0].Name)
	}

	if crashDump.Files[0].Timestamp != "2024-01-15T10:30:00Z" {
		t.Errorf("Expected first crash dump timestamp '2024-01-15T10:30:00Z', got '%s'", crashDump.Files[0].Timestamp)
	}

	if crashDump.Files[0].Size != 5120 {
		t.Errorf("Expected first crash dump size 5120, got %d", crashDump.Files[0].Size)
	}

	if crashDump.Files[1].Name != "crash_2024-01-16.dmp" {
		t.Errorf("Expected second crash dump name 'crash_2024-01-16.dmp', got '%s'", crashDump.Files[1].Name)
	}
}
