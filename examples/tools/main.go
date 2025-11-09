// Package main demonstrates comprehensive usage of the ToolsService
// from the go-contextforge SDK. This example uses a mock HTTP server
// to provide a self-contained demonstration without requiring a live
// ContextForge instance.
//
// Run: go run examples/tools/main.go
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

	fmt.Println("=== ContextForge SDK - Tools Service Example ===\n")

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

	// Step 3: Create a tool
	fmt.Println("2. Creating a new tool...")
	newTool := &contextforge.Tool{
		Name:        "example-calculator",
		Description: contextforge.String("A simple calculator tool for demonstrations"),
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"operation": map[string]any{
					"type": "string",
					"enum": []string{"add", "subtract", "multiply", "divide"},
				},
				"a": map[string]any{"type": "number"},
				"b": map[string]any{"type": "number"},
			},
			"required": []string{"operation", "a", "b"},
		},
		Tags:       []string{"math", "calculator", "example"},
		Visibility: "public",
	}

	// TeamID is passed via options, not in the tool struct
	opts := &contextforge.ToolCreateOptions{
		TeamID:     contextforge.String("team-example"),
		Visibility: contextforge.String("public"),
	}

	createdTool, resp, err := client.Tools.Create(ctx, newTool, opts)
	if err != nil {
		log.Fatalf("Failed to create tool: %v", err)
	}
	fmt.Printf("   ✓ Created tool: %s (ID: %s)\n", createdTool.Name, createdTool.ID)
	fmt.Printf("   ✓ Description: %s\n", *createdTool.Description)
	fmt.Printf("   ✓ Enabled: %v\n", createdTool.Enabled)
	fmt.Printf("   ✓ Rate limit: %d/%d remaining\n\n", resp.Rate.Remaining, resp.Rate.Limit)

	// Step 4: Get the tool by ID
	fmt.Println("3. Retrieving tool by ID...")
	retrievedTool, _, err := client.Tools.Get(ctx, createdTool.ID)
	if err != nil {
		log.Fatalf("Failed to get tool: %v", err)
	}
	fmt.Printf("   ✓ Retrieved: %s\n", retrievedTool.Name)
	fmt.Printf("   ✓ Description: %s\n", *retrievedTool.Description)
	fmt.Printf("   ✓ Enabled: %v\n", retrievedTool.Enabled)
	fmt.Printf("   ✓ Tags: %v\n\n", retrievedTool.Tags)

	// Step 5: List tools with filtering and pagination
	fmt.Println("4. Listing tools with filters...")
	listOpts := &contextforge.ToolListOptions{
		ListOptions: contextforge.ListOptions{
			Limit: 10,
		},
		IncludeInactive: true,
		Tags:            "math,calculator",
		TeamID:          "team-example",
		Visibility:      "public",
	}

	tools, resp, err := client.Tools.List(ctx, listOpts)
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}
	fmt.Printf("   ✓ Found %d tool(s)\n", len(tools))
	for i, tool := range tools {
		fmt.Printf("   %d. %s (ID: %s, Enabled: %v)\n", i+1, tool.Name, tool.ID, tool.Enabled)
	}
	if resp.NextCursor != "" {
		fmt.Printf("   ✓ More results available (cursor: %s)\n", resp.NextCursor)
	}
	fmt.Println()

	// Step 6: Pagination example
	fmt.Println("5. Demonstrating pagination...")
	page := 1
	cursor := ""
	for {
		pageOpts := &contextforge.ToolListOptions{
			ListOptions: contextforge.ListOptions{
				Limit:  2,
				Cursor: cursor,
			},
		}
		pageTools, pageResp, err := client.Tools.List(ctx, pageOpts)
		if err != nil {
			log.Fatalf("Failed to list page: %v", err)
		}
		fmt.Printf("   Page %d: %d tool(s)\n", page, len(pageTools))

		if pageResp.NextCursor == "" {
			break
		}
		cursor = pageResp.NextCursor
		page++
		if page > 3 { // Limit pagination demo
			fmt.Println("   (stopping after 3 pages for demo)")
			break
		}
	}
	fmt.Println()

	// Step 7: Update the tool
	fmt.Println("6. Updating tool...")
	updateTool := &contextforge.Tool{
		Description: contextforge.String("An advanced calculator with additional features"),
		Tags:        []string{"math", "calculator", "example", "advanced"},
	}

	updatedTool, _, err := client.Tools.Update(ctx, createdTool.ID, updateTool)
	if err != nil {
		log.Fatalf("Failed to update tool: %v", err)
	}
	fmt.Printf("   ✓ Updated description: %s\n", *updatedTool.Description)
	fmt.Printf("   ✓ Updated tags: %v\n\n", updatedTool.Tags)

	// Step 8: Toggle tool (disable)
	fmt.Println("7. Toggling tool (disabling)...")
	toggledTool, _, err := client.Tools.Toggle(ctx, createdTool.ID, false)
	if err != nil {
		log.Fatalf("Failed to toggle tool: %v", err)
	}
	fmt.Printf("   ✓ Tool is now enabled: %v\n\n", toggledTool.Enabled)

	// Step 9: Toggle tool (enable)
	fmt.Println("8. Toggling tool (enabling)...")
	toggledTool, _, err = client.Tools.Toggle(ctx, createdTool.ID, true)
	if err != nil {
		log.Fatalf("Failed to toggle tool: %v", err)
	}
	fmt.Printf("   ✓ Tool is now enabled: %v\n\n", toggledTool.Enabled)

	// Step 10: Error handling example
	fmt.Println("9. Demonstrating error handling...")
	_, _, err = client.Tools.Get(ctx, "non-existent-tool-id")
	if err != nil {
		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
			fmt.Printf("   ✓ Caught expected error: HTTP %d - %s\n",
				apiErr.Response.StatusCode, apiErr.Message)
		} else {
			fmt.Printf("   ✓ Caught error: %v\n", err)
		}
	}
	fmt.Println()

	// Step 11: Delete the tool
	fmt.Println("10. Deleting tool...")
	_, err = client.Tools.Delete(ctx, createdTool.ID)
	if err != nil {
		log.Fatalf("Failed to delete tool: %v", err)
	}
	fmt.Printf("   ✓ Tool deleted successfully\n\n")

	fmt.Println("=== Example completed successfully! ===")
	fmt.Println("\nTo use with a real ContextForge instance:")
	fmt.Println("1. Replace server.URL with your ContextForge base URL")
	fmt.Println("2. Use real authentication credentials")
	fmt.Println("3. Adjust team IDs and other parameters to match your setup")
}

// authenticate performs mock authentication and returns a JWT token
func authenticate(baseURL string) string {
	// In a real application, you would:
	// 1. POST to /auth/login with username/password
	// 2. Extract the access_token from the response
	// 3. Use that token for subsequent requests
	//
	// For this mock example, we'll simulate the login request
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
			"access_token": "mock-jwt-token-12345",
			"token_type":   "bearer",
		})
	})

	// Mock tool storage (in-memory)
	tools := make(map[string]*contextforge.Tool)
	var toolCounter int

	// POST /tools - Create tool
	mux.HandleFunc("/tools", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var req struct {
				Tool *contextforge.Tool `json:"tool"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			toolCounter++
			id := fmt.Sprintf("tool-%d", toolCounter)
			now := time.Now()

			tool := req.Tool
			tool.ID = id
			tool.Enabled = true
			tool.CreatedAt = &contextforge.Timestamp{Time: now}
			tool.UpdatedAt = &contextforge.Timestamp{Time: now}

			tools[id] = tool

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-RateLimit-Limit", "1000")
			w.Header().Set("X-RateLimit-Remaining", "995")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", now.Add(time.Hour).Unix()))

			// Return tool directly (not wrapped)
			json.NewEncoder(w).Encode(tool)

		case http.MethodGet:
			// List tools with filtering
			query := r.URL.Query()
			result := []*contextforge.Tool{}

			for _, tool := range tools {
				// Apply filters
				if query.Get("include_inactive") != "true" && !tool.Enabled {
					continue
				}
				if teamID := query.Get("team_id"); teamID != "" && tool.TeamID != nil && *tool.TeamID != teamID {
					continue
				}
				if visibility := query.Get("visibility"); visibility != "" && tool.Visibility != visibility {
					continue
				}
				result = append(result, tool)
			}

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-RateLimit-Limit", "1000")
			w.Header().Set("X-RateLimit-Remaining", "994")

			// Pagination simulation
			limit := 10
			if l := query.Get("limit"); l != "" {
				fmt.Sscanf(l, "%d", &limit)
			}

			if len(result) > limit {
				w.Header().Set("X-Next-Cursor", "mock-cursor-next-page")
				result = result[:limit]
			}

			json.NewEncoder(w).Encode(result)
		}
	})

	// GET /tools/{id} - Get tool
	// PUT /tools/{id} - Update tool
	// DELETE /tools/{id} - Delete tool
	mux.HandleFunc("/tools/", func(w http.ResponseWriter, r *http.Request) {
		// Extract tool ID from path
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 3 {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		toolID := parts[2]

		// Handle toggle endpoint
		if strings.Contains(r.URL.Path, "/toggle") {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			tool, exists := tools[toolID]
			if !exists {
				http.Error(w, `{"message":"Tool not found"}`, http.StatusNotFound)
				return
			}

			// Extract activate parameter from query string
			activate := r.URL.Query().Get("activate") == "true"
			tool.Enabled = activate
			tool.UpdatedAt = &contextforge.Timestamp{Time: time.Now()}

			w.Header().Set("Content-Type", "application/json")
			// Toggle returns wrapped response
			json.NewEncoder(w).Encode(map[string]any{
				"status":  "success",
				"message": "Tool toggled successfully",
				"tool":    tool,
			})
			return
		}

		switch r.Method {
		case http.MethodGet:
			tool, exists := tools[toolID]
			if !exists {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]any{
					"message": "Tool not found",
				})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tool)

		case http.MethodPut:
			tool, exists := tools[toolID]
			if !exists {
				http.Error(w, `{"message":"Tool not found"}`, http.StatusNotFound)
				return
			}

			var req struct {
				Tool *contextforge.Tool `json:"tool"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// Update fields
			if req.Tool.Description != nil {
				tool.Description = req.Tool.Description
			}
			if req.Tool.Tags != nil {
				tool.Tags = req.Tool.Tags
			}
			tool.UpdatedAt = &contextforge.Timestamp{Time: time.Now()}

			w.Header().Set("Content-Type", "application/json")
			// Return tool directly (not wrapped)
			json.NewEncoder(w).Encode(tool)

		case http.MethodDelete:
			if _, exists := tools[toolID]; !exists {
				http.Error(w, `{"message":"Tool not found"}`, http.StatusNotFound)
				return
			}

			delete(tools, toolID)
			w.WriteHeader(http.StatusNoContent)
		}
	})
}
