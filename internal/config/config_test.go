package config

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Check default values
	if config.APIVersion != "2022/06/REST" {
		t.Errorf("Expected API version '2022/06/REST', got '%s'", config.APIVersion)
	}

	if config.BSNBaseURL != "https://api.bsn.cloud" {
		t.Errorf("Expected BSN base URL 'https://api.bsn.cloud', got '%s'", config.BSNBaseURL)
	}

	if config.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", config.Timeout)
	}

	if config.RetryCount != 3 {
		t.Errorf("Expected retry count 3, got %d", config.RetryCount)
	}
}

func TestConfigLoadFromEnv(t *testing.T) {
	// Save original env vars
	originalClientID := os.Getenv("BS_CLIENT_ID")
	originalSecret := os.Getenv("BS_SECRET")
	originalNetwork := os.Getenv("BS_NETWORK")

	// Set test values
	os.Setenv("BS_CLIENT_ID", "test-client-id")
	os.Setenv("BS_SECRET", "test-secret")
	os.Setenv("BS_NETWORK", "test-network")

	// Clean up after test
	defer func() {
		if originalClientID == "" {
			os.Unsetenv("BS_CLIENT_ID")
		} else {
			os.Setenv("BS_CLIENT_ID", originalClientID)
		}
		if originalSecret == "" {
			os.Unsetenv("BS_SECRET")
		} else {
			os.Setenv("BS_SECRET", originalSecret)
		}
		if originalNetwork == "" {
			os.Unsetenv("BS_NETWORK")
		} else {
			os.Setenv("BS_NETWORK", originalNetwork)
		}
	}()

	config := DefaultConfig()
	config.LoadFromEnv()

	if config.ClientID != "test-client-id" {
		t.Errorf("Expected client ID 'test-client-id', got '%s'", config.ClientID)
	}

	if config.ClientSecret != "test-secret" {
		t.Errorf("Expected client secret 'test-secret', got '%s'", config.ClientSecret)
	}

	if config.NetworkName != "test-network" {
		t.Errorf("Expected network name 'test-network', got '%s'", config.NetworkName)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorString string
	}{
		{
			name: "valid config",
			config: &Config{
				ClientID:        "test-id",
				ClientSecret:    "test-secret",
				BSNBaseURL:      "https://api.bsn.cloud",
				ProvisioningURL: "https://provision.bsn.cloud",
				RDWSBaseURL:     "https://ws.bsn.cloud/rest/v1",
				TokenEndpoint:   "https://auth.bsn.cloud/realms/bsncloud/protocol/openid-connect/token",
				Timeout:         30 * time.Second,
				RetryCount:      3,
			},
			expectError: false,
		},
		{
			name: "missing client ID",
			config: &Config{
				ClientSecret:  "test-secret",
				BSNBaseURL:    "https://api.bsn.cloud",
				RDWSBaseURL:   "https://ws.bsn.cloud/rest/v1",
				TokenEndpoint: "https://auth.bsn.cloud/realms/bsncloud/protocol/openid-connect/token",
				Timeout:       30 * time.Second,
				RetryCount:    3,
			},
			expectError: true,
			errorString: "configuration error: ClientID - field is required (suggestion: pass WithCredentials or WithEnv (BS_CLIENT_ID))",
		},
		{
			name: "missing client secret",
			config: &Config{
				ClientID:      "test-id",
				BSNBaseURL:    "https://api.bsn.cloud",
				RDWSBaseURL:   "https://ws.bsn.cloud/rest/v1",
				TokenEndpoint: "https://auth.bsn.cloud/realms/bsncloud/protocol/openid-connect/token",
				Timeout:       30 * time.Second,
				RetryCount:    3,
			},
			expectError: true,
			errorString: "configuration error: ClientSecret - field is required (suggestion: pass WithCredentials or WithEnv (BS_SECRET))",
		},
		{
			name: "invalid timeout",
			config: &Config{
				ClientID:        "test-id",
				ClientSecret:    "test-secret",
				BSNBaseURL:      "https://api.bsn.cloud",
				ProvisioningURL: "https://provision.bsn.cloud",
				RDWSBaseURL:     "https://ws.bsn.cloud/rest/v1",
				TokenEndpoint:   "https://auth.bsn.cloud/realms/bsncloud/protocol/openid-connect/token",
				Timeout:         0,
				RetryCount:      3,
			},
			expectError: true,
			errorString: "configuration error: Timeout - must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				if err == nil {
					t.Error("Expected validation error but got none")
				} else if tt.errorString != "" && err.Error() != tt.errorString {
					t.Errorf("Expected error '%s', got '%s'", tt.errorString, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no validation error but got: %v", err)
				}
			}
		})
	}
}

func TestOptions(t *testing.T) {
	config := DefaultConfig()

	// Test WithCredentials
	opt := WithCredentials("test-id", "test-secret")
	err := opt(config)
	if err != nil {
		t.Fatalf("WithCredentials failed: %v", err)
	}

	if config.ClientID != "test-id" {
		t.Errorf("Expected client ID 'test-id', got '%s'", config.ClientID)
	}

	if config.ClientSecret != "test-secret" {
		t.Errorf("Expected client secret 'test-secret', got '%s'", config.ClientSecret)
	}

	// Test WithNetwork
	opt = WithNetwork("test-network")
	err = opt(config)
	if err != nil {
		t.Fatalf("WithNetwork failed: %v", err)
	}

	if config.NetworkName != "test-network" {
		t.Errorf("Expected network name 'test-network', got '%s'", config.NetworkName)
	}

	// Test WithTimeout
	opt = WithTimeout(60 * time.Second)
	err = opt(config)
	if err != nil {
		t.Fatalf("WithTimeout failed: %v", err)
	}

	if config.Timeout != 60*time.Second {
		t.Errorf("Expected timeout 60s, got %v", config.Timeout)
	}

	// Test invalid timeout
	opt = WithTimeout(0)
	err = opt(config)
	if err == nil {
		t.Error("Expected error for invalid timeout but got none")
	}
}
func TestWithEnvLoadsCredentialsAndNetwork(t *testing.T) {
	t.Setenv("BS_CLIENT_ID", "env-id")
	t.Setenv("BS_SECRET", "env-secret")
	t.Setenv("BS_NETWORK", "env-network")

	cfg := DefaultConfig()
	if err := WithEnv()(cfg); err != nil {
		t.Fatalf("WithEnv: %v", err)
	}
	if cfg.ClientID != "env-id" || cfg.ClientSecret != "env-secret" || cfg.NetworkName != "env-network" {
		t.Errorf("WithEnv did not load all values: %+v", cfg)
	}
}

// Options apply in order, last write wins: an explicit option after WithEnv
// overrides the environment.
func TestExplicitOptionOverridesEnv(t *testing.T) {
	t.Setenv("BS_CLIENT_ID", "env-id")
	t.Setenv("BS_SECRET", "env-secret")

	cfg := DefaultConfig()
	for _, opt := range []Option{WithEnv(), WithCredentials("explicit-id", "explicit-secret")} {
		if err := opt(cfg); err != nil {
			t.Fatalf("option: %v", err)
		}
	}
	if cfg.ClientID != "explicit-id" || cfg.ClientSecret != "explicit-secret" {
		t.Errorf("explicit credentials did not override env: %+v", cfg)
	}
}

type dummyTokenSource struct{}

func (dummyTokenSource) Token(context.Context) (AccessToken, error) {
	return AccessToken{Value: "x", ExpiresAt: time.Now().Add(time.Hour)}, nil
}

// With a TokenSource the SDK owns no credentials, so Validate must not require
// ClientID/ClientSecret.
func TestValidatePassesWithTokenSourceAndNoCredentials(t *testing.T) {
	cfg := DefaultConfig()
	cfg.TokenSource = dummyTokenSource{}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate with TokenSource and no credentials: %v", err)
	}
}

func TestWithTokenSourceRejectsNil(t *testing.T) {
	if err := WithTokenSource(nil)(DefaultConfig()); err == nil {
		t.Error("WithTokenSource(nil) should error")
	}
}
