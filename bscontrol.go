// Package bscontrol provides a comprehensive SDK for interacting with BrightSign Cloud (BSN.cloud)
// and Remote Diagnostic Web Service (RDWS) APIs.
//
// The SDK enables developers to build custom automation tools and applications for BrightSign
// device management with a clean, extensible API that abstracts away implementation details.
//
// Basic usage:
//
//	client, err := bscontrol.New(
//	    bscontrol.WithCredentials(clientID, clientSecret),
//	    bscontrol.WithNetwork("My Network"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	ctx := context.Background()
//
//	// Authenticate
//	if err := client.Authenticate(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
//	// List networks
//	networks, err := client.GetNetworks(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Found %d networks\n", len(networks))
package bscontrol

import (
	"context"
	"time"

	"github.com/brightsign/bscontrol/internal/auth"
	"github.com/brightsign/bscontrol/internal/config"
	"github.com/brightsign/bscontrol/internal/errors"
	"github.com/brightsign/bscontrol/internal/http"
	"github.com/brightsign/bscontrol/internal/services"
	"github.com/brightsign/bscontrol/internal/types"
)

// Re-export public types from internal packages
type (
	// Network represents a BSN.cloud network.
	Network = types.Network

	// NetworkSubscription represents subscription details for a network.
	NetworkSubscription = types.NetworkSubscription

	// NetworkSettings represents network configuration settings.
	NetworkSettings = types.NetworkSettings

	// Device represents a BrightSign device in the network.
	Device = types.Device

	// DeviceSettings represents device configuration settings.
	DeviceSettings = types.DeviceSettings

	// Group represents a device group.
	Group = types.Group

	// GroupList represents a list of groups.
	GroupList = types.GroupList

	// DeviceList represents a paginated list of devices.
	DeviceList = types.DeviceList

	// DeviceStatusEmbed represents the embedded status information in a device.
	DeviceStatusEmbed = types.DeviceStatusEmbed

	// SyncSettings represents synchronization settings for a device.
	SyncSettings = types.SyncSettings

	// SyncPeriod represents a synchronization period configuration.
	SyncPeriod = types.SyncPeriod

	// ScriptInfo represents script-related information for a device.
	ScriptInfo = types.ScriptInfo

	// FirmwareInfo represents firmware version information.
	FirmwareInfo = types.FirmwareInfo

	// StorageInfo represents storage device information.
	StorageInfo = types.StorageInfo

	// NetworkInfo represents network connectivity information.
	NetworkInfo = types.NetworkInfo

	// NetworkInterfaceBSN represents a network interface from BSN.cloud.
	NetworkInterfaceBSN = types.NetworkInterfaceBSN

	// Region represents a rectangular region for screenshots.
	Region = types.Region

	// SnapshotRequest represents a request to capture a device screenshot.
	SnapshotRequest = types.SnapshotRequest

	// SnapshotResponse represents the response from a device snapshot request.
	SnapshotResponse = types.SnapshotResponse

	// DeviceStatus represents the current operational status of a device.
	DeviceStatus = types.DeviceStatus

	// DeviceError represents an error log entry from a device.
	DeviceError = types.DeviceError

	// DeviceErrorList represents a paginated list of device errors.
	DeviceErrorList = types.DeviceErrorList

	// RebootResponse represents the response from a device reboot request.
	RebootResponse = types.RebootResponse

	// RebootType represents the type of reboot to perform.
	RebootType = types.RebootType

	// ReprovisionResponse represents the response from a device re-provision request.
	ReprovisionResponse = types.ReprovisionResponse

	// DWSPasswordRequest represents a request to set the DWS password.
	DWSPasswordRequest = types.DWSPasswordRequest

	// DWSPasswordInfo represents information about the DWS password.
	DWSPasswordInfo = types.DWSPasswordInfo

	// DWSPasswordGetResponse represents the response from getting DWS password info.
	DWSPasswordGetResponse = types.DWSPasswordGetResponse

	// DWSPasswordSetResponse represents the response from setting DWS password.
	DWSPasswordSetResponse = types.DWSPasswordSetResponse

	// BDeployRecord represents a B-Deploy setup record.
	BDeployRecord = types.BDeployRecord

	// BDeployRecordList represents a list of B-Deploy setup records.
	BDeployRecordList = types.BDeployRecordList

	// BDeploySetupRecord represents a complete B-Deploy setup record for creation.
	BDeploySetupRecord = types.BDeploySetupRecord

	// BDeployCreateResponse represents the response from creating a B-Deploy record.
	BDeployCreateResponse = types.BDeployCreateResponse

	// BDeployDeleteResponse represents the response from deleting a B-Deploy record.
	BDeployDeleteResponse = types.BDeployDeleteResponse

	// BDeployInfo represents the B-Deploy section of a setup record.
	BDeployInfo = types.BDeployInfo

	// IdleScreenColor represents RGBA color for idle screen configuration.
	IdleScreenColor = types.IdleScreenColor

	// BSNTokenEntity represents the BSN device registration token.
	BSNTokenEntity = types.BSNTokenEntity

	// NetworkInterface represents a network interface configuration.
	NetworkInterface = types.NetworkInterface

	// NetworkConfig represents network configuration.
	NetworkConfig = types.NetworkConfig

	// NetworkContextRequest represents a request to set the network context.
	NetworkContextRequest = types.NetworkContextRequest

	// BDeployDevice represents a device in the B-Deploy system.
	BDeployDevice = types.BDeployDevice

	// BDeployDeviceRequest represents a request to create or update a B-Deploy device.
	BDeployDeviceRequest = types.BDeployDeviceRequest

	// BDeployDeviceResponse represents the response from B-Deploy device API.
	BDeployDeviceResponse = types.BDeployDeviceResponse

	// BDeployDeviceListResponse represents a list of B-Deploy devices.
	BDeployDeviceListResponse = types.BDeployDeviceListResponse

	// RDWSInfo represents player information from the rDWS /info/ endpoint
	RDWSInfo = types.RDWSInfo

	// RDWSInfoSubResult represents nested result objects in the info response.
	RDWSInfoSubResult = types.RDWSInfoSubResult

	// RDWSNetworkInterface represents a network interface from the player
	RDWSNetworkInterface = types.RDWSNetworkInterface

	// RDWSIPAddress represents an IP address configuration
	RDWSIPAddress = types.RDWSIPAddress

	// RDWSTimeInfo represents time information from the rDWS /time/ endpoint
	RDWSTimeInfo = types.RDWSTimeInfo

	// RDWSTimeSetRequest represents the request body for rDWS PUT /time/
	RDWSTimeSetRequest = types.RDWSTimeSetRequest

	// RDWSHealthInfo represents health status from the rDWS /health/ endpoint
	RDWSHealthInfo = types.RDWSHealthInfo

	// RDWSFileStat represents file statistics from the fs module
	RDWSFileStat = types.RDWSFileStat

	// RDWSFileInfo represents a file or directory entry
	RDWSFileInfo = types.RDWSFileInfo

	// RDWSStorageStats represents storage device statistics
	RDWSStorageStats = types.RDWSStorageStats

	// RDWSStorageInfo represents storage device information
	RDWSStorageInfo = types.RDWSStorageInfo

	// RDWSFileListResult represents the result of listing files
	RDWSFileListResult = types.RDWSFileListResult

	// RDWSFileListResponse represents the response from listing files
	RDWSFileListResponse = types.RDWSFileListResponse

	// RDWSFileUploadItem represents a single file to upload
	RDWSFileUploadItem = types.RDWSFileUploadItem

	// RDWSFileUploadRequest represents the request to upload files
	RDWSFileUploadRequest = types.RDWSFileUploadRequest

	// RDWSFileUploadResult represents the result of file upload
	RDWSFileUploadResult = types.RDWSFileUploadResult

	// RDWSFileUploadResponse represents the response from file upload
	RDWSFileUploadResponse = types.RDWSFileUploadResponse

	// RDWSFileRenameRequest represents the request to rename a file
	RDWSFileRenameRequest = types.RDWSFileRenameRequest

	// RDWSFileOperationResponse represents a generic success/error response for file operations
	RDWSFileOperationResponse = types.RDWSFileOperationResponse

	// DeviceDownload represents a content download on a device
	DeviceDownload = types.DeviceDownload

	// DeviceDownloadList represents a list of device downloads
	DeviceDownloadList = types.DeviceDownloadList

	// RDWSLocalDWSInfo represents the current state of local DWS
	RDWSLocalDWSInfo = types.RDWSLocalDWSInfo

	// RDWSDiagnosticsInfo represents network diagnostics information
	RDWSDiagnosticsInfo = types.RDWSDiagnosticsInfo

	// RDWSDNSLookupResult represents DNS lookup result
	RDWSDNSLookupResult = types.RDWSDNSLookupResult

	// RDWSPingResult represents ping diagnostic result
	RDWSPingResult = types.RDWSPingResult

	// RDWSTraceRouteHop represents a single hop in trace route
	RDWSTraceRouteHop = types.RDWSTraceRouteHop

	// RDWSTraceRouteResult represents trace route diagnostic result
	RDWSTraceRouteResult = types.RDWSTraceRouteResult

	// RDWSNetworkConfig represents network interface configuration
	RDWSNetworkConfig = types.RDWSNetworkConfig

	// RDWSNetworkConfigSetRequest represents the request to set network configuration
	RDWSNetworkConfigSetRequest = types.RDWSNetworkConfigSetRequest

	// RDWSNetworkNeighbor represents a device on the network
	RDWSNetworkNeighbor = types.RDWSNetworkNeighbor

	// RDWSNetworkNeighborhoodResult represents network neighborhood information
	RDWSNetworkNeighborhoodResult = types.RDWSNetworkNeighborhoodResult

	// RDWSPacketCaptureStatus represents packet capture status
	RDWSPacketCaptureStatus = types.RDWSPacketCaptureStatus

	// RDWSPacketCaptureStartRequest represents the request to start packet capture
	RDWSPacketCaptureStartRequest = types.RDWSPacketCaptureStartRequest

	// RDWSTelnetInfo represents telnet status information
	RDWSTelnetInfo = types.RDWSTelnetInfo

	// RDWSSSHInfo represents SSH status information
	RDWSSSHInfo = types.RDWSSSHInfo

	// Subscription represents a device subscription in BSN.cloud
	Subscription = types.Subscription

	// SubscriptionList represents a list of subscriptions with pagination support
	SubscriptionList = types.SubscriptionList

	// SubscriptionCount represents the count of subscriptions
	SubscriptionCount = types.SubscriptionCount

	// SubscriptionOperation represents an operation that can be performed on subscriptions
	SubscriptionOperation = types.SubscriptionOperation

	// SubscriptionOperations represents the list of available operations
	SubscriptionOperations = types.SubscriptionOperations

	// RDWSLogFile represents a single log file from the player
	RDWSLogFile = types.RDWSLogFile

	// RDWSLogs represents the collection of log files from a player
	RDWSLogs = types.RDWSLogs

	// RDWSCrashDumpFile represents a single crash dump file from the player
	RDWSCrashDumpFile = types.RDWSCrashDumpFile

	// RDWSCrashDump represents the collection of crash dump files from a player
	RDWSCrashDump = types.RDWSCrashDump

	// RDWSCrashDumpListEntry is one crash dump the player is holding, as reported
	// by GetCrashDumpFiles.
	RDWSCrashDumpListEntry = types.RDWSCrashDumpListEntry

	// RDWSComponentVersion is the {major,minor,patch,build} shape the player uses
	// for firmware, bootstrap and supervisor versions.
	RDWSComponentVersion = types.RDWSComponentVersion

	// RDWSSupervisorRef is one supervisor the bootstrap enumerated, carrying its
	// full path and whether it is the running build.
	RDWSSupervisorRef = types.RDWSSupervisorRef

	// RDWSSystemInfo is the player's system information, including the version of
	// the supervisor that is actually running.
	RDWSSystemInfo = types.RDWSSystemInfo

	// RDWSStoredSupervisors lists the supervisor builds downloaded to the player.
	RDWSStoredSupervisors = types.RDWSStoredSupervisors

	// RDWSDeleteSupervisorsRequest removes downloaded supervisor builds. Builds and
	// Clear are mutually exclusive.
	RDWSDeleteSupervisorsRequest = types.RDWSDeleteSupervisorsRequest

	// RDWSUpdateSyncResult is the outcome of asking a player to check for a
	// supervisor update now.
	RDWSUpdateSyncResult = types.RDWSUpdateSyncResult

	// RDWSRegistry represents the full player registry
	RDWSRegistry = types.RDWSRegistry

	// RDWSRegistryValue represents a single registry value
	RDWSRegistryValue = types.RDWSRegistryValue

	// RDWSRecoveryURL represents the player's recovery URL setting
	RDWSRecoveryURL = types.RDWSRecoveryURL

	// DeviceWebPage represents a device web page template
	DeviceWebPage = types.DeviceWebPage

	// DeviceWebPageList represents a paginated list of device web pages
	DeviceWebPageList = types.DeviceWebPageList

	// RequestTrace is the timing and outcome of a single HTTP request the SDK
	// made, delivered to a WithRequestTracer callback.
	RequestTrace = types.RequestTrace

	// RequestTracer receives one RequestTrace per request the SDK completes.
	RequestTracer = types.RequestTracer
)

// Re-export configuration options
type Option = config.Option

var (
	// WithCredentials sets the BSN.cloud OAuth2 credentials.
	WithCredentials = config.WithCredentials

	// WithNetwork sets the default network name for device operations.
	WithNetwork = config.WithNetwork

	// WithTimeout sets the HTTP request timeout for API calls.
	WithTimeout = config.WithTimeout

	// WithRetryCount sets the number of retry attempts for failed API requests.
	WithRetryCount = config.WithRetryCount

	// WithDebug enables debug logging of all HTTP requests and responses.
	WithDebug = config.WithDebug

	// WithEndpoints sets custom API endpoints for BSN.cloud and RDWS.
	WithEndpoints = config.WithEndpoints

	// WithTokenEndpoint sets a custom OAuth2 token endpoint.
	WithTokenEndpoint = config.WithTokenEndpoint

	// WithOIDCURL sets the OIDC base URL and derives the token endpoint from it.
	WithOIDCURL = config.WithOIDCURL

	// WithProvisioningEndpoint sets a custom provisioning endpoint.
	WithProvisioningEndpoint = config.WithProvisioningEndpoint

	// WithDeviceSerial sets a default device serial number for single-device operations.
	WithDeviceSerial = config.WithDeviceSerial

	// WithAccessToken sets a pre-loaded access token for session reuse.
	// This allows CLI tools to cache the bearer token between invocations,
	// skipping the OAuth round-trip when the token is still valid.
	WithAccessToken = config.WithAccessToken

	// WithEnv loads credentials and the network from BS_CLIENT_ID/BS_SECRET/
	// BS_NETWORK. New does not read the environment unless this is passed.
	WithEnv = config.WithEnv

	// WithTokenSource supplies a caller-owned TokenSource so the caller owns
	// token acquisition and caching instead of the SDK.
	WithTokenSource = config.WithTokenSource

	// WithRequestTracer registers a per-request timing callback (see
	// RequestTrace). Off by default; with no tracer the SDK does no tracing and
	// adds no overhead.
	WithRequestTracer = config.WithRequestTracer
)

// TokenSource supplies bearer tokens on demand. Implement it to own token
// acquisition and caching; pass the implementation with WithTokenSource.
type TokenSource = config.TokenSource

// AccessToken is a bearer token with its absolute expiry, returned by a
// TokenSource.
type AccessToken = config.AccessToken

// Re-export device list options
type ListOption = services.ListOption
type BDeployListOption = services.BDeployListOption
type BDeployDeviceListOption = services.BDeployDeviceListOption

var (
	// WithPageSize sets the page size for device listing.
	WithPageSize = services.WithPageSize

	// WithMarker sets the pagination marker for device listing.
	WithMarker = services.WithMarker

	// WithFilter sets the filter expression for device listing.
	WithFilter = services.WithFilter

	// WithSort sets the sort expression for device listing.
	WithSort = services.WithSort

	// WithNetworkName sets the network name filter for B-Deploy record listing.
	WithNetworkName = services.WithNetworkName

	// WithUsername sets the username filter for B-Deploy record listing.
	WithUsername = services.WithUsername

	// WithPackageName sets the package name filter for B-Deploy record listing.
	WithPackageName = services.WithPackageName

	// WithBDeployPageSize sets the page size for B-Deploy record listing.
	WithBDeployPageSize = services.WithBDeployPageSize

	// WithBDeployPage sets the page number for B-Deploy record listing.
	WithBDeployPage = services.WithBDeployPage

	// WithSetupName sets the setup name filter for B-Deploy device listing.
	// This filters devices by their associated setup package name using the query[setupName] API parameter.
	WithSetupName = services.WithSetupName
)

// Re-export reboot type constants
const (
	// RebootTypeNormal performs a standard reboot
	RebootTypeNormal = types.RebootTypeNormal
	// RebootTypeCrash performs a reboot and saves a crash report
	RebootTypeCrash = types.RebootTypeCrash
	// RebootTypeFactoryReset performs a factory reset reboot
	RebootTypeFactoryReset = types.RebootTypeFactoryReset
	// RebootTypeDisableAutorun disables autorun script and reboots
	RebootTypeDisableAutorun = types.RebootTypeDisableAutorun
)

// Re-export error checking functions
var (
	// IsAuthenticationError checks if an error is authentication-related.
	IsAuthenticationError = errors.IsAuthenticationError

	// IsNetworkError checks if an error is network-related.
	IsNetworkError = errors.IsNetworkError

	// IsConfigurationError checks if an error is configuration-related.
	IsConfigurationError = errors.IsConfigurationError

	// IsRetryableError checks if an error might succeed on retry.
	IsRetryableError = errors.IsRetryableError
)

// Client is the main SDK client that provides access to all BrightSign services.
//
// It manages authentication, network selection, and provides access to various
// service interfaces for device management, firmware operations, and job orchestration.
type Client struct {
	config      *config.Config
	httpClient  *http.HTTPClient
	authManager *auth.AuthManager

	// Services
	Devices        services.DeviceService
	BDeploy        services.BDeployService
	Provisioning   services.ProvisioningService
	RDWS           services.RDWSService
	Subscriptions  services.SubscriptionService
	DeviceWebPages services.DeviceWebPageService
}

// New creates a new BrightSign SDK client with the given configuration options.
//
// The client must be configured with at least BSN.cloud credentials:
//
//	client, err := bscontrol.New(
//	    bscontrol.WithCredentials(clientID, clientSecret),
//	)
//
// Additional options can be provided for custom endpoints, timeouts, and network selection:
//
//	client, err := bscontrol.New(
//	    bscontrol.WithCredentials(clientID, clientSecret),
//	    bscontrol.WithNetwork("Production Network"),
//	    bscontrol.WithTimeout(60*time.Second),
//	)
//
// New does not read the environment implicitly. To load credentials and the
// network from BS_CLIENT_ID/BS_SECRET/BS_NETWORK, pass WithEnv():
//
//	client, err := bscontrol.New(bscontrol.WithEnv())
func New(opts ...Option) (*Client, error) {
	cfg := config.DefaultConfig()

	// New does not read the environment on its own. Pass WithEnv() to load
	// BS_CLIENT_ID/BS_SECRET/BS_NETWORK, or supply values explicitly.
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Create HTTP client
	httpClient := http.NewHTTPClient(cfg)

	// Create authentication manager
	authManager := auth.NewAuthManager(cfg, httpClient)

	client := &Client{
		config:         cfg,
		httpClient:     httpClient,
		authManager:    authManager,
		Devices:        services.NewDeviceService(cfg, httpClient, authManager),
		BDeploy:        services.NewBDeployService(cfg, httpClient, authManager),
		Provisioning:   services.NewProvisioningService(cfg, httpClient, authManager),
		RDWS:           services.NewRDWSService(cfg, httpClient, authManager),
		Subscriptions:  services.NewSubscriptionService(cfg, httpClient, authManager),
		DeviceWebPages: services.NewDeviceWebPageService(cfg, httpClient, authManager),
	}

	return client, nil
}

// Authenticate performs OAuth2 authentication with BSN.cloud using the configured credentials.
//
// This method must be called before making any API requests. The SDK will automatically
// handle token refresh for subsequent requests.
func (c *Client) Authenticate(ctx context.Context) error {
	return c.authManager.Authenticate(ctx)
}

// SetNetwork sets the active network for subsequent device operations by name.
//
// Many device operations require a network context. This method sets the default
// network for the client session.
func (c *Client) SetNetwork(ctx context.Context, networkName string) error {
	return c.authManager.SetNetwork(ctx, networkName)
}

// SetNetworkByID sets the active network for subsequent device operations by ID.
func (c *Client) SetNetworkByID(ctx context.Context, networkID int) error {
	return c.authManager.SetNetworkByID(ctx, networkID)
}

// GetNetworks retrieves all networks accessible to the authenticated user.
//
// This method requires authentication but does not require a network context.
func (c *Client) GetNetworks(ctx context.Context) ([]Network, error) {
	return c.authManager.GetNetworks(ctx)
}

// GetCurrentNetwork returns the currently selected network.
//
// Returns an error if no network has been selected.
func (c *Client) GetCurrentNetwork(ctx context.Context) (*Network, error) {
	return c.authManager.GetCurrentNetwork()
}

// IsAuthenticated returns true if the client has a valid access token.
func (c *Client) IsAuthenticated() bool {
	return c.authManager.IsAuthenticated()
}

// IsNetworkSet returns true if a network context has been established.
func (c *Client) IsNetworkSet() bool {
	return c.authManager.IsNetworkSet()
}

// GetAccessToken returns the current access token.
// Returns an error if not authenticated.
func (c *Client) GetAccessToken() (string, error) {
	return c.authManager.GetToken()
}

// TokenExpiresAt returns the absolute expiry time of the current access token.
// The zero time means no token is held. This lets callers cache the token
// across invocations without having to decode the JWT themselves.
func (c *Client) TokenExpiresAt() time.Time {
	return c.authManager.ExpiresAt()
}

// Config returns a copy of the client configuration.
func (c *Client) Config() config.Config {
	return *c.config
}

// EnsureReady ensures the client is authenticated and has a network context.
//
// This is a convenience method that calls Authenticate() and then attempts
// to set a network if one is configured but not yet set.
func (c *Client) EnsureReady(ctx context.Context) error {
	// Ensure authentication
	if err := c.Authenticate(ctx); err != nil {
		return err
	}

	// Ensure network context if configured
	if c.config.NetworkName != "" && !c.IsNetworkSet() {
		if err := c.SetNetwork(ctx, c.config.NetworkName); err != nil {
			return err
		}
	}

	return nil
}

// WithAuthentication executes a function with authentication ensured.
//
// This is a convenience method for operations that require authentication
// but not necessarily a network context.
func (c *Client) WithAuthentication(ctx context.Context, fn func() error) error {
	if err := c.Authenticate(ctx); err != nil {
		return err
	}
	return fn()
}

// WithNetworkContext executes a function with both authentication and network context ensured.
//
// This is a convenience method for operations that require both authentication
// and a network to be selected.
func (c *Client) WithNetworkContext(ctx context.Context, fn func() error) error {
	if err := c.EnsureReady(ctx); err != nil {
		return err
	}

	if !c.IsNetworkSet() {
		return errors.NewAuthError("no network selected - use SetNetwork() or configure BS_NETWORK", nil)
	}

	return fn()
}
