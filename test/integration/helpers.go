//go:build integration
// +build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/leefowlercu/go-contextforge/contextforge"
)

const (
	defaultAddress         = "http://localhost:8000/"
	defaultAdminEmail      = "admin@test.local"
	defaultAdminPass       = "testpassword123"
	testToolNamePrefix     = "test-tool"
	testGatewayNamePrefix  = "test-gateway"
	testResourceNamePrefix = "test-resource"
	testServerNamePrefix   = "test-server"
	testAgentNamePrefix    = "test-agent"
	testTeamNamePrefix     = "test-team"
)

// skipIfNotIntegration skips the test if INTEGRATION_TESTS is not set to "true"
func skipIfNotIntegration(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TESTS=true to run.")
	}
}

// getAddress returns the address for the ContextForge API
func getAddress() string {
	if url := os.Getenv("CONTEXTFORGE_ADDR"); url != "" {
		return url
	}
	return defaultAddress
}

// getAdminEmail returns the admin email for authentication
func getAdminEmail() string {
	if email := os.Getenv("CONTEXTFORGE_ADMIN_EMAIL"); email != "" {
		return email
	}
	return defaultAdminEmail
}

// getAdminPassword returns the admin password for authentication
func getAdminPassword() string {
	if pass := os.Getenv("CONTEXTFORGE_ADMIN_PASSWORD"); pass != "" {
		return pass
	}
	return defaultAdminPass
}

// loginResponse represents the response from the login endpoint
type loginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// getTestToken authenticates with the ContextForge API and returns a JWT token
func getTestToken(t *testing.T) string {
	t.Helper()

	address := getAddress()
	loginURL := address + "auth/login"

	loginReq := map[string]string{
		"username": getAdminEmail(),
		"password": getAdminPassword(),
	}

	body, err := json.Marshal(loginReq)
	if err != nil {
		t.Fatalf("Failed to marshal login request: %v", err)
	}

	resp, err := http.Post(loginURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Login failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var loginResp loginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		t.Fatalf("Failed to decode login response: %v", err)
	}

	if loginResp.AccessToken == "" {
		t.Fatal("Login response did not contain access token")
	}

	t.Logf("Successfully obtained JWT token")
	return loginResp.AccessToken
}

// setupClient creates an authenticated ContextForge client for testing
func setupClient(t *testing.T) *contextforge.Client {
	t.Helper()
	skipIfNotIntegration(t)

	token := getTestToken(t)
	client, err := contextforge.NewClient(nil, getAddress(), token)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Logf("Created ContextForge client with address: %s", client.Address.String())
	return client
}

// randomToolName generates a unique tool name for testing
func randomToolName() string {
	return fmt.Sprintf("%s-%d", testToolNamePrefix, time.Now().UnixNano())
}

// minimalToolInput returns a minimal valid tool input for testing
func minimalToolInput() *contextforge.Tool {
	return &contextforge.Tool{
		Name:        randomToolName(),
		Description: contextforge.String("A test tool for integration testing"),
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"input": map[string]any{
					"type":        "string",
					"description": "Test input parameter",
				},
			},
		},
	}
}

// completeToolInput returns a tool input with all optional fields for testing
func completeToolInput() *contextforge.Tool {
	return &contextforge.Tool{
		Name:        randomToolName(),
		Description: contextforge.String("A complete test tool with all fields"),
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"input": map[string]any{
					"type":        "string",
					"description": "Test input parameter",
				},
				"count": map[string]any{
					"type":        "integer",
					"description": "Test count parameter",
				},
			},
			"required": []string{"input"},
		},
		Visibility: "public",
		Tags:       contextforge.NewTags([]string{"test", "integration"}),
		TeamID:     contextforge.String("test-team"),
	}
}

// createTestTool creates a test tool and registers it for cleanup
func createTestTool(t *testing.T, client *contextforge.Client, name string) *contextforge.Tool {
	t.Helper()

	tool := &contextforge.Tool{
		Name:        name,
		Description: contextforge.String("Test tool created by integration test"),
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"input": map[string]any{"type": "string"},
			},
		},
	}

	ctx := context.Background()
	created, _, err := client.Tools.Create(ctx, tool, nil)
	if err != nil {
		t.Fatalf("Failed to create test tool: %v", err)
	}

	t.Logf("Created test tool: %s (ID: %s)", created.Name, created.ID)

	// Register cleanup
	t.Cleanup(func() {
		cleanupTool(t, client, created.ID)
	})

	return created
}

// cleanupTool deletes a tool by ID (ignores errors for cleanup)
func cleanupTool(t *testing.T, client *contextforge.Client, toolID string) {
	t.Helper()

	ctx := context.Background()
	_, err := client.Tools.Delete(ctx, toolID)
	if err != nil {
		t.Logf("Warning: Failed to cleanup tool %s: %v (may already be deleted)", toolID, err)
	} else {
		t.Logf("Cleaned up tool: %s", toolID)
	}
}

// cleanupTools deletes multiple tools by ID (ignores errors for cleanup)
func cleanupTools(t *testing.T, client *contextforge.Client, toolIDs []string) {
	t.Helper()

	for _, toolID := range toolIDs {
		cleanupTool(t, client, toolID)
	}
}

// randomGatewayName generates a unique gateway name for testing
func randomGatewayName() string {
	return fmt.Sprintf("%s-%d", testGatewayNamePrefix, time.Now().UnixNano())
}

// minimalGatewayInput returns a minimal valid gateway input for testing
func minimalGatewayInput() *contextforge.Gateway {
	return &contextforge.Gateway{
		Name:        randomGatewayName(),
		URL:         GetMockMCPServerURL(),
		Description: contextforge.String("A test gateway for integration testing"),
		Transport:   "STREAMABLEHTTP",
	}
}

// completeGatewayInput returns a gateway input with all optional fields for testing
func completeGatewayInput() *contextforge.Gateway {
	return &contextforge.Gateway{
		Name:        randomGatewayName(),
		URL:         GetMockMCPServerURL(),
		Description: contextforge.String("A complete test gateway with all fields"),
		Transport:   "STREAMABLEHTTP",
		Visibility:  contextforge.String("public"),
		Tags:        contextforge.NewTags([]string{"test", "integration"}),
		TeamID:      contextforge.String("test-team"),
		AuthType:    contextforge.String("bearer"),
		AuthToken:   contextforge.String("test-token-123"),
	}
}

// createTestGateway creates a test gateway and registers it for cleanup
func createTestGateway(t *testing.T, client *contextforge.Client, name string) *contextforge.Gateway {
	t.Helper()

	gateway := &contextforge.Gateway{
		Name:        name,
		URL:         GetMockMCPServerURL(),
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

// cleanupGateway deletes a gateway by ID (ignores errors for cleanup)
func cleanupGateway(t *testing.T, client *contextforge.Client, gatewayID string) {
	t.Helper()

	ctx := context.Background()
	_, err := client.Gateways.Delete(ctx, gatewayID)
	if err != nil {
		t.Logf("Warning: Failed to cleanup gateway %s: %v (may already be deleted)", gatewayID, err)
	} else {
		t.Logf("Cleaned up gateway: %s", gatewayID)
	}
}

// cleanupGateways deletes multiple gateways by ID (ignores errors for cleanup)
func cleanupGateways(t *testing.T, client *contextforge.Client, gatewayIDs []string) {
	t.Helper()

	for _, gatewayID := range gatewayIDs {
		cleanupGateway(t, client, gatewayID)
	}
}

// randomResourceName generates a unique resource name for testing
func randomResourceName() string {
	return fmt.Sprintf("%s-%d", testResourceNamePrefix, time.Now().UnixNano())
}

// minimalResourceInput returns a minimal valid resource input for testing
func minimalResourceInput() *contextforge.ResourceCreate {
	return &contextforge.ResourceCreate{
		URI:         fmt.Sprintf("file:///test-%d.txt", time.Now().UnixNano()),
		Name:        randomResourceName(),
		Content:     "test content",
		Description: contextforge.String("A test resource for integration testing"),
	}
}

// completeResourceInput returns a resource input with all optional fields for testing
func completeResourceInput() *contextforge.ResourceCreate {
	return &contextforge.ResourceCreate{
		URI:         fmt.Sprintf("file:///complete-%d.txt", time.Now().UnixNano()),
		Name:        randomResourceName(),
		Content:     "complete test content",
		Description: contextforge.String("A complete test resource with all fields"),
		MimeType:    contextforge.String("text/plain"),
		Tags:        []string{"test", "integration"},
	}
}

// createTestResource creates a test resource and registers it for cleanup
func createTestResource(t *testing.T, client *contextforge.Client, name string) *contextforge.Resource {
	t.Helper()

	resource := &contextforge.ResourceCreate{
		URI:         fmt.Sprintf("file:///%s.txt", name),
		Name:        name,
		Content:     "Test resource created by integration test",
		Description: contextforge.String("Test resource created by integration test"),
	}

	opts := &contextforge.ResourceCreateOptions{
		// Don't set TeamID or Visibility - let API use defaults
	}

	ctx := context.Background()
	created, _, err := client.Resources.Create(ctx, resource, opts)
	if err != nil {
		t.Fatalf("Failed to create test resource: %v", err)
	}

	t.Logf("Created test resource: %s (ID: %s)", created.Name, *created.ID)

	// Register cleanup
	t.Cleanup(func() {
		cleanupResource(t, client, string(*created.ID))
	})

	return created
}

// cleanupResource deletes a resource by ID (ignores errors for cleanup)
func cleanupResource(t *testing.T, client *contextforge.Client, resourceID string) {
	t.Helper()

	ctx := context.Background()
	_, err := client.Resources.Delete(ctx, resourceID)
	if err != nil {
		t.Logf("Warning: Failed to cleanup resource %s: %v (may already be deleted)", resourceID, err)
	} else {
		t.Logf("Cleaned up resource: %s", resourceID)
	}
}

// cleanupResources deletes multiple resources by ID (ignores errors for cleanup)
func cleanupResources(t *testing.T, client *contextforge.Client, resourceIDs []string) {
	t.Helper()

	for _, resourceID := range resourceIDs {
		cleanupResource(t, client, resourceID)
	}
}

// randomServerName generates a unique server name for testing
func randomServerName() string {
	return fmt.Sprintf("%s-%d", testServerNamePrefix, time.Now().UnixNano())
}

// minimalServerInput returns a minimal valid server input for testing
func minimalServerInput() *contextforge.ServerCreate {
	return &contextforge.ServerCreate{
		Name:        randomServerName(),
		Description: contextforge.String("A test server for integration testing"),
	}
}

// completeServerInput returns a server input with all optional fields for testing
func completeServerInput() *contextforge.ServerCreate {
	return &contextforge.ServerCreate{
		Name:        randomServerName(),
		Description: contextforge.String("A complete test server with all fields"),
		Tags:        []string{"test", "integration"},
		Visibility:  contextforge.String("public"),
	}
}

// createTestServer creates a test server and registers it for cleanup
func createTestServer(t *testing.T, client *contextforge.Client, name string) *contextforge.Server {
	t.Helper()

	server := &contextforge.ServerCreate{
		Name:        name,
		Description: contextforge.String("Test server created by integration test"),
	}

	ctx := context.Background()
	created, _, err := client.Servers.Create(ctx, server, nil)
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}

	t.Logf("Created test server: %s (ID: %s)", created.Name, created.ID)

	// Register cleanup
	t.Cleanup(func() {
		cleanupServer(t, client, created.ID)
	})

	return created
}

// cleanupServer deletes a server by ID (ignores errors for cleanup)
func cleanupServer(t *testing.T, client *contextforge.Client, serverID string) {
	t.Helper()

	ctx := context.Background()
	_, err := client.Servers.Delete(ctx, serverID)
	if err != nil {
		t.Logf("Warning: Failed to cleanup server %s: %v (may already be deleted)", serverID, err)
	} else {
		t.Logf("Cleaned up server: %s", serverID)
	}
}

// cleanupServers deletes multiple servers by ID (ignores errors for cleanup)
func cleanupServers(t *testing.T, client *contextforge.Client, serverIDs []string) {
	t.Helper()

	for _, serverID := range serverIDs {
		cleanupServer(t, client, serverID)
	}
}

const testPromptNamePrefix = "test-prompt"

// randomPromptName generates a unique prompt name for testing
func randomPromptName() string {
	return fmt.Sprintf("%s-%d", testPromptNamePrefix, time.Now().UnixNano())
}

// minimalPromptInput returns a minimal valid prompt input for testing
func minimalPromptInput() *contextforge.PromptCreate {
	return &contextforge.PromptCreate{
		Name:     randomPromptName(),
		Template: "Hello {{name}}!",
		Arguments: []contextforge.PromptArgument{
			{Name: "name", Description: contextforge.String("Name to greet"), Required: true},
		},
	}
}

// completePromptInput returns a prompt input with all optional fields for testing
func completePromptInput() *contextforge.PromptCreate {
	return &contextforge.PromptCreate{
		Name:        randomPromptName(),
		Description: contextforge.String("A complete test prompt with all fields"),
		Template:    "Hello {{name}}! You are {{age}} years old.",
		Arguments: []contextforge.PromptArgument{
			{Name: "name", Description: contextforge.String("Name to greet"), Required: true},
			{Name: "age", Description: contextforge.String("Age of person"), Required: false},
		},
		Tags:       []string{"test", "integration"},
		Visibility: contextforge.String("public"),
	}
}

// createTestPrompt creates a test prompt and registers it for cleanup
func createTestPrompt(t *testing.T, client *contextforge.Client, name string) *contextforge.Prompt {
	t.Helper()

	prompt := &contextforge.PromptCreate{
		Name:        name,
		Description: contextforge.String("Test prompt created by integration test"),
		Template:    "Hello {{name}}!",
		Arguments: []contextforge.PromptArgument{
			{Name: "name", Required: true},
		},
	}

	ctx := context.Background()
	created, _, err := client.Prompts.Create(ctx, prompt, nil)
	if err != nil {
		t.Fatalf("Failed to create test prompt: %v", err)
	}

	t.Logf("Created test prompt: %s (ID: %s)", created.Name, created.ID)

	// Register cleanup
	t.Cleanup(func() {
		cleanupPrompt(t, client, created.ID)
	})

	return created
}

// cleanupPrompt deletes a prompt by ID (ignores errors for cleanup)
func cleanupPrompt(t *testing.T, client *contextforge.Client, promptID string) {
	t.Helper()

	ctx := context.Background()
	_, err := client.Prompts.Delete(ctx, promptID)
	if err != nil {
		t.Logf("Warning: Failed to cleanup prompt %s: %v (may already be deleted)", promptID, err)
	} else {
		t.Logf("Cleaned up prompt: %s", promptID)
	}
}

// cleanupPrompts deletes multiple prompts by ID (ignores errors for cleanup)
func cleanupPrompts(t *testing.T, client *contextforge.Client, promptIDs []string) {
	t.Helper()

	for _, promptID := range promptIDs {
		cleanupPrompt(t, client, promptID)
	}
}

// randomAgentName generates a unique agent name for testing
func randomAgentName() string {
	return fmt.Sprintf("%s-%d", testAgentNamePrefix, time.Now().UnixNano())
}

// minimalAgentInput returns a minimal valid agent input for testing
func minimalAgentInput() *contextforge.AgentCreate {
	return &contextforge.AgentCreate{
		Name:        randomAgentName(),
		EndpointURL: "https://example.com/a2a/agent",
		Description: contextforge.String("A test agent for integration testing"),
	}
}

// completeAgentInput returns an agent input with all optional fields for testing
func completeAgentInput() *contextforge.AgentCreate {
	return &contextforge.AgentCreate{
		Name:            randomAgentName(),
		EndpointURL:     "https://example.com/a2a/complete-agent",
		Description:     contextforge.String("A complete test agent with all fields"),
		AgentType:       "custom",
		ProtocolVersion: "1.0",
		Capabilities: map[string]any{
			"streaming": true,
			"async":     false,
		},
		Config: map[string]any{
			"timeout": 30,
			"retries": 3,
		},
		AuthType:   contextforge.String("bearer"),
		AuthValue:  contextforge.String("test-token-123"),
		Tags:       []string{"test", "integration"},
		Visibility: contextforge.String("public"),
	}
}

// createTestAgent creates a test agent and registers it for cleanup
func createTestAgent(t *testing.T, client *contextforge.Client, name string) *contextforge.Agent {
	t.Helper()

	agent := &contextforge.AgentCreate{
		Name:        name,
		EndpointURL: fmt.Sprintf("https://example.com/a2a/%s", name),
		Description: contextforge.String("Test agent created by integration test"),
	}

	ctx := context.Background()
	created, _, err := client.Agents.Create(ctx, agent, nil)
	if err != nil {
		t.Fatalf("Failed to create test agent: %v", err)
	}

	t.Logf("Created test agent: %s (ID: %s)", created.Name, created.ID)

	// Register cleanup
	t.Cleanup(func() {
		cleanupAgent(t, client, created.ID)
	})

	return created
}

// cleanupAgent deletes an agent by ID (ignores errors for cleanup)
func cleanupAgent(t *testing.T, client *contextforge.Client, agentID string) {
	t.Helper()

	ctx := context.Background()
	_, err := client.Agents.Delete(ctx, agentID)
	if err != nil {
		t.Logf("Warning: Failed to cleanup agent %s: %v (may already be deleted)", agentID, err)
	} else {
		t.Logf("Cleaned up agent: %s", agentID)
	}
}

// cleanupAgents deletes multiple agents by ID (ignores errors for cleanup)
func cleanupAgents(t *testing.T, client *contextforge.Client, agentIDs []string) {
	t.Helper()

	for _, agentID := range agentIDs {
		cleanupAgent(t, client, agentID)
	}
}

// Team helpers

// randomTeamName generates a unique team name for testing
func randomTeamName() string {
	return fmt.Sprintf("%s-%d", testTeamNamePrefix, time.Now().UnixNano())
}

// minimalTeamInput returns a minimal valid team input for testing
func minimalTeamInput() *contextforge.TeamCreate {
	return &contextforge.TeamCreate{
		Name: randomTeamName(),
	}
}

// completeTeamInput returns a team input with all optional fields for testing
func completeTeamInput() *contextforge.TeamCreate {
	name := randomTeamName()
	return &contextforge.TeamCreate{
		Name:        name,
		Slug:        contextforge.String(fmt.Sprintf("%s-slug", name)),
		Description: contextforge.String("A complete test team with all fields"),
		Visibility:  contextforge.String("private"),
		MaxMembers:  contextforge.Int(50),
	}
}

// createTestTeam creates a test team and registers it for cleanup
func createTestTeam(t *testing.T, client *contextforge.Client, name string) *contextforge.Team {
	t.Helper()

	team := &contextforge.TeamCreate{
		Name:        name,
		Description: contextforge.String("Test team created by integration test"),
	}

	ctx := context.Background()
	created, _, err := client.Teams.Create(ctx, team)
	if err != nil {
		t.Fatalf("Failed to create test team: %v", err)
	}

	t.Logf("Created test team: %s (ID: %s)", created.Name, created.ID)

	// Register cleanup
	t.Cleanup(func() {
		cleanupTeam(t, client, created.ID)
	})

	return created
}

// cleanupTeam deletes a team by ID (ignores errors for cleanup)
func cleanupTeam(t *testing.T, client *contextforge.Client, teamID string) {
	t.Helper()

	ctx := context.Background()
	_, err := client.Teams.Delete(ctx, teamID)
	if err != nil {
		t.Logf("Warning: Failed to cleanup team %s: %v (may already be deleted)", teamID, err)
	} else {
		t.Logf("Cleaned up team: %s", teamID)
	}
}

// cleanupTeams deletes multiple teams by ID (ignores errors for cleanup)
func cleanupTeams(t *testing.T, client *contextforge.Client, teamIDs []string) {
	t.Helper()

	for _, teamID := range teamIDs {
		cleanupTeam(t, client, teamID)
	}
}
