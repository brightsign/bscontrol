package services

import (
	"context"
	"fmt"

	"github.com/brightsign/bscontrol/internal/auth"
	"github.com/brightsign/bscontrol/internal/config"
	"github.com/brightsign/bscontrol/internal/errors"
	bshttp "github.com/brightsign/bscontrol/internal/http"
	"github.com/brightsign/bscontrol/internal/types"
)

// PresentationService manages presentations through the Presentations REST API
// (2022/06).
type PresentationService interface {
	// List returns one page of presentations. Use WithMarker to continue.
	List(ctx context.Context, opts ...ListOption) (*types.PresentationList, error)

	// ListAll follows pagination and returns every matching presentation.
	ListAll(ctx context.Context, opts ...ListOption) ([]types.Presentation, error)

	// GetCount returns the number of presentations on the network.
	GetCount(ctx context.Context) (*types.PresentationCount, error)

	// GetByID returns one presentation.
	GetByID(ctx context.Context, id int) (*types.Presentation, error)

	// GetByName returns one presentation by its (unique) name.
	GetByName(ctx context.Context, name string) (*types.Presentation, error)

	// Create makes a presentation. Only Name is required; dates and the empty
	// lists the service insists on are filled in for any field left unset.
	Create(ctx context.Context, request *types.PresentationCreateRequest) (*types.Presentation, error)

	// Update replaces a presentation. The same defaults as Create apply.
	Update(ctx context.Context, id int, request *types.PresentationCreateRequest) (*types.Presentation, error)

	// DeleteByID removes one presentation.
	DeleteByID(ctx context.Context, id int) error

	// DeleteByName removes one presentation by name.
	DeleteByName(ctx context.Context, name string) error

	// DeleteByFilter removes every presentation matching a filter expression. The
	// filter is required; build values with EscapeFilterValue.
	DeleteByFilter(ctx context.Context, filter string) (*types.PresentationDeleteResult, error)
}

// presentationService implements the PresentationService interface.
type presentationService struct {
	config      *config.Config
	httpClient  *bshttp.HTTPClient
	authManager *auth.AuthManager
}

// NewPresentationService creates a new presentation service.
func NewPresentationService(cfg *config.Config, httpClient *bshttp.HTTPClient, authManager *auth.AuthManager) PresentationService {
	return &presentationService{config: cfg, httpClient: httpClient, authManager: authManager}
}

func (s *presentationService) baseURL() string {
	return fmt.Sprintf("%s/%s/Presentations", s.config.BSNBaseURL, s.config.APIVersion)
}

func validatePresentationID(id int) error {
	if id <= 0 {
		return errors.NewValidationError("id", id, "presentation ID must be positive")
	}
	return nil
}

func validatePresentationName(name string) error {
	if name == "" {
		return errors.NewValidationError("name", name, "presentation name cannot be empty")
	}
	return nil
}

// withPresentationDefaults returns a copy of req with the placeholders the
// service requires: the null date on both date fields and empty (not null) lists.
// The caller's request is left untouched.
func withPresentationDefaults(req *types.PresentationCreateRequest) *types.PresentationCreateRequest {
	out := *req
	if out.CreationDate == "" {
		out.CreationDate = types.PresentationNullDate
	}
	if out.LastModifiedDate == "" {
		out.LastModifiedDate = types.PresentationNullDate
	}
	if out.Files == nil {
		out.Files = []types.PresentationFile{}
	}
	if out.Applications == nil {
		out.Applications = []any{}
	}
	if out.Dependencies == nil {
		out.Dependencies = []any{}
	}
	if out.Permissions == nil {
		out.Permissions = []any{}
	}
	return &out
}

// List retrieves one page of presentations.
func (s *presentationService) List(ctx context.Context, opts ...ListOption) (*types.PresentationList, error) {
	token, err := ready(ctx, s.authManager)
	if err != nil {
		return nil, err
	}

	var result types.PresentationList
	if err := s.httpClient.GetWithAuth(ctx, token, withQuery(s.baseURL(), listQuery(opts)), &result); err != nil {
		return nil, fmt.Errorf("failed to list presentations: %w", err)
	}
	return &result, nil
}

// ListAll retrieves every matching presentation across pages.
func (s *presentationService) ListAll(ctx context.Context, opts ...ListOption) ([]types.Presentation, error) {
	return collectPages(func(marker string) ([]types.Presentation, bool, string, error) {
		page, err := s.List(ctx, withMarker(opts, marker)...)
		if err != nil {
			return nil, false, "", err
		}
		return page.Items, page.IsTruncated, page.NextMarker, nil
	})
}

// GetCount retrieves the number of presentations. The API answers with a bare
// number.
func (s *presentationService) GetCount(ctx context.Context) (*types.PresentationCount, error) {
	token, err := ready(ctx, s.authManager)
	if err != nil {
		return nil, err
	}

	var count flexCount
	if err := s.httpClient.GetWithAuth(ctx, token, s.baseURL()+"/Count", &count); err != nil {
		return nil, fmt.Errorf("failed to get presentation count: %w", err)
	}
	return &types.PresentationCount{Count: count.N}, nil
}

// GetByID retrieves one presentation.
func (s *presentationService) GetByID(ctx context.Context, id int) (*types.Presentation, error) {
	if err := validatePresentationID(id); err != nil {
		return nil, err
	}
	token, err := ready(ctx, s.authManager)
	if err != nil {
		return nil, err
	}

	var result types.Presentation
	if err := s.httpClient.GetWithAuth(ctx, token, fmt.Sprintf("%s/%d", s.baseURL(), id), &result); err != nil {
		return nil, fmt.Errorf("failed to get presentation %d: %w", id, err)
	}
	return &result, nil
}

// GetByName retrieves one presentation by name.
func (s *presentationService) GetByName(ctx context.Context, name string) (*types.Presentation, error) {
	if err := validatePresentationName(name); err != nil {
		return nil, err
	}
	token, err := ready(ctx, s.authManager)
	if err != nil {
		return nil, err
	}

	var result types.Presentation
	if err := s.httpClient.GetWithAuth(ctx, token, s.baseURL()+"/"+escSegment(name), &result); err != nil {
		return nil, fmt.Errorf("failed to get presentation %q: %w", name, err)
	}
	return &result, nil
}

// Create makes a new presentation.
func (s *presentationService) Create(ctx context.Context, request *types.PresentationCreateRequest) (*types.Presentation, error) {
	if request == nil {
		return nil, errors.NewValidationError("request", request, "presentation create request cannot be nil")
	}
	if err := validatePresentationName(request.Name); err != nil {
		return nil, err
	}
	if request.ID != 0 {
		return nil, errors.NewValidationError("id", request.ID, "a new presentation must have ID 0; use Update to change an existing one")
	}
	token, err := ready(ctx, s.authManager)
	if err != nil {
		return nil, err
	}

	var result types.Presentation
	if err := s.httpClient.PostWithAuth(ctx, token, s.baseURL(), withPresentationDefaults(request), &result); err != nil {
		return nil, fmt.Errorf("failed to create presentation %q: %w", request.Name, err)
	}
	return &result, nil
}

// Update replaces an existing presentation.
func (s *presentationService) Update(ctx context.Context, id int, request *types.PresentationCreateRequest) (*types.Presentation, error) {
	if err := validatePresentationID(id); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, errors.NewValidationError("request", request, "presentation update request cannot be nil")
	}
	if err := validatePresentationName(request.Name); err != nil {
		return nil, err
	}
	if request.ID != 0 && request.ID != id {
		return nil, errors.NewValidationError("id", request.ID, fmt.Sprintf("request ID does not match presentation %d", id))
	}
	token, err := ready(ctx, s.authManager)
	if err != nil {
		return nil, err
	}

	body := withPresentationDefaults(request)
	body.ID = id
	var result types.Presentation
	if err := s.httpClient.PutWithAuth(ctx, token, fmt.Sprintf("%s/%d", s.baseURL(), id), body, &result); err != nil {
		return nil, fmt.Errorf("failed to update presentation %d (%q): %w", id, request.Name, err)
	}
	return &result, nil
}

// DeleteByID removes one presentation.
func (s *presentationService) DeleteByID(ctx context.Context, id int) error {
	if err := validatePresentationID(id); err != nil {
		return err
	}
	token, err := ready(ctx, s.authManager)
	if err != nil {
		return err
	}

	if err := s.httpClient.DeleteWithAuth(ctx, token, fmt.Sprintf("%s/%d", s.baseURL(), id), nil); err != nil {
		return fmt.Errorf("failed to delete presentation %d: %w", id, err)
	}
	return nil
}

// DeleteByName removes one presentation by name.
func (s *presentationService) DeleteByName(ctx context.Context, name string) error {
	if err := validatePresentationName(name); err != nil {
		return err
	}
	token, err := ready(ctx, s.authManager)
	if err != nil {
		return err
	}

	if err := s.httpClient.DeleteWithAuth(ctx, token, s.baseURL()+"/"+escSegment(name), nil); err != nil {
		return fmt.Errorf("failed to delete presentation %q: %w", name, err)
	}
	return nil
}

// DeleteByFilter removes presentations matching a filter.
func (s *presentationService) DeleteByFilter(ctx context.Context, filter string) (*types.PresentationDeleteResult, error) {
	if filter == "" {
		return nil, errors.NewValidationError("filter", filter, "filter cannot be empty for bulk delete")
	}
	token, err := ready(ctx, s.authManager)
	if err != nil {
		return nil, err
	}

	var res flexCount
	if err := s.httpClient.DeleteWithAuth(ctx, token, withQuery(s.baseURL(), "filter="+escQuery(filter)), &res); err != nil {
		return nil, fmt.Errorf("failed to delete presentations with filter %q: %w", filter, err)
	}
	return &types.PresentationDeleteResult{DeletedCount: res.N, DeletedIDs: res.IDs, Errors: res.Errors}, nil
}
