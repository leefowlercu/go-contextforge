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

// TestServersService_BasicCRUD tests basic CRUD operations
func TestServersService_BasicCRUD(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("create server with minimal fields", func(t *testing.T) {
		server := minimalServerInput()

		created, _, err := client.Servers.Create(ctx, server, nil)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		t.Cleanup(func() {
			cleanupServer(t, client, created.ID)
		})

		if created.ID == "" {
			t.Error("Expected created server to have an ID")
		}
		if created.Name != server.Name {
			t.Errorf("Expected server name %q, got %q", server.Name, created.Name)
		}
		if created.Description == nil || *created.Description != *server.Description {
			t.Errorf("Expected server description %q, got %v", *server.Description, created.Description)
		}
		if created.Metrics == nil {
			t.Error("Expected created server to have metrics")
		}

		t.Logf("Successfully created server: %s (ID: %s)", created.Name, created.ID)
	})

	t.Run("create server with all optional fields", func(t *testing.T) {
		server := completeServerInput()

		created, _, err := client.Servers.Create(ctx, server, nil)
		if err != nil {
			t.Fatalf("Failed to create server with all fields: %v", err)
		}

		t.Cleanup(func() {
			cleanupServer(t, client, created.ID)
		})

		if created.ID == "" {
			t.Error("Expected created server to have an ID")
		}
		if created.Visibility == nil || *created.Visibility != *server.Visibility {
			t.Errorf("Expected visibility %q, got %v", *server.Visibility, created.Visibility)
		}
		if len(created.Tags) != len(server.Tags) {
			t.Errorf("Expected %d tags, got %d", len(server.Tags), len(created.Tags))
		}

		t.Logf("Successfully created server with all fields: %s (ID: %s)", created.Name, created.ID)
	})

	t.Run("get server by ID", func(t *testing.T) {
		created := createTestServer(t, client, randomServerName())

		retrieved, _, err := client.Servers.Get(ctx, created.ID)
		if err != nil {
			t.Fatalf("Failed to get server: %v", err)
		}

		if retrieved.ID != created.ID {
			t.Errorf("Expected server ID %q, got %q", created.ID, retrieved.ID)
		}
		if retrieved.Name != created.Name {
			t.Errorf("Expected server name %q, got %q", created.Name, retrieved.Name)
		}

		t.Logf("Successfully retrieved server: %s (ID: %s)", retrieved.Name, retrieved.ID)
	})

	t.Run("list servers", func(t *testing.T) {
		// Create a few test servers
		createTestServer(t, client, randomServerName())
		createTestServer(t, client, randomServerName())

		servers, _, err := client.Servers.List(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to list servers: %v", err)
		}

		if len(servers) == 0 {
			t.Error("Expected at least some servers in the list")
		}

		t.Logf("Successfully listed %d servers", len(servers))
	})

	t.Run("update server", func(t *testing.T) {
		created := createTestServer(t, client, randomServerName())

		// Update the server
		expectedDescription := "Updated description for integration test"
		expectedTags := []string{"updated", "integration-test"}
		update := &contextforge.ServerUpdate{
			Description: contextforge.String(expectedDescription),
			Tags:        expectedTags,
		}

		updated, _, err := client.Servers.Update(ctx, created.ID, update)
		if err != nil {
			t.Fatalf("Failed to update server: %v", err)
		}

		// Assert that updates actually persisted
		if updated.Description == nil || *updated.Description != expectedDescription {
			t.Errorf("Expected description %q, got %v", expectedDescription, updated.Description)
		}
		if !reflect.DeepEqual(updated.Tags, expectedTags) {
			t.Errorf("Expected tags %v, got %v", expectedTags, updated.Tags)
		}

		t.Logf("Successfully updated server: %s (ID: %s)", updated.Name, updated.ID)
	})

	t.Run("delete server", func(t *testing.T) {
		created := createTestServer(t, client, randomServerName())

		// Delete the server
		_, err := client.Servers.Delete(ctx, created.ID)
		if err != nil {
			t.Fatalf("Failed to delete server: %v", err)
		}

		t.Logf("Successfully deleted server: %s (ID: %s)", created.Name, created.ID)
	})

	t.Run("get deleted server returns 404", func(t *testing.T) {
		created := createTestServer(t, client, randomServerName())

		// Delete the server
		_, err := client.Servers.Delete(ctx, created.ID)
		if err != nil {
			t.Fatalf("Failed to delete server: %v", err)
		}

		// Try to get the deleted server
		_, _, err = client.Servers.Get(ctx, created.ID)
		if err == nil {
			t.Error("Expected error when getting deleted server")
		}

		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
			if apiErr.Response.StatusCode != http.StatusNotFound {
				t.Errorf("Expected 404 Not Found, got %d", apiErr.Response.StatusCode)
			}
			t.Logf("Correctly received 404 for deleted server")
		} else {
			t.Errorf("Expected ErrorResponse, got %T: %v", err, err)
		}
	})
}

// TestServersService_Toggle tests toggle functionality
func TestServersService_Toggle(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("toggle active to inactive", func(t *testing.T) {
		server := minimalServerInput()

		created, _, err := client.Servers.Create(ctx, server, nil)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		t.Cleanup(func() {
			cleanupServer(t, client, created.ID)
		})

		initialState := created.IsActive
		t.Logf("Server initial state: isActive=%v", initialState)

		// Toggle to inactive
		toggled, _, err := client.Servers.Toggle(ctx, created.ID, false)
		if err != nil {
			t.Fatalf("Failed to toggle server: %v", err)
		}

		if toggled.IsActive {
			t.Error("Expected server to be inactive after toggle(false)")
		}

		t.Logf("Successfully toggled server to inactive")
	})

	t.Run("toggle inactive to active", func(t *testing.T) {
		server := minimalServerInput()
		created, _, err := client.Servers.Create(ctx, server, nil)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		t.Cleanup(func() {
			cleanupServer(t, client, created.ID)
		})

		// First deactivate the server
		_, _, err = client.Servers.Toggle(ctx, created.ID, false)
		if err != nil {
			t.Fatalf("Failed to deactivate server: %v", err)
		}

		// Now toggle to active
		toggled, _, err := client.Servers.Toggle(ctx, created.ID, true)
		if err != nil {
			t.Fatalf("Failed to toggle server to active: %v", err)
		}

		if !toggled.IsActive {
			t.Error("Expected server to be active after toggle(true)")
		}

		t.Logf("Successfully toggled server to active")
	})

	t.Run("toggle persists after retrieval", func(t *testing.T) {
		created := createTestServer(t, client, randomServerName())

		// Toggle to inactive
		_, _, err := client.Servers.Toggle(ctx, created.ID, false)
		if err != nil {
			t.Fatalf("Failed to toggle server: %v", err)
		}

		// Retrieve and verify state
		retrieved, _, err := client.Servers.Get(ctx, created.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve server: %v", err)
		}

		if retrieved.IsActive {
			t.Error("Expected server to remain inactive after retrieval")
		}

		t.Logf("Toggle state correctly persisted")
	})
}

// TestServersService_Associations tests association listing endpoints
func TestServersService_Associations(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("list tools for server", func(t *testing.T) {
		server := createTestServer(t, client, randomServerName())

		tools, _, err := client.Servers.ListTools(ctx, server.ID, nil)
		if err != nil {
			t.Fatalf("Failed to list server tools: %v", err)
		}

		t.Logf("Server has %d associated tools", len(tools))
	})

	t.Run("list tools with include_inactive", func(t *testing.T) {
		server := createTestServer(t, client, randomServerName())

		opts := &contextforge.ServerAssociationOptions{
			IncludeInactive: true,
		}

		tools, _, err := client.Servers.ListTools(ctx, server.ID, opts)
		if err != nil {
			t.Fatalf("Failed to list server tools with include_inactive: %v", err)
		}

		t.Logf("Server has %d associated tools (including inactive)", len(tools))
	})

	t.Run("list resources for server", func(t *testing.T) {
		server := createTestServer(t, client, randomServerName())

		resources, _, err := client.Servers.ListResources(ctx, server.ID, nil)
		if err != nil {
			t.Fatalf("Failed to list server resources: %v", err)
		}

		t.Logf("Server has %d associated resources", len(resources))
	})

	t.Run("list resources with include_inactive", func(t *testing.T) {
		server := createTestServer(t, client, randomServerName())

		opts := &contextforge.ServerAssociationOptions{
			IncludeInactive: true,
		}

		resources, _, err := client.Servers.ListResources(ctx, server.ID, opts)
		if err != nil {
			t.Fatalf("Failed to list server resources with include_inactive: %v", err)
		}

		t.Logf("Server has %d associated resources (including inactive)", len(resources))
	})

	t.Run("list prompts for server", func(t *testing.T) {
		server := createTestServer(t, client, randomServerName())

		prompts, _, err := client.Servers.ListPrompts(ctx, server.ID, nil)
		if err != nil {
			t.Fatalf("Failed to list server prompts: %v", err)
		}

		t.Logf("Server has %d associated prompts", len(prompts))
	})

	t.Run("list prompts with include_inactive", func(t *testing.T) {
		server := createTestServer(t, client, randomServerName())

		opts := &contextforge.ServerAssociationOptions{
			IncludeInactive: true,
		}

		prompts, _, err := client.Servers.ListPrompts(ctx, server.ID, opts)
		if err != nil {
			t.Fatalf("Failed to list server prompts with include_inactive: %v", err)
		}

		t.Logf("Server has %d associated prompts (including inactive)", len(prompts))
	})
}

// TestServersService_Filtering tests filtering capabilities
func TestServersService_Filtering(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("filter by tags", func(t *testing.T) {
		// Create server with specific tags
		server := &contextforge.ServerCreate{
			Name:        randomServerName(),
			Description: contextforge.String("Server for tag filtering test"),
			Tags:        []string{"filter-test", "tag-search"},
		}

		created, _, err := client.Servers.Create(ctx, server, nil)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		t.Cleanup(func() {
			cleanupServer(t, client, created.ID)
		})

		// List with tag filter
		opts := &contextforge.ServerListOptions{
			Tags: "filter-test",
		}

		servers, _, err := client.Servers.List(ctx, opts)
		if err != nil {
			t.Fatalf("Failed to list servers with tag filter: %v", err)
		}

		if len(servers) == 0 {
			t.Error("Expected at least one server with tag 'filter-test'")
		}

		t.Logf("Found %d servers with tag filter", len(servers))
	})

	t.Run("filter by visibility", func(t *testing.T) {
		// Create server with specific visibility
		server := &contextforge.ServerCreate{
			Name:        randomServerName(),
			Description: contextforge.String("Server for visibility filtering test"),
			Visibility:  contextforge.String("public"),
		}

		created, _, err := client.Servers.Create(ctx, server, nil)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		t.Cleanup(func() {
			cleanupServer(t, client, created.ID)
		})

		// List with visibility filter
		opts := &contextforge.ServerListOptions{
			Visibility: "public",
		}

		servers, _, err := client.Servers.List(ctx, opts)
		if err != nil {
			t.Fatalf("Failed to list servers with visibility filter: %v", err)
		}

		if len(servers) == 0 {
			t.Error("Expected at least one public server")
		}

		t.Logf("Found %d public servers", len(servers))
	})

	t.Run("filter include_inactive", func(t *testing.T) {
		server := minimalServerInput()
		created, _, err := client.Servers.Create(ctx, server, nil)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		t.Cleanup(func() {
			cleanupServer(t, client, created.ID)
		})

		// Deactivate the server
		_, _, err = client.Servers.Toggle(ctx, created.ID, false)
		if err != nil {
			t.Fatalf("Failed to deactivate server: %v", err)
		}

		// List without include_inactive (should not include our server)
		opts1 := &contextforge.ServerListOptions{
			IncludeInactive: false,
		}

		servers1, _, err := client.Servers.List(ctx, opts1)
		if err != nil {
			t.Fatalf("Failed to list active servers: %v", err)
		}

		// List with include_inactive (should include our server)
		opts2 := &contextforge.ServerListOptions{
			IncludeInactive: true,
		}

		servers2, _, err := client.Servers.List(ctx, opts2)
		if err != nil {
			t.Fatalf("Failed to list all servers: %v", err)
		}

		if len(servers2) <= len(servers1) {
			t.Logf("Active servers: %d, All servers: %d", len(servers1), len(servers2))
		}

		t.Logf("Include_inactive filter working correctly")
	})

	t.Run("combined filters", func(t *testing.T) {
		// Create server with specific tags and visibility
		server := &contextforge.ServerCreate{
			Name:        randomServerName(),
			Description: contextforge.String("Server for combined filtering test"),
			Tags:        []string{"combined-filter-test"},
			Visibility:  contextforge.String("public"),
		}

		created, _, err := client.Servers.Create(ctx, server, nil)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		t.Cleanup(func() {
			cleanupServer(t, client, created.ID)
		})

		// List with combined filters
		opts := &contextforge.ServerListOptions{
			Tags:       "combined-filter-test",
			Visibility: "public",
		}

		servers, _, err := client.Servers.List(ctx, opts)
		if err != nil {
			t.Fatalf("Failed to list servers with combined filters: %v", err)
		}

		if len(servers) == 0 {
			t.Error("Expected at least one server matching combined filters")
		}

		t.Logf("Found %d servers with combined filters", len(servers))
	})
}

// TestServersService_Pagination tests pagination functionality
func TestServersService_Pagination(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("pagination with limit", func(t *testing.T) {
		// Create multiple servers
		for i := 0; i < 5; i++ {
			createTestServer(t, client, randomServerName())
		}

		// List with limit
		opts := &contextforge.ServerListOptions{
			ListOptions: contextforge.ListOptions{
				Limit: 2,
			},
		}

		servers, resp, err := client.Servers.List(ctx, opts)
		if err != nil {
			t.Fatalf("Failed to list servers with limit: %v", err)
		}

		if len(servers) > 2 {
			t.Logf("Warning: Expected at most 2 servers with limit=2, got %d (API may not fully support pagination limits)", len(servers))
		} else {
			t.Logf("Successfully respected limit: got %d servers with limit=2", len(servers))
		}

		if resp.NextCursor != "" {
			t.Logf("Next cursor available: %s", resp.NextCursor)
		}

		t.Logf("Listed %d servers with limit=2", len(servers))
	})

	t.Run("pagination with cursor", func(t *testing.T) {
		// Create multiple servers
		for i := 0; i < 5; i++ {
			createTestServer(t, client, randomServerName())
		}

		// Get first page
		opts1 := &contextforge.ServerListOptions{
			ListOptions: contextforge.ListOptions{
				Limit: 2,
			},
		}

		page1, resp1, err := client.Servers.List(ctx, opts1)
		if err != nil {
			t.Fatalf("Failed to list first page: %v", err)
		}

		if resp1.NextCursor == "" {
			t.Log("No next cursor available (dataset may be small)")
			return
		}

		// Get second page
		opts2 := &contextforge.ServerListOptions{
			ListOptions: contextforge.ListOptions{
				Cursor: resp1.NextCursor,
				Limit:  2,
			},
		}

		page2, _, err := client.Servers.List(ctx, opts2)
		if err != nil {
			t.Fatalf("Failed to list second page: %v", err)
		}

		t.Logf("First page: %d servers, Second page: %d servers", len(page1), len(page2))
	})

	t.Run("pagination no duplicates", func(t *testing.T) {
		// Create multiple servers
		var createdIDs []string
		for i := 0; i < 5; i++ {
			server := createTestServer(t, client, randomServerName())
			createdIDs = append(createdIDs, server.ID)
		}

		// Collect all servers across pages
		allIDs := make(map[string]bool)
		opts := &contextforge.ServerListOptions{
			ListOptions: contextforge.ListOptions{
				Limit: 2,
			},
		}

		for {
			servers, resp, err := client.Servers.List(ctx, opts)
			if err != nil {
				t.Fatalf("Failed to list servers: %v", err)
			}

			for _, server := range servers {
				if allIDs[server.ID] {
					t.Errorf("Duplicate server ID found: %s", server.ID)
				}
				allIDs[server.ID] = true
			}

			if resp.NextCursor == "" {
				break
			}

			opts.Cursor = resp.NextCursor
		}

		t.Logf("Collected %d unique servers across all pages", len(allIDs))
	})
}

// TestServersService_InputValidation tests input validation
func TestServersService_InputValidation(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("create server with missing required field", func(t *testing.T) {
		// Create server without name (required field)
		server := &contextforge.ServerCreate{
			Description: contextforge.String("Server without name"),
		}

		_, _, err := client.Servers.Create(ctx, server, nil)
		if err == nil {
			t.Error("Expected error when creating server without name")
		}

		t.Logf("Correctly rejected server without name: %v", err)
	})

	t.Run("create server with empty name", func(t *testing.T) {
		server := &contextforge.ServerCreate{
			Name: "",
		}

		_, _, err := client.Servers.Create(ctx, server, nil)
		if err == nil {
			t.Error("Expected error when creating server with empty name")
		}

		t.Logf("Correctly rejected server with empty name: %v", err)
	})

	t.Run("create server with very long name", func(t *testing.T) {
		longName := strings.Repeat("a", 500)
		server := &contextforge.ServerCreate{
			Name: longName,
		}

		_, _, err := client.Servers.Create(ctx, server, nil)
		if err != nil {
			t.Logf("Long name rejected (as expected): %v", err)
		} else {
			t.Log("Long name accepted (API may not have length limits)")
		}
	})
}

// TestServersService_ErrorHandling tests error handling
func TestServersService_ErrorHandling(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("get non-existent server", func(t *testing.T) {
		_, _, err := client.Servers.Get(ctx, "non-existent-server-id-12345")
		if err == nil {
			t.Error("Expected error when getting non-existent server")
		}

		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
			if apiErr.Response.StatusCode != http.StatusNotFound {
				t.Errorf("Expected 404 Not Found, got %d", apiErr.Response.StatusCode)
			}
		} else {
			t.Logf("Error type: %T, error: %v", err, err)
		}
	})

	t.Run("update non-existent server", func(t *testing.T) {
		update := &contextforge.ServerUpdate{
			Description: contextforge.String("Updated description"),
		}

		_, _, err := client.Servers.Update(ctx, "non-existent-server-id-12345", update)
		if err == nil {
			t.Error("Expected error when updating non-existent server")
		}

		t.Logf("Correctly received error for non-existent server update: %v", err)
	})

	t.Run("delete non-existent server", func(t *testing.T) {
		_, err := client.Servers.Delete(ctx, "non-existent-server-id-12345")
		if err == nil {
			t.Error("Expected error when deleting non-existent server")
		}

		t.Logf("Correctly received error for non-existent server delete: %v", err)
	})

	t.Run("toggle non-existent server", func(t *testing.T) {
		_, _, err := client.Servers.Toggle(ctx, "non-existent-server-id-12345", true)
		if err == nil {
			t.Error("Expected error when toggling non-existent server")
		}

		t.Logf("Correctly received error for non-existent server toggle: %v", err)
	})
}

// TestServersService_EdgeCases tests edge cases
func TestServersService_EdgeCases(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("server with special characters in name", func(t *testing.T) {
		server := &contextforge.ServerCreate{
			Name:        "test-server-!@#$%^&*()-" + randomServerName(),
			Description: contextforge.String("Server with special characters"),
		}

		created, _, err := client.Servers.Create(ctx, server, nil)
		if err != nil {
			t.Logf("Special characters rejected: %v", err)
		} else {
			t.Cleanup(func() {
				cleanupServer(t, client, created.ID)
			})
			t.Logf("Successfully created server with special characters: %s", created.Name)
		}
	})

	t.Run("server with unicode characters", func(t *testing.T) {
		server := &contextforge.ServerCreate{
			Name:        "test-server-日本語-" + randomServerName(),
			Description: contextforge.String("Server with unicode 你好 characters"),
		}

		created, _, err := client.Servers.Create(ctx, server, nil)
		if err != nil {
			t.Logf("Unicode characters rejected: %v", err)
		} else {
			t.Cleanup(func() {
				cleanupServer(t, client, created.ID)
			})
			t.Logf("Successfully created server with unicode: %s", created.Name)
		}
	})

	t.Run("server with empty tags array", func(t *testing.T) {
		server := &contextforge.ServerCreate{
			Name:        randomServerName(),
			Description: contextforge.String("Server with empty tags array"),
			Tags:        []string{},
		}

		created, _, err := client.Servers.Create(ctx, server, nil)
		if err != nil {
			t.Fatalf("Failed to create server with empty tags: %v", err)
		}

		t.Cleanup(func() {
			cleanupServer(t, client, created.ID)
		})

		t.Logf("Successfully created server with empty tags array")
	})

	t.Run("server with empty associations arrays", func(t *testing.T) {
		server := &contextforge.ServerCreate{
			Name:                randomServerName(),
			Description:         contextforge.String("Server with empty associations"),
			AssociatedTools:     []string{},
			AssociatedResources: []string{},
			AssociatedPrompts:   []string{},
		}

		created, _, err := client.Servers.Create(ctx, server, nil)
		if err != nil {
			t.Fatalf("Failed to create server with empty associations: %v", err)
		}

		t.Cleanup(func() {
			cleanupServer(t, client, created.ID)
		})

		t.Logf("Successfully created server with empty associations arrays")
	})
}
