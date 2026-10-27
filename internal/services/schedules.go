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

// ScheduleService manages the presentations scheduled on a regular group
// (Groups/Regular/{name}/schedule).
type ScheduleService interface {
	// GetGroupSchedule returns every scheduled presentation on a group.
	GetGroupSchedule(ctx context.Context, groupName string) (*types.ScheduledPresentationList, error)

	// AddScheduledPresentation schedules a presentation on a group. The returned
	// entry carries the schedule ID to pass to DeleteScheduledPresentation.
	AddScheduledPresentation(ctx context.Context, groupName string, schedule *types.ScheduledPresentation) (*types.ScheduledPresentation, error)

	// DeleteScheduledPresentation removes one schedule entry from a group.
	DeleteScheduledPresentation(ctx context.Context, groupName string, scheduleID int) error
}

// scheduleService implements the ScheduleService interface.
type scheduleService struct {
	config      *config.Config
	httpClient  *bshttp.HTTPClient
	authManager *auth.AuthManager
}

// NewScheduleService creates a new schedule service.
func NewScheduleService(cfg *config.Config, httpClient *bshttp.HTTPClient, authManager *auth.AuthManager) ScheduleService {
	return &scheduleService{config: cfg, httpClient: httpClient, authManager: authManager}
}

func validateGroupName(name string) error {
	if name == "" {
		return errors.NewValidationError("groupName", name, "group name cannot be empty")
	}
	return nil
}

// scheduleURL builds the group's schedule collection URL with the name escaped.
//
// NOTE: docs/all-apis.md writes the segment as "Schedule"; this uses the
// lowercase "schedule" gopurple has been calling against the live service. The
// service routes case-insensitively, so either spelling reaches the same place.
func (s *scheduleService) scheduleURL(groupName string) string {
	return fmt.Sprintf("%s/%s/Groups/Regular/%s/schedule/", s.config.BSNBaseURL, s.config.APIVersion, escSegment(groupName))
}

// GetGroupSchedule retrieves a group's schedule.
func (s *scheduleService) GetGroupSchedule(ctx context.Context, groupName string) (*types.ScheduledPresentationList, error) {
	if err := validateGroupName(groupName); err != nil {
		return nil, err
	}
	token, err := ready(ctx, s.authManager)
	if err != nil {
		return nil, err
	}

	var result types.ScheduledPresentationList
	if err := s.httpClient.GetWithAuth(ctx, token, s.scheduleURL(groupName), &result); err != nil {
		return nil, fmt.Errorf("failed to get schedule for group %q: %w", groupName, err)
	}
	return &result, nil
}

// AddScheduledPresentation schedules a presentation on a group.
func (s *scheduleService) AddScheduledPresentation(ctx context.Context, groupName string, schedule *types.ScheduledPresentation) (*types.ScheduledPresentation, error) {
	if err := validateGroupName(groupName); err != nil {
		return nil, err
	}
	if schedule == nil {
		return nil, errors.NewValidationError("schedule", schedule, "schedule cannot be nil")
	}
	if schedule.PresentationID <= 0 {
		return nil, errors.NewValidationError("presentationId", schedule.PresentationID, "presentation ID must be positive")
	}
	token, err := ready(ctx, s.authManager)
	if err != nil {
		return nil, err
	}

	var result types.ScheduledPresentation
	if err := s.httpClient.PostWithAuth(ctx, token, s.scheduleURL(groupName), schedule, &result); err != nil {
		return nil, fmt.Errorf("failed to add scheduled presentation %d to group %q: %w", schedule.PresentationID, groupName, err)
	}
	return &result, nil
}

// DeleteScheduledPresentation removes one schedule entry.
func (s *scheduleService) DeleteScheduledPresentation(ctx context.Context, groupName string, scheduleID int) error {
	if err := validateGroupName(groupName); err != nil {
		return err
	}
	if scheduleID <= 0 {
		return errors.NewValidationError("scheduleID", scheduleID, "schedule ID must be positive")
	}
	token, err := ready(ctx, s.authManager)
	if err != nil {
		return err
	}

	deleteURL := fmt.Sprintf("%s%d/", s.scheduleURL(groupName), scheduleID)
	if err := s.httpClient.DeleteWithAuth(ctx, token, deleteURL, nil); err != nil {
		return fmt.Errorf("failed to delete schedule %d from group %q: %w", scheduleID, groupName, err)
	}
	return nil
}
