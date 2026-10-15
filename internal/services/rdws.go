package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/brightsign/bscontrol/internal/auth"
	"github.com/brightsign/bscontrol/internal/config"
	"github.com/brightsign/bscontrol/internal/errors"
	"github.com/brightsign/bscontrol/internal/http"
	"github.com/brightsign/bscontrol/internal/types"
)

// rDWS URLs interpolate caller-supplied values (serial, host, path, registry
// key, ...) into both the query string and the path. These helpers percent-
// encode those values so a crafted value cannot inject extra query parameters
// (for example a second destinationName that redirects the request to a
// different player) or break out of its path segment. For well-formed inputs
// the encoding is a no-op, so the wire format is unchanged.

// escQuery encodes a value placed in the URL query string.
func escQuery(value string) string { return url.QueryEscape(value) }

// escSegment encodes a value placed in a single URL path segment (slashes in
// the value are themselves encoded, since a single segment has none).
func escSegment(value string) string { return url.PathEscape(value) }

// escPath encodes a multi-segment device path, percent-encoding each segment
// while preserving the slash separators and any leading slash.
func escPath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

// RDWSService provides remote Diagnostic Web Server (rDWS) operations.
// These operations query player information, time, health status, manage files, and perform diagnostics remotely.
type RDWSService interface {
	// Information and Status
	GetInfo(ctx context.Context, serial string) (*types.RDWSInfo, error)
	GetTime(ctx context.Context, serial string) (*types.RDWSTimeInfo, error)
	SetTime(ctx context.Context, serial string, request *types.RDWSTimeSetRequest) (bool, error)
	GetHealth(ctx context.Context, serial string) (*types.RDWSHealthInfo, error)

	// File Management
	ListFiles(ctx context.Context, serial string, path string) (*types.RDWSFileListResponse, error)
	UploadFile(ctx context.Context, serial string, path string, fileName string, fileContents string, fileType string) (bool, error)
	CreateFolder(ctx context.Context, serial string, path string) (bool, error)
	RenameFile(ctx context.Context, serial string, path string, newName string) (bool, error)
	DeleteFile(ctx context.Context, serial string, path string) (bool, error)

	// Control
	GetLocalDWS(ctx context.Context, serial string) (*types.RDWSLocalDWSInfo, error)
	SetLocalDWS(ctx context.Context, serial string, enabled bool) (bool, error)

	// Diagnostics
	GetDiagnostics(ctx context.Context, serial string) (*types.RDWSDiagnosticsInfo, error)
	DNSLookup(ctx context.Context, serial string, domain string) (*types.RDWSDNSLookupResult, error)
	Ping(ctx context.Context, serial string, host string) (*types.RDWSPingResult, error)
	TraceRoute(ctx context.Context, serial string, host string) (*types.RDWSTraceRouteResult, error)
	GetNetworkConfig(ctx context.Context, serial string, iface string) (*types.RDWSNetworkConfig, error)
	SetNetworkConfig(ctx context.Context, serial string, iface string, request *types.RDWSNetworkConfigSetRequest) (bool, error)
	GetNetworkNeighborhood(ctx context.Context, serial string) (*types.RDWSNetworkNeighborhoodResult, error)
	GetPacketCaptureStatus(ctx context.Context, serial string) (*types.RDWSPacketCaptureStatus, error)
	StartPacketCapture(ctx context.Context, serial string, request *types.RDWSPacketCaptureStartRequest) (string, error)
	StopPacketCapture(ctx context.Context, serial string) (string, error)
	GetTelnetStatus(ctx context.Context, serial string) (*types.RDWSTelnetInfo, error)
	SetTelnetStatus(ctx context.Context, serial string, enabled bool, port int) (bool, error)
	GetSSHStatus(ctx context.Context, serial string) (*types.RDWSSSHInfo, error)
	SetSSHStatus(ctx context.Context, serial string, enabled bool, port int, password string) (bool, error)

	// Storage Management
	ReformatStorage(ctx context.Context, serial string, deviceName string) (bool, error)

	// Custom Commands
	SendCustomData(ctx context.Context, serial string, data string) (bool, error)

	// Firmware Management
	DownloadFirmware(ctx context.Context, serial string, firmwareURL string, autoReboot *bool) (bool, error)

	// Registry Management
	GetRegistry(ctx context.Context, serial string) (*types.RDWSRegistry, error)
	GetRegistryValue(ctx context.Context, serial string, section string, key string) (*types.RDWSRegistryValue, error)
	SetRegistryValue(ctx context.Context, serial string, section string, key string, value string) (bool, error)
	DeleteRegistryValue(ctx context.Context, serial string, section string, key string) (bool, error)
	FlushRegistry(ctx context.Context, serial string) (bool, error)
	GetRecoveryURL(ctx context.Context, serial string) (*types.RDWSRecoveryURL, error)
	SetRecoveryURL(ctx context.Context, serial string, recoveryURL string) (bool, error)

	// Logs and Diagnostics
	GetLogs(ctx context.Context, serial string) (*types.RDWSLogs, error)
	GetCrashDump(ctx context.Context, serial string) (*types.RDWSCrashDump, error)
	GetCrashDumpFiles(ctx context.Context, serial string) ([]types.RDWSCrashDumpListEntry, error)

	// Supervisor Update and System Inspection
	TriggerUpdateSync(ctx context.Context, serial string) (*types.RDWSUpdateSyncResult, error)
	GetStoredSupervisors(ctx context.Context, serial string) (*types.RDWSStoredSupervisors, error)
	DeleteSupervisors(ctx context.Context, serial string, request *types.RDWSDeleteSupervisorsRequest) (bool, error)
	// GetSystemInfo is EXPERIMENTAL — see its implementation for why.
	GetSystemInfo(ctx context.Context, serial string) (*types.RDWSSystemInfo, error)
}

// rdwsService implements the RDWSService interface.
type rdwsService struct {
	config      *config.Config
	httpClient  *http.HTTPClient
	authManager *auth.AuthManager
}

// NewRDWSService creates a new rDWS service.
func NewRDWSService(cfg *config.Config, httpClient *http.HTTPClient, authManager *auth.AuthManager) RDWSService {
	return &rdwsService{
		config:      cfg,
		httpClient:  httpClient,
		authManager: authManager,
	}
}

// GetInfo retrieves general information about a player via rDWS.
// This includes hardware details, network configuration, firmware version, and more.
func (s *rdwsService) GetInfo(ctx context.Context, serial string) (*types.RDWSInfo, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Build the rDWS info endpoint URL
	infoURL := fmt.Sprintf("%s/info/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escQuery(serial))

	// Make the API request
	var response types.RDWSInfoResponse
	err = s.httpClient.GetWithAuth(ctx, token, infoURL, &response)
	if err != nil {
		return nil, errors.NewAPIError(0, "rdws_info_failed",
			fmt.Sprintf("Failed to get info for device with serial '%s'", serial), err.Error())
	}

	return &response.Data.Result, nil
}

// GetTime retrieves the current date and time configured on a player.
func (s *rdwsService) GetTime(ctx context.Context, serial string) (*types.RDWSTimeInfo, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Build the rDWS time endpoint URL
	timeURL := fmt.Sprintf("%s/time/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escQuery(serial))

	// Make the API request
	var response types.RDWSTimeResponse
	err = s.httpClient.GetWithAuth(ctx, token, timeURL, &response)
	if err != nil {
		return nil, errors.NewAPIError(0, "rdws_time_failed",
			fmt.Sprintf("Failed to get time for device with serial '%s'", serial), err.Error())
	}

	return &response.Data.Result, nil
}

// SetTime sets the date and time on a player.
func (s *rdwsService) SetTime(ctx context.Context, serial string, request *types.RDWSTimeSetRequest) (bool, error) {
	if serial == "" {
		return false, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	if request == nil {
		return false, errors.NewValidationError("request", request, "time set request cannot be nil")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return false, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return false, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return false, err
	}

	// Build the rDWS time endpoint URL
	timeURL := fmt.Sprintf("%s/time/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escQuery(serial))

	// Build request body - wrap in data envelope
	requestBody := map[string]interface{}{
		"data": request,
	}

	// Make the API request
	var response types.RDWSTimeSetResponse
	err = s.httpClient.PutWithAuth(ctx, token, timeURL, requestBody, &response)
	if err != nil {
		return false, errors.NewAPIError(0, "rdws_time_set_failed",
			fmt.Sprintf("Failed to set time for device with serial '%s'", serial), err.Error())
	}

	return response.Data.Result, nil
}

// GetHealth retrieves the current health status of a player.
// This is primarily used to determine if a player can respond to WebSocket requests.
func (s *rdwsService) GetHealth(ctx context.Context, serial string) (*types.RDWSHealthInfo, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Build the rDWS health endpoint URL
	healthURL := fmt.Sprintf("%s/health/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escQuery(serial))

	// Make the API request
	var response types.RDWSHealthResponse
	err = s.httpClient.GetWithAuth(ctx, token, healthURL, &response)
	if err != nil {
		return nil, errors.NewAPIError(0, "rdws_health_failed",
			fmt.Sprintf("Failed to get health for device with serial '%s'", serial), err.Error())
	}

	return &response.Data.Result, nil
}

// ListFiles lists the directories and/or files in a path on the player.
func (s *rdwsService) ListFiles(ctx context.Context, serial string, path string) (*types.RDWSFileListResponse, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}
	if path == "" {
		return nil, errors.NewValidationError("path", path, "file path cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Build the rDWS files list endpoint URL
	filesURL := fmt.Sprintf("%s/files/%s/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escPath(path), escQuery(serial))

	// Make the API request
	var response types.RDWSFileListResponse
	err = s.httpClient.GetWithAuth(ctx, token, filesURL, &response)
	if err != nil {
		return nil, errors.NewAPIError(0, "rdws_files_list_failed",
			fmt.Sprintf("Failed to list files for device with serial '%s' at path '%s'", serial, path), err.Error())
	}

	return &response, nil
}

// UploadFile uploads a file to the player storage.
// fileContents should be either plain text or Data URL (base64-encoded) format.
func (s *rdwsService) UploadFile(ctx context.Context, serial string, path string, fileName string, fileContents string, fileType string) (bool, error) {
	if serial == "" {
		return false, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}
	if path == "" {
		return false, errors.NewValidationError("path", path, "upload path cannot be empty")
	}
	if fileName == "" {
		return false, errors.NewValidationError("fileName", fileName, "file name cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return false, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return false, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return false, err
	}

	// Build the request
	var request types.RDWSFileUploadRequest
	request.Data.FileUploadPath = path
	request.Data.Files = []types.RDWSFileUploadItem{
		{
			FileName:     fileName,
			FileContents: fileContents,
			FileType:     fileType,
		},
	}

	// Build the rDWS files upload endpoint URL
	filesURL := fmt.Sprintf("%s/files%s?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escPath(path), escQuery(serial))

	// Make the API request
	var response types.RDWSFileUploadResponse
	err = s.httpClient.PutWithAuth(ctx, token, filesURL, request, &response)
	if err != nil {
		return false, errors.NewAPIError(0, "rdws_file_upload_failed",
			fmt.Sprintf("Failed to upload file '%s' to device with serial '%s'", fileName, serial), err.Error())
	}

	return response.Data.Result.Success, nil
}

// CreateFolder creates a new folder on the player storage.
func (s *rdwsService) CreateFolder(ctx context.Context, serial string, path string) (bool, error) {
	if serial == "" {
		return false, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}
	if path == "" {
		return false, errors.NewValidationError("path", path, "folder path cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return false, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return false, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return false, err
	}

	// Build the rDWS folder create endpoint URL (path should end with /)
	folderURL := fmt.Sprintf("%s/files%s/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escPath(path), escQuery(serial))

	// Make the API request (PUT with no body creates a folder)
	var response types.RDWSFileOperationResponse
	err = s.httpClient.PutWithAuth(ctx, token, folderURL, nil, &response)
	if err != nil {
		return false, errors.NewAPIError(0, "rdws_folder_create_failed",
			fmt.Sprintf("Failed to create folder at '%s' on device with serial '%s'", path, serial), err.Error())
	}

	return response.Data.Result.Success, nil
}

// RenameFile renames a file on the player storage.
func (s *rdwsService) RenameFile(ctx context.Context, serial string, path string, newName string) (bool, error) {
	if serial == "" {
		return false, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}
	if path == "" {
		return false, errors.NewValidationError("path", path, "file path cannot be empty")
	}
	if newName == "" {
		return false, errors.NewValidationError("newName", newName, "new file name cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return false, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return false, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return false, err
	}

	// Build the request
	var request types.RDWSFileRenameRequest
	request.Data.Name = newName

	// Build the rDWS file rename endpoint URL
	filesURL := fmt.Sprintf("%s/files/%s?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escPath(path), escQuery(serial))

	// Make the API request
	var response types.RDWSFileOperationResponse
	err = s.httpClient.PostWithAuth(ctx, token, filesURL, request, &response)
	if err != nil {
		return false, errors.NewAPIError(0, "rdws_file_rename_failed",
			fmt.Sprintf("Failed to rename file '%s' on device with serial '%s'", path, serial), err.Error())
	}

	return response.Data.Result.Success, nil
}

// DeleteFile deletes a file from the player storage.
func (s *rdwsService) DeleteFile(ctx context.Context, serial string, path string) (bool, error) {
	if serial == "" {
		return false, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}
	if path == "" {
		return false, errors.NewValidationError("path", path, "file path cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return false, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return false, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return false, err
	}

	// Build the rDWS file delete endpoint URL
	filesURL := fmt.Sprintf("%s/files/%s?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escPath(path), escQuery(serial))

	// Make the API request
	var response types.RDWSFileOperationResponse
	err = s.httpClient.DeleteWithAuth(ctx, token, filesURL, &response)
	if err != nil {
		return false, errors.NewAPIError(0, "rdws_file_delete_failed",
			fmt.Sprintf("Failed to delete file '%s' on device with serial '%s'", path, serial), err.Error())
	}

	return response.Data.Result.Success, nil
}

// GetLocalDWS retrieves the current state of local DWS on a player.
func (s *rdwsService) GetLocalDWS(ctx context.Context, serial string) (*types.RDWSLocalDWSInfo, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Build the rDWS local-dws endpoint URL
	localDWSURL := fmt.Sprintf("%s/control/local-dws/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escQuery(serial))

	// Make the API request
	var response types.RDWSLocalDWSResponse
	err = s.httpClient.GetWithAuth(ctx, token, localDWSURL, &response)
	if err != nil {
		return nil, errors.NewAPIError(0, "rdws_local_dws_get_failed",
			fmt.Sprintf("Failed to get local DWS status for device with serial '%s'", serial), err.Error())
	}

	return &response.Data.Result, nil
}

// SetLocalDWS enables or disables local DWS on a player.
func (s *rdwsService) SetLocalDWS(ctx context.Context, serial string, enabled bool) (bool, error) {
	if serial == "" {
		return false, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return false, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return false, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return false, err
	}

	// Build the request
	var request types.RDWSLocalDWSSetRequest
	request.Data.Enabled = enabled

	// Build the rDWS local-dws endpoint URL
	localDWSURL := fmt.Sprintf("%s/control/local-dws/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escQuery(serial))

	// Make the API request
	var response types.RDWSLocalDWSSetResponse
	err = s.httpClient.PutWithAuth(ctx, token, localDWSURL, request, &response)
	if err != nil {
		return false, errors.NewAPIError(0, "rdws_local_dws_set_failed",
			fmt.Sprintf("Failed to set local DWS status for device with serial '%s'", serial), err.Error())
	}

	return response.Data.Result.Success, nil
}

// GetDiagnostics runs network diagnostics on a player.
func (s *rdwsService) GetDiagnostics(ctx context.Context, serial string) (*types.RDWSDiagnosticsInfo, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Build the rDWS diagnostics endpoint URL
	diagnosticsURL := fmt.Sprintf("%s/diagnostics/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escQuery(serial))

	// Make the API request
	var response types.RDWSDiagnosticsResponse
	err = s.httpClient.GetWithAuth(ctx, token, diagnosticsURL, &response)
	if err != nil {
		return nil, errors.NewAPIError(0, "rdws_diagnostics_failed",
			fmt.Sprintf("Failed to run diagnostics for device with serial '%s'", serial), err.Error())
	}

	return &response.Data.Result, nil
}

// DNSLookup tests name resolution on a specified DNS address.
func (s *rdwsService) DNSLookup(ctx context.Context, serial string, domain string) (*types.RDWSDNSLookupResult, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}
	if domain == "" {
		return nil, errors.NewValidationError("domain", domain, "domain cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Build the rDWS DNS lookup endpoint URL
	dnsURL := fmt.Sprintf("%s/diagnostics/dns-lookup/%s/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escSegment(domain), escQuery(serial))

	// Make the API request
	var response types.RDWSDNSLookupResponse
	err = s.httpClient.GetWithAuth(ctx, token, dnsURL, &response)
	if err != nil {
		return nil, errors.NewAPIError(0, "rdws_dns_lookup_failed",
			fmt.Sprintf("Failed to perform DNS lookup for domain '%s' on device with serial '%s'", domain, serial), err.Error())
	}

	return &response.Data.Result, nil
}

// Ping pings a specified IP or DNS address on the local network.
func (s *rdwsService) Ping(ctx context.Context, serial string, host string) (*types.RDWSPingResult, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}
	if host == "" {
		return nil, errors.NewValidationError("host", host, "host cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Build the rDWS ping endpoint URL
	pingURL := fmt.Sprintf("%s/diagnostics/ping/%s/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escSegment(host), escQuery(serial))

	// Make the API request
	var response types.RDWSPingResponse
	err = s.httpClient.GetWithAuth(ctx, token, pingURL, &response)
	if err != nil {
		return nil, errors.NewAPIError(0, "rdws_ping_failed",
			fmt.Sprintf("Failed to ping host '%s' from device with serial '%s'", host, serial), err.Error())
	}

	return &response.Data.Result, nil
}

// TraceRoute performs a trace-route diagnostic on a specified IP or DNS address.
func (s *rdwsService) TraceRoute(ctx context.Context, serial string, host string) (*types.RDWSTraceRouteResult, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}
	if host == "" {
		return nil, errors.NewValidationError("host", host, "host cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Build the rDWS trace-route endpoint URL
	traceURL := fmt.Sprintf("%s/diagnostics/trace-route/%s/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escSegment(host), escQuery(serial))

	// Make the API request
	var response types.RDWSTraceRouteResponse
	err = s.httpClient.GetWithAuth(ctx, token, traceURL, &response)
	if err != nil {
		return nil, errors.NewAPIError(0, "rdws_trace_route_failed",
			fmt.Sprintf("Failed to trace route to host '%s' from device with serial '%s'", host, serial), err.Error())
	}

	return &response.Data.Result, nil
}

// GetNetworkConfig retrieves network interface settings for a player.
func (s *rdwsService) GetNetworkConfig(ctx context.Context, serial string, iface string) (*types.RDWSNetworkConfig, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}
	if iface == "" {
		return nil, errors.NewValidationError("interface", iface, "network interface cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Build the rDWS network configuration endpoint URL
	netConfigURL := fmt.Sprintf("%s/diagnostics/network-configuration/%s/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escSegment(iface), escQuery(serial))

	// Make the API request
	var response types.RDWSNetworkConfigResponse
	err = s.httpClient.GetWithAuth(ctx, token, netConfigURL, &response)
	if err != nil {
		return nil, errors.NewAPIError(0, "rdws_network_config_get_failed",
			fmt.Sprintf("Failed to get network configuration for interface '%s' on device with serial '%s'", iface, serial), err.Error())
	}

	return &response.Data.Result, nil
}

// SetNetworkConfig applies test network configuration to a player.
func (s *rdwsService) SetNetworkConfig(ctx context.Context, serial string, iface string, request *types.RDWSNetworkConfigSetRequest) (bool, error) {
	if serial == "" {
		return false, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}
	if iface == "" {
		return false, errors.NewValidationError("interface", iface, "network interface cannot be empty")
	}
	if request == nil {
		return false, errors.NewValidationError("request", request, "network configuration request cannot be nil")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return false, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return false, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return false, err
	}

	// Build the rDWS network configuration endpoint URL
	netConfigURL := fmt.Sprintf("%s/diagnostics/network-configuration/%s/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escSegment(iface), escQuery(serial))

	// Make the API request
	var response types.RDWSNetworkConfigSetResponse
	err = s.httpClient.PutWithAuth(ctx, token, netConfigURL, request, &response)
	if err != nil {
		return false, errors.NewAPIError(0, "rdws_network_config_set_failed",
			fmt.Sprintf("Failed to set network configuration for interface '%s' on device with serial '%s'", iface, serial), err.Error())
	}

	return response.Data.Result.Success, nil
}

// GetNetworkNeighborhood retrieves information about the player's network neighborhood.
func (s *rdwsService) GetNetworkNeighborhood(ctx context.Context, serial string) (*types.RDWSNetworkNeighborhoodResult, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Build the rDWS network neighborhood endpoint URL
	neighborhoodURL := fmt.Sprintf("%s/diagnostics/network-neighborhood/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escQuery(serial))

	// Make the API request
	var response types.RDWSNetworkNeighborhoodResponse
	err = s.httpClient.GetWithAuth(ctx, token, neighborhoodURL, &response)
	if err != nil {
		return nil, errors.NewAPIError(0, "rdws_network_neighborhood_failed",
			fmt.Sprintf("Failed to get network neighborhood for device with serial '%s'", serial), err.Error())
	}

	return &response.Data.Result, nil
}

// GetPacketCaptureStatus gets the current status of a packet capture operation.
func (s *rdwsService) GetPacketCaptureStatus(ctx context.Context, serial string) (*types.RDWSPacketCaptureStatus, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Build the rDWS packet capture endpoint URL
	packetCaptureURL := fmt.Sprintf("%s/diagnostics/packet-capture/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escQuery(serial))

	// Make the API request
	var response types.RDWSPacketCaptureResponse
	err = s.httpClient.GetWithAuth(ctx, token, packetCaptureURL, &response)
	if err != nil {
		return nil, errors.NewAPIError(0, "rdws_packet_capture_status_failed",
			fmt.Sprintf("Failed to get packet capture status for device with serial '%s'", serial), err.Error())
	}

	return &response.Data.Result, nil
}

// StartPacketCapture starts a packet capture operation on the player.
func (s *rdwsService) StartPacketCapture(ctx context.Context, serial string, request *types.RDWSPacketCaptureStartRequest) (string, error) {
	if serial == "" {
		return "", errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}
	if request == nil {
		return "", errors.NewValidationError("request", request, "packet capture request cannot be nil")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return "", err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return "", err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return "", err
	}

	// Build the rDWS packet capture endpoint URL
	packetCaptureURL := fmt.Sprintf("%s/diagnostics/packet-capture/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escQuery(serial))

	// Make the API request
	var response types.RDWSPacketCaptureStartResponse
	err = s.httpClient.PostWithAuth(ctx, token, packetCaptureURL, request, &response)
	if err != nil {
		return "", errors.NewAPIError(0, "rdws_packet_capture_start_failed",
			fmt.Sprintf("Failed to start packet capture on device with serial '%s'", serial), err.Error())
	}

	return response.Data.Result.FilePath, nil
}

// StopPacketCapture stops a running packet capture operation on the player.
func (s *rdwsService) StopPacketCapture(ctx context.Context, serial string) (string, error) {
	if serial == "" {
		return "", errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return "", err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return "", err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return "", err
	}

	// Build the rDWS packet capture endpoint URL
	packetCaptureURL := fmt.Sprintf("%s/diagnostics/packet-capture/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escQuery(serial))

	// Make the API request
	var response types.RDWSPacketCaptureStopResponse
	err = s.httpClient.DeleteWithAuth(ctx, token, packetCaptureURL, &response)
	if err != nil {
		return "", errors.NewAPIError(0, "rdws_packet_capture_stop_failed",
			fmt.Sprintf("Failed to stop packet capture on device with serial '%s'", serial), err.Error())
	}

	return response.Data.Result.FilePath, nil
}

// GetTelnetStatus gets telnet information (enabled status and port number).
func (s *rdwsService) GetTelnetStatus(ctx context.Context, serial string) (*types.RDWSTelnetInfo, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Build the rDWS telnet endpoint URL
	telnetURL := fmt.Sprintf("%s/diagnostics/telnet/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escQuery(serial))

	// Make the API request
	var response types.RDWSTelnetResponse
	err = s.httpClient.GetWithAuth(ctx, token, telnetURL, &response)
	if err != nil {
		return nil, errors.NewAPIError(0, "rdws_telnet_get_failed",
			fmt.Sprintf("Failed to get telnet status for device with serial '%s'", serial), err.Error())
	}

	return &response.Data.Result, nil
}

// SetTelnetStatus enables or disables telnet on the player.
func (s *rdwsService) SetTelnetStatus(ctx context.Context, serial string, enabled bool, port int) (bool, error) {
	if serial == "" {
		return false, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return false, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return false, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return false, err
	}

	// Build the request
	var request types.RDWSTelnetSetRequest
	request.Data.Enabled = enabled
	if port > 0 {
		request.Data.Port = port
	}

	// Build the rDWS telnet endpoint URL
	telnetURL := fmt.Sprintf("%s/diagnostics/telnet/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escQuery(serial))

	// Make the API request
	var response types.RDWSTelnetSetResponse
	err = s.httpClient.PutWithAuth(ctx, token, telnetURL, request, &response)
	if err != nil {
		return false, errors.NewAPIError(0, "rdws_telnet_set_failed",
			fmt.Sprintf("Failed to set telnet status for device with serial '%s'", serial), err.Error())
	}

	return response.Data.Result.Success, nil
}

// GetSSHStatus gets SSH information (enabled status and port number).
func (s *rdwsService) GetSSHStatus(ctx context.Context, serial string) (*types.RDWSSSHInfo, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Build the rDWS SSH endpoint URL
	sshURL := fmt.Sprintf("%s/diagnostics/ssh/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escQuery(serial))

	// Make the API request
	var response types.RDWSSSHResponse
	err = s.httpClient.GetWithAuth(ctx, token, sshURL, &response)
	if err != nil {
		return nil, errors.NewAPIError(0, "rdws_ssh_get_failed",
			fmt.Sprintf("Failed to get SSH status for device with serial '%s'", serial), err.Error())
	}

	return &response.Data.Result, nil
}

// SetSSHStatus enables or disables SSH on the player.
// If password is non-empty, it will be set. If password is empty, the existing password is not changed.
func (s *rdwsService) SetSSHStatus(ctx context.Context, serial string, enabled bool, port int, password string) (bool, error) {
	if serial == "" {
		return false, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return false, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return false, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return false, err
	}

	// Build the request
	var request types.RDWSSSHSetRequest
	request.Data.Enabled = enabled
	if port > 0 {
		request.Data.Port = port
	}
	if password != "" {
		request.Data.Password = password
	}

	// Build the rDWS SSH endpoint URL
	sshURL := fmt.Sprintf("%s/diagnostics/ssh/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escQuery(serial))

	// Make the API request
	var response types.RDWSSSHSetResponse
	err = s.httpClient.PutWithAuth(ctx, token, sshURL, request, &response)
	if err != nil {
		return false, errors.NewAPIError(0, "rdws_ssh_set_failed",
			fmt.Sprintf("Failed to set SSH status for device with serial '%s'", serial), err.Error())
	}

	return response.Data.Result.Success, nil
}

// ReformatStorage reformats the specified storage device on a player.
// WARNING: This operation will ERASE ALL DATA on the specified storage device.
// Common device names: "sd", "ssd", "usb"
func (s *rdwsService) ReformatStorage(ctx context.Context, serial string, deviceName string) (bool, error) {
	if serial == "" {
		return false, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}
	if deviceName == "" {
		return false, errors.NewValidationError("deviceName", deviceName, "storage device name cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return false, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return false, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return false, err
	}

	// Build the rDWS storage reformat endpoint URL
	storageURL := fmt.Sprintf("%s/storage/%s?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escSegment(deviceName), escQuery(serial))

	// Make the API request
	var response types.RDWSStorageReformatResponse
	err = s.httpClient.DeleteWithAuth(ctx, token, storageURL, &response)
	if err != nil {
		return false, errors.NewAPIError(0, "rdws_storage_reformat_failed",
			fmt.Sprintf("Failed to reformat storage device '%s' on device with serial '%s'", deviceName, serial), err.Error())
	}

	return response.Data.Result.Success, nil
}

// SendCustomData sends custom data to a player via UDP port 5000.
// This allows sending custom commands or data to player applications listening on UDP port 5000.
func (s *rdwsService) SendCustomData(ctx context.Context, serial string, data string) (bool, error) {
	if serial == "" {
		return false, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}
	if data == "" {
		return false, errors.NewValidationError("data", data, "custom data cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return false, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return false, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return false, err
	}

	// Build the request
	var request types.RDWSCustomDataRequest
	request.Data.Data = data

	// Build the rDWS custom data endpoint URL
	customURL := fmt.Sprintf("%s/custom/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escQuery(serial))

	// Make the API request
	var response types.RDWSCustomDataResponse
	err = s.httpClient.PutWithAuth(ctx, token, customURL, request, &response)
	if err != nil {
		return false, errors.NewAPIError(0, "rdws_custom_data_failed",
			fmt.Sprintf("Failed to send custom data to device with serial '%s'", serial), err.Error())
	}

	return response.Data.Result.Success, nil
}

// DownloadFirmware downloads and applies a firmware update to a player.
// The player will download the firmware from the specified URL and apply it.
// If autoReboot is nil or true, the player will reboot automatically after the firmware update is applied.
// If autoReboot is false, the player will NOT automatically reboot and will require manual reboot.
func (s *rdwsService) DownloadFirmware(ctx context.Context, serial string, firmwareURL string, autoReboot *bool) (bool, error) {
	if serial == "" {
		return false, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}
	if firmwareURL == "" {
		return false, errors.NewValidationError("firmwareURL", firmwareURL, "firmware URL cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return false, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return false, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return false, err
	}

	// Build the rDWS firmware download endpoint URL with query parameters
	firmwareDownloadURL := fmt.Sprintf("%s/download-firmware/?destinationType=player&destinationName=%s&url=%s",
		s.config.RDWSBaseURL,
		escQuery(serial),
		url.QueryEscape(firmwareURL))

	// Add autoReboot parameter if specified
	if autoReboot != nil {
		if *autoReboot {
			firmwareDownloadURL += "&autoReboot=true"
		} else {
			firmwareDownloadURL += "&autoReboot=false"
		}
	}

	// Make the API request using GET (not POST)
	var response types.RDWSFirmwareDownloadResponse
	err = s.httpClient.GetWithAuth(ctx, token, firmwareDownloadURL, &response)
	if err != nil {
		return false, errors.NewAPIError(0, "rdws_firmware_download_failed",
			fmt.Sprintf("Failed to initiate firmware download on device with serial '%s'", serial), err.Error())
	}

	return response.Data.Result.Success, nil
}

// GetRegistry retrieves the complete player registry dump.
func (s *rdwsService) GetRegistry(ctx context.Context, serial string) (*types.RDWSRegistry, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Build the rDWS registry endpoint URL
	registryURL := fmt.Sprintf("%s/registry/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escQuery(serial))

	// Make the API request
	var response types.RDWSRegistryResponse
	err = s.httpClient.GetWithAuth(ctx, token, registryURL, &response)
	if err != nil {
		return nil, errors.NewAPIError(0, "rdws_registry_get_failed",
			fmt.Sprintf("Failed to get registry from device with serial '%s'", serial), err.Error())
	}

	return &response.Data.Result, nil
}

// GetRegistryValue retrieves a specific value from the player registry.
func (s *rdwsService) GetRegistryValue(ctx context.Context, serial string, section string, key string) (*types.RDWSRegistryValue, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}
	if section == "" {
		return nil, errors.NewValidationError("section", section, "registry section cannot be empty")
	}
	if key == "" {
		return nil, errors.NewValidationError("key", key, "registry key cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Build the rDWS registry value endpoint URL
	registryURL := fmt.Sprintf("%s/registry/%s/%s/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escSegment(section), escSegment(key), escQuery(serial))

	// Make the API request
	var response types.RDWSRegistryValueResponse
	err = s.httpClient.GetWithAuth(ctx, token, registryURL, &response)
	if err != nil {
		return nil, errors.NewAPIError(0, "rdws_registry_value_get_failed",
			fmt.Sprintf("Failed to get registry value '%s/%s' from device with serial '%s'", section, key, serial), err.Error())
	}

	return &types.RDWSRegistryValue{
		Section: section,
		Key:     key,
		Value:   response.Data.Result.Value,
	}, nil
}

// SetRegistryValue sets a specific value in the player registry.
func (s *rdwsService) SetRegistryValue(ctx context.Context, serial string, section string, key string, value string) (bool, error) {
	if serial == "" {
		return false, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}
	if section == "" {
		return false, errors.NewValidationError("section", section, "registry section cannot be empty")
	}
	if key == "" {
		return false, errors.NewValidationError("key", key, "registry key cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return false, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return false, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return false, err
	}

	// Build the request
	var request types.RDWSRegistrySetRequest
	request.Data.Value = value

	// Build the rDWS registry set endpoint URL
	registryURL := fmt.Sprintf("%s/registry/%s/%s/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escSegment(section), escSegment(key), escQuery(serial))

	// Make the API request
	var response types.RDWSRegistrySetResponse
	err = s.httpClient.PutWithAuth(ctx, token, registryURL, request, &response)
	if err != nil {
		return false, errors.NewAPIError(0, "rdws_registry_value_set_failed",
			fmt.Sprintf("Failed to set registry value '%s/%s' on device with serial '%s'", section, key, serial), err.Error())
	}

	return response.Data.Result.Success, nil
}

// DeleteRegistryValue deletes a key-value pair from the player registry.
func (s *rdwsService) DeleteRegistryValue(ctx context.Context, serial string, section string, key string) (bool, error) {
	if serial == "" {
		return false, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}
	if section == "" {
		return false, errors.NewValidationError("section", section, "registry section cannot be empty")
	}
	if key == "" {
		return false, errors.NewValidationError("key", key, "registry key cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return false, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return false, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return false, err
	}

	// Build the rDWS registry delete endpoint URL
	registryURL := fmt.Sprintf("%s/registry/%s/%s/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escSegment(section), escSegment(key), escQuery(serial))

	// Make the API request
	var response types.RDWSRegistryDeleteResponse
	err = s.httpClient.DeleteWithAuth(ctx, token, registryURL, &response)
	if err != nil {
		return false, errors.NewAPIError(0, "rdws_registry_value_delete_failed",
			fmt.Sprintf("Failed to delete registry value '%s/%s' from device with serial '%s'", section, key, serial), err.Error())
	}

	return response.Data.Result.Success, nil
}

// FlushRegistry flushes the player registry immediately to disk.
func (s *rdwsService) FlushRegistry(ctx context.Context, serial string) (bool, error) {
	if serial == "" {
		return false, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return false, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return false, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return false, err
	}

	// Build the rDWS registry flush endpoint URL
	registryURL := fmt.Sprintf("%s/registry/flush/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escQuery(serial))

	// Make the API request
	var response types.RDWSRegistryFlushResponse
	err = s.httpClient.PutWithAuth(ctx, token, registryURL, nil, &response)
	if err != nil {
		return false, errors.NewAPIError(0, "rdws_registry_flush_failed",
			fmt.Sprintf("Failed to flush registry on device with serial '%s'", serial), err.Error())
	}

	return response.Data.Result.Success, nil
}

// GetRecoveryURL retrieves the recovery URL from the player registry.
func (s *rdwsService) GetRecoveryURL(ctx context.Context, serial string) (*types.RDWSRecoveryURL, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Build the rDWS recovery URL endpoint URL
	recoveryURL := fmt.Sprintf("%s/registry/recovery_url/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escQuery(serial))

	// Make the API request
	var response types.RDWSRecoveryURLResponse
	err = s.httpClient.GetWithAuth(ctx, token, recoveryURL, &response)
	if err != nil {
		return nil, errors.NewAPIError(0, "rdws_recovery_url_get_failed",
			fmt.Sprintf("Failed to get recovery URL from device with serial '%s'", serial), err.Error())
	}

	return &types.RDWSRecoveryURL{
		URL: response.Data.Result.URL,
	}, nil
}

// SetRecoveryURL sets the recovery URL in the player registry.
func (s *rdwsService) SetRecoveryURL(ctx context.Context, serial string, recoveryURL string) (bool, error) {
	if serial == "" {
		return false, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}
	if recoveryURL == "" {
		return false, errors.NewValidationError("recoveryURL", recoveryURL, "recovery URL cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return false, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return false, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return false, err
	}

	// Build the request
	var request types.RDWSRecoveryURLSetRequest
	request.Data.URL = recoveryURL

	// Build the rDWS recovery URL set endpoint URL
	recoveryURLEndpoint := fmt.Sprintf("%s/registry/recovery_url/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escQuery(serial))

	// Make the API request
	var response types.RDWSRecoveryURLSetResponse
	err = s.httpClient.PutWithAuth(ctx, token, recoveryURLEndpoint, request, &response)
	if err != nil {
		return false, errors.NewAPIError(0, "rdws_recovery_url_set_failed",
			fmt.Sprintf("Failed to set recovery URL on device with serial '%s'", serial), err.Error())
	}

	return response.Data.Result.Success, nil
}

// GetLogs retrieves log files from the player
func (s *rdwsService) GetLogs(ctx context.Context, serial string) (*types.RDWSLogs, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Build the rDWS logs endpoint URL
	logsEndpoint := fmt.Sprintf("%s/logs/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escQuery(serial))

	// Make the API request
	var response types.RDWSLogsResponse
	err = s.httpClient.GetWithAuth(ctx, token, logsEndpoint, &response)
	if err != nil {
		return nil, errors.NewAPIError(0, "rdws_logs_failed",
			fmt.Sprintf("Failed to get logs from device with serial '%s'", serial), err.Error())
	}

	logs, parseErr := parseLogsResult(response.Data.Result)
	if parseErr != nil {
		return nil, errors.NewAPIError(0, "rdws_logs_parse_failed",
			fmt.Sprintf("Failed to parse logs response from device with serial '%s'", serial), parseErr.Error())
	}

	return logs, nil
}

// parseLogsResult interprets the data.result payload of a GET /logs response.
//
// The player returns its kernel log (dmesg) as a plain JSON string, so a string
// result is the SUCCESS case. An earlier implementation treated any string as a
// device error message, which made GetLogs fail on every successful read and
// surface the log text as the error detail — the method could not succeed at all.
//
// Device-side failures do not reach here: they carry a non-2xx status and are
// converted by the HTTP client's error handling before the body is parsed. Some
// firmware may return a structured {"logs":[...]} list instead, which is still
// accepted.
func parseLogsResult(raw json.RawMessage) (*types.RDWSLogs, error) {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return &types.RDWSLogs{Text: text}, nil
	}

	var result types.RDWSLogsResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return &types.RDWSLogs{Files: result.Logs}, nil
}

// GetCrashDump retrieves crash dump files from the player
func (s *rdwsService) GetCrashDump(ctx context.Context, serial string) (*types.RDWSCrashDump, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Build the rDWS crash dump endpoint URL
	crashDumpEndpoint := fmt.Sprintf("%s/crash-dump/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, escQuery(serial))

	// Make the API request
	var response types.RDWSCrashDumpResponse
	err = s.httpClient.GetWithAuth(ctx, token, crashDumpEndpoint, &response)
	if err != nil {
		return nil, errors.NewAPIError(0, "rdws_crash_dump_failed",
			fmt.Sprintf("Failed to get crash dump from device with serial '%s'", serial), err.Error())
	}

	// Check if result is an error string or a success object
	var errorString string
	if err := json.Unmarshal(response.Data.Result, &errorString); err == nil {
		// Result is a string (error message)
		return nil, errors.NewAPIError(0, "rdws_crash_dump_error",
			fmt.Sprintf("Device returned error for serial '%s'", serial), errorString)
	}

	// Try to unmarshal as success response
	var result types.RDWSCrashDumpResult
	if err := json.Unmarshal(response.Data.Result, &result); err != nil {
		return nil, errors.NewAPIError(0, "rdws_crash_dump_parse_failed",
			fmt.Sprintf("Failed to parse crash dump response from device with serial '%s'", serial), err.Error())
	}

	// Convert response to return type
	crashDump := &types.RDWSCrashDump{
		Files: result.Dumps,
	}

	return crashDump, nil
}
