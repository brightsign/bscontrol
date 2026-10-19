package services

import (
	"context"
	"fmt"
	"net/url"

	"github.com/brightsign/bscontrol/internal/auth"
	"github.com/brightsign/bscontrol/internal/config"
	"github.com/brightsign/bscontrol/internal/errors"
	"github.com/brightsign/bscontrol/internal/http"
	"github.com/brightsign/bscontrol/internal/types"
)

// ProvisioningService provides player provisioning operations including token generation.
type ProvisioningService interface {
	GenerateDeviceToken(ctx context.Context) (*types.BSNTokenEntity, error)
	ValidateDeviceToken(ctx context.Context, token string) (*types.BSNTokenEntity, error)
	RevokeRegistrationToken(ctx context.Context, token string) error
}

// provisioningService implements the ProvisioningService interface.
type provisioningService struct {
	config      *config.Config
	httpClient  *http.HTTPClient
	authManager *auth.AuthManager
}

// NewProvisioningService creates a new provisioning service.
func NewProvisioningService(cfg *config.Config, httpClient *http.HTTPClient, authManager *auth.AuthManager) ProvisioningService {
	return &provisioningService{
		config:      cfg,
		httpClient:  httpClient,
		authManager: authManager,
	}
}

const provisioningTokenPath = "/2022/06/REST/Provisioning/Setups/Tokens/"

// GenerateDeviceToken generates a new device registration token for the current network.
//
// This token allows BrightSign players to register themselves with BSN.cloud.
// The token is valid for 2 years by default and has "cert" scope, allowing
// multiple devices to use the same token.
//
// Required scope: bsn.api.main.devices.setups.token.create
//
// The returned token should be embedded in B-Deploy setup records to enable
// player registration during provisioning.
func (s *provisioningService) GenerateDeviceToken(ctx context.Context) (*types.BSNTokenEntity, error) {
	// Ensure we have authentication
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Make the API request - POST to generate token
	var response types.BSNTokenEntity
	var provisioningTokenURL = fmt.Sprintf("%s%s", s.config.BSNBaseURL, provisioningTokenPath)
	err = s.httpClient.PostWithAuth(ctx, token, provisioningTokenURL, nil, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to generate device registration token: %w", err)
	}

	// Validate response has required fields
	if response.Token == "" {
		return nil, errors.NewAppError("invalid_token_response",
			"API returned empty token", "token field is empty")
	}

	if response.Scope == "" {
		// Default to "cert" if not provided
		response.Scope = "cert"
	}

	return &response, nil
}

// ValidateDeviceToken validates a device registration token and retrieves its metadata.
//
// Required scope: bsn.api.main.devices.setups.token.validate
func (s *provisioningService) ValidateDeviceToken(ctx context.Context, tokenValue string) (*types.BSNTokenEntity, error) {
	if tokenValue == "" {
		return nil, errors.NewValidationError("token", tokenValue, "token cannot be empty")
	}

	// Ensure we have authentication
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Build the token validation endpoint
	validateURL := fmt.Sprintf("%s%s%s/", s.config.BSNBaseURL, provisioningTokenPath, url.PathEscape(tokenValue))

	// Make the API request
	var response types.BSNTokenEntity
	err = s.httpClient.GetWithAuth(ctx, token, validateURL, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to validate device registration token: %w", err)
	}

	return &response, nil
}

// RevokeRegistrationToken revokes a device registration token on the current network.
//
// Once revoked, players can no longer use the token to register with BSN.cloud.
// Tokens with "cert" scope are shared by every device provisioned with them, so
// revoking one breaks registration for all of them.
//
// Required scope: bsn.api.main.devices.setups.token.delete
func (s *provisioningService) RevokeRegistrationToken(ctx context.Context, tokenValue string) error {
	if tokenValue == "" {
		return errors.NewValidationError("token", tokenValue, "token cannot be empty")
	}

	// Ensure we have authentication
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return err
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return err
	}

	// Build the token revocation endpoint
	revokeURL := fmt.Sprintf("%s%s%s/", s.config.BSNBaseURL, provisioningTokenPath, url.PathEscape(tokenValue))

	// Make the API request - DELETE returns no content on success
	err = s.httpClient.DeleteWithAuth(ctx, token, revokeURL, nil)
	if err != nil {
		return fmt.Errorf("failed to revoke device registration token: %w", err)
	}

	return nil
}
