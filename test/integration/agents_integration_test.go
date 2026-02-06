//go:build integration
// +build integration

package integration

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/leefowlercu/go-contextforge/contextforge"
)

// TestAgentsService_BasicCRUD tests basic CRUD operations
func TestAgentsService_BasicCRUD(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("create agent with minimal fields", func(t *testing.T) {
		agent := minimalAgentInput()

		created, _, err := client.Agents.Create(ctx, agent, nil)
		if err != nil {
			t.Fatalf("Failed to create agent: %v", err)
		}

		t.Cleanup(func() {
			cleanupAgent(t, client, created.ID)
		})

		if created.ID == "" {
			t.Error("Expected created agent to have an ID")
		}
		if created.Name != agent.Name {
			t.Errorf("Expected agent name %q, got %q", agent.Name, created.Name)
		}
		if created.EndpointURL != agent.EndpointURL {
			t.Errorf("Expected endpoint URL %q, got %q", agent.EndpointURL, created.EndpointURL)
		}
		if created.Description == nil || *created.Description != *agent.Description {
			t.Errorf("Expected agent description %q, got %v", *agent.Description, created.Description)
		}
		if created.Metrics == nil {
			t.Log("Created agent omitted metrics (allowed in v1.0.0-BETA-2)")
		}

		t.Logf("Successfully created agent: %s (ID: %s)", created.Name, created.ID)
	})

	t.Run("create agent with all optional fields", func(t *testing.T) {
		agent := completeAgentInput()

		created, _, err := client.Agents.Create(ctx, agent, nil)
		if err != nil {
			t.Fatalf("Failed to create agent with all fields: %v", err)
		}

		t.Cleanup(func() {
			cleanupAgent(t, client, created.ID)
		})

		if created.ID == "" {
			t.Error("Expected created agent to have an ID")
		}
		if created.AgentType != agent.AgentType {
			t.Errorf("Expected agent type %q, got %q", agent.AgentType, created.AgentType)
		}
		if created.ProtocolVersion != agent.ProtocolVersion {
			t.Errorf("Expected protocol version %q, got %q", agent.ProtocolVersion, created.ProtocolVersion)
		}
		if created.Capabilities == nil {
			t.Error("Expected created agent to have capabilities")
		}
		if created.Config == nil {
			t.Error("Expected created agent to have config")
		}
		if created.Visibility == nil || *created.Visibility != *agent.Visibility {
			t.Errorf("Expected visibility %q, got %v", *agent.Visibility, created.Visibility)
		}
		if len(created.Tags) != len(agent.Tags) {
			t.Errorf("Expected %d tags, got %d", len(agent.Tags), len(created.Tags))
		}

		t.Logf("Successfully created agent with all fields: %s (ID: %s)", created.Name, created.ID)
	})

	t.Run("get agent by ID", func(t *testing.T) {
		created := createTestAgent(t, client, randomAgentName())

		retrieved, _, err := client.Agents.Get(ctx, created.ID)
		if err != nil {
			t.Fatalf("Failed to get agent: %v", err)
		}

		if retrieved.ID != created.ID {
			t.Errorf("Expected agent ID %q, got %q", created.ID, retrieved.ID)
		}
		if retrieved.Name != created.Name {
			t.Errorf("Expected agent name %q, got %q", created.Name, retrieved.Name)
		}

		t.Logf("Successfully retrieved agent: %s (ID: %s)", retrieved.Name, retrieved.ID)
	})

	t.Run("list agents", func(t *testing.T) {
		// Create a few test agents
		createTestAgent(t, client, randomAgentName())
		createTestAgent(t, client, randomAgentName())

		agents, _, err := client.Agents.List(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to list agents: %v", err)
		}

		if len(agents) == 0 {
			t.Error("Expected at least some agents in the list")
		}

		t.Logf("Successfully listed %d agents", len(agents))
	})

	t.Run("update agent", func(t *testing.T) {
		created := createTestAgent(t, client, randomAgentName())

		// Update the agent
		expectedDescription := "Updated description for integration test"
		expectedTags := []string{"updated", "integration-test"}
		update := &contextforge.AgentUpdate{
			Description: contextforge.String(expectedDescription),
			Tags:        expectedTags,
		}

		updated, _, err := client.Agents.Update(ctx, created.ID, update)
		if err != nil {
			t.Fatalf("Failed to update agent: %v", err)
		}

		// Assert that updates actually persisted
		if updated.Description == nil || *updated.Description != expectedDescription {
			t.Errorf("Expected description %q, got %v", expectedDescription, updated.Description)
		}
		actualTagNames := contextforge.TagNames(updated.Tags)
		if !reflect.DeepEqual(actualTagNames, expectedTags) {
			t.Errorf("Expected tags %v, got %v", expectedTags, actualTagNames)
		}

		t.Logf("Successfully updated agent: %s (ID: %s)", updated.Name, updated.ID)
	})

	t.Run("delete agent", func(t *testing.T) {
		created := createTestAgent(t, client, randomAgentName())

		// Delete the agent
		_, err := client.Agents.Delete(ctx, created.ID)
		if err != nil {
			t.Fatalf("Failed to delete agent: %v", err)
		}

		t.Logf("Successfully deleted agent: %s (ID: %s)", created.Name, created.ID)
	})

	t.Run("get deleted agent returns 404", func(t *testing.T) {
		created := createTestAgent(t, client, randomAgentName())

		// Delete the agent
		_, err := client.Agents.Delete(ctx, created.ID)
		if err != nil {
			t.Fatalf("Failed to delete agent: %v", err)
		}

		// Try to get the deleted agent
		_, _, err = client.Agents.Get(ctx, created.ID)
		if err == nil {
			t.Error("Expected error when getting deleted agent")
		}

		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
			if apiErr.Response.StatusCode != http.StatusNotFound {
				t.Errorf("Expected 404 Not Found, got %d", apiErr.Response.StatusCode)
			}
			t.Logf("Correctly received 404 for deleted agent")
		} else {
			t.Errorf("Expected ErrorResponse, got %T: %v", err, err)
		}
	})
}

// TestAgentsService_Toggle tests toggle functionality
func TestAgentsService_Toggle(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("toggle enabled to disabled", func(t *testing.T) {
		agent := minimalAgentInput()

		created, _, err := client.Agents.Create(ctx, agent, nil)
		if err != nil {
			t.Fatalf("Failed to create agent: %v", err)
		}

		t.Cleanup(func() {
			cleanupAgent(t, client, created.ID)
		})

		initialState := created.Enabled
		t.Logf("Agent initial state: enabled=%v", initialState)

		// Toggle to disabled
		toggled, _, err := client.Agents.SetState(ctx, created.ID, false)
		if err != nil {
			t.Fatalf("Failed to toggle agent: %v", err)
		}

		if toggled.Enabled {
			t.Error("Expected agent to be disabled after toggle")
		}

		t.Logf("Successfully toggled agent to disabled")
	})

	t.Run("toggle disabled to enabled", func(t *testing.T) {
		agent := minimalAgentInput()

		created, _, err := client.Agents.Create(ctx, agent, nil)
		if err != nil {
			t.Fatalf("Failed to create agent: %v", err)
		}

		t.Cleanup(func() {
			cleanupAgent(t, client, created.ID)
		})

		// First disable
		_, _, err = client.Agents.SetState(ctx, created.ID, false)
		if err != nil {
			t.Fatalf("Failed to disable agent: %v", err)
		}

		// Then enable
		toggled, _, err := client.Agents.SetState(ctx, created.ID, true)
		if err != nil {
			t.Fatalf("Failed to toggle agent: %v", err)
		}

		if !toggled.Enabled {
			t.Error("Expected agent to be enabled after toggle")
		}

		t.Logf("Successfully toggled agent to enabled")
	})
}

// TestAgentsService_Pagination tests skip/limit pagination
func TestAgentsService_Pagination(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	// Create multiple test agents
	for i := 0; i < 5; i++ {
		createTestAgent(t, client, randomAgentName())
	}

	t.Run("list with limit", func(t *testing.T) {
		opts := &contextforge.AgentListOptions{
			Limit: 2,
		}

		agents, _, err := client.Agents.List(ctx, opts)
		if err != nil {
			t.Fatalf("Failed to list agents with limit: %v", err)
		}

		if len(agents) > 2 {
			t.Errorf("Expected at most 2 agents, got %d", len(agents))
		}

		t.Logf("Successfully listed %d agents with limit=2", len(agents))
	})

	t.Run("list with cursor and limit", func(t *testing.T) {
		// Get first page
		firstPage, firstResp, err := client.Agents.List(ctx, &contextforge.AgentListOptions{Limit: 2})
		if err != nil {
			t.Fatalf("Failed to list first page: %v", err)
		}
		if firstResp == nil {
			t.Fatal("Expected pagination response metadata on first page")
		}
		if firstResp.NextCursor == "" {
			t.Fatal("Expected non-empty next cursor for first page")
		}

		// Get second page
		secondPage, _, err := client.Agents.List(ctx, &contextforge.AgentListOptions{
			Cursor: firstResp.NextCursor,
			Limit:  2,
		})
		if err != nil {
			t.Fatalf("Failed to list second page: %v", err)
		}

		// Verify pages are different
		if len(firstPage) > 0 && len(secondPage) > 0 {
			if firstPage[0].ID == secondPage[0].ID {
				t.Error("Expected different agents on different pages")
			}
		}

		t.Logf("Successfully retrieved different pages: first=%d, second=%d", len(firstPage), len(secondPage))
	})
}

// TestAgentsService_Filtering tests list filtering options
func TestAgentsService_Filtering(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("filter by tags", func(t *testing.T) {
		// Create agent with specific tags
		agent := minimalAgentInput()
		agent.Tags = []string{"filterable", "test-tag"}

		created, _, err := client.Agents.Create(ctx, agent, nil)
		if err != nil {
			t.Fatalf("Failed to create agent: %v", err)
		}

		t.Cleanup(func() {
			cleanupAgent(t, client, created.ID)
		})

		// List with tag filter
		opts := &contextforge.AgentListOptions{
			Tags: "filterable",
		}

		agents, _, err := client.Agents.List(ctx, opts)
		if err != nil {
			t.Fatalf("Failed to list agents with tags filter: %v", err)
		}

		found := false
		for _, a := range agents {
			if a.ID == created.ID {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected to find created agent in filtered list")
		}

		t.Logf("Successfully filtered agents by tags")
	})

	t.Run("filter by visibility", func(t *testing.T) {
		agent := minimalAgentInput()
		agent.Visibility = contextforge.String("public")

		opts := &contextforge.AgentCreateOptions{
			Visibility: contextforge.String("public"),
		}

		created, _, err := client.Agents.Create(ctx, agent, opts)
		if err != nil {
			t.Fatalf("Failed to create agent: %v", err)
		}

		t.Cleanup(func() {
			cleanupAgent(t, client, created.ID)
		})

		// List with visibility filter
		listOpts := &contextforge.AgentListOptions{
			Visibility: "public",
		}

		agents, _, err := client.Agents.List(ctx, listOpts)
		if err != nil {
			t.Fatalf("Failed to list agents with visibility filter: %v", err)
		}

		found := false
		for _, a := range agents {
			if a.ID == created.ID {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected to find created agent in filtered list")
		}

		t.Logf("Successfully filtered agents by visibility")
	})

	t.Run("include inactive agents", func(t *testing.T) {
		// Create and toggle agent to inactive
		agent := minimalAgentInput()

		created, _, err := client.Agents.Create(ctx, agent, nil)
		if err != nil {
			t.Fatalf("Failed to create agent: %v", err)
		}

		t.Cleanup(func() {
			cleanupAgent(t, client, created.ID)
		})

		// Disable the agent
		_, _, err = client.Agents.SetState(ctx, created.ID, false)
		if err != nil {
			t.Fatalf("Failed to disable agent: %v", err)
		}

		// List without include_inactive
		agents, _, err := client.Agents.List(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to list agents: %v", err)
		}

		foundInactive := false
		for _, a := range agents {
			if a.ID == created.ID {
				foundInactive = true
				break
			}
		}

		// List with include_inactive
		opts := &contextforge.AgentListOptions{
			IncludeInactive: true,
		}

		agentsWithInactive, _, err := client.Agents.List(ctx, opts)
		if err != nil {
			t.Fatalf("Failed to list agents with include_inactive: %v", err)
		}

		foundWithFlag := false
		for _, a := range agentsWithInactive {
			if a.ID == created.ID {
				foundWithFlag = true
				break
			}
		}

		if !foundWithFlag {
			t.Error("Expected to find inactive agent when include_inactive=true")
		}

		t.Logf("Successfully tested include_inactive filter: without flag=%v, with flag=%v", foundInactive, foundWithFlag)
	})
}

// TestAgentsService_Invoke tests agent invocation
func TestAgentsService_Invoke(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("invoke agent with parameters", func(t *testing.T) {
		// Note: This test requires a real or mock agent endpoint
		// For now, we'll test that the SDK properly sends the request
		// The actual invocation may fail if the endpoint doesn't exist

		agent := minimalAgentInput()

		created, _, err := client.Agents.Create(ctx, agent, nil)
		if err != nil {
			t.Fatalf("Failed to create agent: %v", err)
		}

		t.Cleanup(func() {
			cleanupAgent(t, client, created.ID)
		})

		req := &contextforge.AgentInvokeRequest{
			Parameters: map[string]any{
				"query": "test query",
				"mode":  "sync",
			},
			InteractionType: "query",
		}

		// Invoke agent (may fail if endpoint doesn't exist, which is okay)
		result, _, err := client.Agents.Invoke(ctx, created.Name, req)
		if err != nil {
			// This is expected if the agent endpoint doesn't actually exist
			t.Logf("Invoke failed as expected (endpoint doesn't exist): %v", err)
		} else {
			t.Logf("Invoke succeeded with result: %+v", result)
		}
	})

	t.Run("invoke agent with nil request", func(t *testing.T) {
		agent := minimalAgentInput()

		created, _, err := client.Agents.Create(ctx, agent, nil)
		if err != nil {
			t.Fatalf("Failed to create agent: %v", err)
		}

		t.Cleanup(func() {
			cleanupAgent(t, client, created.ID)
		})

		// Invoke with nil request (may fail if endpoint doesn't exist)
		_, _, err = client.Agents.Invoke(ctx, created.Name, nil)
		if err != nil {
			t.Logf("Invoke with nil request failed as expected: %v", err)
		}
	})
}

// TestAgentsService_ErrorHandling tests error scenarios
func TestAgentsService_ErrorHandling(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("get non-existent agent returns 404", func(t *testing.T) {
		_, _, err := client.Agents.Get(ctx, "non-existent-agent-id")
		if err == nil {
			t.Error("Expected error when getting non-existent agent")
		}

		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
			if apiErr.Response.StatusCode != http.StatusNotFound {
				t.Errorf("Expected 404 Not Found, got %d", apiErr.Response.StatusCode)
			}
			t.Logf("Correctly received 404 for non-existent agent")
		}
	})

	t.Run("delete non-existent agent returns 404", func(t *testing.T) {
		_, err := client.Agents.Delete(ctx, "non-existent-agent-id")
		if err == nil {
			t.Error("Expected error when deleting non-existent agent")
		}

		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
			if apiErr.Response.StatusCode != http.StatusNotFound {
				t.Errorf("Expected 404 Not Found, got %d", apiErr.Response.StatusCode)
			}
			t.Logf("Correctly received 404 for non-existent agent")
		}
	})

	t.Run("toggle non-existent agent returns error", func(t *testing.T) {
		_, _, err := client.Agents.SetState(ctx, "non-existent-agent-id", true)
		if err == nil {
			t.Error("Expected error when setting state for non-existent agent")
		}

		t.Logf("Correctly received error for non-existent agent state change: %v", err)
	})
}

// TestAgentsService_EdgeCases tests edge cases and special scenarios
func TestAgentsService_EdgeCases(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("create agent with authentication", func(t *testing.T) {
		// CONTEXTFORGE-008: Agent bearer auth requires auth_token field - see docs/upstream-bugs/contextforge-008-agent-auth-field-name.md
		t.Skip("CONTEXTFORGE-008: Agent bearer auth requires auth_token instead of auth_value")

		agent := minimalAgentInput()
		agent.AuthType = contextforge.String("bearer")
		agent.AuthValue = contextforge.String("test-secret-token")

		created, _, err := client.Agents.Create(ctx, agent, nil)
		if err != nil {
			t.Fatalf("Failed to create agent with auth: %v", err)
		}

		t.Cleanup(func() {
			cleanupAgent(t, client, created.ID)
		})

		// AuthValue should be encrypted by API, so we shouldn't see the original value
		if created.AuthType == nil || *created.AuthType != "bearer" {
			t.Errorf("Expected auth type %q, got %v", "bearer", created.AuthType)
		}

		t.Logf("Successfully created agent with authentication")
	})

	t.Run("update agent capabilities", func(t *testing.T) {
		created := createTestAgent(t, client, randomAgentName())

		update := &contextforge.AgentUpdate{
			Capabilities: map[string]any{
				"streaming": true,
				"batch":     false,
			},
		}

		updated, _, err := client.Agents.Update(ctx, created.ID, update)
		if err != nil {
			t.Fatalf("Failed to update agent capabilities: %v", err)
		}

		// Assert that capabilities were actually updated
		if updated.Capabilities == nil {
			t.Fatal("Expected updated agent to have capabilities")
		}
		if streaming, ok := updated.Capabilities["streaming"].(bool); !ok || !streaming {
			t.Errorf("Expected capabilities[\"streaming\"] to be true, got %v", updated.Capabilities["streaming"])
		}
		if batch, ok := updated.Capabilities["batch"].(bool); !ok || batch {
			t.Errorf("Expected capabilities[\"batch\"] to be false, got %v", updated.Capabilities["batch"])
		}

		t.Logf("Successfully updated agent capabilities: %+v", updated.Capabilities)
	})

	t.Run("update agent config", func(t *testing.T) {
		created := createTestAgent(t, client, randomAgentName())

		update := &contextforge.AgentUpdate{
			Config: map[string]any{
				"timeout": 60,
				"retries": 5,
			},
		}

		updated, _, err := client.Agents.Update(ctx, created.ID, update)
		if err != nil {
			t.Fatalf("Failed to update agent config: %v", err)
		}

		// Assert that config was actually updated
		if updated.Config == nil {
			t.Fatal("Expected updated agent to have config")
		}
		// Config values might be float64 due to JSON unmarshaling
		if timeout, ok := updated.Config["timeout"].(float64); !ok || int(timeout) != 60 {
			t.Errorf("Expected config[\"timeout\"] to be 60, got %v (type: %T)", updated.Config["timeout"], updated.Config["timeout"])
		}
		if retries, ok := updated.Config["retries"].(float64); !ok || int(retries) != 5 {
			t.Errorf("Expected config[\"retries\"] to be 5, got %v (type: %T)", updated.Config["retries"], updated.Config["retries"])
		}

		t.Logf("Successfully updated agent config: %+v", updated.Config)
	})
}

// TestAgentsService_SetState tests the preferred /state endpoint.
func TestAgentsService_SetState(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("set agent state disabled then enabled", func(t *testing.T) {
		created := createTestAgent(t, client, randomAgentName())

		disabled, _, err := client.Agents.SetState(ctx, created.ID, false)
		if err != nil {
			t.Fatalf("Failed to disable agent via SetState: %v", err)
		}
		if disabled == nil {
			t.Fatal("SetState returned nil agent on disable")
		}
		if disabled.Enabled {
			t.Errorf("Expected disabled agent, got enabled=%v", disabled.Enabled)
		}

		enabled, _, err := client.Agents.SetState(ctx, created.ID, true)
		if err != nil {
			t.Fatalf("Failed to enable agent via SetState: %v", err)
		}
		if enabled == nil {
			t.Fatal("SetState returned nil agent on enable")
		}
		if !enabled.Enabled {
			t.Errorf("Expected enabled agent, got enabled=%v", enabled.Enabled)
		}
	})
}
