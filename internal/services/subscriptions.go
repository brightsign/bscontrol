package services

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/brightsign/bscontrol/internal/auth"
	"github.com/brightsign/bscontrol/internal/config"
	"github.com/brightsign/bscontrol/internal/http"
	"github.com/brightsign/bscontrol/internal/types"
)

// SubscriptionService provides subscription management operations.
type SubscriptionService interface {
	List(ctx context.Context, opts ...ListOption) (*types.SubscriptionList, error)
	GetCount(ctx context.Context) (*types.SubscriptionCount, error)
	GetOperations(ctx context.Context) (*types.SubscriptionOperations, error)
}

// subscriptionService implements the SubscriptionService interface.
type subscriptionService struct {
	config      *config.Config
	httpClient  *http.HTTPClient
	authManager *auth.AuthManager
}

// NewSubscriptionService creates a new subscription service.
func NewSubscriptionService(cfg *config.Config, httpClient *http.HTTPClient, authManager *auth.AuthManager) SubscriptionService {
	return &subscriptionService{
		config:      cfg,
		httpClient:  httpClient,
		authManager: authManager,
	}
}

// List retrieves a list of device subscriptions with optional filtering and pagination.
func (s *subscriptionService) List(ctx context.Context, opts ...ListOption) (*types.SubscriptionList, error) {
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
	baseURL := fmt.Sprintf("%s/%s/Subscriptions", s.config.BSNBaseURL, s.config.APIVersion)
	if len(params) > 0 {
		baseURL += "?" + params.Encode()
	}

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Make the request
	var result types.SubscriptionList
	if err := s.httpClient.GetWithAuth(ctx, token, baseURL, &result); err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	return &result, nil
}

// GetCount retrieves the number of subscription instances on the network.
func (s *subscriptionService) GetCount(ctx context.Context) (*types.SubscriptionCount, error) {
	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Build URL
	url := fmt.Sprintf("%s/%s/Subscriptions/Count", s.config.BSNBaseURL, s.config.APIVersion)

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Make the request
	var result types.SubscriptionCount
	if err := s.httpClient.GetWithAuth(ctx, token, url, &result); err != nil {
		return nil, fmt.Errorf("failed to get subscription count: %w", err)
	}

	return &result, nil
}

// GetOperations returns operational permissions granted to roles for subscriptions.
func (s *subscriptionService) GetOperations(ctx context.Context) (*types.SubscriptionOperations, error) {
	// Ensure we have authentication and network context
	if err := s.authManager.EnsureValid(ctx); err != nil {
		return nil, err
	}

	if err := s.authManager.EnsureNetworkSet(ctx); err != nil {
		return nil, err
	}

	// Build URL
	url := fmt.Sprintf("%s/%s/Subscriptions/Operations", s.config.BSNBaseURL, s.config.APIVersion)

	// Get access token
	token, err := s.authManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Make the request
	var result types.SubscriptionOperations
	if err := s.httpClient.GetWithAuth(ctx, token, url, &result); err != nil {
		return nil, fmt.Errorf("failed to get subscription operations: %w", err)
	}

	return &result, nil
}
