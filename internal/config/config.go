package config

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/brightsign/bscontrol/internal/errors"
	"github.com/brightsign/bscontrol/internal/types"
)

// AccessToken is a bearer token together with its absolute expiry time.
type AccessToken struct {
	Value     string
	ExpiresAt time.Time
}

// TokenSource supplies bearer tokens on demand. Callers implement it to own
// token acquisition and caching (for example, to share one cached token across
// processes), instead of letting the SDK run the client-credentials grant.
//
// Token must return a currently-valid token or an error. The SDK calls it
// whenever it needs a token and treats a token within its refresh margin of
// expiry as needing a fresh call.
type TokenSource interface {
	Token(ctx context.Context) (AccessToken, error)
}

// Config holds all configuration for the BSN.cloud SDK client.
type Config struct {
	// Authentication credentials
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`

	// Network and API settings
	NetworkName string `json:"network_name,omitempty"`
	APIVersion  string `json:"api_version"`

	// Endpoint URLs
	BSNBaseURL      string `json:"bsn_base_url"`
	ProvisioningURL string `json:"provisioning_url"`
	RDWSBaseURL     string `json:"rdws_base_url"`
	TokenEndpoint   string `json:"token_endpoint"`

	// HTTP client settings
	Timeout    time.Duration `json:"timeout"`
	RetryCount int           `json:"retry_count"`
	Debug      bool          `json:"debug"` // Enable debug logging of HTTP requests/responses

	// Optional device settings
	DeviceSerial string `json:"device_serial,omitempty"`

	// Pre-loaded access token (for session reuse across CLI invocations)
	AccessToken string    `json:"-"`
	ExpiresAt   time.Time `json:"-"`

	// Optional caller-supplied token source. When set, the SDK obtains tokens
	// from it instead of running the client-credentials grant itself.
	TokenSource TokenSource `json:"-"`

	// Optional per-request timing callback. When set, the SDK enables transport
	// tracing and reports a RequestTrace for every request. Nil (the default)
	// means no tracing and no overhead.
	RequestTracer types.RequestTracer `json:"-"`
}

// DefaultConfig returns a Config with sensible default values.
func DefaultConfig() *Config {
	return &Config{
		APIVersion:      "2022/06/REST",
		BSNBaseURL:      "https://api.bsn.cloud",
		ProvisioningURL: "https://provision.bsn.cloud",
		RDWSBaseURL:     "https://ws.bsn.cloud/rest/v1",
		TokenEndpoint:   "https://auth.bsn.cloud/realms/bsncloud/protocol/openid-connect/token",
		Timeout:         30 * time.Second,
		RetryCount:      3,
	}
}

// WithEnv loads credentials and the network name from the environment:
// BS_CLIENT_ID, BS_SECRET, and BS_NETWORK. It is opt-in: New does not read the
// environment unless this option is passed.
//
// Options are applied in order and the last write wins, so place WithEnv before
// explicit options to use the environment as a base that those options override,
// or after them to let the environment take precedence.
func WithEnv() Option {
	return func(c *Config) error {
		c.LoadFromEnv()
		return nil
	}
}

// WithTokenSource supplies a caller-owned TokenSource. When set, the SDK obtains
// bearer tokens from it instead of running the client-credentials grant, which
// lets the caller own token caching and sharing. ClientID/ClientSecret are then
// not required.
func WithTokenSource(ts TokenSource) Option {
	return func(c *Config) error {
		if ts == nil {
			return fmt.Errorf("token source cannot be nil")
		}
		c.TokenSource = ts
		return nil
	}
}

// LoadFromEnv loads configuration values from environment variables.
// Environment variables take precedence over any existing values in the config.
func (c *Config) LoadFromEnv() {
	if clientID := os.Getenv("BS_CLIENT_ID"); clientID != "" {
		c.ClientID = clientID
	}
	if clientSecret := os.Getenv("BS_SECRET"); clientSecret != "" {
		c.ClientSecret = clientSecret
	}
	if networkName := os.Getenv("BS_NETWORK"); networkName != "" {
		c.NetworkName = networkName
	}
}

// Validate checks that the configuration contains all required fields and valid values.
func (c *Config) Validate() error {
	// Credentials are required only when the SDK acquires tokens itself. A
	// caller-supplied TokenSource owns acquisition, so it needs neither.
	if c.TokenSource == nil {
		if c.ClientID == "" {
			return errors.NewConfigError("ClientID", "field is required", "pass WithCredentials or WithEnv (BS_CLIENT_ID)")
		}

		if c.ClientSecret == "" {
			return errors.NewConfigError("ClientSecret", "field is required", "pass WithCredentials or WithEnv (BS_SECRET)")
		}
	}

	if c.BSNBaseURL == "" {
		return errors.NewConfigError("BSNBaseURL", "field is required", "")
	}

	if c.ProvisioningURL == "" {
		return errors.NewConfigError("ProvisioningURL", "field is required", "")
	}

	if c.RDWSBaseURL == "" {
		return errors.NewConfigError("RDWSBaseURL", "field is required", "")
	}

	if c.TokenEndpoint == "" {
		return errors.NewConfigError("TokenEndpoint", "field is required", "")
	}

	if c.Timeout <= 0 {
		return errors.NewConfigError("Timeout", "must be positive", "")
	}

	if c.RetryCount < 0 {
		return errors.NewConfigError("RetryCount", "cannot be negative", "")
	}

	return nil
}

// Option is a function that configures the SDK client.
type Option func(*Config) error

// WithCredentials sets the BSN.cloud OAuth2 credentials.
//
// These credentials are required for authentication. They can be obtained
// from the BSN.cloud Admin Panel under API Access.
func WithCredentials(clientID, clientSecret string) Option {
	return func(c *Config) error {
		c.ClientID = clientID
		c.ClientSecret = clientSecret
		return nil
	}
}

// WithNetwork sets the default network name for device operations.
//
// Many device operations require a network context. Setting this allows
// the SDK to automatically select the specified network.
func WithNetwork(networkName string) Option {
	return func(c *Config) error {
		c.NetworkName = networkName
		return nil
	}
}

// WithRequestTracer registers a callback invoked once per HTTP request the SDK
// completes, carrying the request's timing and outcome (see types.RequestTrace).
//
// Tracing is off unless a tracer is set: with no tracer the SDK does not enable
// transport tracing and adds no overhead. The callback runs synchronously on the
// request goroutine and must not block.
func WithRequestTracer(tracer types.RequestTracer) Option {
	return func(c *Config) error {
		c.RequestTracer = tracer
		return nil
	}
}

// WithTimeout sets the HTTP request timeout for API calls.
//
// This applies to all HTTP requests made by the SDK. The default is 30 seconds.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Config) error {
		if timeout <= 0 {
			return errors.NewConfigError("Timeout", "must be positive", "")
		}
		c.Timeout = timeout
		return nil
	}
}

// WithRetryCount sets the number of retry attempts for failed API requests.
//
// The SDK will automatically retry failed requests up to this number of times.
// The default is 3 retries.
func WithRetryCount(count int) Option {
	return func(c *Config) error {
		if count < 0 {
			return errors.NewConfigError("RetryCount", "cannot be negative", "")
		}
		c.RetryCount = count
		return nil
	}
}

// WithDebug enables debug logging of all HTTP requests and responses.
//
// When enabled, the SDK will print detailed information about every API call
// including request/response headers and bodies. This is useful for debugging
// but should not be enabled in production as it may expose sensitive data.
func WithDebug(debug bool) Option {
	return func(c *Config) error {
		c.Debug = debug
		return nil
	}
}

// WithEndpoints sets custom API endpoints for BSN.cloud and RDWS.
//
// This is primarily useful for testing or when using private cloud deployments.
func WithEndpoints(bsnURL, rdwsURL string) Option {
	return func(c *Config) error {
		if bsnURL == "" {
			return fmt.Errorf("BSN base URL cannot be empty")
		}
		if rdwsURL == "" {
			return fmt.Errorf("RDWS base URL cannot be empty")
		}
		c.BSNBaseURL = bsnURL
		c.RDWSBaseURL = rdwsURL
		return nil
	}
}

// WithTokenEndpoint sets a custom OAuth2 token endpoint.
//
// This is primarily useful for testing or when using private cloud deployments.
func WithTokenEndpoint(endpoint string) Option {
	return func(c *Config) error {
		if endpoint == "" {
			return fmt.Errorf("token endpoint cannot be empty")
		}
		c.TokenEndpoint = endpoint
		return nil
	}
}

// WithDeviceSerial sets a default device serial number for single-device operations.
//
// This is optional and can be useful when the SDK is primarily used to manage
// a single device.
func WithDeviceSerial(serial string) Option {
	return func(c *Config) error {
		c.DeviceSerial = serial
		return nil
	}
}

// WithAccessToken sets a pre-loaded access token for session reuse.
//
// This allows CLI tools to cache the bearer token between invocations,
// skipping the OAuth round-trip when the token is still valid. The expiresAt
// parameter should be the absolute time when the token expires.
//
// If the token is expired or near expiry, the SDK will automatically
// re-authenticate using client credentials.
func WithAccessToken(token string, expiresAt time.Time) Option {
	return func(c *Config) error {
		if token == "" {
			return nil // silently ignore empty token, will fall back to normal auth
		}
		c.AccessToken = token
		c.ExpiresAt = expiresAt
		return nil
	}
}

// WithOIDCURL sets the OIDC base URL and derives the token endpoint from it.
//
// This is useful when you have the OIDC URL (e.g., from Lookout config) rather than
// the direct token endpoint. The token endpoint is constructed as:
// {oidcURL}/protocol/openid-connect/token
func WithOIDCURL(oidcURL string) Option {
	return func(c *Config) error {
		if oidcURL == "" {
			return fmt.Errorf("OIDC URL cannot be empty")
		}
		// Construct token endpoint from OIDC base URL
		c.TokenEndpoint = oidcURL + "/protocol/openid-connect/token"
		return nil
	}
}

// WithProvisioningEndpoint sets a custom provisioning endpoint.
//
// This is primarily useful for testing or when using private cloud deployments.
func WithProvisioningEndpoint(url string) Option {
	return func(c *Config) error {
		if url == "" {
			return fmt.Errorf("provisioning endpoint cannot be empty")
		}
		c.ProvisioningURL = url
		return nil
	}
}
