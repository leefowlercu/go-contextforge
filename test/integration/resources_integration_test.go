//go:build integration
// +build integration

package integration

import (
	"context"
	"net/http"
	"testing"

	"github.com/leefowlercu/go-contextforge/contextforge"
)

func TestResourcesService_BasicCRUD(t *testing.T) {
	client := setupClient(t)
	ctx := context.Background()

	t.Run("create resource with minimal fields", func(t *testing.T) {
		resource := minimalResourceInput()

		created, _, err := client.Resources.Create(ctx, resource, nil)
		if err != nil {
			t.Fatalf("Failed to create resource: %v", err)
		}

		// Cleanup
		t.Cleanup(func() {
			cleanupResource(t, client, string(*created.ID))
		})

		if created.ID == nil || *created.ID == "" {
			t.Error("Expected created resource to have an ID")
		}
		if created.Name != resource.Name {
			t.Errorf("Expected resource name %q, got %q", resource.Name, created.Name)
		}
		if created.URI == "" {
			t.Error("Expected resource URI to be set")
		}

		t.Logf("Successfully created resource: %s (ID: %s)", created.Name, *created.ID)
	})

	t.Run("create resource with all optional fields", func(t *testing.T) {
		resource := completeResourceInput()

		opts := &contextforge.ResourceCreateOptions{
			TeamID:     nil, // Let API assign
			Visibility: contextforge.String("public"),
		}

		created, _, err := client.Resources.Create(ctx, resource, opts)
		if err != nil {
			t.Fatalf("Failed to create resource: %v", err)
		}

		// Cleanup
		t.Cleanup(func() {
			cleanupResource(t, client, string(*created.ID))
		})

		if created.Name != resource.Name {
			t.Errorf("Expected resource name %q, got %q", resource.Name, created.Name)
		}
		if created.MimeType == nil || *created.MimeType != *resource.MimeType {
			t.Errorf("Expected mimeType %q, got %v", *resource.MimeType, created.MimeType)
		}
		if len(created.Tags) != len(resource.Tags) {
			t.Errorf("Expected %d tags, got %d", len(resource.Tags), len(created.Tags))
		}

		t.Logf("Successfully created resource with all fields: %s (ID: %s)", created.Name, *created.ID)
	})

	t.Run("list resources", func(t *testing.T) {
		// Create a test resource
		createTestResource(t, client, randomResourceName())

		resources, _, err := client.Resources.List(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to list resources: %v", err)
		}

		if len(resources) == 0 {
			t.Error("Expected at least some resources in the list")
		}

		t.Logf("Successfully listed %d resources", len(resources))
	})

	t.Run("update resource", func(t *testing.T) {
		created := createTestResource(t, client, randomResourceName())

		update := &contextforge.ResourceUpdate{
			Description: contextforge.String("Updated description for integration test"),
			Tags:        []string{"updated", "integration-test"},
		}

		updated, _, err := client.Resources.Update(ctx, string(*created.ID), update)
		if err != nil {
			t.Fatalf("Failed to update resource: %v", err)
		}

		if updated.Description == nil || *updated.Description != *update.Description {
			t.Errorf("Expected description %q, got %v", *update.Description, updated.Description)
		}
		if len(updated.Tags) != len(update.Tags) {
			t.Errorf("Expected %d tags, got %d", len(update.Tags), len(updated.Tags))
		}

		t.Logf("Successfully updated resource: %s (ID: %s)", updated.Name, *updated.ID)
	})

	t.Run("delete resource", func(t *testing.T) {
		created := createTestResource(t, client, randomResourceName())

		_, err := client.Resources.Delete(ctx, string(*created.ID))
		if err != nil {
			t.Fatalf("Failed to delete resource: %v", err)
		}

		t.Logf("Successfully deleted resource: %s (ID: %s)", created.Name, *created.ID)
	})

	t.Run("list empty results", func(t *testing.T) {
		// List with invalid cursor that may return no results
		opts := &contextforge.ResourceListOptions{
			ListOptions: contextforge.ListOptions{
				Cursor: "invalid-cursor-xyz",
			},
		}

		resources, _, err := client.Resources.List(ctx, opts)
		// API may return error or empty list for invalid cursor
		if err == nil {
			t.Logf("List with invalid cursor returned %d resources", len(resources))
		}
	})
}

func TestResourcesService_Toggle(t *testing.T) {
	client := setupClient(t)
	ctx := context.Background()

	t.Run("toggle active to inactive", func(t *testing.T) {
		// Create an active resource
		resource := minimalResourceInput()
		created, _, err := client.Resources.Create(ctx, resource, nil)
		if err != nil {
			t.Fatalf("Failed to create resource: %v", err)
		}

		// Cleanup
		t.Cleanup(func() {
			cleanupResource(t, client, string(*created.ID))
		})

		// Verify it's active
		if !created.IsActive {
			t.Error("Expected newly created resource to be active")
		}

		// Toggle to inactive
		toggled, _, err := client.Resources.Toggle(ctx, string(*created.ID), false)
		if err != nil {
			t.Fatalf("Failed to toggle resource: %v", err)
		}

		if toggled.IsActive {
			t.Error("Expected resource to be inactive after toggle")
		}

		t.Logf("Successfully toggled resource from active to inactive")
	})

	t.Run("toggle inactive to active", func(t *testing.T) {
		// Create resource and toggle to inactive
		resource := minimalResourceInput()
		created, _, err := client.Resources.Create(ctx, resource, nil)
		if err != nil {
			t.Fatalf("Failed to create resource: %v", err)
		}

		// Cleanup
		t.Cleanup(func() {
			cleanupResource(t, client, string(*created.ID))
		})

		// Toggle to inactive first
		_, _, err = client.Resources.Toggle(ctx, string(*created.ID), false)
		if err != nil {
			t.Fatalf("Failed to toggle resource to inactive: %v", err)
		}

		// Toggle back to active
		toggled, _, err := client.Resources.Toggle(ctx, string(*created.ID), true)
		if err != nil {
			t.Fatalf("Failed to toggle resource to active: %v", err)
		}

		if !toggled.IsActive {
			t.Error("Expected resource to be active after toggle")
		}

		t.Logf("Successfully toggled resource from inactive to active")
	})
}

func TestResourcesService_Templates(t *testing.T) {
	client := setupClient(t)
	ctx := context.Background()

	t.Run("list templates", func(t *testing.T) {
		result, _, err := client.Resources.ListTemplates(ctx)
		if err != nil {
			t.Fatalf("Failed to list templates: %v", err)
		}

		// API may or may not have templates configured
		t.Logf("ListTemplates returned %d templates", len(result.Templates))

		// Verify result structure is returned (templates may be nil or empty)
		if result == nil {
			t.Error("Expected non-nil result")
		}
	})
}

func TestResourcesService_InputValidation(t *testing.T) {
	client := setupClient(t)
	ctx := context.Background()

	t.Run("create resource missing required name", func(t *testing.T) {
		resource := &contextforge.ResourceCreate{
			URI:     "file:///test.txt",
			Content: "test",
			// Name is missing
		}

		_, _, err := client.Resources.Create(ctx, resource, nil)
		if err == nil {
			t.Error("Expected error when creating resource without name")
		}

		t.Logf("Correctly rejected resource without name: %v", err)
	})

	t.Run("create resource missing required URI", func(t *testing.T) {
		resource := &contextforge.ResourceCreate{
			Name:    "test-resource",
			Content: "test",
			// URI is missing
		}

		_, _, err := client.Resources.Create(ctx, resource, nil)
		if err == nil {
			t.Error("Expected error when creating resource without URI")
		}

		t.Logf("Correctly rejected resource without URI: %v", err)
	})

	t.Run("create resource missing required content", func(t *testing.T) {
		resource := &contextforge.ResourceCreate{
			URI:  "file:///test.txt",
			Name: "test-resource",
			// Content is missing
		}

		_, _, err := client.Resources.Create(ctx, resource, nil)
		if err == nil {
			t.Error("Expected error when creating resource without content")
		}

		t.Logf("Correctly rejected resource without content: %v", err)
	})
}

func TestResourcesService_ErrorHandling(t *testing.T) {
	client := setupClient(t)
	ctx := context.Background()

	t.Run("update non-existent resource", func(t *testing.T) {
		update := &contextforge.ResourceUpdate{
			Description: contextforge.String("Updated"),
		}

		_, _, err := client.Resources.Update(ctx, "non-existent-resource-id-xyz", update)
		if err == nil {
			t.Error("Expected error when updating non-existent resource")
		}

		t.Logf("Correctly rejected update of non-existent resource: %v", err)
	})

	t.Run("delete non-existent resource", func(t *testing.T) {
		_, err := client.Resources.Delete(ctx, "non-existent-resource-id-xyz")
		if err == nil {
			t.Error("Expected error when deleting non-existent resource")
		}

		t.Logf("Correctly rejected deletion of non-existent resource: %v", err)
	})

	t.Run("toggle non-existent resource", func(t *testing.T) {
		_, _, err := client.Resources.Toggle(ctx, "non-existent-resource-id-xyz", true)
		if err == nil {
			t.Error("Expected error when toggling non-existent resource")
		}

		t.Logf("Correctly rejected toggle of non-existent resource: %v", err)
	})

	t.Run("invalid authentication", func(t *testing.T) {
		// Create client with invalid token
		invalidClient, err := contextforge.NewClient(nil, client.BaseURL.String(), "invalid-token-xyz")
		if err != nil {
			t.Fatalf("Failed to create invalid client: %v", err)
		}

		_, _, err = invalidClient.Resources.List(ctx, nil)
		if err == nil {
			t.Error("Expected error with invalid token")
		}

		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
			if apiErr.Response.StatusCode != http.StatusUnauthorized {
				t.Errorf("Expected 401 Unauthorized, got %d", apiErr.Response.StatusCode)
			}
		}

		t.Logf("Correctly received 401 error for invalid token")
	})
}
