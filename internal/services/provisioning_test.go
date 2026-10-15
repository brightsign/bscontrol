package services

import (
	"context"
	"testing"

	"github.com/brightsign/bscontrol/internal/auth"
	"github.com/brightsign/bscontrol/internal/config"
	"github.com/brightsign/bscontrol/internal/http"
)

// TestProvisioningServiceInterface verifies that our implementation satisfies the interface
func TestProvisioningServiceInterface(t *testing.T) {
	var _ ProvisioningService = (*provisioningService)(nil)
}

// Helper function to create a test provisioning service
func createTestProvisioningService() ProvisioningService {
	cfg := config.DefaultConfig()
	cfg.ClientID = "test-id"
	cfg.ClientSecret = "test-secret"

	httpClient := http.NewHTTPClient(cfg)
	authManager := auth.NewAuthManager(cfg, httpClient)

	return NewProvisioningService(cfg, httpClient, authManager)
}

func TestProvisioningService_RevokeRegistrationToken(t *testing.T) {
	service := createTestProvisioningService()
	ctx := context.Background()

	// Test with empty token
	err := service.RevokeRegistrationToken(ctx, "")
	if err == nil {
		t.Error("Expected error when revoking with empty token")
	}

	// Test without authentication should fail
	err = service.RevokeRegistrationToken(ctx, "test-registration-token")
	if err == nil {
		t.Error("Expected error when revoking token without authentication")
	}
}

func TestProvisioningService_ValidateDeviceToken(t *testing.T) {
	service := createTestProvisioningService()
	ctx := context.Background()

	// Test with empty token
	_, err := service.ValidateDeviceToken(ctx, "")
	if err == nil {
		t.Error("Expected error when validating with empty token")
	}

	// Test without authentication should fail
	_, err = service.ValidateDeviceToken(ctx, "test-registration-token")
	if err == nil {
		t.Error("Expected error when validating token without authentication")
	}
}
