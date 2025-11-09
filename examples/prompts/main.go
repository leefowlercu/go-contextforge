// Package main demonstrates comprehensive usage of the PromptsService
// from the go-contextforge SDK. This example highlights prompts using integer IDs
// (unlike other services) and documents known upstream API bugs. Uses a mock HTTP
// server for self-contained demonstration.
//
// Known Upstream Bugs (ContextForge v0.8.0):
// - CONTEXTFORGE-001: Toggle returns stale isActive state
// - CONTEXTFORGE-002: API accepts empty template field
// - CONTEXTFORGE-003: Toggle returns 400 instead of 404 for non-existent prompts
//
// Run: go run examples/prompts/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
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

	fmt.Println("=== ContextForge SDK - Prompts Service Example ===")

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

	// Step 3: Create a prompt with template arguments
	fmt.Println("2. Creating a new prompt with template arguments...")
	// Note: API uses snake_case for create (like Resources/Servers)
	newPrompt := &contextforge.PromptCreate{
		Name:        "analyze-code",
		Description: contextforge.String("Analyzes code for potential issues and improvements"),
		Template:    "Analyze the following {{language}} code:\n\n{{code}}\n\nProvide suggestions for improvements.",
		Arguments: []contextforge.PromptArgument{
			{
				Name:        "language",
				Description: contextforge.String("Programming language of the code"),
				Required:    true,
			},
			{
				Name:        "code",
				Description: contextforge.String("The code to analyze"),
				Required:    true,
			},
		},
		Tags:       []string{"code", "analysis", "review"},
		TeamID:     contextforge.String("team-example"),
		Visibility: contextforge.String("public"),
	}

	createdPrompt, resp, err := client.Prompts.Create(ctx, newPrompt, nil)
	if err != nil {
		log.Fatalf("Failed to create prompt: %v", err)
	}
	fmt.Printf("   ✓ Created prompt: %s (ID: %d)\n", createdPrompt.Name, createdPrompt.ID)
	fmt.Printf("   ✓ Template: %s\n", createdPrompt.Template[:50]+"...")
	fmt.Printf("   ✓ Arguments: %d\n", len(createdPrompt.Arguments))
	for i, arg := range createdPrompt.Arguments {
		fmt.Printf("     %d. %s (required: %v)\n", i+1, arg.Name, arg.Required)
	}
	fmt.Printf("   ✓ Rate limit: %d/%d remaining\n\n", resp.Rate.Remaining, resp.Rate.Limit)

	// Step 4: Create a simple prompt without arguments
	fmt.Println("3. Creating a simple prompt without arguments...")
	simplePrompt := &contextforge.PromptCreate{
		Name:        "greeting",
		Description: contextforge.String("A friendly greeting prompt"),
		Template:    "Hello! How can I assist you today?",
		Tags:        []string{"greeting", "simple"},
		TeamID:      contextforge.String("team-example"),
		Visibility:  contextforge.String("public"),
	}

	createdPrompt2, _, err := client.Prompts.Create(ctx, simplePrompt, nil)
	if err != nil {
		log.Fatalf("Failed to create prompt: %v", err)
	}
	fmt.Printf("   ✓ Created prompt: %s (ID: %d)\n\n", createdPrompt2.Name, createdPrompt2.ID)

	// Step 4: List prompts with filtering
	// Note: PromptsService does not have a Get method (only List, Create, Update, Delete, Toggle)
	fmt.Println("4. Listing prompts with filters...")
	listOpts := &contextforge.PromptListOptions{
		ListOptions: contextforge.ListOptions{
			Limit: 10,
		},
		IncludeInactive: true,
		Tags:            "code,analysis",
		TeamID:          "team-example",
		Visibility:      "public",
	}

	prompts, resp, err := client.Prompts.List(ctx, listOpts)
	if err != nil {
		log.Fatalf("Failed to list prompts: %v", err)
	}
	fmt.Printf("   ✓ Found %d prompt(s)\n", len(prompts))
	for i, prompt := range prompts {
		fmt.Printf("   %d. %s (ID: %d, Active: %v)\n", i+1, prompt.Name, prompt.ID, prompt.IsActive)
	}
	fmt.Println()

	// Step 5: Update a prompt
	fmt.Println("5. Updating prompt...")
	// Note: API uses camelCase for update (inconsistent with create)
	updatePrompt := &contextforge.PromptUpdate{
		Description: contextforge.String("Advanced code analysis with detailed recommendations"),
		Tags:        []string{"code", "analysis", "review", "advanced"},
	}

	updatedPrompt, _, err := client.Prompts.Update(ctx, createdPrompt.ID, updatePrompt)
	if err != nil {
		log.Fatalf("Failed to update prompt: %v", err)
	}
	fmt.Printf("   ✓ Updated name: %s\n", updatedPrompt.Name)
	fmt.Printf("   ✓ Updated description: %s\n", *updatedPrompt.Description)
	fmt.Printf("   ✓ Updated tags: %v\n\n", updatedPrompt.Tags)

	// Step 6: Pagination example
	fmt.Println("6. Demonstrating pagination...")
	page := 1
	cursor := ""
	for {
		pageOpts := &contextforge.PromptListOptions{
			ListOptions: contextforge.ListOptions{
				Limit:  1,
				Cursor: cursor,
			},
		}
		pagePrompts, pageResp, err := client.Prompts.List(ctx, pageOpts)
		if err != nil {
			log.Fatalf("Failed to list page: %v", err)
		}
		fmt.Printf("   Page %d: %d prompt(s)\n", page, len(pagePrompts))

		if pageResp.NextCursor == "" || len(pagePrompts) == 0 {
			break
		}
		cursor = pageResp.NextCursor
		page++
		if page > 2 { // Limit pagination demo
			fmt.Println("   (stopping after 2 pages for demo)")
			break
		}
	}
	fmt.Println()

	// Step 7: Toggle prompt (deactivate)
	fmt.Println("7. Toggling prompt (deactivating)...")
	fmt.Println("   NOTE: CONTEXTFORGE-001 bug - toggle may return stale isActive state")
	toggledPrompt, _, err := client.Prompts.Toggle(ctx, createdPrompt.ID, false)
	if err != nil {
		log.Fatalf("Failed to toggle prompt: %v", err)
	}
	fmt.Printf("   ✓ Prompt is now active: %v\n", toggledPrompt.IsActive)
	if toggledPrompt.IsActive != false {
		fmt.Println("   ⚠ WARNING: Toggle returned stale state (known bug CONTEXTFORGE-001)")
		fmt.Println("   ⚠ The database was updated correctly, but response shows old value")
	}
	fmt.Println()

	// Step 8: Toggle prompt (reactivate)
	fmt.Println("8. Toggling prompt (reactivating)...")
	toggledPrompt, _, err = client.Prompts.Toggle(ctx, createdPrompt.ID, true)
	if err != nil {
		log.Fatalf("Failed to toggle prompt: %v", err)
	}
	fmt.Printf("   ✓ Prompt is now active: %v\n\n", toggledPrompt.IsActive)

	// Step 9: Demonstrate upstream bug - empty template accepted
	fmt.Println("9. Demonstrating CONTEXTFORGE-002 (empty template accepted)...")
	fmt.Println("   NOTE: API should reject prompts without template, but currently accepts them")
	emptyTemplatePrompt := &contextforge.PromptCreate{
		Name:        "invalid-prompt",
		Description: contextforge.String("Prompt with missing template (should fail)"),
		Template:    "", // Empty template - API bug may allow this
		Tags:        []string{"bug", "test"},
		TeamID:      contextforge.String("team-example"),
		Visibility:  contextforge.String("private"),
	}

	invalidPrompt, _, err := client.Prompts.Create(ctx, emptyTemplatePrompt, nil)
	if err != nil {
		fmt.Printf("   ✓ API correctly rejected empty template: %v\n", err)
	} else {
		fmt.Printf("   ⚠ WARNING: API accepted prompt without template (ID: %d)\n", invalidPrompt.ID)
		fmt.Println("   ⚠ This is CONTEXTFORGE-002 bug - prompts should require template")
		// Clean up invalid prompt
		client.Prompts.Delete(ctx, invalidPrompt.ID)
	}
	fmt.Println()

	// Step 10: Demonstrate upstream bug - toggle non-existent returns 400
	fmt.Println("10. Demonstrating CONTEXTFORGE-003 (toggle returns 400 vs 404)...")
	fmt.Println("   NOTE: Toggle non-existent prompt returns 400, should return 404")
	_, _, err = client.Prompts.Toggle(ctx, 99999, false)
	if err != nil {
		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
			fmt.Printf("   ✓ Got error: HTTP %d\n", apiErr.Response.StatusCode)
			if apiErr.Response.StatusCode == 400 {
				fmt.Println("   ⚠ WARNING: Got 400 Bad Request (should be 404 Not Found)")
				fmt.Println("   ⚠ This is CONTEXTFORGE-003 bug - toggle should return 404")
			}
		}
	}
	fmt.Println()

	// Step 11: Delete prompts
	fmt.Println("11. Deleting prompts...")
	for _, id := range []int{createdPrompt.ID, createdPrompt2.ID} {
		_, err = client.Prompts.Delete(ctx, id)
		if err != nil {
			log.Fatalf("Failed to delete prompt %d: %v", id, err)
		}
		fmt.Printf("   ✓ Deleted prompt: %d\n", id)
	}
	fmt.Println()

	fmt.Println("=== Example completed successfully! ===")
	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("• Prompts use integer IDs (not strings like other services)")
	fmt.Println("• No Get method - use List to retrieve prompts")
	fmt.Println("• Template with argument placeholders ({{language}}, {{code}})")
	fmt.Println("• Required and optional arguments")
	fmt.Println("• Create uses snake_case, Update uses camelCase (API inconsistency)")
	fmt.Println("\nKnown Upstream Bugs (ContextForge v0.8.0):")
	fmt.Println("• CONTEXTFORGE-001: Toggle returns stale isActive state")
	fmt.Println("• CONTEXTFORGE-002: API accepts empty template field")
	fmt.Println("• CONTEXTFORGE-003: Toggle returns 400 instead of 404 for non-existent")
	fmt.Println("\nSee docs/upstream-bugs/ for detailed bug reports")
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
			"access_token": "mock-jwt-token-55555",
			"token_type":   "bearer",
		})
	})

	// Mock prompt storage (in-memory)
	prompts := make(map[int]*contextforge.Prompt)
	var promptCounter int

	// POST /prompts - Create prompt
	// GET /prompts - List prompts
	mux.HandleFunc("/prompts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var req struct {
				Prompt *contextforge.PromptCreate `json:"prompt"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// CONTEXTFORGE-002: API should validate template is present, but doesn't
			// For this mock, we'll accept it to demonstrate the bug
			promptCounter++
			id := promptCounter
			now := time.Now()

			prompt := &contextforge.Prompt{
				ID:          id,
				Name:        req.Prompt.Name,
				Description: req.Prompt.Description,
				Template:    req.Prompt.Template,
				Arguments:   req.Prompt.Arguments,
				Tags:        req.Prompt.Tags,
				TeamID:      req.Prompt.TeamID,
				Visibility:  req.Prompt.Visibility,
				IsActive:    true,
				CreatedAt:   &contextforge.Timestamp{Time: now},
				UpdatedAt:   &contextforge.Timestamp{Time: now},
			}
			prompts[id] = prompt

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-RateLimit-Limit", "1000")
			w.Header().Set("X-RateLimit-Remaining", "995")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", now.Add(time.Hour).Unix()))

			// Return prompt directly (not wrapped)
			json.NewEncoder(w).Encode(prompt)

		case http.MethodGet:
			query := r.URL.Query()
			result := []*contextforge.Prompt{}

			for _, prompt := range prompts {
				// Apply filters
				if query.Get("include_inactive") != "true" && !prompt.IsActive {
					continue
				}
				if teamID := query.Get("team_id"); teamID != "" && prompt.TeamID != nil && *prompt.TeamID != teamID {
					continue
				}
				if visibility := query.Get("visibility"); visibility != "" && prompt.Visibility != nil && *prompt.Visibility != visibility {
					continue
				}
				result = append(result, prompt)
			}

			w.Header().Set("Content-Type", "application/json")

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

	// GET /prompts/{id} - Get prompt
	// PUT /prompts/{id} - Update prompt
	// DELETE /prompts/{id} - Delete prompt
	mux.HandleFunc("/prompts/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 3 {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		promptID, err := strconv.Atoi(parts[2])
		if err != nil {
			http.Error(w, "Invalid prompt ID", http.StatusBadRequest)
			return
		}

		// Handle toggle endpoint
		if len(parts) == 4 && parts[3] == "toggle" {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			prompt, exists := prompts[promptID]
			if !exists {
				// CONTEXTFORGE-003: Should return 404, but returns 400
				http.Error(w, `{"message":"Prompt not found"}`, http.StatusBadRequest)
				return
			}

			// Extract activate parameter from query string
			activate := r.URL.Query().Get("activate") == "true"

			// CONTEXTFORGE-001: Simulate stale state bug
			// We'll update the database but return the OLD value
			oldState := prompt.IsActive
			prompt.IsActive = activate
			prompt.UpdatedAt = &contextforge.Timestamp{Time: time.Now()}

			// Create response with stale isActive (bug simulation)
			stalePrompt := *prompt
			stalePrompt.IsActive = oldState

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"prompt": &stalePrompt})
			return
		}

		switch r.Method {
		case http.MethodGet:
			prompt, exists := prompts[promptID]
			if !exists {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]any{
					"message": "Prompt not found",
				})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(prompt)

		case http.MethodPut:
			prompt, exists := prompts[promptID]
			if !exists {
				http.Error(w, `{"message":"Prompt not found"}`, http.StatusNotFound)
				return
			}

			// Note: Update request is NOT wrapped (unlike Create)
			var req contextforge.PromptUpdate
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// Update fields
			if req.Name != nil {
				prompt.Name = *req.Name
			}
			if req.Description != nil {
				prompt.Description = req.Description
			}
			if req.Tags != nil {
				prompt.Tags = req.Tags
			}
			prompt.UpdatedAt = &contextforge.Timestamp{Time: time.Now()}

			w.Header().Set("Content-Type", "application/json")
			// Note: Update response is NOT wrapped
			json.NewEncoder(w).Encode(prompt)

		case http.MethodDelete:
			if _, exists := prompts[promptID]; !exists {
				http.Error(w, `{"message":"Prompt not found"}`, http.StatusNotFound)
				return
			}

			delete(prompts, promptID)
			w.WriteHeader(http.StatusNoContent)
		}
	})
}
