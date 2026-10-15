package services

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/brightsign/bscontrol/internal/auth"
	"github.com/brightsign/bscontrol/internal/config"
	"github.com/brightsign/bscontrol/internal/errors"
	"github.com/brightsign/bscontrol/internal/http"
	"github.com/brightsign/bscontrol/internal/types"
)

// DeviceService provides device management operations.
type DeviceService interface {
	List(ctx context.Context, opts ...ListOption) (*types.DeviceList, error)
	Get(ctx context.Context, serial string) (*types.Device, error)
	GetByID(ctx context.Context, id int) (*types.Device, error)
	Update(ctx context.Context, id int, device *types.Device) (*types.Device, error)
	UpdateBySerial(ctx context.Context, serial string, device *types.Device) (*types.Device, error)
	Delete(ctx context.Context, id int) error
	DeleteBySerial(ctx context.Context, serial string) error
	GetStatus(ctx context.Context, id int) (*types.DeviceStatus, error)
	GetStatusBySerial(ctx context.Context, serial string) (*types.DeviceStatus, error)
	GetErrors(ctx context.Context, id int, opts ...ListOption) (*types.DeviceErrorList, error)
	GetErrorsBySerial(ctx context.Context, serial string, opts ...ListOption) (*types.DeviceErrorList, error)
	Reboot(ctx context.Context, id int, rebootType types.RebootType) (*types.RebootResponse, error)
	RebootBySerial(ctx context.Context, serial string, rebootType types.RebootType) (*types.RebootResponse, error)
	TakeSnapshot(ctx context.Context, id int, request *types.SnapshotRequest) (*types.SnapshotResponse, error)
	TakeSnapshotBySerial(ctx context.Context, serial string, request *types.SnapshotRequest) (*types.SnapshotResponse, error)
	Reprovision(ctx context.Context, id int) (*types.ReprovisionResponse, error)
	ReprovisionBySerial(ctx context.Context, serial string) (*types.ReprovisionResponse, error)
	GetDWSPassword(ctx context.Context, id int) (*types.DWSPasswordGetResponse, error)
	GetDWSPasswordBySerial(ctx context.Context, serial string) (*types.DWSPasswordGetResponse, error)
	SetDWSPassword(ctx context.Context, id int, request *types.DWSPasswordRequest) (*types.DWSPasswordSetResponse, error)
	SetDWSPasswordBySerial(ctx context.Context, serial string, request *types.DWSPasswordRequest) (*types.DWSPasswordSetResponse, error)
	ListGroups(ctx context.Context) (*types.GroupList, error)
	GetGroup(ctx context.Context, id int) (*types.Group, error)
	GetGroupByName(ctx context.Context, name string) (*types.Group, error)
	CreateGroup(ctx context.Context, name string) (*types.Group, error)
	UpdateGroup(ctx context.Context, id int, group *types.Group) (*types.Group, error)
	DeleteGroup(ctx context.Context, id int) error
	GetDownloads(ctx context.Context, id int) (*types.DeviceDownloadList, error)
	GetDownloadsBySerial(ctx context.Context, serial string) (*types.DeviceDownloadList, error)
	GetOperations(ctx context.Context, id int) (*types.DeviceOperationList, error)
	GetOperationsBySerial(ctx context.Context, serial string) (*types.DeviceOperationList, error)
}

// deviceService implements the DeviceService interface.
type deviceService struct {
	config      *config.Config
	httpClient  *http.HTTPClient
	authManager *auth.AuthManager
}

// NewDeviceService creates a new device service.
func NewDeviceService(cfg *config.Config, httpClient *http.HTTPClient, authManager *auth.AuthManager) DeviceService {
	return &deviceService{
		config:      cfg,
		httpClient:  httpClient,
		authManager: authManager,
	}
}

// List retrieves a list of devices with optional filtering and pagination.
func (s *deviceService) List(ctx context.Context, opts ...ListOption) (*types.DeviceList, error) {
	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Build query parameters from options
	config := &listConfig{}
	for _, opt := range opts {
		opt.apply(config)
	}

	params := url.Values{}
	if config.pageSize > 0 {
		params.Set("pageSize", strconv.Itoa(config.pageSize))
	}
	if config.marker != "" {
		params.Set("marker", config.marker)
	}
	if config.filter != "" {
		params.Set("filter", config.filter)
	}
	if config.sort != "" {
		params.Set("sort", config.sort)
	}

	// Build URL
	baseURL := fmt.Sprintf("%s/%s/Devices", s.config.BSNBaseURL, s.config.APIVersion)
	if len(params) > 0 {
		baseURL += "?" + params.Encode()
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Make the API request
	var deviceList types.DeviceList
	err = s.httpClient.GetWithAuth(ctx, token, baseURL, &deviceList)
	if err != nil {
		return nil, errors.NewAPIError(0, "device_list_failed", "Failed to list devices", err.Error())
	}

	return &deviceList, nil
}

// Get retrieves a device by its serial number.
func (s *deviceService) Get(ctx context.Context, serial string) (*types.Device, error) {
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

	// Build URL - using direct serial endpoint format (serial is a path segment)
	deviceURL := fmt.Sprintf("%s/%s/Devices/%s",
		s.config.BSNBaseURL, s.config.APIVersion, url.PathEscape(serial))

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Make the API request
	var device types.Device
	err = s.httpClient.GetWithAuth(ctx, token, deviceURL, &device)
	if err != nil {
		return nil, errors.NewAPIError(0, "device_get_failed",
			fmt.Sprintf("Failed to get device with serial '%s'", serial), err.Error())
	}

	return &device, nil
}

// GetByID retrieves a device by its ID.
func (s *deviceService) GetByID(ctx context.Context, id int) (*types.Device, error) {
	if id <= 0 {
		return nil, errors.NewValidationError("id", id, "device ID must be positive")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Build URL
	deviceURL := fmt.Sprintf("%s/%s/Devices/%d",
		s.config.BSNBaseURL, s.config.APIVersion, id)

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Make the API request
	var device types.Device
	err = s.httpClient.GetWithAuth(ctx, token, deviceURL, &device)
	if err != nil {
		return nil, errors.NewAPIError(0, "device_get_failed",
			fmt.Sprintf("Failed to get device with ID %d", id), err.Error())
	}

	return &device, nil
}

// Update updates a device by its ID.
func (s *deviceService) Update(ctx context.Context, id int, device *types.Device) (*types.Device, error) {
	if id <= 0 {
		return nil, errors.NewValidationError("id", id, "device ID must be positive")
	}
	if device == nil {
		return nil, errors.NewValidationError("device", "nil", "device cannot be nil")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Build URL
	deviceURL := fmt.Sprintf("%s/%s/Devices/%d",
		s.config.BSNBaseURL, s.config.APIVersion, id)

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Make the API request
	var updatedDevice types.Device
	err = s.httpClient.PutWithAuth(ctx, token, deviceURL, device, &updatedDevice)
	if err != nil {
		return nil, errors.NewAPIError(0, "device_update_failed",
			fmt.Sprintf("Failed to update device with ID %d", id), err.Error())
	}

	return &updatedDevice, nil
}

// UpdateBySerial updates a device by its serial number.
func (s *deviceService) UpdateBySerial(ctx context.Context, serial string, device *types.Device) (*types.Device, error) {
	// First get the device to find its ID
	existingDevice, err := s.Get(ctx, serial)
	if err != nil {
		return nil, err
	}

	// Update using the ID
	return s.Update(ctx, existingDevice.ID, device)
}

// Delete removes a device from the network by ID.
func (s *deviceService) Delete(ctx context.Context, id int) error {
	if id <= 0 {
		return errors.NewValidationError("id", fmt.Sprintf("%d", id), "device ID must be positive")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return err
	}

	// Build URL
	deleteURL := fmt.Sprintf("%s/%s/Devices/%d",
		s.config.BSNBaseURL, s.config.APIVersion, id)

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return err
	}

	// Make the API request - DELETE returns no content on success
	err = s.httpClient.DeleteWithAuth(ctx, token, deleteURL, nil)
	if err != nil {
		return errors.NewAPIError(0, "device_delete_failed", "Failed to delete device", err.Error())
	}

	return nil
}

// DeleteBySerial removes a device from the network by serial number.
func (s *deviceService) DeleteBySerial(ctx context.Context, serial string) error {
	if serial == "" {
		return errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	// First get the device to find its ID
	existingDevice, err := s.Get(ctx, serial)
	if err != nil {
		return err
	}

	// Delete using the ID
	return s.Delete(ctx, existingDevice.ID)
}

// ListGroups retrieves all device groups in the network.
func (s *deviceService) ListGroups(ctx context.Context) (*types.GroupList, error) {
	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Build URL
	groupsURL := fmt.Sprintf("%s/%s/Groups/Regular",
		s.config.BSNBaseURL, s.config.APIVersion)

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Make the API request
	var groups types.GroupList
	err = s.httpClient.GetWithAuth(ctx, token, groupsURL, &groups)
	if err != nil {
		return nil, errors.NewAPIError(0, "groups_list_failed", "Failed to list groups", err.Error())
	}

	return &groups, nil
}

// CreateGroup creates a new device group.
func (s *deviceService) CreateGroup(ctx context.Context, name string) (*types.Group, error) {
	if name == "" {
		return nil, errors.NewValidationError("name", name, "group name cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Build URL
	groupsURL := fmt.Sprintf("%s/%s/Groups/Regular",
		s.config.BSNBaseURL, s.config.APIVersion)

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Prepare request body
	groupRequest := map[string]string{"name": name}

	// Make the API request
	var group types.Group
	err = s.httpClient.PostWithAuth(ctx, token, groupsURL, groupRequest, &group)
	if err != nil {
		return nil, errors.NewAPIError(0, "group_create_failed",
			fmt.Sprintf("Failed to create group '%s'", name), err.Error())
	}

	return &group, nil
}

// GetGroup retrieves a specific device group by ID.
func (s *deviceService) GetGroup(ctx context.Context, id int) (*types.Group, error) {
	if id <= 0 {
		return nil, errors.NewValidationError("id", fmt.Sprintf("%d", id), "group ID must be positive")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Build URL
	groupURL := fmt.Sprintf("%s/%s/Groups/Regular/%d",
		s.config.BSNBaseURL, s.config.APIVersion, id)

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Make the API request
	var group types.Group
	err = s.httpClient.GetWithAuth(ctx, token, groupURL, &group)
	if err != nil {
		return nil, errors.NewAPIError(0, "group_get_failed",
			fmt.Sprintf("Failed to get group with ID %d", id), err.Error())
	}

	return &group, nil
}

// GetGroupByName retrieves a specific device group by name.
func (s *deviceService) GetGroupByName(ctx context.Context, name string) (*types.Group, error) {
	if name == "" {
		return nil, errors.NewValidationError("name", name, "group name cannot be empty")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Build URL - name needs to be URL encoded
	groupURL := fmt.Sprintf("%s/%s/Groups/Regular/%s/",
		s.config.BSNBaseURL, s.config.APIVersion, url.PathEscape(name))

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Make the API request
	var group types.Group
	err = s.httpClient.GetWithAuth(ctx, token, groupURL, &group)
	if err != nil {
		return nil, errors.NewAPIError(0, "group_get_failed",
			fmt.Sprintf("Failed to get group with name '%s'", name), err.Error())
	}

	return &group, nil
}

// UpdateGroup updates an existing device group.
func (s *deviceService) UpdateGroup(ctx context.Context, id int, group *types.Group) (*types.Group, error) {
	if id <= 0 {
		return nil, errors.NewValidationError("id", fmt.Sprintf("%d", id), "group ID must be positive")
	}
	if group == nil {
		return nil, errors.NewValidationError("group", "nil", "group cannot be nil")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Build URL
	groupURL := fmt.Sprintf("%s/%s/Groups/Regular/%d",
		s.config.BSNBaseURL, s.config.APIVersion, id)

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Make the API request
	var updatedGroup types.Group
	err = s.httpClient.PutWithAuth(ctx, token, groupURL, group, &updatedGroup)
	if err != nil {
		return nil, errors.NewAPIError(0, "group_update_failed",
			fmt.Sprintf("Failed to update group with ID %d", id), err.Error())
	}

	return &updatedGroup, nil
}

// DeleteGroup removes a device group from the network.
func (s *deviceService) DeleteGroup(ctx context.Context, id int) error {
	if id <= 0 {
		return errors.NewValidationError("id", fmt.Sprintf("%d", id), "group ID must be positive")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return err
	}

	// Build URL
	groupURL := fmt.Sprintf("%s/%s/Groups/Regular/%d",
		s.config.BSNBaseURL, s.config.APIVersion, id)

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return err
	}

	// Make the API request - DELETE returns no content on success
	err = s.httpClient.DeleteWithAuth(ctx, token, groupURL, nil)
	if err != nil {
		return errors.NewAPIError(0, "group_delete_failed",
			fmt.Sprintf("Failed to delete group with ID %d", id), err.Error())
	}

	return nil
}

// GetDownloads retrieves the list of content downloads for a device by device ID.
func (s *deviceService) GetDownloads(ctx context.Context, id int) (*types.DeviceDownloadList, error) {
	if id <= 0 {
		return nil, errors.NewValidationError("id", fmt.Sprintf("%d", id), "device ID must be positive")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Build URL
	downloadsURL := fmt.Sprintf("%s/%s/Devices/%d/Downloads",
		s.config.BSNBaseURL, s.config.APIVersion, id)

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Make the API request
	var downloadList types.DeviceDownloadList
	err = s.httpClient.GetWithAuth(ctx, token, downloadsURL, &downloadList)
	if err != nil {
		return nil, errors.NewAPIError(0, "device_downloads_get_failed",
			fmt.Sprintf("Failed to get downloads for device with ID %d", id), err.Error())
	}

	return &downloadList, nil
}

// GetDownloadsBySerial retrieves the list of content downloads for a device by serial number.
func (s *deviceService) GetDownloadsBySerial(ctx context.Context, serial string) (*types.DeviceDownloadList, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	// First get the device to retrieve its ID
	device, err := s.Get(ctx, serial)
	if err != nil {
		return nil, err
	}

	// Use the ID to get downloads
	return s.GetDownloads(ctx, device.ID)
}

// GetOperations retrieves the list of operations for a device by device ID.
func (s *deviceService) GetOperations(ctx context.Context, id int) (*types.DeviceOperationList, error) {
	if id <= 0 {
		return nil, errors.NewValidationError("id", fmt.Sprintf("%d", id), "device ID must be positive")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Build URL
	operationsURL := fmt.Sprintf("%s/%s/Devices/%d/Operations",
		s.config.BSNBaseURL, s.config.APIVersion, id)

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Make the API request
	var operationList types.DeviceOperationList
	err = s.httpClient.GetWithAuth(ctx, token, operationsURL, &operationList)
	if err != nil {
		return nil, errors.NewAPIError(0, "device_operations_get_failed",
			fmt.Sprintf("Failed to get operations for device with ID %d", id), err.Error())
	}

	return &operationList, nil
}

// GetOperationsBySerial retrieves the list of operations for a device by serial number.
func (s *deviceService) GetOperationsBySerial(ctx context.Context, serial string) (*types.DeviceOperationList, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	// First get the device to retrieve its ID
	device, err := s.Get(ctx, serial)
	if err != nil {
		return nil, err
	}

	// Use the ID to get operations
	return s.GetOperations(ctx, device.ID)
}

// ListOption represents an option for device listing.
type ListOption interface {
	apply(*listConfig)
}

// listConfig holds configuration for device listing.
type listConfig struct {
	pageSize int
	marker   string
	filter   string
	sort     string
}

// optionFunc is a function that implements ListOption.
type optionFunc func(*listConfig)

func (f optionFunc) apply(c *listConfig) {
	f(c)
}

// WithPageSize sets the page size for device listing.
func WithPageSize(size int) ListOption {
	return optionFunc(func(c *listConfig) {
		c.pageSize = size
	})
}

// WithMarker sets the pagination marker for device listing.
func WithMarker(marker string) ListOption {
	return optionFunc(func(c *listConfig) {
		c.marker = marker
	})
}

// WithFilter sets the filter expression for device listing.
func WithFilter(expression string) ListOption {
	return optionFunc(func(c *listConfig) {
		c.filter = expression
	})
}

// WithSort sets the sort expression for device listing.
func WithSort(expression string) ListOption {
	return optionFunc(func(c *listConfig) {
		c.sort = expression
	})
}

// GetStatus retrieves the current operational status of a device by ID.
func (s *deviceService) GetStatus(ctx context.Context, id int) (*types.DeviceStatus, error) {
	if id <= 0 {
		return nil, errors.NewValidationError("id", id, "device ID must be positive")
	}

	// Get the device details which includes status
	device, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get device details: %w", err)
	}

	// Convert the embedded status to DeviceStatus
	status := &types.DeviceStatus{
		DeviceID:      strconv.Itoa(device.ID),
		Serial:        device.Serial,
		Model:         device.Model,
		HealthStatus:  "Unknown",
		UptimeDisplay: "Unknown",
	}

	// Extract information from embedded status if available
	if device.Status != nil {
		status.HealthStatus = device.Status.Health
		status.UptimeDisplay = device.Status.Uptime
		status.LastHealthCheck = device.Status.LastModifiedDate

		if device.Status.Firmware != nil {
			status.FirmwareVersion = device.Status.Firmware.Version
		}

		if device.Status.Network != nil {
			// Extract first IP from interfaces if available
			if len(device.Status.Network.Interfaces) > 0 {
				for _, iface := range device.Status.Network.Interfaces {
					if iface.Enabled && len(iface.IP) > 0 {
						status.IPAddress = iface.IP[0]
						status.ConnectionType = iface.Type
						break
					}
				}
			}
			// Fallback to external IP if available
			if status.IPAddress == "" && device.Status.Network.ExternalIP != "" {
				status.IPAddress = device.Status.Network.ExternalIP
			}
		}

		// Parse uptime to seconds if possible
		if device.Status.Uptime != "" {
			// Try to parse uptime string (e.g., "1d 2h 3m" or similar)
			// For now, just use the string display
			status.UptimeDisplay = device.Status.Uptime
		}

		// Determine online status based on health
		status.IsOnline = device.Status.Health == "Healthy" || device.Status.Health == "Good"
		status.Status = device.Status.Health
		status.LastSeen = device.Status.LastModifiedDate
	}

	return status, nil
}

// GetStatusBySerial retrieves the current operational status of a device by serial number.
func (s *deviceService) GetStatusBySerial(ctx context.Context, serial string) (*types.DeviceStatus, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	// First get the device to obtain its ID
	device, err := s.Get(ctx, serial)
	if err != nil {
		return nil, fmt.Errorf("failed to get device by serial: %w", err)
	}

	// Use the device ID to get status
	return s.GetStatus(ctx, device.ID)
}

// GetErrors retrieves error logs and diagnostic information for a device by ID.
// Note: Based on the API documentation, device errors might not have a dedicated endpoint.
// This implementation attempts to use the /Errors endpoint if it exists, otherwise returns empty list.
func (s *deviceService) GetErrors(ctx context.Context, id int, opts ...ListOption) (*types.DeviceErrorList, error) {
	if id <= 0 {
		return nil, errors.NewValidationError("id", id, "device ID must be positive")
	}

	// First get the device to obtain its serial number for better error messages
	device, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get device details: %w", err)
	}

	// Use the GetErrorsBySerial method to do the actual work
	return s.GetErrorsBySerial(ctx, device.Serial, opts...)
}

// GetErrorsBySerial retrieves error logs and diagnostic information for a device by serial number.
func (s *deviceService) GetErrorsBySerial(ctx context.Context, serial string, opts ...ListOption) (*types.DeviceErrorList, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	// First get the device to obtain its ID
	device, err := s.Get(ctx, serial)
	if err != nil {
		return nil, fmt.Errorf("failed to get device by serial: %w", err)
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Build query parameters from options
	config := &listConfig{}
	for _, opt := range opts {
		opt.apply(config)
	}

	params := url.Values{}
	if config.pageSize > 0 {
		params.Set("pageSize", strconv.Itoa(config.pageSize))
	}
	if config.marker != "" {
		params.Set("marker", config.marker)
	}
	if config.filter != "" {
		params.Set("filter", config.filter)
	}
	if config.sort != "" {
		params.Set("sort", config.sort)
	}

	// Build URL - try the Errors endpoint
	baseURL := fmt.Sprintf("%s/%s/Devices/%d/Errors",
		s.config.BSNBaseURL, s.config.APIVersion, device.ID)
	if len(params) > 0 {
		baseURL += "?" + params.Encode()
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Make the API request
	var errorList types.DeviceErrorList
	err = s.httpClient.GetWithAuth(ctx, token, baseURL, &errorList)
	if err != nil {
		// If the endpoint doesn't exist, return empty list
		// This is because the API documentation shows device errors might be
		// retrieved through other means (e.g., Downloads, Operations, or RDWS logs)
		if httpErr, ok := err.(*errors.APIError); ok && httpErr.StatusCode == 404 {
			return &types.DeviceErrorList{
				Items:       []types.DeviceError{},
				IsTruncated: false,
				TotalCount:  0,
			}, nil
		}
		return nil, errors.NewAPIError(0, "device_errors_failed",
			fmt.Sprintf("Failed to get errors for device with serial '%s'", serial), err.Error())
	}

	return &errorList, nil
}

// Reboot initiates a remote reboot of the device by ID.
// This uses the RDWS (Remote Diagnostic Web Service) API to send a reboot command.
func (s *deviceService) Reboot(ctx context.Context, id int, rebootType types.RebootType) (*types.RebootResponse, error) {
	if id <= 0 {
		return nil, errors.NewValidationError("id", id, "device ID must be positive")
	}

	// First get the device to obtain its serial number for better error messages
	device, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get device details: %w", err)
	}

	// Use the RebootBySerial method to do the actual reboot
	return s.RebootBySerial(ctx, device.Serial, rebootType)
}

// RebootBySerial initiates a remote reboot of the device by serial number.
func (s *deviceService) RebootBySerial(ctx context.Context, serial string, rebootType types.RebootType) (*types.RebootResponse, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	// First get the device to obtain its ID (needed for the API call)
	device, err := s.Get(ctx, serial)
	if err != nil {
		return nil, fmt.Errorf("failed to get device by serial: %w", err)
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

	// Build the proper rDWS reboot endpoint according to documentation
	// PUT /rest/v1/control/reboot/?destinationType=player&destinationName={{deviceSerial}}
	rebootURL := fmt.Sprintf("%s/control/reboot/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, serial)

	// Build request body based on reboot type
	var requestBody interface{}
	switch rebootType {
	case types.RebootTypeNormal:
		// No body for normal reboot
		requestBody = nil
	case types.RebootTypeCrash:
		// Request Example (Crash Report)
		requestBody = map[string]interface{}{
			"data": map[string]interface{}{
				"crash_report": true,
			},
		}
	case types.RebootTypeFactoryReset:
		// Request Example (Factory Reset)
		requestBody = map[string]interface{}{
			"data": map[string]interface{}{
				"factory_reset": true,
			},
		}
	case types.RebootTypeDisableAutorun:
		// Request Example (Disable Autorun)
		requestBody = map[string]interface{}{
			"data": map[string]interface{}{
				"autorun": "disable",
			},
		}
	default:
		return nil, errors.NewValidationError("rebootType", rebootType, "invalid reboot type")
	}

	// Make the API request - PUT with appropriate body
	var rawResponse struct {
		Data struct {
			Result struct {
				Success bool   `json:"success"`
				Message string `json:"message"`
				Reboot  bool   `json:"reboot,omitempty"`
			} `json:"result"`
		} `json:"data"`
		Route  string `json:"route"`
		Method string `json:"method"`
	}

	err = s.httpClient.PutWithAuth(ctx, token, rebootURL, requestBody, &rawResponse)
	if err != nil {
		return nil, errors.NewAPIError(0, "device_reboot_failed",
			fmt.Sprintf("Failed to reboot device with serial '%s'", serial), err.Error())
	}

	// Convert the rDWS response to our RebootResponse format
	rebootResponse := &types.RebootResponse{
		DeviceID:  strconv.Itoa(device.ID),
		Serial:    serial,
		Status:    "success",
		Message:   rawResponse.Data.Result.Message,
		Timestamp: time.Now(),
	}

	// Set status based on API response
	if rawResponse.Data.Result.Success {
		rebootResponse.Status = "success"
	} else {
		rebootResponse.Status = "failed"
	}

	// If reboot time is indicated, set it
	if rawResponse.Data.Result.Reboot {
		rebootResponse.RebootTime = time.Now()
	}

	return rebootResponse, nil
}

// TakeSnapshot initiates a remote screenshot of the device by ID.
func (s *deviceService) TakeSnapshot(ctx context.Context, id int, request *types.SnapshotRequest) (*types.SnapshotResponse, error) {
	if id <= 0 {
		return nil, errors.NewValidationError("id", id, "device ID must be positive")
	}

	// First get the device to obtain its serial number
	device, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get device details: %w", err)
	}

	// Use the TakeSnapshotBySerial method to do the actual snapshot
	return s.TakeSnapshotBySerial(ctx, device.Serial, request)
}

// TakeSnapshotBySerial initiates a remote screenshot of the device by serial number.
func (s *deviceService) TakeSnapshotBySerial(ctx context.Context, serial string, request *types.SnapshotRequest) (*types.SnapshotResponse, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	if request == nil {
		request = &types.SnapshotRequest{}
	}

	// Set defaults if not provided
	if request.Format == "" {
		request.Format = "png"
	}
	if request.Quality == 0 {
		request.Quality = 90
	}
	if request.Output == "" {
		request.Output = "base64"
	}
	if request.Compression == "" {
		request.Compression = "medium"
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

	// Build the rDWS snapshot endpoint according to documentation
	// POST /rest/v1/snapshot/?destinationType=player&destinationName={{deviceSerial}}
	snapshotURL := fmt.Sprintf("%s/snapshot/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, serial)

	// Build request body
	requestBody := map[string]interface{}{
		"data": request,
	}

	// Make the API request - POST with request body
	var rawResponse struct {
		Data struct {
			Result types.SnapshotResponse `json:"result"`
		} `json:"data"`
		Route  string `json:"route"`
		Method string `json:"method"`
	}

	err = s.httpClient.PostWithAuth(ctx, token, snapshotURL, requestBody, &rawResponse)
	if err != nil {
		return nil, errors.NewAPIError(0, "device_snapshot_failed",
			fmt.Sprintf("Failed to take snapshot of device with serial '%s'", serial), err.Error())
	}

	// Return the snapshot response
	return &rawResponse.Data.Result, nil
}

// Reprovision initiates a remote re-provision of the device by ID.
func (s *deviceService) Reprovision(ctx context.Context, id int) (*types.ReprovisionResponse, error) {
	if id <= 0 {
		return nil, errors.NewValidationError("id", id, "device ID must be positive")
	}

	// First get the device to obtain its serial number
	device, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get device details: %w", err)
	}

	// Use the ReprovisionBySerial method to do the actual re-provision
	return s.ReprovisionBySerial(ctx, device.Serial)
}

// ReprovisionBySerial initiates a remote re-provision of the device by serial number.
func (s *deviceService) ReprovisionBySerial(ctx context.Context, serial string) (*types.ReprovisionResponse, error) {
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

	// Build the rDWS re-provision endpoint according to documentation
	// GET /rest/v1/re-provision/?destinationType=player&destinationName={{deviceSerial}}
	reprovisionURL := fmt.Sprintf("%s/re-provision/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, serial)

	// Make the API request - GET with no body
	var rawResponse struct {
		Data struct {
			Result types.ReprovisionResponse `json:"result"`
		} `json:"data"`
		Route  string `json:"route"`
		Method string `json:"method"`
	}

	err = s.httpClient.GetWithAuth(ctx, token, reprovisionURL, &rawResponse)
	if err != nil {
		return nil, errors.NewAPIError(0, "device_reprovision_failed",
			fmt.Sprintf("Failed to re-provision device with serial '%s'", serial), err.Error())
	}

	// Return the re-provision response
	return &rawResponse.Data.Result, nil
}

// GetDWSPassword retrieves DWS password information by device ID.
func (s *deviceService) GetDWSPassword(ctx context.Context, id int) (*types.DWSPasswordGetResponse, error) {
	if id <= 0 {
		return nil, errors.NewValidationError("id", id, "device ID must be positive")
	}

	// First get the device to obtain its serial number
	device, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get device details: %w", err)
	}

	// Use the GetDWSPasswordBySerial method to do the actual work
	return s.GetDWSPasswordBySerial(ctx, device.Serial)
}

// GetDWSPasswordBySerial retrieves DWS password information by device serial.
func (s *deviceService) GetDWSPasswordBySerial(ctx context.Context, serial string) (*types.DWSPasswordGetResponse, error) {
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

	// Build the rDWS DWS password endpoint according to documentation
	// GET /rest/v1/control/dws-password/?destinationType=player&destinationName={{deviceSerial}}
	dwsPasswordURL := fmt.Sprintf("%s/control/dws-password/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, serial)

	// Make the API request - GET with no body
	var rawResponse struct {
		Data struct {
			Result types.DWSPasswordGetResponse `json:"result"`
		} `json:"data"`
		Route  string `json:"route"`
		Method string `json:"method"`
	}

	err = s.httpClient.GetWithAuth(ctx, token, dwsPasswordURL, &rawResponse)
	if err != nil {
		return nil, errors.NewAPIError(0, "device_dws_password_get_failed",
			fmt.Sprintf("Failed to get DWS password info for device with serial '%s'", serial), err.Error())
	}

	// Return the DWS password response
	return &rawResponse.Data.Result, nil
}

// SetDWSPassword sets DWS password by device ID.
func (s *deviceService) SetDWSPassword(ctx context.Context, id int, request *types.DWSPasswordRequest) (*types.DWSPasswordSetResponse, error) {
	if id <= 0 {
		return nil, errors.NewValidationError("id", id, "device ID must be positive")
	}

	// First get the device to obtain its serial number
	device, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get device details: %w", err)
	}

	// Use the SetDWSPasswordBySerial method to do the actual work
	return s.SetDWSPasswordBySerial(ctx, device.Serial, request)
}

// SetDWSPasswordBySerial sets DWS password by device serial.
func (s *deviceService) SetDWSPasswordBySerial(ctx context.Context, serial string, request *types.DWSPasswordRequest) (*types.DWSPasswordSetResponse, error) {
	if serial == "" {
		return nil, errors.NewValidationError("serial", serial, "device serial cannot be empty")
	}

	if request == nil {
		return nil, errors.NewValidationError("request", request, "DWS password request cannot be nil")
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

	// Build the rDWS DWS password endpoint according to documentation
	// PUT /rest/v1/control/dws-password/?destinationType=player&destinationName={{deviceSerial}}
	dwsPasswordURL := fmt.Sprintf("%s/control/dws-password/?destinationType=player&destinationName=%s", s.config.RDWSBaseURL, serial)

	// Build request body
	requestBody := map[string]interface{}{
		"data": request,
	}

	// Make the API request - PUT with request body
	var rawResponse struct {
		Data struct {
			Result types.DWSPasswordSetResponse `json:"result"`
		} `json:"data"`
		Route  string `json:"route"`
		Method string `json:"method"`
	}

	err = s.httpClient.PutWithAuth(ctx, token, dwsPasswordURL, requestBody, &rawResponse)
	if err != nil {
		return nil, errors.NewAPIError(0, "device_dws_password_set_failed",
			fmt.Sprintf("Failed to set DWS password for device with serial '%s'", serial), err.Error())
	}

	// Return the DWS password response
	return &rawResponse.Data.Result, nil
}
