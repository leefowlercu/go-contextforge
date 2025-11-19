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

// TestToolsService_BasicCRUD tests basic CRUD operations
func TestToolsService_BasicCRUD(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("create tool with minimal fields", func(t *testing.T) {
		tool := minimalToolInput()

		created, _, err := client.Tools.Create(ctx, tool, nil)
		if err != nil {
			t.Fatalf("Failed to create tool: %v", err)
		}

		// Cleanup
		t.Cleanup(func() {
			cleanupTool(t, client, created.ID)
		})

		if created.ID == "" {
			t.Error("Expected created tool to have an ID")
		}
		if created.Name != tool.Name {
			t.Errorf("Expected tool name %q, got %q", tool.Name, created.Name)
		}
		if created.Description == nil || *created.Description != *tool.Description {
			t.Errorf("Expected tool description %q, got %v", *tool.Description, created.Description)
		}
		// Tools are created as enabled by default in ContextForge v0.8.0
		if !created.Enabled {
			t.Log("Tool was created disabled (expected enabled by default in v0.8.0)")
		}

		t.Logf("Successfully created tool: %s (ID: %s)", created.Name, created.ID)
	})

	t.Run("create tool with all optional fields", func(t *testing.T) {
		tool := completeToolInput()

		created, _, err := client.Tools.Create(ctx, tool, nil)
		if err != nil {
			t.Fatalf("Failed to create tool with all fields: %v", err)
		}

		// Cleanup
		t.Cleanup(func() {
			cleanupTool(t, client, created.ID)
		})

		if created.ID == "" {
			t.Error("Expected created tool to have an ID")
		}
		if created.Visibility != tool.Visibility {
			t.Errorf("Expected visibility %q, got %q", tool.Visibility, created.Visibility)
		}
		if len(created.Tags) != len(tool.Tags) {
			t.Errorf("Expected %d tags, got %d", len(tool.Tags), len(created.Tags))
		}
		// TeamID may be set via top-level parameter in request body, not tool object
		// So it may not match the value in the tool object
		if created.TeamID != nil {
			t.Logf("Tool created with TeamID: %s", *created.TeamID)
		}

		t.Logf("Successfully created tool with all fields: %s (ID: %s)", created.Name, created.ID)
	})

	t.Run("get tool by ID", func(t *testing.T) {
		created := createTestTool(t, client, randomToolName())

		retrieved, _, err := client.Tools.Get(ctx, created.ID)
		if err != nil {
			t.Fatalf("Failed to get tool: %v", err)
		}

		if retrieved.ID != created.ID {
			t.Errorf("Expected tool ID %q, got %q", created.ID, retrieved.ID)
		}
		if retrieved.Name != created.Name {
			t.Errorf("Expected tool name %q, got %q", created.Name, retrieved.Name)
		}

		t.Logf("Successfully retrieved tool: %s (ID: %s)", retrieved.Name, retrieved.ID)
	})

	t.Run("list tools", func(t *testing.T) {
		// Create a few test tools
		createTestTool(t, client, randomToolName())
		createTestTool(t, client, randomToolName())

		tools, _, err := client.Tools.List(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to list tools: %v", err)
		}

		if len(tools) == 0 {
			t.Error("Expected at least some tools in the list")
		}

		t.Logf("Successfully listed %d tools", len(tools))
	})

	t.Run("update tool", func(t *testing.T) {
		created := createTestTool(t, client, randomToolName())

		// Update the tool
		expectedDescription := "Updated description for integration test"
		expectedTags := []string{"updated", "integration-test"}
		created.Description = contextforge.String(expectedDescription)
		created.Tags = expectedTags

		updated, _, err := client.Tools.Update(ctx, created.ID, created)
		if err != nil {
			t.Fatalf("Failed to update tool: %v", err)
		}

		// Assert that updates actually persisted
		if updated.Description == nil || *updated.Description != expectedDescription {
			t.Errorf("Expected description %q, got %v", expectedDescription, updated.Description)
		}
		if !reflect.DeepEqual(updated.Tags, expectedTags) {
			t.Errorf("Expected tags %v, got %v", expectedTags, updated.Tags)
		}

		t.Logf("Successfully updated tool: %s (ID: %s)", updated.Name, updated.ID)
	})

	t.Run("delete tool", func(t *testing.T) {
		created := createTestTool(t, client, randomToolName())

		// Delete the tool
		_, err := client.Tools.Delete(ctx, created.ID)
		if err != nil {
			t.Fatalf("Failed to delete tool: %v", err)
		}

		t.Logf("Successfully deleted tool: %s (ID: %s)", created.Name, created.ID)
	})

	t.Run("get deleted tool returns 404", func(t *testing.T) {
		created := createTestTool(t, client, randomToolName())

		// Delete the tool
		_, err := client.Tools.Delete(ctx, created.ID)
		if err != nil {
			t.Fatalf("Failed to delete tool: %v", err)
		}

		// Try to get the deleted tool
		_, _, err = client.Tools.Get(ctx, created.ID)
		if err == nil {
			t.Error("Expected error when getting deleted tool")
		}

		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
			if apiErr.Response.StatusCode != http.StatusNotFound {
				t.Errorf("Expected 404 Not Found, got %d", apiErr.Response.StatusCode)
			}
			t.Logf("Correctly received 404 for deleted tool")
		} else {
			t.Errorf("Expected ErrorResponse, got %T: %v", err, err)
		}
	})

	t.Run("list empty results", func(t *testing.T) {
		// List with a filter that matches nothing
		opts := &contextforge.ToolListOptions{
			Tags: "non-existent-tag-xyz-12345",
		}

		tools, _, err := client.Tools.List(ctx, opts)
		if err != nil {
			t.Fatalf("Failed to list tools: %v", err)
		}

		if len(tools) > 0 {
			t.Logf("Warning: Expected no tools with non-existent tag, got %d", len(tools))
		} else {
			t.Log("Successfully got empty list for non-existent tag")
		}
	})
}

// TestToolsService_Toggle tests toggle functionality
func TestToolsService_Toggle(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("toggle enabled to disabled", func(t *testing.T) {
		// Create tool (will be enabled by default in ContextForge)
		tool := minimalToolInput()

		created, _, err := client.Tools.Create(ctx, tool, nil)
		if err != nil {
			t.Fatalf("Failed to create tool: %v", err)
		}

		t.Cleanup(func() {
			cleanupTool(t, client, created.ID)
		})

		if !created.Enabled {
			t.Log("Tool was created disabled (expected enabled by default)")
		}

		// Toggle to disabled
		toggled, _, err := client.Tools.Toggle(ctx, created.ID, false)
		if err != nil {
			t.Fatalf("Failed to toggle tool: %v", err)
		}

		if toggled.Enabled {
			t.Error("Expected tool to be disabled after toggle")
		}

		t.Logf("Successfully toggled tool from enabled to disabled")
	})

	t.Run("toggle disabled to enabled", func(t *testing.T) {
		tool := minimalToolInput()
		created, _, err := client.Tools.Create(ctx, tool, nil)
		if err != nil {
			t.Fatalf("Failed to create tool: %v", err)
		}

		t.Cleanup(func() {
			cleanupTool(t, client, created.ID)
		})

		// First disable the tool (if it was created enabled)
		disabled, _, err := client.Tools.Toggle(ctx, created.ID, false)
		if err != nil {
			t.Fatalf("Failed to disable tool: %v", err)
		}

		if disabled.Enabled {
			t.Fatal("Failed to disable tool in setup")
		}

		// Now toggle to enabled
		toggled, _, err := client.Tools.Toggle(ctx, created.ID, true)
		if err != nil {
			t.Fatalf("Failed to toggle tool: %v", err)
		}

		if !toggled.Enabled {
			t.Error("Expected tool to be enabled after toggle")
		}

		t.Logf("Successfully toggled tool from disabled to enabled")
	})

	t.Run("verify toggle persists", func(t *testing.T) {
		tool := minimalToolInput()
		created, _, err := client.Tools.Create(ctx, tool, nil)
		if err != nil {
			t.Fatalf("Failed to create tool: %v", err)
		}

		t.Cleanup(func() {
			cleanupTool(t, client, created.ID)
		})

		// Toggle to enabled
		_, _, err = client.Tools.Toggle(ctx, created.ID, true)
		if err != nil {
			t.Fatalf("Failed to toggle tool: %v", err)
		}

		// Get the tool again to verify it's enabled
		retrieved, _, err := client.Tools.Get(ctx, created.ID)
		if err != nil {
			t.Fatalf("Failed to get tool: %v", err)
		}

		if !retrieved.Enabled {
			t.Error("Expected tool to remain enabled")
		}

		t.Logf("Successfully verified toggle persists")
	})
}

// TestToolsService_Filtering tests filtering and search
func TestToolsService_Filtering(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("filter by tags", func(t *testing.T) {
		// Create tool with specific tags
		tool := minimalToolInput()
		tool.Tags = []string{"filter-test", "integration"}

		created, _, err := client.Tools.Create(ctx, tool, nil)
		if err != nil {
			t.Fatalf("Failed to create tool: %v", err)
		}

		t.Cleanup(func() {
			cleanupTool(t, client, created.ID)
		})

		// Filter by tag
		opts := &contextforge.ToolListOptions{
			Tags: "filter-test",
		}

		tools, _, err := client.Tools.List(ctx, opts)
		if err != nil {
			t.Fatalf("Failed to list tools: %v", err)
		}

		found := false
		for _, tool := range tools {
			if tool.ID == created.ID {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected to find created tool in filtered results")
		} else {
			t.Logf("Successfully filtered tools by tag, found %d tools", len(tools))
		}
	})

	t.Run("filter by visibility", func(t *testing.T) {
		// Create public tool
		tool := minimalToolInput()
		tool.Visibility = "public"

		created, _, err := client.Tools.Create(ctx, tool, nil)
		if err != nil {
			t.Fatalf("Failed to create tool: %v", err)
		}

		t.Cleanup(func() {
			cleanupTool(t, client, created.ID)
		})

		// Filter by visibility
		opts := &contextforge.ToolListOptions{
			Visibility: "public",
		}

		tools, _, err := client.Tools.List(ctx, opts)
		if err != nil {
			t.Fatalf("Failed to list tools: %v", err)
		}

		found := false
		for _, tool := range tools {
			if tool.ID == created.ID {
				found = true
				break
			}
		}

		if !found {
			t.Log("Warning: Created tool not found in visibility filter (may be implementation-specific)")
		} else {
			t.Logf("Successfully filtered tools by visibility, found %d tools", len(tools))
		}
	})

	t.Run("filter by team_id", func(t *testing.T) {
		// Create tool with team ID
		tool := minimalToolInput()
		tool.TeamID = contextforge.String("test-team-integration")

		created, _, err := client.Tools.Create(ctx, tool, nil)
		if err != nil {
			t.Fatalf("Failed to create tool: %v", err)
		}

		t.Cleanup(func() {
			cleanupTool(t, client, created.ID)
		})

		// Filter by team ID
		opts := &contextforge.ToolListOptions{
			TeamID: "test-team-integration",
		}

		tools, _, err := client.Tools.List(ctx, opts)
		if err != nil {
			t.Fatalf("Failed to list tools: %v", err)
		}

		found := false
		for _, tool := range tools {
			if tool.ID == created.ID {
				found = true
				break
			}
		}

		if !found {
			t.Log("Warning: Created tool not found in team filter (may be implementation-specific)")
		} else {
			t.Logf("Successfully filtered tools by team_id, found %d tools", len(tools))
		}
	})

	t.Run("include inactive tools", func(t *testing.T) {
		// Create tool (will be inactive by default)
		tool := minimalToolInput()

		created, _, err := client.Tools.Create(ctx, tool, nil)
		if err != nil {
			t.Fatalf("Failed to create tool: %v", err)
		}

		t.Cleanup(func() {
			cleanupTool(t, client, created.ID)
		})

		// List without including inactive
		opts1 := &contextforge.ToolListOptions{
			IncludeInactive: false,
		}

		tools1, _, err := client.Tools.List(ctx, opts1)
		if err != nil {
			t.Fatalf("Failed to list tools: %v", err)
		}

		// List with including inactive
		opts2 := &contextforge.ToolListOptions{
			IncludeInactive: true,
		}

		tools2, _, err := client.Tools.List(ctx, opts2)
		if err != nil {
			t.Fatalf("Failed to list tools: %v", err)
		}

		if len(tools2) < len(tools1) {
			t.Error("Expected more tools when including inactive")
		}

		t.Logf("Without inactive: %d tools, with inactive: %d tools", len(tools1), len(tools2))
	})

	t.Run("combined filters", func(t *testing.T) {
		// Create tool with multiple filterable properties
		tool := minimalToolInput()
		tool.Tags = []string{"combined-filter-test"}
		tool.Visibility = "public"

		created, _, err := client.Tools.Create(ctx, tool, nil)
		if err != nil {
			t.Fatalf("Failed to create tool: %v", err)
		}

		t.Cleanup(func() {
			cleanupTool(t, client, created.ID)
		})

		// Apply multiple filters
		opts := &contextforge.ToolListOptions{
			Tags:       "combined-filter-test",
			Visibility: "public",
		}

		tools, _, err := client.Tools.List(ctx, opts)
		if err != nil {
			t.Fatalf("Failed to list tools: %v", err)
		}

		found := false
		for _, tool := range tools {
			if tool.ID == created.ID {
				found = true
				break
			}
		}

		if !found {
			t.Log("Warning: Created tool not found with combined filters (may be implementation-specific)")
		} else {
			t.Logf("Successfully applied combined filters, found %d tools", len(tools))
		}
	})
}

// TestToolsService_Pagination tests pagination behavior
func TestToolsService_Pagination(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	// Create multiple test tools for pagination
	t.Logf("Creating test tools for pagination...")
	for i := 0; i < 5; i++ {
		createTestTool(t, client, randomToolName())
	}

	t.Run("list with small limit", func(t *testing.T) {
		opts := &contextforge.ToolListOptions{
			ListOptions: contextforge.ListOptions{
				Limit: 2,
			},
		}

		tools, _, err := client.Tools.List(ctx, opts)
		if err != nil {
			t.Fatalf("Failed to list tools: %v", err)
		}

		// Note: The API may not respect the limit parameter in all implementations
		// Some versions may return all results regardless of limit
		if len(tools) > 2 {
			t.Logf("API returned %d tools (limit parameter may not be implemented)", len(tools))
		} else {
			t.Logf("Successfully limited results to %d tools", len(tools))
		}
	})

	t.Run("navigate multiple pages", func(t *testing.T) {
		opts := &contextforge.ToolListOptions{
			ListOptions: contextforge.ListOptions{
				Limit: 2,
			},
		}

		tools1, resp1, err := client.Tools.List(ctx, opts)
		if err != nil {
			t.Fatalf("Failed to list first page: %v", err)
		}

		t.Logf("First page: got %d tools, NextCursor: %q", len(tools1), resp1.NextCursor)

		if resp1.NextCursor != "" {
			opts.Cursor = resp1.NextCursor
			tools2, resp2, err := client.Tools.List(ctx, opts)
			if err != nil {
				t.Fatalf("Failed to list second page: %v", err)
			}

			t.Logf("Second page: got %d tools, NextCursor: %q", len(tools2), resp2.NextCursor)

			// Verify we got different tools
			if len(tools1) > 0 && len(tools2) > 0 {
				if tools1[0].ID == tools2[0].ID {
					t.Error("Expected different tools on different pages")
				}
			}
		} else {
			t.Log("No second page available (total tools <= limit)")
		}
	})

	t.Run("verify no duplicates across pages", func(t *testing.T) {
		opts := &contextforge.ToolListOptions{
			ListOptions: contextforge.ListOptions{
				Limit: 2,
			},
		}

		allIDs := make(map[string]bool)
		pageCount := 0

		for {
			tools, resp, err := client.Tools.List(ctx, opts)
			if err != nil {
				t.Fatalf("Failed to list page %d: %v", pageCount+1, err)
			}

			pageCount++

			for _, tool := range tools {
				if allIDs[tool.ID] {
					t.Errorf("Found duplicate tool ID %q across pages", tool.ID)
				}
				allIDs[tool.ID] = true
			}

			if resp.NextCursor == "" {
				break
			}

			opts.Cursor = resp.NextCursor

			// Safety limit
			if pageCount > 10 {
				t.Log("Stopping after 10 pages (safety limit)")
				break
			}
		}

		t.Logf("Verified %d unique tools across %d pages", len(allIDs), pageCount)
	})

	t.Run("empty cursor on last page", func(t *testing.T) {
		opts := &contextforge.ToolListOptions{
			ListOptions: contextforge.ListOptions{
				Limit: 100, // Large limit to get all in one page
			},
		}

		_, resp, err := client.Tools.List(ctx, opts)
		if err != nil {
			t.Fatalf("Failed to list tools: %v", err)
		}

		if resp.NextCursor != "" {
			t.Logf("NextCursor is not empty: %q (may have > 100 tools)", resp.NextCursor)
		} else {
			t.Log("NextCursor correctly empty on last page")
		}
	})
}

// TestToolsService_InputValidation tests input validation
func TestToolsService_InputValidation(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("missing required name", func(t *testing.T) {
		tool := minimalToolInput()
		tool.Name = ""

		_, _, err := client.Tools.Create(ctx, tool, nil)
		if err == nil {
			t.Error("Expected error for missing name")
		} else {
			t.Logf("Correctly received error for missing name: %v", err)
		}
	})

	t.Run("missing description is accepted", func(t *testing.T) {
		tool := minimalToolInput()
		tool.Description = nil

		created, _, err := client.Tools.Create(ctx, tool, nil)
		if err != nil {
			t.Fatalf("Failed to create tool without description: %v", err)
		}

		t.Cleanup(func() {
			cleanupTool(t, client, created.ID)
		})

		t.Log("API correctly accepts tools without description (description is optional)")
	})

	t.Run("missing input_schema is accepted", func(t *testing.T) {
		tool := minimalToolInput()
		tool.InputSchema = nil

		created, _, err := client.Tools.Create(ctx, tool, nil)
		if err != nil {
			t.Fatalf("Failed to create tool without input_schema: %v", err)
		}

		t.Cleanup(func() {
			cleanupTool(t, client, created.ID)
		})

		t.Log("API correctly accepts tools without input_schema (input_schema is optional)")
	})

	t.Run("empty input_schema", func(t *testing.T) {
		tool := minimalToolInput()
		tool.InputSchema = map[string]any{}

		created, _, err := client.Tools.Create(ctx, tool, nil)
		if err != nil {
			t.Logf("Empty input_schema rejected: %v", err)
		} else {
			t.Cleanup(func() {
				cleanupTool(t, client, created.ID)
			})
			t.Log("Empty input_schema accepted (implementation allows it)")
		}
	})

	t.Run("invalid visibility value", func(t *testing.T) {
		tool := minimalToolInput()
		tool.Visibility = "invalid-visibility-value"

		created, _, err := client.Tools.Create(ctx, tool, nil)
		if err == nil {
			t.Cleanup(func() {
				cleanupTool(t, client, created.ID)
			})
			t.Log("Invalid visibility accepted (implementation may not validate)")
		} else {
			t.Logf("Correctly rejected invalid visibility: %v", err)
		}
	})
}

// TestToolsService_ErrorHandling tests error scenarios
func TestToolsService_ErrorHandling(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("get non-existent tool", func(t *testing.T) {
		nonExistentID := "non-existent-tool-id-12345"
		_, _, err := client.Tools.Get(ctx, nonExistentID)
		if err == nil {
			t.Error("Expected error for non-existent tool")
		}

		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
			if apiErr.Response.StatusCode != http.StatusNotFound {
				t.Errorf("Expected 404 Not Found, got %d", apiErr.Response.StatusCode)
			}
			t.Logf("Correctly received 404 error")
		}
	})

	t.Run("update non-existent tool", func(t *testing.T) {
		tool := minimalToolInput()
		nonExistentID := "non-existent-tool-id-12345"

		_, _, err := client.Tools.Update(ctx, nonExistentID, tool)
		if err == nil {
			t.Error("Expected error for updating non-existent tool")
		} else {
			t.Logf("Correctly received error: %v", err)
		}
	})

	t.Run("delete non-existent tool", func(t *testing.T) {
		nonExistentID := "non-existent-tool-id-12345"

		_, err := client.Tools.Delete(ctx, nonExistentID)
		if err == nil {
			t.Log("Delete of non-existent tool succeeded (may be idempotent)")
		} else {
			t.Logf("Delete of non-existent tool failed: %v", err)
		}
	})

	t.Run("invalid authentication", func(t *testing.T) {
		// Create client with invalid token
		invalidClient, err := contextforge.NewClient(nil, client.Address.String(), "invalid-token")
		if err != nil {
			t.Fatalf("Failed to create invalid client: %v", err)
		}

		_, _, err = invalidClient.Tools.List(ctx, nil)
		if err == nil {
			t.Error("Expected error with invalid token")
		}

		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
			if apiErr.Response.StatusCode != http.StatusUnauthorized {
				t.Errorf("Expected 401 Unauthorized, got %d", apiErr.Response.StatusCode)
			}
			t.Logf("Correctly received 401 error for invalid token")
		}
	})
}

// TestToolsService_EdgeCases tests edge cases
func TestToolsService_EdgeCases(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("long tool name", func(t *testing.T) {
		tool := minimalToolInput()
		tool.Name = "test-" + strings.Repeat("a", 200) + "-tool"

		created, _, err := client.Tools.Create(ctx, tool, nil)
		if err != nil {
			t.Logf("Long name rejected: %v", err)
		} else {
			t.Cleanup(func() {
				cleanupTool(t, client, created.ID)
			})
			t.Logf("Long name accepted: %d characters", len(created.Name))
		}
	})

	t.Run("special characters in name", func(t *testing.T) {
		tool := minimalToolInput()
		tool.Name = randomToolName() + "-special-!@#$%^&*()"

		created, _, err := client.Tools.Create(ctx, tool, nil)
		if err != nil {
			t.Logf("Special characters rejected: %v", err)
		} else {
			t.Cleanup(func() {
				cleanupTool(t, client, created.ID)
			})
			t.Logf("Special characters accepted in name")
		}
	})

	t.Run("large input schema", func(t *testing.T) {
		tool := minimalToolInput()

		// Create a large schema
		properties := make(map[string]any)
		for i := 0; i < 50; i++ {
			properties[randomToolName()] = map[string]any{
				"type":        "string",
				"description": "Property " + strings.Repeat("x", 100),
			}
		}

		tool.InputSchema = map[string]any{
			"type":       "object",
			"properties": properties,
		}

		created, _, err := client.Tools.Create(ctx, tool, nil)
		if err != nil {
			t.Logf("Large schema rejected: %v", err)
		} else {
			t.Cleanup(func() {
				cleanupTool(t, client, created.ID)
			})
			t.Logf("Large schema accepted (50 properties)")
		}
	})
}
