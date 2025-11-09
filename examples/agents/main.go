// Package main demonstrates comprehensive usage of the AgentsService
// from the go-contextforge SDK. This example highlights A2A (Agent-to-Agent)
// agent management including unique features like skip/limit pagination and
// agent invocation. Uses a mock HTTP server for self-contained demonstration.
//
// Run: go run examples/agents/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/leefowlercu/go-contextforge/contextforge"
)

func main() {
	// Create mock server with all necessary endpoints
	mux := http.NewServeMux()
	setupMockEndpoints(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	fmt.Println("=== ContextForge SDK - Agents Service Example ===\n")

	// Step 1: Authentication
	fmt.Println("1. Authenticating...")
	token := authenticate(server.URL)
	fmt.Printf("   ✓ Obtained JWT token: %s...\n\n", token[:20])

	// Step 2: Create client
	// To use a real ContextForge instance, replace server.URL with:
	// "https://your-contextforge-instance.com"
	client, err := contextforge.NewClientWithBaseURL(nil, server.URL, token)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Step 3: Create a basic agent
	fmt.Println("2. Creating a basic A2A agent...")
	// Note: Create uses snake_case fields (endpoint_url, agent_type, team_id)
	newAgent := &contextforge.AgentCreate{
		Name:            "data-processor",
		EndpointURL:     "https://agent1.example.com/a2a",
		Description:     contextforge.String("Processes and transforms data records"),
		AgentType:       "generic",
		ProtocolVersion: "1.0",
		Capabilities: map[string]any{
			"streaming": true,
			"batch":     true,
		},
		Config: map[string]any{
			"timeout": 30,
			"retries": 3,
		},
		Tags: []string{"data", "processing"},
	}

	createdAgent1, resp, err := client.Agents.Create(ctx, newAgent, nil)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	fmt.Printf("   ✓ Created agent: %s (ID: %s)\n", createdAgent1.Name, createdAgent1.ID)
	fmt.Printf("   ✓ Endpoint: %s\n", createdAgent1.EndpointURL)
	fmt.Printf("   ✓ Type: %s, Protocol: %s\n", createdAgent1.AgentType, createdAgent1.ProtocolVersion)
	fmt.Printf("   ✓ Enabled: %v, Reachable: %v\n", createdAgent1.Enabled, createdAgent1.Reachable)
	fmt.Printf("   ✓ Rate limit: %d/%d remaining\n\n", resp.Rate.Remaining, resp.Rate.Limit)

	// Step 4: Create an agent with authentication
	fmt.Println("3. Creating an agent with API key authentication...")
	authAgent := &contextforge.AgentCreate{
		Name:            "secure-analyzer",
		EndpointURL:     "https://agent2.example.com/a2a",
		Description:     contextforge.String("Secure data analyzer with API key authentication"),
		AgentType:       "analyzer",
		ProtocolVersion: "1.0",
		AuthType:        contextforge.String("api_key"),
		AuthValue:       contextforge.String("secret-key-12345"), // Will be encrypted by API
		Tags:            []string{"security", "analysis"},
	}

	createdAgent2, _, err := client.Agents.Create(ctx, authAgent, nil)
	if err != nil {
		log.Fatalf("Failed to create agent with auth: %v", err)
	}
	fmt.Printf("   ✓ Created agent: %s (ID: %s)\n", createdAgent2.Name, createdAgent2.ID)
	if createdAgent2.AuthType != nil {
		fmt.Printf("   ✓ Auth Type: %s\n", *createdAgent2.AuthType)
		fmt.Println("   ✓ Auth Value encrypted by API (not returned in response)")
	}
	fmt.Println()

	// Step 5: Get a specific agent
	fmt.Println("4. Retrieving agent by ID...")
	retrievedAgent, _, err := client.Agents.Get(ctx, createdAgent1.ID)
	if err != nil {
		log.Fatalf("Failed to get agent: %v", err)
	}
	fmt.Printf("   ✓ Retrieved: %s\n", retrievedAgent.Name)
	fmt.Printf("   ✓ Slug: %s\n", retrievedAgent.Slug)
	if retrievedAgent.Description != nil {
		fmt.Printf("   ✓ Description: %s\n", *retrievedAgent.Description)
	}
	if retrievedAgent.Capabilities != nil {
		fmt.Printf("   ✓ Capabilities: %+v\n", retrievedAgent.Capabilities)
	}
	if retrievedAgent.Config != nil {
		fmt.Printf("   ✓ Config: %+v\n", retrievedAgent.Config)
	}
	fmt.Println()

	// Step 6: List all agents
	fmt.Println("5. Listing all agents...")
	agents, _, err := client.Agents.List(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to list agents: %v", err)
	}
	fmt.Printf("   ✓ Found %d agent(s)\n", len(agents))
	for i, agent := range agents {
		fmt.Printf("   %d. %s (ID: %s, Enabled: %v)\n", i+1, agent.Name, agent.ID, agent.Enabled)
	}
	fmt.Println()

	// Step 7: List agents with filtering
	fmt.Println("6. Listing agents with filters...")
	listOpts := &contextforge.AgentListOptions{
		IncludeInactive: true,
		Tags:            "data,processing",
		Visibility:      "public",
	}

	filteredAgents, _, err := client.Agents.List(ctx, listOpts)
	if err != nil {
		log.Fatalf("Failed to list filtered agents: %v", err)
	}
	fmt.Printf("   ✓ Found %d agent(s) with filters\n", len(filteredAgents))
	for i, agent := range filteredAgents {
		fmt.Printf("   %d. %s (Tags: %v)\n", i+1, agent.Name, agent.Tags)
	}
	fmt.Println()

	// Step 8: Demonstrate skip/limit pagination (unique to agents)
	fmt.Println("7. Demonstrating skip/limit pagination...")
	fmt.Println("   NOTE: Agents use skip/limit (offset-based) pagination")
	fmt.Println("   This differs from cursor-based pagination in other services")
	page := 1
	for skip := 0; skip < 4; skip += 2 {
		pageOpts := &contextforge.AgentListOptions{
			Skip:  skip,
			Limit: 2,
		}
		pageAgents, _, err := client.Agents.List(ctx, pageOpts)
		if err != nil {
			log.Fatalf("Failed to list page: %v", err)
		}
		fmt.Printf("   Page %d (skip=%d, limit=2): %d agent(s)\n", page, skip, len(pageAgents))
		if len(pageAgents) == 0 {
			break
		}
		page++
	}
	fmt.Println()

	// Step 9: Update an agent
	fmt.Println("8. Updating agent...")
	// Note: Update uses camelCase fields (inconsistent with Create)
	updateAgent := &contextforge.AgentUpdate{
		Description: contextforge.String("Advanced data processor with enhanced capabilities"),
		Tags:        []string{"data", "processing", "advanced"},
		Capabilities: map[string]any{
			"streaming": true,
			"batch":     true,
			"parallel":  true,
		},
	}

	updatedAgent, _, err := client.Agents.Update(ctx, createdAgent1.ID, updateAgent)
	if err != nil {
		log.Fatalf("Failed to update agent: %v", err)
	}
	fmt.Printf("   ✓ Updated description: %s\n", *updatedAgent.Description)
	fmt.Printf("   ✓ Updated tags: %v\n", updatedAgent.Tags)
	if updatedAgent.Capabilities != nil {
		fmt.Printf("   ✓ Updated capabilities: %+v\n", updatedAgent.Capabilities)
	}
	fmt.Println()

	// Step 10: Toggle agent (disable)
	fmt.Println("9. Toggling agent (disabling)...")
	toggledAgent, _, err := client.Agents.Toggle(ctx, createdAgent1.ID, false)
	if err != nil {
		log.Fatalf("Failed to toggle agent: %v", err)
	}
	fmt.Printf("   ✓ Agent is now enabled: %v\n", toggledAgent.Enabled)
	fmt.Printf("   ✓ Agent reachable status: %v (read-only, not affected by toggle)\n\n", toggledAgent.Reachable)

	// Step 11: Toggle agent (enable)
	fmt.Println("10. Toggling agent (enabling)...")
	toggledAgent, _, err = client.Agents.Toggle(ctx, createdAgent1.ID, true)
	if err != nil {
		log.Fatalf("Failed to toggle agent: %v", err)
	}
	fmt.Printf("   ✓ Agent is now enabled: %v\n\n", toggledAgent.Enabled)

	// Step 12: Invoke an agent
	fmt.Println("11. Invoking agent with parameters...")
	fmt.Println("   NOTE: Invoke uses agent name (not ID) as identifier")
	invokeReq := &contextforge.AgentInvokeRequest{
		Parameters: map[string]any{
			"input": "sample data to process",
			"options": map[string]any{
				"format":   "json",
				"validate": true,
			},
		},
		InteractionType: "query",
	}

	result, _, err := client.Agents.Invoke(ctx, createdAgent1.Name, invokeReq)
	if err != nil {
		// In this mock example, invoke might succeed or fail depending on mock implementation
		fmt.Printf("   ⚠ Invoke returned error (expected in mock): %v\n", err)
	} else {
		fmt.Printf("   ✓ Invoke succeeded with result:\n")
		if status, ok := result["status"]; ok {
			fmt.Printf("      Status: %v\n", status)
		}
		if data, ok := result["result"]; ok {
			fmt.Printf("      Result: %v\n", data)
		}
		if execTime, ok := result["execution_time"]; ok {
			fmt.Printf("      Execution time: %v ms\n", execTime)
		}
	}
	fmt.Println()

	// Step 13: Error handling example
	fmt.Println("12. Demonstrating error handling...")
	_, _, err = client.Agents.Get(ctx, "non-existent-agent-id")
	if err != nil {
		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
			fmt.Printf("   ✓ Caught expected error: HTTP %d - %s\n",
				apiErr.Response.StatusCode, apiErr.Message)
		} else {
			fmt.Printf("   ✓ Caught error: %v\n", err)
		}
	}
	fmt.Println()

	// Step 14: Delete agents
	fmt.Println("13. Deleting agents...")
	for _, id := range []string{createdAgent1.ID, createdAgent2.ID} {
		_, err = client.Agents.Delete(ctx, id)
		if err != nil {
			log.Fatalf("Failed to delete agent %s: %v", id, err)
		}
		fmt.Printf("   ✓ Deleted agent: %s\n", id)
	}
	fmt.Println()

	fmt.Println("=== Example completed successfully! ===")
	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("• A2A agent CRUD operations")
	fmt.Println("• Skip/limit (offset-based) pagination instead of cursor-based")
	fmt.Println("• Agent invocation with parameters (uses name, not ID)")
	fmt.Println("• Authentication configuration (AuthType/AuthValue)")
	fmt.Println("• Complex types (Capabilities and Config maps)")
	fmt.Println("• Agent types and protocol versions")
	fmt.Println("• Toggle enabled/disabled state")
	fmt.Println("• Enabled vs Reachable distinction")
	fmt.Println("\nAPI Inconsistencies:")
	fmt.Println("• Create uses snake_case (endpoint_url, agent_type, team_id)")
	fmt.Println("• Update uses camelCase (endpointUrl, agentType, teamId)")
	fmt.Println("• Create request is wrapped: {\"agent\": {...}}")
	fmt.Println("• Update request is unwrapped (direct body)")
	fmt.Println("\nTo use with a real ContextForge instance:")
	fmt.Println("1. Replace server.URL with your ContextForge base URL")
	fmt.Println("2. Use real authentication credentials")
	fmt.Println("3. Adjust agent endpoints to match real A2A agents")
}

// authenticate performs mock authentication and returns a JWT token
func authenticate(baseURL string) string {
	loginURL := baseURL + "/auth/login"
	payload := strings.NewReader(`{"email":"admin@example.com","password":"secret"}`)

	resp, err := http.Post(loginURL, "application/json", payload)
	if err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}
	defer resp.Body.Close()

	var authResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		log.Fatalf("Failed to decode auth response: %v", err)
	}

	return authResp.AccessToken
}

// setupMockEndpoints configures all the mock HTTP endpoints
func setupMockEndpoints(mux *http.ServeMux) {
	// Mock authentication endpoint
	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "mock-jwt-token-99999",
			"token_type":   "bearer",
		})
	})

	// Mock agent storage (in-memory)
	agents := make(map[string]*contextforge.Agent)
	agentsByName := make(map[string]*contextforge.Agent)
	var agentCounter int

	// POST /a2a - Create agent
	// GET /a2a - List agents
	mux.HandleFunc("/a2a", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var req struct {
				Agent *contextforge.AgentCreate `json:"agent"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			agentCounter++
			id := fmt.Sprintf("agent-%d", agentCounter)
			now := time.Now()

			// Generate slug from name if not provided
			slug := req.Agent.Name
			if req.Agent.Slug != nil {
				slug = *req.Agent.Slug
			}

			agent := &contextforge.Agent{
				ID:              id,
				Name:            req.Agent.Name,
				Slug:            slug,
				Description:     req.Agent.Description,
				EndpointURL:     req.Agent.EndpointURL,
				AgentType:       req.Agent.AgentType,
				ProtocolVersion: req.Agent.ProtocolVersion,
				Capabilities:    req.Agent.Capabilities,
				Config:          req.Agent.Config,
				AuthType:        req.Agent.AuthType,
				// Don't return AuthValue (it's encrypted by API)
				Enabled:   true,
				Reachable: true, // Simulated connectivity status
				Tags:      req.Agent.Tags,
				TeamID:    req.Agent.TeamID,
				Visibility: req.Agent.Visibility,
				CreatedAt: &contextforge.Timestamp{Time: now},
				UpdatedAt: &contextforge.Timestamp{Time: now},
				Metrics: &contextforge.AgentMetrics{
					TotalExecutions:      0,
					SuccessfulExecutions: 0,
					FailedExecutions:     0,
					FailureRate:          0.0,
				},
			}

			agents[id] = agent
			agentsByName[agent.Name] = agent

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-RateLimit-Limit", "1000")
			w.Header().Set("X-RateLimit-Remaining", "995")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", now.Add(time.Hour).Unix()))

			// Return agent directly (not wrapped)
			json.NewEncoder(w).Encode(agent)

		case http.MethodGet:
			query := r.URL.Query()
			result := []*contextforge.Agent{}

			for _, agent := range agents {
				// Apply filters
				if query.Get("include_inactive") != "true" && !agent.Enabled {
					continue
				}
				if teamID := query.Get("team_id"); teamID != "" && agent.TeamID != nil && *agent.TeamID != teamID {
					continue
				}
				if visibility := query.Get("visibility"); visibility != "" && agent.Visibility != nil && *agent.Visibility != visibility {
					continue
				}
				result = append(result, agent)
			}

			// Handle skip/limit pagination
			skip := 0
			if s := query.Get("skip"); s != "" {
				fmt.Sscanf(s, "%d", &skip)
			}

			limit := 100
			if l := query.Get("limit"); l != "" {
				fmt.Sscanf(l, "%d", &limit)
			}

			// Apply skip and limit
			if skip >= len(result) {
				result = []*contextforge.Agent{}
			} else {
				result = result[skip:]
				if len(result) > limit {
					result = result[:limit]
				}
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		}
	})

	// GET /a2a/{id} - Get agent
	// PUT /a2a/{id} - Update agent
	// DELETE /a2a/{id} - Delete agent
	// POST /a2a/{id}/toggle - Toggle agent
	// POST /a2a/{name}/invoke - Invoke agent
	mux.HandleFunc("/a2a/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 3 {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		identifier := parts[2] // Could be ID or name

		// Handle toggle endpoint
		if len(parts) == 4 && parts[3] == "toggle" {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			agent, exists := agents[identifier]
			if !exists {
				http.Error(w, `{"message":"Agent not found"}`, http.StatusNotFound)
				return
			}

			// Extract activate parameter from query string
			activate := r.URL.Query().Get("activate") == "true"
			agent.Enabled = activate
			agent.UpdatedAt = &contextforge.Timestamp{Time: time.Now()}

			w.Header().Set("Content-Type", "application/json")
			// Return agent directly (not wrapped)
			json.NewEncoder(w).Encode(agent)
			return
		}

		// Handle invoke endpoint (uses name, not ID)
		if len(parts) == 4 && parts[3] == "invoke" {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			// Look up by name
			agent, exists := agentsByName[identifier]
			if !exists {
				http.Error(w, `{"message":"Agent not found"}`, http.StatusNotFound)
				return
			}

			// Parse invoke request
			var invokeReq contextforge.AgentInvokeRequest
			if r.Body != nil {
				json.NewDecoder(r.Body).Decode(&invokeReq)
			}

			// Update agent metrics and last interaction
			agent.LastInteraction = &contextforge.Timestamp{Time: time.Now()}
			if agent.Metrics != nil {
				agent.Metrics.TotalExecutions++
				agent.Metrics.SuccessfulExecutions++
			}

			// Return mock invoke response
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"status": "success",
				"result": map[string]any{
					"processed": true,
					"data":      "transformed output from " + agent.Name,
				},
				"execution_time": 123,
			})
			return
		}

		// Handle standard CRUD operations (by ID)
		switch r.Method {
		case http.MethodGet:
			agent, exists := agents[identifier]
			if !exists {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]any{
					"message": "Agent not found",
				})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(agent)

		case http.MethodPut:
			agent, exists := agents[identifier]
			if !exists {
				http.Error(w, `{"message":"Agent not found"}`, http.StatusNotFound)
				return
			}

			// Update request is NOT wrapped (unlike Create)
			var req contextforge.AgentUpdate
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// Update fields
			if req.Name != nil {
				agent.Name = *req.Name
			}
			if req.Description != nil {
				agent.Description = req.Description
			}
			if req.EndpointURL != nil {
				agent.EndpointURL = *req.EndpointURL
			}
			if req.AgentType != nil {
				agent.AgentType = *req.AgentType
			}
			if req.ProtocolVersion != nil {
				agent.ProtocolVersion = *req.ProtocolVersion
			}
			if req.Capabilities != nil {
				agent.Capabilities = req.Capabilities
			}
			if req.Config != nil {
				agent.Config = req.Config
			}
			if req.Tags != nil {
				agent.Tags = req.Tags
			}
			agent.UpdatedAt = &contextforge.Timestamp{Time: time.Now()}

			w.Header().Set("Content-Type", "application/json")
			// Return agent directly (not wrapped)
			json.NewEncoder(w).Encode(agent)

		case http.MethodDelete:
			if _, exists := agents[identifier]; !exists {
				http.Error(w, `{"message":"Agent not found"}`, http.StatusNotFound)
				return
			}

			// Remove from both maps
			if agent, ok := agents[identifier]; ok {
				delete(agentsByName, agent.Name)
			}
			delete(agents, identifier)
			w.WriteHeader(http.StatusNoContent)
		}
	})
}
