package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/brightsign/bscontrol/internal/config"
	"github.com/brightsign/bscontrol/internal/errors"
	"github.com/brightsign/bscontrol/internal/http"
	"github.com/brightsign/bscontrol/internal/types"
)

// refreshMargin is how close to expiry a token may get before the SDK treats it
// as needing refresh. A call that starts inside this margin re-authenticates
// first, so it never races the token expiring mid-flight.
const refreshMargin = 30 * time.Second

// AuthManager handles OAuth2 authentication and network selection for BSN.cloud.
type AuthManager struct {
	config      *config.Config
	httpClient  *http.HTTPClient
	mu          sync.RWMutex
	accessToken string
	expiresAt   time.Time

	// networkSet / currentNetwork track what is applied to the CURRENT server
	// session. A re-authentication clears them because the new session starts
	// with no network.
	networkSet     bool
	currentNetwork *types.Network

	// desiredNetworkName / desiredNetworkID record the network the caller last
	// asked for. They SURVIVE re-authentication, so after a token refresh the
	// SDK re-applies the caller's chosen network instead of falling back to the
	// environment. This is the fix for the cross-network wire-up hazard: the
	// network you select is the network you keep, for the life of the client.
	desiredNetworkName string
	desiredNetworkID   int
}

// NewAuthManager creates a new authentication manager.
func NewAuthManager(cfg *config.Config, httpClient *http.HTTPClient) *AuthManager {
	am := &AuthManager{
		config:     cfg,
		httpClient: httpClient,
		// Seed the desired network from configuration (WithNetwork/WithEnv), so a
		// configured network is still selected lazily on the first call that
		// needs one. An explicit SetNetwork later replaces this.
		desiredNetworkName: cfg.NetworkName,
	}

	// If a pre-loaded access token was provided, use it
	if cfg.AccessToken != "" && !cfg.ExpiresAt.IsZero() {
		am.accessToken = cfg.AccessToken
		am.expiresAt = cfg.ExpiresAt
	}

	return am
}

// Authenticate performs OAuth2 client credentials authentication.
func (a *AuthManager) Authenticate(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if we already have a valid token
	if a.accessToken != "" && time.Until(a.expiresAt) > refreshMargin {
		return nil
	}

	// A caller-supplied token source owns acquisition and caching; otherwise run
	// the client-credentials grant ourselves.
	if a.config.TokenSource != nil {
		tok, err := a.config.TokenSource.Token(ctx)
		if err != nil {
			return errors.NewAuthError("token source failed", err)
		}
		if tok.Value == "" {
			return errors.NewAuthError("token source returned an empty token", nil)
		}
		a.accessToken = tok.Value
		a.expiresAt = tok.ExpiresAt
	} else {
		formData := map[string]string{
			"grant_type": "client_credentials",
		}

		var tokenResp types.TokenResponse
		err := a.httpClient.PostFormWithAuth(
			ctx,
			a.config.ClientID,
			a.config.ClientSecret,
			a.config.TokenEndpoint,
			formData,
			&tokenResp,
		)
		if err != nil {
			return errors.NewAuthError("failed to get access token", err)
		}

		a.accessToken = tokenResp.AccessToken
		a.expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	// The new token begins a fresh server session with no network applied. The
	// desired network is retained (see struct doc) and re-applied on the next
	// call that needs a network context.
	a.networkSet = false
	a.currentNetwork = nil

	return nil
}

// SetNetwork sets the active network for API operations.
func (a *AuthManager) SetNetwork(ctx context.Context, networkName string) error {
	if networkName == "" {
		return errors.NewValidationError("networkName", networkName, "network name cannot be empty")
	}

	// Ensure we have a valid token
	if err := a.EnsureValid(ctx); err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Record the caller's choice so it survives a later token refresh.
	a.desiredNetworkName = networkName
	a.desiredNetworkID = 0

	// Check if we already have this network set
	if a.networkSet && a.currentNetwork != nil && a.currentNetwork.Name == networkName {
		return nil
	}

	// Set the network via API
	networkReq := types.NetworkRequest{
		Name: networkName,
	}

	url := fmt.Sprintf("%s/%s/Self/Session/Network", a.config.BSNBaseURL, a.config.APIVersion)
	err := a.httpClient.PutWithAuth(ctx, a.accessToken, url, networkReq, nil)
	if err != nil {
		return errors.NewAuthError(fmt.Sprintf("failed to set network '%s'", networkName), err)
	}

	// Update network state
	a.networkSet = true
	a.currentNetwork = &types.Network{Name: networkName}

	return nil
}

// SetNetworkByID sets the active network by ID.
func (a *AuthManager) SetNetworkByID(ctx context.Context, networkID int) error {
	if networkID <= 0 {
		return errors.NewValidationError("networkID", networkID, "network ID must be positive")
	}

	// Ensure we have a valid token
	if err := a.EnsureValid(ctx); err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Record the caller's choice so it survives a later token refresh.
	a.desiredNetworkID = networkID
	a.desiredNetworkName = ""

	// Check if we already have this network set
	if a.networkSet && a.currentNetwork != nil && a.currentNetwork.ID == networkID {
		return nil
	}

	// Set the network via API
	networkReq := types.NetworkRequest{
		ID: networkID,
	}

	url := fmt.Sprintf("%s/%s/Self/Session/Network", a.config.BSNBaseURL, a.config.APIVersion)
	err := a.httpClient.PutWithAuth(ctx, a.accessToken, url, networkReq, nil)
	if err != nil {
		return errors.NewAuthError(fmt.Sprintf("failed to set network ID %d", networkID), err)
	}

	// Update network state
	a.networkSet = true
	a.currentNetwork = &types.Network{ID: networkID}

	return nil
}

// GetNetworks retrieves the list of available networks.
func (a *AuthManager) GetNetworks(ctx context.Context) ([]types.Network, error) {
	// Ensure we have a valid token
	if err := a.EnsureValid(ctx); err != nil {
		return nil, err
	}

	a.mu.RLock()
	token := a.accessToken
	a.mu.RUnlock()

	url := fmt.Sprintf("%s/%s/Self/Networks", a.config.BSNBaseURL, a.config.APIVersion)
	var networks []types.Network
	err := a.httpClient.GetWithAuth(ctx, token, url, &networks)
	if err != nil {
		return nil, errors.NewAuthError("failed to get networks", err)
	}

	return networks, nil
}

// ExpiresAt returns the absolute expiry time of the current access token, or
// the zero time if no token is held. Callers can use this to cache the token
// across invocations without decoding the JWT themselves.
func (a *AuthManager) ExpiresAt() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.expiresAt
}

// DesiredNetworkName returns the name of the network the caller last selected
// (via SetNetwork, or configured through WithNetwork/WithEnv), whether or not it
// is currently applied to the server session. It is empty when no network was
// selected by name. Callers that need the network name for a query filter read
// it here so there is a single source of network truth across the client.
func (a *AuthManager) DesiredNetworkName() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.desiredNetworkName
}

// GetToken returns the current access token.
func (a *AuthManager) GetToken() (string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.accessToken == "" {
		return "", errors.NewAuthError("not authenticated", nil)
	}

	if time.Until(a.expiresAt) <= refreshMargin {
		return "", errors.NewAuthError("token expired", nil)
	}

	return a.accessToken, nil
}

// IsAuthenticated returns true if the client has a valid access token.
func (a *AuthManager) IsAuthenticated() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.accessToken != "" && time.Until(a.expiresAt) > refreshMargin
}

// IsNetworkSet returns true if a network context has been established.
func (a *AuthManager) IsNetworkSet() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.networkSet
}

// GetCurrentNetwork returns the currently selected network.
func (a *AuthManager) GetCurrentNetwork() (*types.Network, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.networkSet || a.currentNetwork == nil {
		return nil, errors.NewAuthError("no network selected", nil)
	}

	// Return a copy to prevent external modifications
	network := *a.currentNetwork
	return &network, nil
}

// EnsureValid ensures that we have a valid access token.
func (a *AuthManager) EnsureValid(ctx context.Context) error {
	a.mu.RLock()
	needsRefresh := a.accessToken == "" || time.Until(a.expiresAt) <= refreshMargin
	a.mu.RUnlock()

	if needsRefresh {
		return a.Authenticate(ctx)
	}

	return nil
}

// EnsureNetworkSet ensures that a network context has been established on the
// current server session, re-applying the caller's desired network if a token
// refresh has reset the session.
//
// It re-applies ONLY the network the caller explicitly asked for (via
// SetNetwork/SetNetworkByID) or configured (WithNetwork/WithEnv). It never reads
// the environment here, so a mid-flight refresh cannot silently switch the
// client to a different network than the one in use.
func (a *AuthManager) EnsureNetworkSet(ctx context.Context) error {
	a.mu.RLock()
	networkSet := a.networkSet
	desiredName := a.desiredNetworkName
	desiredID := a.desiredNetworkID
	a.mu.RUnlock()

	if networkSet {
		return nil
	}

	switch {
	case desiredName != "":
		return a.SetNetwork(ctx, desiredName)
	case desiredID > 0:
		return a.SetNetworkByID(ctx, desiredID)
	default:
		return errors.NewAuthError("no network selected", nil)
	}
}

// WithValidToken executes a function with a valid access token.
func (a *AuthManager) WithValidToken(ctx context.Context, fn func(token string) error) error {
	if err := a.EnsureValid(ctx); err != nil {
		return err
	}

	token, err := a.GetToken()
	if err != nil {
		return err
	}

	return fn(token)
}
