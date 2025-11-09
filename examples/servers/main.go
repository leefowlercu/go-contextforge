// Package main demonstrates comprehensive usage of the ServersService
// from the go-contextforge SDK. This example showcases server CRUD operations
// plus unique association methods (ListTools, ListResources, ListPrompts).
// Uses a mock HTTP server for self-contained demonstration.
//
// Run: go run examples/servers/main.go
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

	fmt.Println("=== ContextForge SDK - Servers Service Example ===")

	// Step 1: Authentication
	fmt.Println("1. Authenticating...")
	token := authenticate(server.URL)
	fmt.Printf("   ✓ Obtained JWT token: %s...\n\n", token[:20])

	// Step 2: Create client
	// To use a real ContextForge instance, replace server.URL with:
	// "https://your-contextforge-instance.com"
	client, err := contextforge.NewClient(nil, server.URL, token)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Step 3: Create an MCP server
	fmt.Println("2. Creating a new MCP server...")
	newServer := &contextforge.ServerCreate{
		Name:        "example-mcp-server",
		Description: contextforge.String("An example MCP server for demonstration"),
		Icon:        contextforge.String("server"),
		Tags:        []string{"mcp", "example"},
	}

	// TeamID and Visibility can be passed via options
	opts := &contextforge.ServerCreateOptions{
		TeamID:     contextforge.String("team-example"),
		Visibility: contextforge.String("public"),
	}

	createdServer, resp, err := client.Servers.Create(ctx, newServer, opts)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	fmt.Printf("   ✓ Created server: %s (ID: %s)\n", createdServer.Name, createdServer.ID)
	fmt.Printf("   ✓ Description: %s\n", *createdServer.Description)
	fmt.Printf("   ✓ Active: %v\n", createdServer.IsActive)
	fmt.Printf("   ✓ Rate limit: %d/%d remaining\n\n", resp.Rate.Remaining, resp.Rate.Limit)

	// Step 4: Get the server by ID
	fmt.Println("3. Retrieving server by ID...")
	retrievedServer, _, err := client.Servers.Get(ctx, createdServer.ID)
	if err != nil {
		log.Fatalf("Failed to get server: %v", err)
	}
	fmt.Printf("   ✓ Retrieved: %s\n", retrievedServer.Name)
	fmt.Printf("   ✓ Description: %s\n", *retrievedServer.Description)
	fmt.Printf("   ✓ Active: %v\n", retrievedServer.IsActive)
	fmt.Printf("   ✓ Tags: %v\n\n", retrievedServer.Tags)

	// Step 5: List servers with filtering
	fmt.Println("4. Listing servers with filters...")
	listOpts := &contextforge.ServerListOptions{
		ListOptions: contextforge.ListOptions{
			Limit: 10,
		},
		IncludeInactive: true,
		Tags:            "mcp,example",
		TeamID:          "team-example",
		Visibility:      "public",
	}

	servers, resp, err := client.Servers.List(ctx, listOpts)
	if err != nil {
		log.Fatalf("Failed to list servers: %v", err)
	}
	fmt.Printf("   ✓ Found %d server(s)\n", len(servers))
	for i, svr := range servers {
		fmt.Printf("   %d. %s (ID: %s, Active: %v)\n", i+1, svr.Name, svr.ID, svr.IsActive)
	}
	fmt.Println()

	// Step 6: Update the server
	fmt.Println("5. Updating server...")
	updateServer := &contextforge.ServerUpdate{
		Description: contextforge.String("An advanced MCP server with enhanced features"),
		Tags:        []string{"mcp", "example", "advanced"},
		Icon:        contextforge.String("server-enhanced"),
	}

	updatedServer, _, err := client.Servers.Update(ctx, createdServer.ID, updateServer)
	if err != nil {
		log.Fatalf("Failed to update server: %v", err)
	}
	fmt.Printf("   ✓ Updated description: %s\n", *updatedServer.Description)
	fmt.Printf("   ✓ Updated tags: %v\n", updatedServer.Tags)
	fmt.Printf("   ✓ Updated icon: %s\n\n", *updatedServer.Icon)

	// Step 7: Toggle server (deactivate)
	fmt.Println("6. Toggling server (deactivating)...")
	toggledServer, _, err := client.Servers.Toggle(ctx, createdServer.ID, false)
	if err != nil {
		log.Fatalf("Failed to toggle server: %v", err)
	}
	fmt.Printf("   ✓ Server is now active: %v\n\n", toggledServer.IsActive)

	// Step 8: Toggle server (reactivate)
	fmt.Println("7. Toggling server (reactivating)...")
	toggledServer, _, err = client.Servers.Toggle(ctx, createdServer.ID, true)
	if err != nil {
		log.Fatalf("Failed to toggle server: %v", err)
	}
	fmt.Printf("   ✓ Server is now active: %v\n\n", toggledServer.IsActive)

	// Step 9: List tools associated with this server
	fmt.Println("8. Listing tools provided by server...")
	toolOpts := &contextforge.ServerAssociationOptions{
		IncludeInactive: true,
	}
	tools, _, err := client.Servers.ListTools(ctx, createdServer.ID, toolOpts)
	if err != nil {
		log.Fatalf("Failed to list server tools: %v", err)
	}
	fmt.Printf("   ✓ Server provides %d tool(s):\n", len(tools))
	for i, tool := range tools {
		fmt.Printf("   %d. %s", i+1, tool.Name)
		if tool.Description != nil {
			fmt.Printf(" - %s", *tool.Description)
		}
		fmt.Println()
	}
	fmt.Println()

	// Step 10: List resources associated with this server
	fmt.Println("9. Listing resources provided by server...")
	resourceOpts := &contextforge.ServerAssociationOptions{
		IncludeInactive: true,
	}
	resources, _, err := client.Servers.ListResources(ctx, createdServer.ID, resourceOpts)
	if err != nil {
		log.Fatalf("Failed to list server resources: %v", err)
	}
	fmt.Printf("   ✓ Server provides %d resource(s):\n", len(resources))
	for i, resource := range resources {
		fmt.Printf("   %d. %s", i+1, resource.Name)
		if resource.Description != nil {
			fmt.Printf(" - %s", *resource.Description)
		}
		fmt.Println()
	}
	fmt.Println()

	// Step 11: List prompts associated with this server
	fmt.Println("10. Listing prompts provided by server...")
	promptOpts := &contextforge.ServerAssociationOptions{
		IncludeInactive: true,
	}
	prompts, _, err := client.Servers.ListPrompts(ctx, createdServer.ID, promptOpts)
	if err != nil {
		log.Fatalf("Failed to list server prompts: %v", err)
	}
	fmt.Printf("   ✓ Server provides %d prompt(s):\n", len(prompts))
	for i, prompt := range prompts {
		fmt.Printf("   %d. %s", i+1, prompt.Name)
		if prompt.Description != nil {
			fmt.Printf(" - %s", *prompt.Description)
		}
		fmt.Println()
	}
	fmt.Println()

	// Step 12: Pagination example for associations
	fmt.Println("11. Demonstrating association pagination...")
	page := 1
	// Note: ServerAssociationOptions doesn't have Cursor/Limit fields like ListOptions
	// Pagination would be done differently if supported by the API
	allTools, _, err := client.Servers.ListTools(ctx, createdServer.ID, nil)
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}
	fmt.Printf("   Page %d: %d tool(s) total\n", page, len(allTools))
	fmt.Println()

	// Step 13: Error handling example
	fmt.Println("12. Demonstrating error handling...")
	_, _, err = client.Servers.Get(ctx, "non-existent-server-id")
	if err != nil {
		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
			fmt.Printf("   ✓ Caught expected error: HTTP %d - %s\n",
				apiErr.Response.StatusCode, apiErr.Message)
		} else {
			fmt.Printf("   ✓ Caught error: %v\n", err)
		}
	}
	fmt.Println()

	// Step 14: Delete the server
	fmt.Println("13. Deleting server...")
	_, err = client.Servers.Delete(ctx, createdServer.ID)
	if err != nil {
		log.Fatalf("Failed to delete server: %v", err)
	}
	fmt.Printf("   ✓ Server deleted successfully\n\n")

	fmt.Println("=== Example completed successfully! ===")
	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("• CRUD operations for MCP servers")
	fmt.Println("• Toggle active/inactive state")
	fmt.Println("• List tools associated with a server")
	fmt.Println("• List resources associated with a server")
	fmt.Println("• List prompts associated with a server")
	fmt.Println("• Filtering by tags, team, visibility")
	fmt.Println("\nNote: Servers use ServerCreate/ServerUpdate types (snake_case vs camelCase)")
	fmt.Println("Note: Toggle returns server directly (not nested like Tools)")
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
			"access_token": "mock-jwt-token-67890",
			"token_type":   "bearer",
		})
	})

	// Mock server storage (in-memory)
	servers := make(map[string]*contextforge.Server)
	var serverCounter int

	// Mock associated entities
	mockTools := []*contextforge.Tool{
		{
			ID:          "tool-1",
			Name:        "read_file",
			Description: contextforge.String("Read a file from the filesystem"),
			Enabled:     true,
		},
		{
			ID:          "tool-2",
			Name:        "write_file",
			Description: contextforge.String("Write content to a file"),
			Enabled:     true,
		},
		{
			ID:          "tool-3",
			Name:        "list_directory",
			Description: contextforge.String("List files in a directory"),
			Enabled:     true,
		},
	}

	// Create FlexibleID variables
	resourceID1 := contextforge.FlexibleID("resource-1")
	resourceID2 := contextforge.FlexibleID("resource-2")

	mockResources := []*contextforge.Resource{
		{
			ID:          &resourceID1,
			Name:        "file://config.json",
			Description: contextforge.String("Configuration file"),
			URI:         "file://config.json",
			MimeType:    contextforge.String("application/json"),
			IsActive:    true,
		},
		{
			ID:          &resourceID2,
			Name:        "file://README.md",
			Description: contextforge.String("Readme file"),
			URI:         "file://README.md",
			MimeType:    contextforge.String("text/markdown"),
			IsActive:    true,
		},
	}

	mockPrompts := []*contextforge.Prompt{
		{
			ID:          1,
			Name:        "analyze-file",
			Description: contextforge.String("Analyze the contents of a file"),
			Template:    "Analyze the following file: {{file}}",
			Arguments:   []contextforge.PromptArgument{},
			IsActive:    true,
		},
		{
			ID:          2,
			Name:        "summarize-directory",
			Description: contextforge.String("Summarize contents of a directory"),
			Template:    "Summarize files in: {{directory}}",
			Arguments:   []contextforge.PromptArgument{},
			IsActive:    true,
		},
	}

	// POST /servers - Create server
	// GET /servers - List servers
	mux.HandleFunc("/servers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var req struct {
				Server *contextforge.ServerCreate `json:"server"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			serverCounter++
			id := fmt.Sprintf("server-%d", serverCounter)
			now := time.Now()

			server := &contextforge.Server{
				ID:          id,
				Name:        req.Server.Name,
				Description: req.Server.Description,
				Icon:        req.Server.Icon,
				Tags:        req.Server.Tags,
				IsActive:    true,
				CreatedAt:   &contextforge.Timestamp{Time: now},
				UpdatedAt:   &contextforge.Timestamp{Time: now},
			}

			// Copy organizational fields if present in req.Server
			if req.Server.TeamID != nil {
				server.TeamID = req.Server.TeamID
			}
			if req.Server.Visibility != nil {
				server.Visibility = req.Server.Visibility
			}

			servers[id] = server

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-RateLimit-Limit", "1000")
			w.Header().Set("X-RateLimit-Remaining", "995")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", now.Add(time.Hour).Unix()))

			// Return server directly (not wrapped)
			json.NewEncoder(w).Encode(server)

		case http.MethodGet:
			query := r.URL.Query()
			result := []*contextforge.Server{}

			for _, server := range servers {
				// Apply filters
				if query.Get("include_inactive") != "true" && !server.IsActive {
					continue
				}
				if teamID := query.Get("team_id"); teamID != "" && server.TeamID != nil && *server.TeamID != teamID {
					continue
				}
				if visibility := query.Get("visibility"); visibility != "" && server.Visibility != nil && *server.Visibility != visibility {
					continue
				}
				result = append(result, server)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		}
	})

	// GET /servers/{id} - Get server
	// PUT /servers/{id} - Update server
	// DELETE /servers/{id} - Delete server
	// POST /servers/{id}/toggle - Toggle server
	// GET /servers/{id}/tools - List tools
	// GET /servers/{id}/resources - List resources
	// GET /servers/{id}/prompts - List prompts
	mux.HandleFunc("/servers/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 3 {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		serverID := parts[2]

		// Handle association endpoints
		if len(parts) == 4 {
			switch parts[3] {
			case "toggle":
				if r.Method != http.MethodPost {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					return
				}

				server, exists := servers[serverID]
				if !exists {
					http.Error(w, `{"message":"Server not found"}`, http.StatusNotFound)
					return
				}

				// Extract activate parameter from query string
				activate := r.URL.Query().Get("activate") == "true"
				server.IsActive = activate
				server.UpdatedAt = &contextforge.Timestamp{Time: time.Now()}

				// Note: Servers toggle returns direct response (not nested like Tools)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(server)
				return

			case "tools":
				if r.Method != http.MethodGet {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					return
				}

				if _, exists := servers[serverID]; !exists {
					http.Error(w, `{"message":"Server not found"}`, http.StatusNotFound)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(mockTools)
				return

			case "resources":
				if r.Method != http.MethodGet {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					return
				}

				if _, exists := servers[serverID]; !exists {
					http.Error(w, `{"message":"Server not found"}`, http.StatusNotFound)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(mockResources)
				return

			case "prompts":
				if r.Method != http.MethodGet {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					return
				}

				if _, exists := servers[serverID]; !exists {
					http.Error(w, `{"message":"Server not found"}`, http.StatusNotFound)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(mockPrompts)
				return
			}
		}

		switch r.Method {
		case http.MethodGet:
			server, exists := servers[serverID]
			if !exists {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]any{
					"message": "Server not found",
				})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(server)

		case http.MethodPut:
			server, exists := servers[serverID]
			if !exists {
				http.Error(w, `{"message":"Server not found"}`, http.StatusNotFound)
				return
			}

			// Note: Update request is NOT wrapped (unlike Create)
			var req contextforge.ServerUpdate
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// Update fields
			if req.Description != nil {
				server.Description = req.Description
			}
			if req.Icon != nil {
				server.Icon = req.Icon
			}
			if req.Tags != nil {
				server.Tags = req.Tags
			}
			server.UpdatedAt = &contextforge.Timestamp{Time: time.Now()}

			w.Header().Set("Content-Type", "application/json")
			// Return server directly (not wrapped)
			json.NewEncoder(w).Encode(server)

		case http.MethodDelete:
			if _, exists := servers[serverID]; !exists {
				http.Error(w, `{"message":"Server not found"}`, http.StatusNotFound)
				return
			}

			delete(servers, serverID)
			w.WriteHeader(http.StatusNoContent)
		}
	})
}
