//go:build integration
// +build integration

package integration

import (
	"context"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/leefowlercu/go-contextforge/contextforge"
)

// gatewayTestMockServer is a dedicated mock MCP server for gateway tests
// This prevents URL conflicts with other test suites
var gatewayTestMockServer *MockMCPServer

// initGatewayMockServer initializes the gateway-specific mock server (called once)
func initGatewayMockServer(t *testing.T) {
	if gatewayTestMockServer != nil {
		return // Already initialized
	}

	gatewayTestMockServer = NewMockMCPServer()

	// Register cleanup to close the server when tests complete
	t.Cleanup(func() {
		if gatewayTestMockServer != nil {
			gatewayTestMockServer.Close()
			gatewayTestMockServer = nil
		}
	})
}

// getGatewayTestMockServerURL returns the URL of the gateway-specific mock server
func getGatewayTestMockServerURL(t *testing.T) string {
	initGatewayMockServer(t)
	if gatewayTestMockServer == nil {
		return GetMockMCPServerURL() // Fallback to global
	}
	return gatewayTestMockServer.URL
}

// gatewayMinimalInput returns a minimal gateway input for testing
// Uses the gateway-specific mock server to avoid URL conflicts
func gatewayMinimalInput(t *testing.T) *contextforge.Gateway {
	return &contextforge.Gateway{
		Name:        randomGatewayName(),
		URL:         getGatewayTestMockServerURL(t),
		Description: contextforge.String("A test gateway for integration testing"),
		Transport:   "STREAMABLEHTTP",
	}
}

// gatewayCompleteInput returns a gateway input with all optional fields for testing
// Uses the gateway-specific mock server to avoid URL conflicts
func gatewayCompleteInput(t *testing.T) *contextforge.Gateway {
	return &contextforge.Gateway{
		Name:        randomGatewayName(),
		URL:         getGatewayTestMockServerURL(t),
		Description: contextforge.String("A complete test gateway with all fields"),
		Transport:   "STREAMABLEHTTP",
		Visibility:  contextforge.String("public"),
		Tags:        contextforge.NewTags([]string{"test", "integration"}),
		TeamID:      contextforge.String("test-team"),
		AuthType:    contextforge.String("bearer"),
		AuthToken:   contextforge.String("test-token-123"),
	}
}

// gatewayCreate creates a test gateway and registers it for cleanup
// Uses the gateway-specific mock server to avoid URL conflicts
func gatewayCreate(t *testing.T, client *contextforge.Client, name string) *contextforge.Gateway {
	t.Helper()

	gateway := &contextforge.Gateway{
		Name:        name,
		URL:         getGatewayTestMockServerURL(t),
		Description: contextforge.String("Test gateway created by integration test"),
		Transport:   "STREAMABLEHTTP",
	}

	ctx := context.Background()
	created, _, err := client.Gateways.Create(ctx, gateway, nil)
	if err != nil {
		t.Fatalf("Failed to create test gateway: %v", err)
	}

	t.Logf("Created test gateway: %s (ID: %s)", created.Name, *created.ID)

	// Register cleanup
	t.Cleanup(func() {
		cleanupGateway(t, client, *created.ID)
	})

	return created
}

// TestGatewaysService_BasicCRUD tests basic CRUD operations
func TestGatewaysService_BasicCRUD(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("create gateway with minimal fields", func(t *testing.T) {
		gateway := gatewayMinimalInput(t)

		created, _, err := client.Gateways.Create(ctx, gateway, nil)
		if err != nil {
			t.Fatalf("Failed to create gateway: %v", err)
		}

		// Cleanup
		t.Cleanup(func() {
			cleanupGateway(t, client, *created.ID)
		})

		if created.ID == nil || *created.ID == "" {
			t.Error("Expected created gateway to have an ID")
		}
		if created.Name != gateway.Name {
			t.Errorf("Expected gateway name %q, got %q", gateway.Name, created.Name)
		}
		// Note: ContextForge normalizes 127.0.0.1 to localhost, so we don't check exact URL match
		if created.URL == "" {
			t.Error("Expected gateway URL to be set")
		}
		if created.Description == nil || *created.Description != *gateway.Description {
			t.Errorf("Expected gateway description %q, got %v", *gateway.Description, created.Description)
		}

		t.Logf("Successfully created gateway: %s (ID: %s)", created.Name, *created.ID)
	})

	t.Run("create gateway with all optional fields", func(t *testing.T) {
		// CONTEXTFORGE-007: Gateway tags not persisted - see docs/upstream-bugs/contextforge-007-gateway-tags-not-persisted.md
		t.Skip("CONTEXTFORGE-007: Gateway tags not persisted on create")

		gateway := gatewayCompleteInput(t)

		created, _, err := client.Gateways.Create(ctx, gateway, nil)
		if err != nil {
			t.Fatalf("Failed to create gateway with all fields: %v", err)
		}

		// Cleanup
		t.Cleanup(func() {
			cleanupGateway(t, client, *created.ID)
		})

		if created.ID == nil || *created.ID == "" {
			t.Error("Expected created gateway to have an ID")
		}
		if created.Transport != gateway.Transport {
			t.Errorf("Expected transport %q, got %q", gateway.Transport, created.Transport)
		}
		if created.Visibility != nil && gateway.Visibility != nil && *created.Visibility != *gateway.Visibility {
			t.Errorf("Expected visibility %q, got %q", *gateway.Visibility, *created.Visibility)
		}
		if len(created.Tags) != len(gateway.Tags) {
			t.Errorf("Expected %d tags, got %d", len(gateway.Tags), len(created.Tags))
		}
		if created.AuthType != nil && gateway.AuthType != nil {
			t.Logf("Gateway created with AuthType: %s", *created.AuthType)
		}

		t.Logf("Successfully created gateway with all fields: %s (ID: %s)", created.Name, *created.ID)
	})

	t.Run("get gateway by ID", func(t *testing.T) {
		created := gatewayCreate(t, client, randomGatewayName())

		retrieved, _, err := client.Gateways.Get(ctx, *created.ID)
		if err != nil {
			t.Fatalf("Failed to get gateway: %v", err)
		}

		if *retrieved.ID != *created.ID {
			t.Errorf("Expected gateway ID %q, got %q", *created.ID, *retrieved.ID)
		}
		if retrieved.Name != created.Name {
			t.Errorf("Expected gateway name %q, got %q", created.Name, retrieved.Name)
		}

		t.Logf("Successfully retrieved gateway: %s (ID: %s)", retrieved.Name, *retrieved.ID)
	})

	t.Run("list gateways", func(t *testing.T) {
		// Create a test gateway (note: we can only create one per mock server URL)
		gatewayCreate(t, client, randomGatewayName())

		gateways, _, err := client.Gateways.List(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to list gateways: %v", err)
		}

		if len(gateways) == 0 {
			t.Error("Expected at least some gateways in the list")
		}

		t.Logf("Successfully listed %d gateways", len(gateways))
	})

	t.Run("update gateway", func(t *testing.T) {
		// CONTEXTFORGE-007: Gateway tags not persisted - see docs/upstream-bugs/contextforge-007-gateway-tags-not-persisted.md
		t.Skip("CONTEXTFORGE-007: Gateway tags not persisted on update")

		created := gatewayCreate(t, client, randomGatewayName())

		// Update the gateway
		expectedDescription := "Updated description for integration test"
		expectedTagNames := []string{"updated", "integration-test"}
		created.Description = contextforge.String(expectedDescription)
		created.Tags = contextforge.NewTags(expectedTagNames)

		updated, _, err := client.Gateways.Update(ctx, *created.ID, created)
		if err != nil {
			t.Fatalf("Failed to update gateway: %v", err)
		}

		// Assert that updates actually persisted
		if updated.Description == nil || *updated.Description != expectedDescription {
			t.Errorf("Expected description %q, got %v", expectedDescription, updated.Description)
		}
		actualTagNames := contextforge.TagNames(updated.Tags)
		if !reflect.DeepEqual(actualTagNames, expectedTagNames) {
			t.Errorf("Expected tags %v, got %v", expectedTagNames, actualTagNames)
		}

		t.Logf("Successfully updated gateway: %s (ID: %s)", updated.Name, *updated.ID)
	})

	t.Run("delete gateway", func(t *testing.T) {
		created := gatewayCreate(t, client, randomGatewayName())

		// Delete the gateway
		_, err := client.Gateways.Delete(ctx, *created.ID)
		if err != nil {
			t.Fatalf("Failed to delete gateway: %v", err)
		}

		t.Logf("Successfully deleted gateway: %s (ID: %s)", created.Name, *created.ID)
	})

	t.Run("get deleted gateway returns 404", func(t *testing.T) {
		created := gatewayCreate(t, client, randomGatewayName())

		// Delete the gateway
		_, err := client.Gateways.Delete(ctx, *created.ID)
		if err != nil {
			t.Fatalf("Failed to delete gateway: %v", err)
		}

		// Try to get the deleted gateway
		_, _, err = client.Gateways.Get(ctx, *created.ID)
		if err == nil {
			t.Error("Expected error when getting deleted gateway")
		}

		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
			// ContextForge may return 404 or 500 for deleted gateways
			if apiErr.Response.StatusCode != http.StatusNotFound && apiErr.Response.StatusCode != http.StatusInternalServerError {
				t.Errorf("Expected 404 or 500 for deleted gateway, got %d", apiErr.Response.StatusCode)
			}
			t.Logf("Correctly received error (status %d) for deleted gateway", apiErr.Response.StatusCode)
		} else {
			t.Errorf("Expected ErrorResponse, got %T: %v", err, err)
		}
	})

	t.Run("list empty results", func(t *testing.T) {
		// List with pagination that may return no results
		opts := &contextforge.GatewayListOptions{
			ListOptions: contextforge.ListOptions{
				Limit:  1,
				Cursor: "non-existent-cursor-xyz-12345",
			},
		}

		gateways, _, err := client.Gateways.List(ctx, opts)
		if err != nil {
			t.Fatalf("Failed to list gateways: %v", err)
		}

		t.Logf("List with invalid cursor returned %d gateways", len(gateways))
	})
}

// TestGatewaysService_Toggle tests toggle functionality
func TestGatewaysService_Toggle(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("toggle enabled to disabled", func(t *testing.T) {
		gateway := gatewayMinimalInput(t)

		created, _, err := client.Gateways.Create(ctx, gateway, nil)
		if err != nil {
			t.Fatalf("Failed to create gateway: %v", err)
		}

		t.Cleanup(func() {
			cleanupGateway(t, client, *created.ID)
		})

		// Toggle to disabled
		toggled, _, err := client.Gateways.Toggle(ctx, *created.ID, false)
		if err != nil {
			t.Fatalf("Failed to toggle gateway: %v", err)
		}

		if toggled.Enabled {
			t.Error("Expected gateway to be disabled after toggle")
		}

		t.Logf("Successfully toggled gateway from enabled to disabled")
	})

	t.Run("toggle disabled to enabled", func(t *testing.T) {
		gateway := gatewayMinimalInput(t)
		created, _, err := client.Gateways.Create(ctx, gateway, nil)
		if err != nil {
			t.Fatalf("Failed to create gateway: %v", err)
		}

		t.Cleanup(func() {
			cleanupGateway(t, client, *created.ID)
		})

		// First disable the gateway
		disabled, _, err := client.Gateways.Toggle(ctx, *created.ID, false)
		if err != nil {
			t.Fatalf("Failed to disable gateway: %v", err)
		}

		if disabled.Enabled {
			t.Fatal("Failed to disable gateway in setup")
		}

		// Now toggle to enabled
		toggled, _, err := client.Gateways.Toggle(ctx, *created.ID, true)
		if err != nil {
			t.Fatalf("Failed to toggle gateway: %v", err)
		}

		if !toggled.Enabled {
			t.Error("Expected gateway to be enabled after toggle")
		}

		t.Logf("Successfully toggled gateway from disabled to enabled")
	})

	t.Run("verify toggle persists", func(t *testing.T) {
		gateway := gatewayMinimalInput(t)
		created, _, err := client.Gateways.Create(ctx, gateway, nil)
		if err != nil {
			t.Fatalf("Failed to create gateway: %v", err)
		}

		t.Cleanup(func() {
			cleanupGateway(t, client, *created.ID)
		})

		// Toggle to enabled
		_, _, err = client.Gateways.Toggle(ctx, *created.ID, true)
		if err != nil {
			t.Fatalf("Failed to toggle gateway: %v", err)
		}

		// Get the gateway again to verify it's enabled
		retrieved, _, err := client.Gateways.Get(ctx, *created.ID)
		if err != nil {
			t.Fatalf("Failed to get gateway: %v", err)
		}

		if !retrieved.Enabled {
			t.Error("Expected gateway to remain enabled")
		}

		t.Logf("Successfully verified toggle persists")
	})
}

// TestGatewaysService_Filtering tests filtering
func TestGatewaysService_Filtering(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("filter by include_inactive", func(t *testing.T) {
		// Create gateway and disable it
		gateway := gatewayMinimalInput(t)
		created, _, err := client.Gateways.Create(ctx, gateway, nil)
		if err != nil {
			t.Fatalf("Failed to create gateway: %v", err)
		}

		t.Cleanup(func() {
			cleanupGateway(t, client, *created.ID)
		})

		// Disable the gateway
		_, _, err = client.Gateways.Toggle(ctx, *created.ID, false)
		if err != nil {
			t.Fatalf("Failed to disable gateway: %v", err)
		}

		// List without include_inactive (should not include disabled)
		gateways, _, err := client.Gateways.List(ctx, &contextforge.GatewayListOptions{
			IncludeInactive: false,
		})
		if err != nil {
			t.Fatalf("Failed to list gateways: %v", err)
		}

		foundDisabled := false
		for _, gw := range gateways {
			if gw.ID != nil && *gw.ID == *created.ID {
				foundDisabled = true
				break
			}
		}

		if foundDisabled {
			t.Log("Warning: Disabled gateway found in list without include_inactive=true")
		}

		// List with include_inactive (should include disabled)
		allGateways, _, err := client.Gateways.List(ctx, &contextforge.GatewayListOptions{
			IncludeInactive: true,
		})
		if err != nil {
			t.Fatalf("Failed to list gateways with include_inactive: %v", err)
		}

		foundInAll := false
		for _, gw := range allGateways {
			if gw.ID != nil && *gw.ID == *created.ID {
				foundInAll = true
				break
			}
		}

		if !foundInAll {
			t.Error("Expected to find disabled gateway in list with include_inactive=true")
		} else {
			t.Logf("Successfully found disabled gateway with include_inactive=true")
		}
	})
}

// TestGatewaysService_InputValidation tests input validation
func TestGatewaysService_InputValidation(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("create gateway missing required name", func(t *testing.T) {
		gateway := &contextforge.Gateway{
			URL:         "http://localhost:8000",
			Description: contextforge.String("Gateway without name"),
		}

		_, _, err := client.Gateways.Create(ctx, gateway, nil)
		if err == nil {
			t.Error("Expected error when creating gateway without name")
		} else {
			t.Logf("Correctly rejected gateway without name: %v", err)
		}
	})

	t.Run("create gateway missing required url", func(t *testing.T) {
		gateway := &contextforge.Gateway{
			Name:        randomGatewayName(),
			Description: contextforge.String("Gateway without URL"),
		}

		_, _, err := client.Gateways.Create(ctx, gateway, nil)
		if err == nil {
			t.Error("Expected error when creating gateway without URL")
		} else {
			t.Logf("Correctly rejected gateway without URL: %v", err)
		}
	})

	t.Run("create gateway with invalid url format", func(t *testing.T) {
		gateway := &contextforge.Gateway{
			Name:        randomGatewayName(),
			URL:         "not-a-valid-url",
			Description: contextforge.String("Gateway with invalid URL"),
		}

		_, _, err := client.Gateways.Create(ctx, gateway, nil)
		if err == nil {
			t.Log("Warning: API accepted invalid URL format")
		} else {
			t.Logf("Correctly rejected invalid URL format: %v", err)
		}
	})
}

// TestGatewaysService_ErrorHandling tests error handling
func TestGatewaysService_ErrorHandling(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("get non-existent gateway", func(t *testing.T) {
		_, _, err := client.Gateways.Get(ctx, "non-existent-gateway-id-xyz")
		if err == nil {
			t.Error("Expected error when getting non-existent gateway")
		}

		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
			// ContextForge may return 404 or 500 for non-existent gateways
			if apiErr.Response.StatusCode != http.StatusNotFound && apiErr.Response.StatusCode != http.StatusInternalServerError {
				t.Errorf("Expected 404 or 500 for non-existent gateway, got %d", apiErr.Response.StatusCode)
			}
			t.Logf("Correctly received error (status %d) for non-existent gateway", apiErr.Response.StatusCode)
		} else {
			t.Logf("Got error (not ErrorResponse): %v", err)
		}
	})

	t.Run("update non-existent gateway", func(t *testing.T) {
		gateway := gatewayMinimalInput(t)
		_, _, err := client.Gateways.Update(ctx, "non-existent-gateway-id-xyz", gateway)
		if err == nil {
			t.Error("Expected error when updating non-existent gateway")
		} else {
			t.Logf("Correctly rejected update of non-existent gateway: %v", err)
		}
	})

	t.Run("delete non-existent gateway", func(t *testing.T) {
		_, err := client.Gateways.Delete(ctx, "non-existent-gateway-id-xyz")
		if err == nil {
			t.Log("Warning: API accepted deletion of non-existent gateway")
		} else {
			t.Logf("Correctly rejected deletion of non-existent gateway: %v", err)
		}
	})

	t.Run("toggle non-existent gateway", func(t *testing.T) {
		_, _, err := client.Gateways.Toggle(ctx, "non-existent-gateway-id-xyz", true)
		if err == nil {
			t.Error("Expected error when toggling non-existent gateway")
		} else {
			t.Logf("Correctly rejected toggle of non-existent gateway: %v", err)
		}
	})
}

// TestGatewaysService_EdgeCases tests edge cases
func TestGatewaysService_EdgeCases(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("long gateway name", func(t *testing.T) {
		gateway := &contextforge.Gateway{
			Name:        strings.Repeat("a", 200),
			URL:         GetMockMCPServerURL(),
			Description: contextforge.String("Gateway with long name"),
		}

		created, _, err := client.Gateways.Create(ctx, gateway, nil)
		if err != nil {
			t.Logf("API rejected long gateway name: %v", err)
		} else {
			t.Cleanup(func() {
				cleanupGateway(t, client, *created.ID)
			})
			t.Logf("API accepted long gateway name (%d chars)", len(created.Name))
		}
	})

	t.Run("special characters in name", func(t *testing.T) {
		gateway := &contextforge.Gateway{
			Name:        "test-gateway-!@#$%^&*()",
			URL:         GetMockMCPServerURL(),
			Description: contextforge.String("Gateway with special characters"),
		}

		created, _, err := client.Gateways.Create(ctx, gateway, nil)
		if err != nil {
			t.Logf("API rejected special characters in name: %v", err)
		} else {
			t.Cleanup(func() {
				cleanupGateway(t, client, *created.ID)
			})
			t.Logf("API accepted special characters in name")
		}
	})

	t.Run("complex auth configuration", func(t *testing.T) {
		gateway := &contextforge.Gateway{
			Name:        randomGatewayName(),
			URL:         GetMockMCPServerURL(),
			Description: contextforge.String("Gateway with complex auth"),
			AuthType:    contextforge.String("headers"),
			AuthHeaders: []map[string]string{
				{"X-Custom-Auth": "token1"},
				{"X-API-Key": "key123"},
			},
			PassthroughHeaders: []string{"Authorization", "X-Request-ID"},
		}

		created, _, err := client.Gateways.Create(ctx, gateway, nil)
		if err != nil {
			t.Logf("API rejected complex auth configuration: %v", err)
		} else {
			t.Cleanup(func() {
				cleanupGateway(t, client, *created.ID)
			})
			t.Logf("API accepted complex auth configuration")
		}
	})

	t.Run("large passthrough headers list", func(t *testing.T) {
		headers := make([]string, 50)
		for i := 0; i < 50; i++ {
			headers[i] = "X-Header-" + string(rune('A'+i%26))
		}

		gateway := &contextforge.Gateway{
			Name:               randomGatewayName(),
			URL:                GetMockMCPServerURL(),
			Description:        contextforge.String("Gateway with many passthrough headers"),
			PassthroughHeaders: headers,
		}

		created, _, err := client.Gateways.Create(ctx, gateway, nil)
		if err != nil {
			t.Logf("API rejected large passthrough headers list: %v", err)
		} else {
			t.Cleanup(func() {
				cleanupGateway(t, client, *created.ID)
			})
			t.Logf("API accepted %d passthrough headers", len(headers))
		}
	})
}
