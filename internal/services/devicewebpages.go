package services

import (
	"context"
	"fmt"

	"github.com/brightsign/bscontrol/internal/auth"
	"github.com/brightsign/bscontrol/internal/config"
	"github.com/brightsign/bscontrol/internal/errors"
	"github.com/brightsign/bscontrol/internal/http"
	"github.com/brightsign/bscontrol/internal/types"
)

// DeviceWebPageService provides device web page management operations.
type DeviceWebPageService interface {
	// List retrieves all device web pages
	List(ctx context.Context) (*types.DeviceWebPageList, error)

	// GetByID retrieves a specific device web page by ID
	GetByID(ctx context.Context, id int) (*types.DeviceWebPage, error)

	// GetDefault retrieves the default presentation web page
	GetDefault(ctx context.Context) (*types.DeviceWebPage, error)
}

// deviceWebPageService implements the DeviceWebPageService interface.
type deviceWebPageService struct {
	config      *config.Config
	httpClient  *http.HTTPClient
	authManager *auth.AuthManager
}

// NewDeviceWebPageService creates a new device web page service.
func NewDeviceWebPageService(cfg *config.Config, httpClient *http.HTTPClient, authManager *auth.AuthManager) DeviceWebPageService {
	return &deviceWebPageService{
		config:      cfg,
		httpClient:  httpClient,
		authManager: authManager,
	}
}

// List retrieves all device web pages.
func (s *deviceWebPageService) List(ctx context.Context) (*types.DeviceWebPageList, error) {
	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Build URL
	webPagesURL := fmt.Sprintf("%s/%s/DeviceWebPages/",
		s.config.BSNBaseURL, s.config.APIVersion)

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Make the request
	var result types.DeviceWebPageList
	if err := s.httpClient.GetWithAuth(ctx, token, webPagesURL, &result); err != nil {
		return nil, fmt.Errorf("failed to list device web pages: %w", err)
	}

	return &result, nil
}

// GetByID retrieves a specific device web page by ID.
func (s *deviceWebPageService) GetByID(ctx context.Context, id int) (*types.DeviceWebPage, error) {
	if id <= 0 {
		return nil, errors.NewValidationError("id", id, "device web page ID must be positive")
	}

	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Build URL
	webPageURL := fmt.Sprintf("%s/%s/DeviceWebPages/%d/",
		s.config.BSNBaseURL, s.config.APIVersion, id)

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Make the request
	var result types.DeviceWebPage
	if err := s.httpClient.GetWithAuth(ctx, token, webPageURL, &result); err != nil {
		return nil, fmt.Errorf("failed to get device web page with ID %d: %w", id, err)
	}

	return &result, nil
}

// GetDefault retrieves the default presentation web page.
// This is required when creating presentations to reference the default device web page.
func (s *deviceWebPageService) GetDefault(ctx context.Context) (*types.DeviceWebPage, error) {
	// List all web pages
	list, err := s.List(ctx)
	if err != nil {
		return nil, err
	}

	// Find the default presentation web page
	for _, page := range list.Items {
		if page.Name == "Default_PresentationWebPage" {
			return &page, nil
		}
	}

	return nil, errors.NewAPIError(404, "devicewebpage_default_not_found",
		"Default device web page not found", "No web page with name 'Default_PresentationWebPage' exists")
}
