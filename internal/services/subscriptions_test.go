package services

import (
	"context"
	"testing"
	"time"

	"github.com/brightsign/bscontrol/internal/auth"
	"github.com/brightsign/bscontrol/internal/config"
	"github.com/brightsign/bscontrol/internal/http"
	"github.com/brightsign/bscontrol/internal/types"
)

func TestSubscriptionService_List(t *testing.T) {
	// Create test client
	cfg := config.DefaultConfig()
	cfg.ClientID = "test-id"
	cfg.ClientSecret = "test-secret"

	httpClient := http.NewHTTPClient(cfg)
	authManager := auth.NewAuthManager(cfg, httpClient)

	subscriptionService := NewSubscriptionService(cfg, httpClient, authManager)

	ctx := context.Background()

	// Test without authentication should fail
	_, err := subscriptionService.List(ctx)
	if err == nil {
		t.Error("Expected error when listing subscriptions without authentication")
	}
}

func TestSubscriptionService_ListWithOptions(t *testing.T) {
	// Create test client
	cfg := config.DefaultConfig()
	cfg.ClientID = "test-id"
	cfg.ClientSecret = "test-secret"

	httpClient := http.NewHTTPClient(cfg)
	authManager := auth.NewAuthManager(cfg, httpClient)

	subscriptionService := NewSubscriptionService(cfg, httpClient, authManager)

	ctx := context.Background()

	// Test with options (will fail without authentication)
	_, err := subscriptionService.List(ctx,
		WithPageSize(10),
		WithFilter("status=active"),
		WithSort("startDate"),
		WithMarker("next-page"),
	)
	if err == nil {
		t.Error("Expected error when listing subscriptions without authentication")
	}
}

func TestSubscriptionService_GetCount(t *testing.T) {
	// Create test client
	cfg := config.DefaultConfig()
	cfg.ClientID = "test-id"
	cfg.ClientSecret = "test-secret"

	httpClient := http.NewHTTPClient(cfg)
	authManager := auth.NewAuthManager(cfg, httpClient)

	subscriptionService := NewSubscriptionService(cfg, httpClient, authManager)

	ctx := context.Background()

	// Test without authentication should fail
	_, err := subscriptionService.GetCount(ctx)
	if err == nil {
		t.Error("Expected error when getting subscription count without authentication")
	}
}

func TestSubscriptionService_GetOperations(t *testing.T) {
	// Create test client
	cfg := config.DefaultConfig()
	cfg.ClientID = "test-id"
	cfg.ClientSecret = "test-secret"

	httpClient := http.NewHTTPClient(cfg)
	authManager := auth.NewAuthManager(cfg, httpClient)

	subscriptionService := NewSubscriptionService(cfg, httpClient, authManager)

	ctx := context.Background()

	// Test without authentication should fail
	_, err := subscriptionService.GetOperations(ctx)
	if err == nil {
		t.Error("Expected error when getting subscription operations without authentication")
	}
}

func TestSubscriptionServiceInterface(t *testing.T) {
	// Test that our implementation satisfies the interface
	var _ SubscriptionService = (*subscriptionService)(nil)
}

// Test Subscription structure
func TestSubscriptionStructure(t *testing.T) {
	endDate := time.Now().Add(365 * 24 * time.Hour)
	subscription := types.Subscription{
		ID:               123,
		DeviceSerial:     "ABC123DEF456",
		DeviceID:         456,
		Type:             "Content",
		Status:           "active",
		StartDate:        time.Now(),
		EndDate:          &endDate,
		CreationDate:     time.Now(),
		LastModifiedDate: time.Now(),
		AutoRenew:        true,
	}

	if subscription.ID != 123 {
		t.Errorf("Expected ID 123, got %d", subscription.ID)
	}
	if subscription.DeviceSerial != "ABC123DEF456" {
		t.Errorf("Expected DeviceSerial 'ABC123DEF456', got '%s'", subscription.DeviceSerial)
	}
	if subscription.Type != "Content" {
		t.Errorf("Expected Type 'Content', got '%s'", subscription.Type)
	}
	if subscription.Status != "active" {
		t.Errorf("Expected Status 'active', got '%s'", subscription.Status)
	}
	if !subscription.AutoRenew {
		t.Error("Expected AutoRenew to be true")
	}
	if subscription.EndDate == nil {
		t.Error("Expected EndDate to be set")
	}
}

// Test SubscriptionList structure
func TestSubscriptionListStructure(t *testing.T) {
	subscriptionList := types.SubscriptionList{
		Items: []types.Subscription{
			{
				ID:           1,
				DeviceSerial: "DEV001",
				Type:         "Content",
				Status:       "active",
				StartDate:    time.Now(),
			},
			{
				ID:           2,
				DeviceSerial: "DEV002",
				Type:         "Control",
				Status:       "active",
				StartDate:    time.Now(),
			},
		},
		IsTruncated: true,
		NextMarker:  "marker123",
		TotalCount:  50,
	}

	if len(subscriptionList.Items) != 2 {
		t.Errorf("Expected 2 subscriptions, got %d", len(subscriptionList.Items))
	}
	if !subscriptionList.IsTruncated {
		t.Error("Expected list to be truncated")
	}
	if subscriptionList.NextMarker != "marker123" {
		t.Errorf("Expected NextMarker 'marker123', got '%s'", subscriptionList.NextMarker)
	}
	if subscriptionList.TotalCount != 50 {
		t.Errorf("Expected TotalCount 50, got %d", subscriptionList.TotalCount)
	}
}

// Test SubscriptionCount structure
func TestSubscriptionCountStructure(t *testing.T) {
	count := types.SubscriptionCount{
		Count: 42,
	}

	if count.Count != 42 {
		t.Errorf("Expected Count 42, got %d", count.Count)
	}
}

// Test SubscriptionOperation structure
func TestSubscriptionOperationStructure(t *testing.T) {
	operation := types.SubscriptionOperation{
		Name:        "create",
		Description: "Create a new subscription",
		Allowed:     true,
	}

	if operation.Name != "create" {
		t.Errorf("Expected Name 'create', got '%s'", operation.Name)
	}
	if operation.Description != "Create a new subscription" {
		t.Errorf("Expected Description 'Create a new subscription', got '%s'", operation.Description)
	}
	if !operation.Allowed {
		t.Error("Expected Allowed to be true")
	}
}

// Test SubscriptionOperations structure
func TestSubscriptionOperationsStructure(t *testing.T) {
	operations := types.SubscriptionOperations{
		Operations: []types.SubscriptionOperation{
			{
				Name:        "create",
				Description: "Create a new subscription",
				Allowed:     true,
			},
			{
				Name:        "update",
				Description: "Update an existing subscription",
				Allowed:     true,
			},
			{
				Name:        "delete",
				Description: "Delete a subscription",
				Allowed:     false,
			},
		},
	}

	if len(operations.Operations) != 3 {
		t.Errorf("Expected 3 operations, got %d", len(operations.Operations))
	}

	// Check first operation
	if operations.Operations[0].Name != "create" {
		t.Errorf("Expected first operation name 'create', got '%s'", operations.Operations[0].Name)
	}
	if !operations.Operations[0].Allowed {
		t.Error("Expected first operation to be allowed")
	}

	// Check last operation (not allowed)
	if operations.Operations[2].Name != "delete" {
		t.Errorf("Expected last operation name 'delete', got '%s'", operations.Operations[2].Name)
	}
	if operations.Operations[2].Allowed {
		t.Error("Expected last operation to not be allowed")
	}
}

// Test subscription list with various statuses
func TestSubscriptionWithDifferentStatuses(t *testing.T) {
	statuses := []string{"active", "expired", "pending", "cancelled"}

	for _, status := range statuses {
		subscription := types.Subscription{
			ID:        1,
			Type:      "Content",
			Status:    status,
			StartDate: time.Now(),
		}

		if subscription.Status != status {
			t.Errorf("Expected Status '%s', got '%s'", status, subscription.Status)
		}
	}
}

// Test subscription with different types
func TestSubscriptionWithDifferentTypes(t *testing.T) {
	subscriptionTypes := []string{"Content", "Control", "Premium"}

	for _, subType := range subscriptionTypes {
		subscription := types.Subscription{
			ID:        1,
			Type:      subType,
			Status:    "active",
			StartDate: time.Now(),
		}

		if subscription.Type != subType {
			t.Errorf("Expected Type '%s', got '%s'", subType, subscription.Type)
		}
	}
}

// Test subscription without optional fields
func TestSubscriptionWithoutOptionalFields(t *testing.T) {
	subscription := types.Subscription{
		ID:               1,
		Type:             "Content",
		Status:           "active",
		StartDate:        time.Now(),
		CreationDate:     time.Now(),
		LastModifiedDate: time.Now(),
	}

	// These fields should be optional
	if subscription.DeviceSerial != "" {
		t.Error("Expected DeviceSerial to be empty when not set")
	}
	if subscription.DeviceID != 0 {
		t.Error("Expected DeviceID to be 0 when not set")
	}
	if subscription.EndDate != nil {
		t.Error("Expected EndDate to be nil when not set")
	}
	if subscription.AutoRenew {
		t.Error("Expected AutoRenew to be false when not set")
	}
}

// Test empty subscription list
func TestEmptySubscriptionList(t *testing.T) {
	subscriptionList := types.SubscriptionList{
		Items:       []types.Subscription{},
		IsTruncated: false,
		NextMarker:  "",
		TotalCount:  0,
	}

	if len(subscriptionList.Items) != 0 {
		t.Errorf("Expected 0 subscriptions, got %d", len(subscriptionList.Items))
	}
	if subscriptionList.IsTruncated {
		t.Error("Expected list not to be truncated")
	}
	if subscriptionList.TotalCount != 0 {
		t.Errorf("Expected TotalCount 0, got %d", subscriptionList.TotalCount)
	}
}
