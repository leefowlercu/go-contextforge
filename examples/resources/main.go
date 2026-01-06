// Package main demonstrates comprehensive usage of the ResourcesService
// from the go-contextforge SDK. This example highlights API inconsistencies
// between create (snake_case fields) and update (camelCase fields), plus
// the ListTemplates method. Uses a mock HTTP server for self-contained demonstration.
//
// Run: go run examples/resources/main.go
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

	fmt.Println("=== ContextForge SDK - Resources Service ===")

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

	// Step 3: List available resource templates
	fmt.Println("2. Listing available resource templates...")
	templates, _, err := client.Resources.ListTemplates(ctx)
	if err != nil {
		log.Fatalf("Failed to list templates: %v", err)
	}
	fmt.Printf("   ✓ Found %d template(s):\n", len(templates.Templates))
	for i, template := range templates.Templates {
		fmt.Printf("   %d. %s - %s\n", i+1, template.Name, template.Description)
		fmt.Printf("      URI: %s\n", template.URI)
		fmt.Printf("      MIME Type: %s\n", template.MimeType)
	}
	fmt.Println()

	// Step 4: Create a resource
	fmt.Println("3. Creating a new resource...")
	// IMPORTANT: ResourceCreate uses snake_case fields (mime_type)
	newResource := &contextforge.ResourceCreate{
		URI:         "file:///etc/app/config.json",
		Name:        "example-config-file",
		Content:     `{"setting": "value"}`,
		Description: contextforge.String("A configuration file resource for demonstration"),
		MimeType:    contextforge.String("application/json"),
		Tags:        []string{"config", "json", "example"},
	}

	// TeamID and Visibility can be passed via options
	opts := &contextforge.ResourceCreateOptions{
		TeamID:     contextforge.String("team-example"),
		Visibility: contextforge.String("public"),
	}

	createdResource, resp, err := client.Resources.Create(ctx, newResource, opts)
	if err != nil {
		log.Fatalf("Failed to create resource: %v", err)
	}
	fmt.Printf("   ✓ Created resource: %s (ID: %s)\n", createdResource.Name, (*createdResource.ID).String())
	fmt.Printf("   ✓ URI: %s\n", createdResource.URI)
	if createdResource.MimeType != nil {
		fmt.Printf("   ✓ MIME Type: %s\n", *createdResource.MimeType)
	}
	fmt.Printf("   ✓ Active: %v\n", createdResource.IsActive)
	fmt.Printf("   ✓ Rate limit: %d/%d remaining\n\n", resp.Rate.Remaining, resp.Rate.Limit)

	// Step 5: List resources with filtering
	fmt.Println("4. Listing resources with filters...")
	listOpts := &contextforge.ResourceListOptions{
		ListOptions: contextforge.ListOptions{
			Limit: 10,
		},
		IncludeInactive: true,
		Tags:            "config,json",
		TeamID:          "team-example",
		Visibility:      "public",
	}

	resources, resp, err := client.Resources.List(ctx, listOpts)
	if err != nil {
		log.Fatalf("Failed to list resources: %v", err)
	}
	fmt.Printf("   ✓ Found %d resource(s)\n", len(resources))
	for i, resource := range resources {
		fmt.Printf("   %d. %s (ID: %s, Active: %v)\n", i+1, resource.Name, (*resource.ID).String(), resource.IsActive)
		fmt.Printf("      URI: %s\n", resource.URI)
	}
	fmt.Println()

	// Step 6: Update the resource
	fmt.Println("5. Updating resource...")
	// IMPORTANT: ResourceUpdate uses camelCase fields (mimeType)
	// This is an API inconsistency - Create uses snake_case, Update uses camelCase
	updateResource := &contextforge.ResourceUpdate{
		Description: contextforge.String("An advanced configuration with additional metadata"),
		Tags:        []string{"config", "json", "example", "advanced"},
		// Note: MimeType would use camelCase if we were updating it
	}

	updatedResource, _, err := client.Resources.Update(ctx, (*createdResource.ID).String(), updateResource)
	if err != nil {
		log.Fatalf("Failed to update resource: %v", err)
	}
	if updatedResource.Description != nil {
		fmt.Printf("   ✓ Updated description: %s\n", *updatedResource.Description)
	}
	fmt.Printf("   ✓ Updated tags: %v\n\n", updatedResource.Tags)

	// Step 7: Pagination example
	fmt.Println("6. Demonstrating pagination...")
	page := 1
	cursor := ""
	for {
		pageOpts := &contextforge.ResourceListOptions{
			ListOptions: contextforge.ListOptions{
				Limit:  2,
				Cursor: cursor,
			},
		}
		pageResources, pageResp, err := client.Resources.List(ctx, pageOpts)
		if err != nil {
			log.Fatalf("Failed to list page: %v", err)
		}
		fmt.Printf("   Page %d: %d resource(s)\n", page, len(pageResources))

		if pageResp.NextCursor == "" || len(pageResources) == 0 {
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

	// Step 8: Toggle resource (deactivate)
	fmt.Println("7. Toggling resource (deactivating)...")
	// Note: Resources toggle has complex response unwrapping due to snake_case response
	toggledResource, _, err := client.Resources.Toggle(ctx, (*createdResource.ID).String(), false)
	if err != nil {
		log.Fatalf("Failed to toggle resource: %v", err)
	}
	fmt.Printf("   ✓ Resource is now active: %v\n\n", toggledResource.IsActive)

	// Step 9: Toggle resource (reactivate)
	fmt.Println("8. Toggling resource (reactivating)...")
	toggledResource, _, err = client.Resources.Toggle(ctx, (*createdResource.ID).String(), true)
	if err != nil {
		log.Fatalf("Failed to toggle resource: %v", err)
	}
	fmt.Printf("   ✓ Resource is now active: %v\n\n", toggledResource.IsActive)

	// Step 10: Delete the resource
	fmt.Println("9. Deleting resource...")
	_, err = client.Resources.Delete(ctx, (*createdResource.ID).String())
	if err != nil {
		log.Fatalf("Failed to delete resource: %v", err)
	}
	fmt.Printf("   ✓ Resource deleted successfully\n\n")

	fmt.Println("=== Example completed successfully! ===")
	fmt.Println("\nKey API Quirks Demonstrated:")
	fmt.Println("• ResourceCreate uses snake_case (mime_type)")
	fmt.Println("• ResourceUpdate uses camelCase (mimeType)")
	fmt.Println("• Toggle response has complex unwrapping (snake_case)")
	fmt.Println("• ListTemplates provides available resource templates")
	fmt.Println("\nNote: This is an API inconsistency that the SDK handles internally")
}

// authenticate performs mock authentication and returns a JWT token
func authenticate(address string) string {
	loginURL := address + "/auth/login"
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
			"access_token": "mock-jwt-token-11111",
			"token_type":   "bearer",
		})
	})

	// Mock resource storage (in-memory)
	resources := make(map[string]*contextforge.Resource)
	var resourceCounter int

	// GET /resources/templates/list - List templates
	mux.HandleFunc("/resources/templates/list", func(w http.ResponseWriter, r *http.Request) {
		templates := &contextforge.ListResourceTemplatesResult{
			Templates: []contextforge.ResourceTemplate{
				{
					Name:        "File Resource",
					Description: "A file-based resource template",
					URI:         "file://{path}",
					MimeType:    "text/plain",
				},
				{
					Name:        "HTTP Resource",
					Description: "An HTTP/HTTPS resource template",
					URI:         "https://{domain}/{path}",
					MimeType:    "application/json",
				},
				{
					Name:        "Database Resource",
					Description: "A database connection resource template",
					URI:         "postgres://{host}:{port}/{database}",
					MimeType:    "application/sql",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(templates)
	})

	// POST /resources - Create resource
	// GET /resources - List resources
	mux.HandleFunc("/resources", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var req struct {
				Resource *contextforge.ResourceCreate `json:"resource"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			resourceCounter++
			id := contextforge.FlexibleID(fmt.Sprintf("resource-%d", resourceCounter))
			now := time.Now()

			resource := &contextforge.Resource{
				ID:          &id,
				URI:         req.Resource.URI,
				Name:        req.Resource.Name,
				Description: req.Resource.Description,
				MimeType:    req.Resource.MimeType,
				Tags:        contextforge.NewTags(req.Resource.Tags),
				IsActive:    true,
				CreatedAt:   &contextforge.Timestamp{Time: now},
				UpdatedAt:   &contextforge.Timestamp{Time: now},
			}

			resources[id.String()] = resource

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-RateLimit-Limit", "1000")
			w.Header().Set("X-RateLimit-Remaining", "995")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", now.Add(time.Hour).Unix()))

			// Return resource directly (not wrapped)
			json.NewEncoder(w).Encode(resource)

		case http.MethodGet:
			query := r.URL.Query()
			result := []*contextforge.Resource{}

			for _, resource := range resources {
				// Apply filters
				if query.Get("include_inactive") != "true" && !resource.IsActive {
					continue
				}
				if teamID := query.Get("team_id"); teamID != "" && resource.TeamID != nil && *resource.TeamID != teamID {
					continue
				}
				if visibility := query.Get("visibility"); visibility != "" && resource.Visibility != nil && *resource.Visibility != visibility {
					continue
				}
				result = append(result, resource)
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

	// GET /resources/{id} - Get resource
	// PUT /resources/{id} - Update resource
	// DELETE /resources/{id} - Delete resource
	mux.HandleFunc("/resources/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 3 {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		// Skip /resources/templates/list
		if parts[2] == "templates" {
			return
		}

		resourceID := parts[2]

		// Handle toggle endpoint
		if len(parts) == 4 && parts[3] == "toggle" {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			resource, exists := resources[resourceID]
			if !exists {
				http.Error(w, `{"message":"Resource not found"}`, http.StatusNotFound)
				return
			}

			// Extract activate parameter from query string
			activate := r.URL.Query().Get("activate") == "true"
			resource.IsActive = activate
			resource.UpdatedAt = &contextforge.Timestamp{Time: time.Now()}

			// Resources toggle has complex response with status/message and snake_case
			// Note: The SDK handles converting this to the standard Resource format
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"status":  "success",
				"message": "Resource toggled successfully",
				"resource": map[string]any{
					"id":          (*resource.ID).String(),
					"uri":         resource.URI,
					"name":        resource.Name,
					"description": resource.Description,
					"mime_type":   resource.MimeType, // Note: snake_case in toggle response
					"is_active":   resource.IsActive, // Note: snake_case in toggle response
					"tags":        resource.Tags,
					"created_at":  resource.CreatedAt,
					"updated_at":  resource.UpdatedAt,
				},
			})
			return
		}

		switch r.Method {
		case http.MethodGet:
			resource, exists := resources[resourceID]
			if !exists {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]any{
					"message": "Resource not found",
				})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resource)

		case http.MethodPut:
			resource, exists := resources[resourceID]
			if !exists {
				http.Error(w, `{"message":"Resource not found"}`, http.StatusNotFound)
				return
			}

			// Note: Update request is NOT wrapped (unlike Create)
			var req contextforge.ResourceUpdate
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// Update fields
			if req.Description != nil {
				resource.Description = req.Description
			}
			if req.Tags != nil {
				resource.Tags = contextforge.NewTags(req.Tags)
			}
			if req.MimeType != nil {
				resource.MimeType = req.MimeType
			}
			resource.UpdatedAt = &contextforge.Timestamp{Time: time.Now()}

			w.Header().Set("Content-Type", "application/json")
			// Return resource directly (not wrapped)
			json.NewEncoder(w).Encode(resource)

		case http.MethodDelete:
			if _, exists := resources[resourceID]; !exists {
				http.Error(w, `{"message":"Resource not found"}`, http.StatusNotFound)
				return
			}

			delete(resources, resourceID)
			w.WriteHeader(http.StatusNoContent)
		}
	})
}
