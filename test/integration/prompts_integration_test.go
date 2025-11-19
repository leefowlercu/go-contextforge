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

// TestPromptsService_BasicCRUD tests basic CRUD operations
func TestPromptsService_BasicCRUD(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("create prompt with minimal fields", func(t *testing.T) {
		prompt := minimalPromptInput()

		created, _, err := client.Prompts.Create(ctx, prompt, nil)
		if err != nil {
			t.Fatalf("Failed to create prompt: %v", err)
		}

		t.Cleanup(func() {
			cleanupPrompt(t, client, created.ID)
		})

		if created.ID == 0 {
			t.Error("Expected created prompt to have an ID")
		}
		if created.Name != prompt.Name {
			t.Errorf("Expected prompt name %q, got %q", prompt.Name, created.Name)
		}
		if created.Template != prompt.Template {
			t.Errorf("Expected prompt template %q, got %q", prompt.Template, created.Template)
		}
		if created.Metrics == nil {
			t.Error("Expected created prompt to have metrics")
		}

		t.Logf("Successfully created prompt: %s (ID: %d)", created.Name, created.ID)
	})

	t.Run("create prompt with all optional fields", func(t *testing.T) {
		prompt := completePromptInput()

		created, _, err := client.Prompts.Create(ctx, prompt, nil)
		if err != nil {
			t.Fatalf("Failed to create prompt with all fields: %v", err)
		}

		t.Cleanup(func() {
			cleanupPrompt(t, client, created.ID)
		})

		if created.ID == 0 {
			t.Error("Expected created prompt to have an ID")
		}
		if created.Visibility == nil || *created.Visibility != *prompt.Visibility {
			t.Errorf("Expected visibility %q, got %v", *prompt.Visibility, created.Visibility)
		}
		if len(created.Tags) != len(prompt.Tags) {
			t.Errorf("Expected %d tags, got %d", len(prompt.Tags), len(created.Tags))
		}
		if len(created.Arguments) != len(prompt.Arguments) {
			t.Errorf("Expected %d arguments, got %d", len(prompt.Arguments), len(created.Arguments))
		}

		t.Logf("Successfully created prompt with all fields: %s (ID: %d)", created.Name, created.ID)
	})

	t.Run("list prompts", func(t *testing.T) {
		// Create a few test prompts
		createTestPrompt(t, client, randomPromptName())
		createTestPrompt(t, client, randomPromptName())

		prompts, _, err := client.Prompts.List(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to list prompts: %v", err)
		}

		if len(prompts) == 0 {
			t.Error("Expected at least some prompts in the list")
		}

		t.Logf("Successfully listed %d prompts", len(prompts))
	})

	t.Run("update prompt", func(t *testing.T) {
		created := createTestPrompt(t, client, randomPromptName())

		// Update the prompt
		expectedDescription := "Updated description for integration test"
		expectedTags := []string{"updated", "integration-test"}
		update := &contextforge.PromptUpdate{
			Description: contextforge.String(expectedDescription),
			Tags:        expectedTags,
		}

		updated, _, err := client.Prompts.Update(ctx, created.ID, update)
		if err != nil {
			t.Fatalf("Failed to update prompt: %v", err)
		}

		// Assert that updates actually persisted
		if updated.Description == nil || *updated.Description != expectedDescription {
			t.Errorf("Expected description %q, got %v", expectedDescription, updated.Description)
		}
		if !reflect.DeepEqual(updated.Tags, expectedTags) {
			t.Errorf("Expected tags %v, got %v", expectedTags, updated.Tags)
		}

		t.Logf("Successfully updated prompt: %s (ID: %d)", updated.Name, updated.ID)
	})

	t.Run("delete prompt", func(t *testing.T) {
		created := createTestPrompt(t, client, randomPromptName())

		// Delete the prompt
		_, err := client.Prompts.Delete(ctx, created.ID)
		if err != nil {
			t.Fatalf("Failed to delete prompt: %v", err)
		}

		t.Logf("Successfully deleted prompt: %s (ID: %d)", created.Name, created.ID)
	})

	t.Run("list deleted prompt returns empty or excludes it", func(t *testing.T) {
		created := createTestPrompt(t, client, randomPromptName())
		promptID := created.ID

		// Delete the prompt
		_, err := client.Prompts.Delete(ctx, promptID)
		if err != nil {
			t.Fatalf("Failed to delete prompt: %v", err)
		}

		// List prompts - the deleted one should not be in active list
		prompts, _, err := client.Prompts.List(ctx, &contextforge.PromptListOptions{
			IncludeInactive: false,
		})
		if err != nil {
			t.Fatalf("Failed to list prompts: %v", err)
		}

		// Verify the deleted prompt is not in the active list
		for _, p := range prompts {
			if p.ID == promptID {
				t.Errorf("Deleted prompt %d should not be in active list", promptID)
			}
		}

		t.Logf("Correctly excluded deleted prompt from active list")
	})
}

// TestPromptsService_Toggle tests toggle functionality
func TestPromptsService_Toggle(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("toggle active to inactive", func(t *testing.T) {
		prompt := minimalPromptInput()

		created, _, err := client.Prompts.Create(ctx, prompt, nil)
		if err != nil {
			t.Fatalf("Failed to create prompt: %v", err)
		}

		t.Cleanup(func() {
			cleanupPrompt(t, client, created.ID)
		})

		initialState := created.IsActive
		t.Logf("Prompt initial state: isActive=%v", initialState)

		// Toggle to inactive
		toggled, _, err := client.Prompts.Toggle(ctx, created.ID, false)
		if err != nil {
			t.Fatalf("Failed to toggle prompt: %v", err)
		}

		if toggled.IsActive {
			t.Errorf("Expected prompt to be inactive after toggle(false), got isActive=%v", toggled.IsActive)
		}

		t.Logf("Successfully toggled prompt to inactive")
	})

	t.Run("toggle inactive to active", func(t *testing.T) {
		t.Skip("Skipping due to upstream ContextForge bug - see docs/upstream-bugs/prompt-toggle.md")

		prompt := minimalPromptInput()

		created, _, err := client.Prompts.Create(ctx, prompt, nil)
		if err != nil {
			t.Fatalf("Failed to create prompt: %v", err)
		}

		t.Cleanup(func() {
			cleanupPrompt(t, client, created.ID)
		})

		// Toggle to inactive first
		_, _, err = client.Prompts.Toggle(ctx, created.ID, false)
		if err != nil {
			t.Fatalf("Failed to toggle prompt to inactive: %v", err)
		}

		// Toggle back to active
		toggled, _, err := client.Prompts.Toggle(ctx, created.ID, true)
		if err != nil {
			t.Fatalf("Failed to toggle prompt to active: %v", err)
		}

		if !toggled.IsActive {
			t.Errorf("Expected prompt to be active after toggle(true), got isActive=%v", toggled.IsActive)
		}

		t.Logf("Successfully toggled prompt to active")
	})

	t.Run("toggle state persists", func(t *testing.T) {
		prompt := minimalPromptInput()

		created, _, err := client.Prompts.Create(ctx, prompt, nil)
		if err != nil {
			t.Fatalf("Failed to create prompt: %v", err)
		}

		t.Cleanup(func() {
			cleanupPrompt(t, client, created.ID)
		})

		// Toggle to inactive
		_, _, err = client.Prompts.Toggle(ctx, created.ID, false)
		if err != nil {
			t.Fatalf("Failed to toggle prompt: %v", err)
		}

		// List prompts with include_inactive to verify state
		prompts, _, err := client.Prompts.List(ctx, &contextforge.PromptListOptions{
			IncludeInactive: true,
		})
		if err != nil {
			t.Fatalf("Failed to list prompts: %v", err)
		}

		// Find our prompt in the list
		var found *contextforge.Prompt
		for _, p := range prompts {
			if p.ID == created.ID {
				found = p
				break
			}
		}

		if found == nil {
			t.Fatal("Created prompt not found in list")
		}

		if found.IsActive {
			t.Errorf("Expected prompt to remain inactive, got isActive=%v", found.IsActive)
		}

		t.Logf("Toggle state correctly persisted")
	})
}

// TestPromptsService_Filtering tests filtering options
func TestPromptsService_Filtering(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("filter by tags", func(t *testing.T) {
		// Create prompt with specific tag
		prompt := minimalPromptInput()
		prompt.Tags = []string{"test-filter-tag"}

		created, _, err := client.Prompts.Create(ctx, prompt, nil)
		if err != nil {
			t.Fatalf("Failed to create prompt: %v", err)
		}

		t.Cleanup(func() {
			cleanupPrompt(t, client, created.ID)
		})

		// List with tag filter
		prompts, _, err := client.Prompts.List(ctx, &contextforge.PromptListOptions{
			Tags: "test-filter-tag",
		})
		if err != nil {
			t.Fatalf("Failed to list prompts with tag filter: %v", err)
		}

		// Verify our prompt is in the results
		found := false
		for _, p := range prompts {
			if p.ID == created.ID {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected to find prompt with matching tag")
		}

		t.Logf("Successfully filtered prompts by tag")
	})

	t.Run("filter by visibility", func(t *testing.T) {
		// Create prompt with specific visibility
		prompt := minimalPromptInput()
		prompt.Visibility = contextforge.String("public")

		created, _, err := client.Prompts.Create(ctx, prompt, nil)
		if err != nil {
			t.Fatalf("Failed to create prompt: %v", err)
		}

		t.Cleanup(func() {
			cleanupPrompt(t, client, created.ID)
		})

		// List with visibility filter
		prompts, _, err := client.Prompts.List(ctx, &contextforge.PromptListOptions{
			Visibility: "public",
		})
		if err != nil {
			t.Fatalf("Failed to list prompts with visibility filter: %v", err)
		}

		// Verify we got some results
		if len(prompts) == 0 {
			t.Error("Expected at least one prompt with public visibility")
		}

		t.Logf("Successfully filtered prompts by visibility")
	})

	t.Run("include inactive prompts", func(t *testing.T) {
		prompt := minimalPromptInput()

		created, _, err := client.Prompts.Create(ctx, prompt, nil)
		if err != nil {
			t.Fatalf("Failed to create prompt: %v", err)
		}

		t.Cleanup(func() {
			cleanupPrompt(t, client, created.ID)
		})

		// Toggle to inactive
		_, _, err = client.Prompts.Toggle(ctx, created.ID, false)
		if err != nil {
			t.Fatalf("Failed to toggle prompt: %v", err)
		}

		// List without include_inactive (should not find it)
		activePrompts, _, err := client.Prompts.List(ctx, &contextforge.PromptListOptions{
			IncludeInactive: false,
		})
		if err != nil {
			t.Fatalf("Failed to list active prompts: %v", err)
		}

		foundInActive := false
		for _, p := range activePrompts {
			if p.ID == created.ID {
				foundInActive = true
				break
			}
		}

		// List with include_inactive (should find it)
		allPrompts, _, err := client.Prompts.List(ctx, &contextforge.PromptListOptions{
			IncludeInactive: true,
		})
		if err != nil {
			t.Fatalf("Failed to list all prompts: %v", err)
		}

		foundInAll := false
		for _, p := range allPrompts {
			if p.ID == created.ID {
				foundInAll = true
				break
			}
		}

		if foundInActive {
			t.Error("Inactive prompt should not be in active list")
		}

		if !foundInAll {
			t.Error("Inactive prompt should be in list when include_inactive=true")
		}

		t.Logf("Successfully tested include_inactive filter")
	})
}

// TestPromptsService_Pagination tests cursor-based pagination
func TestPromptsService_Pagination(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("pagination with cursor", func(t *testing.T) {
		// Create several test prompts
		var createdIDs []int
		for i := 0; i < 5; i++ {
			created := createTestPrompt(t, client, randomPromptName())
			createdIDs = append(createdIDs, created.ID)
		}

		// List with small limit
		prompts, resp, err := client.Prompts.List(ctx, &contextforge.PromptListOptions{
			ListOptions: contextforge.ListOptions{
				Limit: 2,
			},
		})
		if err != nil {
			t.Fatalf("Failed to list prompts: %v", err)
		}

		firstPageCount := len(prompts)
		t.Logf("First page returned %d prompts", firstPageCount)

		// If there's a next cursor, fetch next page
		if resp.NextCursor != "" {
			nextPrompts, _, err := client.Prompts.List(ctx, &contextforge.PromptListOptions{
				ListOptions: contextforge.ListOptions{
					Limit:  2,
					Cursor: resp.NextCursor,
				},
			})
			if err != nil {
				t.Fatalf("Failed to list next page: %v", err)
			}

			t.Logf("Second page returned %d prompts", len(nextPrompts))

			// Verify pages don't overlap
			firstIDs := make(map[int]bool)
			for _, p := range prompts {
				firstIDs[p.ID] = true
			}

			for _, p := range nextPrompts {
				if firstIDs[p.ID] {
					t.Errorf("Prompt %d appears in both pages (pagination overlap)", p.ID)
				}
			}

			t.Logf("Successfully paginated prompts with no overlap")
		} else {
			t.Logf("No next cursor (all results fit in first page)")
		}
	})
}

// TestPromptsService_InputValidation tests input validation
func TestPromptsService_InputValidation(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("create prompt without required name", func(t *testing.T) {
		prompt := &contextforge.PromptCreate{
			Template: "Hello {{name}}!",
		}

		_, _, err := client.Prompts.Create(ctx, prompt, nil)
		if err == nil {
			t.Error("Expected error when creating prompt without name")
		}

		t.Logf("Correctly rejected prompt without name: %v", err)
	})

	t.Run("create prompt without required template", func(t *testing.T) {
		t.Skip("Skipping due to upstream ContextForge bug - see docs/upstream-bugs/prompt-validation-missing-template.md")

		prompt := &contextforge.PromptCreate{
			Name: randomPromptName(),
		}

		_, _, err := client.Prompts.Create(ctx, prompt, nil)
		if err == nil {
			t.Error("Expected error when creating prompt without template")
		}

		t.Logf("Correctly rejected prompt without template: %v", err)
	})
}

// TestPromptsService_ErrorHandling tests error scenarios
func TestPromptsService_ErrorHandling(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("update non-existent prompt returns 404", func(t *testing.T) {
		update := &contextforge.PromptUpdate{
			Description: contextforge.String("This should fail"),
		}

		_, _, err := client.Prompts.Update(ctx, 99999999, update)
		if err == nil {
			t.Error("Expected error when updating non-existent prompt")
		}

		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
			if apiErr.Response.StatusCode != http.StatusNotFound {
				t.Errorf("Expected 404 Not Found, got %d", apiErr.Response.StatusCode)
			}
			t.Logf("Correctly received 404 for non-existent prompt")
		} else {
			t.Logf("Received error (may not be ErrorResponse): %v", err)
		}
	})

	t.Run("delete non-existent prompt returns 404", func(t *testing.T) {
		_, err := client.Prompts.Delete(ctx, 99999999)
		if err == nil {
			t.Error("Expected error when deleting non-existent prompt")
		}

		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
			if apiErr.Response.StatusCode != http.StatusNotFound {
				t.Errorf("Expected 404 Not Found, got %d", apiErr.Response.StatusCode)
			}
			t.Logf("Correctly received 404 for non-existent prompt")
		} else {
			t.Logf("Received error (may not be ErrorResponse): %v", err)
		}
	})

	t.Run("toggle non-existent prompt returns 404", func(t *testing.T) {
		t.Skip("Skipping due to upstream ContextForge bug - see docs/upstream-bugs/prompt-toggle-error-code.md")

		_, _, err := client.Prompts.Toggle(ctx, 99999999, true)
		if err == nil {
			t.Error("Expected error when toggling non-existent prompt")
		}

		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
			if apiErr.Response.StatusCode != http.StatusNotFound {
				t.Errorf("Expected 404 Not Found, got %d", apiErr.Response.StatusCode)
			}
			t.Logf("Correctly received 404 for non-existent prompt")
		} else {
			t.Logf("Received error (may not be ErrorResponse): %v", err)
		}
	})
}

// TestPromptsService_EdgeCases tests edge cases
func TestPromptsService_EdgeCases(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("create prompt with long name", func(t *testing.T) {
		prompt := minimalPromptInput()
		prompt.Name = "test-prompt-with-very-long-name-" + strings.Repeat("x", 100)

		created, _, err := client.Prompts.Create(ctx, prompt, nil)
		if err != nil {
			t.Fatalf("Failed to create prompt with long name: %v", err)
		}

		t.Cleanup(func() {
			cleanupPrompt(t, client, created.ID)
		})

		t.Logf("Successfully created prompt with long name")
	})

	t.Run("create prompt with special characters in name", func(t *testing.T) {
		prompt := minimalPromptInput()
		prompt.Name = "test-prompt-special-!@#$%^&*()"

		created, _, err := client.Prompts.Create(ctx, prompt, nil)
		if err != nil {
			// Some special characters may not be allowed
			t.Logf("Prompt with special characters rejected (expected): %v", err)
			return
		}

		t.Cleanup(func() {
			cleanupPrompt(t, client, created.ID)
		})

		t.Logf("Successfully created prompt with special characters")
	})

	t.Run("create prompt with large template", func(t *testing.T) {
		prompt := minimalPromptInput()
		prompt.Template = "Large template: " + strings.Repeat("Hello {{name}}! ", 1000)

		created, _, err := client.Prompts.Create(ctx, prompt, nil)
		if err != nil {
			t.Fatalf("Failed to create prompt with large template: %v", err)
		}

		t.Cleanup(func() {
			cleanupPrompt(t, client, created.ID)
		})

		t.Logf("Successfully created prompt with large template")
	})

	t.Run("create prompt with many arguments", func(t *testing.T) {
		prompt := minimalPromptInput()
		prompt.Arguments = []contextforge.PromptArgument{
			{Name: "arg1", Required: true},
			{Name: "arg2", Required: false},
			{Name: "arg3", Required: true},
			{Name: "arg4", Required: false},
			{Name: "arg5", Required: true},
		}
		prompt.Template = "Hello {{arg1}} {{arg2}} {{arg3}} {{arg4}} {{arg5}}"

		created, _, err := client.Prompts.Create(ctx, prompt, nil)
		if err != nil {
			t.Fatalf("Failed to create prompt with many arguments: %v", err)
		}

		t.Cleanup(func() {
			cleanupPrompt(t, client, created.ID)
		})

		if len(created.Arguments) != len(prompt.Arguments) {
			t.Errorf("Expected %d arguments, got %d", len(prompt.Arguments), len(created.Arguments))
		}

		t.Logf("Successfully created prompt with many arguments")
	})
}

func TestPromptsService_GetRenderedPrompt(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("get rendered prompt with arguments", func(t *testing.T) {
		// Create a test prompt with arguments
		prompt := &contextforge.PromptCreate{
			Name:        randomPromptName(),
			Description: contextforge.String("Integration test prompt for Get method"),
			Template:    "Hello {{name}}! Welcome to {{topic}}.",
			Arguments: []contextforge.PromptArgument{
				{Name: "name", Description: contextforge.String("User's name"), Required: true},
				{Name: "topic", Description: contextforge.String("Topic of discussion"), Required: true},
			},
			Tags: []string{"integration-test", "get-test"},
		}

		created, _, err := client.Prompts.Create(ctx, prompt, nil)
		if err != nil {
			t.Fatalf("Failed to create prompt: %v", err)
		}

		t.Cleanup(func() {
			cleanupPrompt(t, client, created.ID)
		})

		// Get the rendered prompt with arguments
		args := map[string]string{
			"name":  "Alice",
			"topic": "Go programming",
		}

		result, _, err := client.Prompts.Get(ctx, created.Name, args)
		if err != nil {
			t.Fatalf("Failed to get rendered prompt: %v", err)
		}

		// Verify result structure
		if result.Messages == nil || len(result.Messages) == 0 {
			t.Error("Expected at least one message in the result")
		}

		if result.Description != nil {
			t.Logf("Prompt description: %s", *result.Description)
		}

		// Check that the template was rendered with the arguments
		if len(result.Messages) > 0 && result.Messages[0].Content != nil {
			if result.Messages[0].Content.Text != nil {
				renderedText := *result.Messages[0].Content.Text
				if !strings.Contains(renderedText, "Alice") || !strings.Contains(renderedText, "Go programming") {
					t.Errorf("Expected rendered text to contain arguments, got: %s", renderedText)
				}
				t.Logf("Successfully rendered prompt: %s", renderedText)
			}
		}
	})

	t.Run("get rendered prompt without arguments", func(t *testing.T) {
		// Create a simple prompt without required arguments
		prompt := &contextforge.PromptCreate{
			Name:        randomPromptName(),
			Description: contextforge.String("Simple prompt without arguments"),
			Template:    "This is a simple prompt without any variables.",
			Tags:        []string{"integration-test", "get-test"},
		}

		created, _, err := client.Prompts.Create(ctx, prompt, nil)
		if err != nil {
			t.Fatalf("Failed to create prompt: %v", err)
		}

		t.Cleanup(func() {
			cleanupPrompt(t, client, created.ID)
		})

		// Get the rendered prompt with empty arguments
		result, _, err := client.Prompts.Get(ctx, created.Name, nil)
		if err != nil {
			t.Fatalf("Failed to get rendered prompt: %v", err)
		}

		// Verify result
		if result.Messages == nil || len(result.Messages) == 0 {
			t.Error("Expected at least one message in the result")
		}

		t.Logf("Successfully retrieved prompt without arguments")
	})

	t.Run("get prompt using GetNoArgs convenience method", func(t *testing.T) {
		// Create a simple prompt
		prompt := &contextforge.PromptCreate{
			Name:        randomPromptName(),
			Description: contextforge.String("Prompt for GetNoArgs test"),
			Template:    "Welcome to the integration test!",
			Tags:        []string{"integration-test", "get-test"},
		}

		created, _, err := client.Prompts.Create(ctx, prompt, nil)
		if err != nil {
			t.Fatalf("Failed to create prompt: %v", err)
		}

		t.Cleanup(func() {
			cleanupPrompt(t, client, created.ID)
		})

		// Use GetNoArgs method
		result, _, err := client.Prompts.GetNoArgs(ctx, created.Name)
		if err != nil {
			t.Fatalf("Failed to get prompt with GetNoArgs: %v", err)
		}

		// Verify result
		if result.Messages == nil || len(result.Messages) == 0 {
			t.Error("Expected at least one message in the result")
		}

		if result.Description != nil && *result.Description != *prompt.Description {
			t.Errorf("Expected description %q, got %q", *prompt.Description, *result.Description)
		}

		t.Logf("Successfully retrieved prompt using GetNoArgs")
	})

	t.Run("get non-existent prompt returns error", func(t *testing.T) {
		_, _, err := client.Prompts.Get(ctx, "non-existent-prompt-xyz", nil)
		if err == nil {
			t.Error("Expected error when getting non-existent prompt")
		}

		// The API may return 422 instead of 404 for non-existent prompts
		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
			if apiErr.Response.StatusCode != http.StatusNotFound && apiErr.Response.StatusCode != http.StatusUnprocessableEntity {
				t.Logf("Note: Expected 404 or 422, got %d (may be API behavior)", apiErr.Response.StatusCode)
			}
		}

		t.Logf("Correctly received error for non-existent prompt: %v", err)
	})

	t.Run("get prompt with missing required arguments", func(t *testing.T) {
		// Create a prompt with required arguments
		prompt := &contextforge.PromptCreate{
			Name:        randomPromptName(),
			Description: contextforge.String("Prompt with required args"),
			Template:    "Hello {{name}}!",
			Arguments: []contextforge.PromptArgument{
				{Name: "name", Description: contextforge.String("User's name"), Required: true},
			},
			Tags: []string{"integration-test", "get-test"},
		}

		created, _, err := client.Prompts.Create(ctx, prompt, nil)
		if err != nil {
			t.Fatalf("Failed to create prompt: %v", err)
		}

		t.Cleanup(func() {
			cleanupPrompt(t, client, created.ID)
		})

		// Try to get without providing required arguments
		_, _, err = client.Prompts.Get(ctx, created.Name, nil)
		if err == nil {
			t.Log("Note: API may allow missing required arguments (renders template as-is)")
		} else {
			t.Logf("API rejected missing required arguments: %v", err)
		}
	})
}
