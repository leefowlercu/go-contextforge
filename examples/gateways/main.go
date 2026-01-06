// Package main demonstrates comprehensive usage of the GatewaysService
// from the go-contextforge SDK. This example showcases various authentication
// types (none, basic, bearer, api_key, oauth) and highlights that gateways
// use unwrapped request bodies. Uses a mock HTTP server for self-contained demonstration.
//
// Run: go run examples/gateways/main.go
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

	fmt.Println("=== ContextForge SDK - Gateways Service Example ===")

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

	// Step 3: Create a gateway with no authentication
	fmt.Println("2. Creating gateway with no authentication...")
	noAuthGateway := &contextforge.Gateway{
		Name:        "public-gateway",
		URL:         "https://api.example.com",
		Description: contextforge.String("A public gateway with no authentication"),
		AuthType:    contextforge.String("none"),
		Tags:        contextforge.NewTags([]string{"public", "example"}),
	}

	// TeamID and Visibility can be passed via options
	opts := &contextforge.GatewayCreateOptions{
		TeamID:     contextforge.String("team-example"),
		Visibility: contextforge.String("public"),
	}

	createdGateway1, resp, err := client.Gateways.Create(ctx, noAuthGateway, opts)
	if err != nil {
		log.Fatalf("Failed to create gateway: %v", err)
	}
	fmt.Printf("   ✓ Created: %s (ID: %s)\n", createdGateway1.Name, *createdGateway1.ID)
	fmt.Printf("   ✓ Auth Type: %s\n", *createdGateway1.AuthType)
	fmt.Printf("   ✓ Enabled: %v\n", createdGateway1.Enabled)
	fmt.Printf("   ✓ Rate limit: %d/%d remaining\n\n", resp.Rate.Remaining, resp.Rate.Limit)

	// Step 4: Create a gateway with basic authentication
	fmt.Println("3. Creating gateway with basic authentication...")
	basicAuthGateway := &contextforge.Gateway{
		Name:         "basic-auth-gateway",
		URL:          "https://api.private.example.com",
		Description:  contextforge.String("A gateway using HTTP Basic Authentication"),
		AuthType:     contextforge.String("basic"),
		AuthUsername: contextforge.String("admin"),
		AuthPassword: contextforge.String("secret123"),
		Tags:         contextforge.NewTags([]string{"basic-auth", "private"}),
	}

	createdGateway2, _, err := client.Gateways.Create(ctx, basicAuthGateway, nil)
	if err != nil {
		log.Fatalf("Failed to create gateway: %v", err)
	}
	fmt.Printf("   ✓ Created: %s (ID: %s)\n", createdGateway2.Name, *createdGateway2.ID)
	fmt.Printf("   ✓ Auth Type: %s\n", *createdGateway2.AuthType)
	if createdGateway2.AuthUsername != nil {
		fmt.Printf("   ✓ Username: %s\n\n", *createdGateway2.AuthUsername)
	}

	// Step 5: Create a gateway with bearer token authentication
	fmt.Println("4. Creating gateway with bearer token authentication...")
	bearerAuthGateway := &contextforge.Gateway{
		Name:        "bearer-auth-gateway",
		URL:         "https://api.secure.example.com",
		Description: contextforge.String("A gateway using Bearer token authentication"),
		AuthType:    contextforge.String("bearer"),
		AuthToken:   contextforge.String("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."),
		Tags:        contextforge.NewTags([]string{"bearer-auth", "jwt"}),
	}

	createdGateway3, _, err := client.Gateways.Create(ctx, bearerAuthGateway, nil)
	if err != nil {
		log.Fatalf("Failed to create gateway: %v", err)
	}
	fmt.Printf("   ✓ Created: %s (ID: %s)\n", createdGateway3.Name, *createdGateway3.ID)
	fmt.Printf("   ✓ Auth Type: %s\n", *createdGateway3.AuthType)
	if createdGateway3.AuthToken != nil {
		fmt.Printf("   ✓ Token: %s...\n\n", (*createdGateway3.AuthToken)[:20])
	}

	// Step 6: Create a gateway with API key authentication
	fmt.Println("5. Creating gateway with API key authentication...")
	apiKeyGateway := &contextforge.Gateway{
		Name:        "apikey-gateway",
		URL:         "https://api.partner.example.com",
		Description: contextforge.String("A gateway using API key in headers"),
		AuthType:    contextforge.String("api_key"),
		AuthHeaders: []map[string]string{
			{"X-API-Key": "abc123def456"},
			{"X-Client-ID": "client-12345"},
		},
		Tags: contextforge.NewTags([]string{"api-key", "partner"}),
	}

	createdGateway4, _, err := client.Gateways.Create(ctx, apiKeyGateway, nil)
	if err != nil {
		log.Fatalf("Failed to create gateway: %v", err)
	}
	fmt.Printf("   ✓ Created: %s (ID: %s)\n", createdGateway4.Name, *createdGateway4.ID)
	fmt.Printf("   ✓ Auth Type: %s\n", *createdGateway4.AuthType)
	fmt.Printf("   ✓ Headers: %v\n\n", createdGateway4.AuthHeaders)

	// Step 7: Create a gateway with OAuth configuration
	fmt.Println("6. Creating gateway with OAuth authentication...")
	oauthGateway := &contextforge.Gateway{
		Name:        "oauth-gateway",
		URL:         "https://api.oauth.example.com",
		Description: contextforge.String("A gateway using OAuth 2.0 authentication"),
		AuthType:    contextforge.String("oauth"),
		OAuthConfig: map[string]any{
			"client_id":     "oauth-client-123",
			"client_secret": "oauth-secret-456",
			"token_url":     "https://auth.example.com/oauth/token",
			"scope":         "read write",
		},
		Tags: contextforge.NewTags([]string{"oauth", "oauth2"}),
	}

	createdGateway5, _, err := client.Gateways.Create(ctx, oauthGateway, nil)
	if err != nil {
		log.Fatalf("Failed to create gateway: %v", err)
	}
	fmt.Printf("   ✓ Created: %s (ID: %s)\n", createdGateway5.Name, *createdGateway5.ID)
	fmt.Printf("   ✓ Auth Type: %s\n", *createdGateway5.AuthType)
	if clientID, ok := createdGateway5.OAuthConfig["client_id"].(string); ok {
		fmt.Printf("   ✓ OAuth Client ID: %s\n", clientID)
	}
	if tokenURL, ok := createdGateway5.OAuthConfig["token_url"].(string); ok {
		fmt.Printf("   ✓ Token URL: %s\n\n", tokenURL)
	}

	// Step 8: List all gateways with filtering
	fmt.Println("7. Listing all gateways with filters...")
	listOpts := &contextforge.GatewayListOptions{
		ListOptions: contextforge.ListOptions{
			Limit: 10,
		},
		IncludeInactive: true,
	}

	gateways, _, err := client.Gateways.List(ctx, listOpts)
	if err != nil {
		log.Fatalf("Failed to list gateways: %v", err)
	}
	fmt.Printf("   ✓ Found %d gateway(s):\n", len(gateways))
	for i, gw := range gateways {
		authType := "unknown"
		if gw.AuthType != nil {
			authType = *gw.AuthType
		}
		fmt.Printf("   %d. %s (Auth: %s, Enabled: %v)\n", i+1, gw.Name, authType, gw.Enabled)
	}
	fmt.Println()

	// Step 9: Get gateway by ID
	fmt.Println("8. Retrieving gateway by ID...")
	retrievedGateway, _, err := client.Gateways.Get(ctx, *createdGateway1.ID)
	if err != nil {
		log.Fatalf("Failed to get gateway: %v", err)
	}
	fmt.Printf("   ✓ Retrieved: %s\n", retrievedGateway.Name)
	fmt.Printf("   ✓ URL: %s\n", retrievedGateway.URL)
	if retrievedGateway.AuthType != nil {
		fmt.Printf("   ✓ Auth Type: %s\n", *retrievedGateway.AuthType)
	}
	fmt.Printf("   ✓ Enabled: %v\n\n", retrievedGateway.Enabled)

	// Step 10: Update a gateway
	fmt.Println("9. Updating gateway...")
	updateGateway := &contextforge.Gateway{
		Description: contextforge.String("An updated public gateway with enhanced features"),
		Tags:        contextforge.NewTags([]string{"public", "example", "updated"}),
	}

	updatedGateway, _, err := client.Gateways.Update(ctx, *createdGateway1.ID, updateGateway)
	if err != nil {
		log.Fatalf("Failed to update gateway: %v", err)
	}
	if updatedGateway.Description != nil {
		fmt.Printf("   ✓ Updated description: %s\n", *updatedGateway.Description)
	}
	fmt.Printf("   ✓ Updated tags: %v\n\n", updatedGateway.Tags)

	// Step 11: Toggle gateway (disable)
	fmt.Println("10. Toggling gateway (disabling)...")
	toggledGateway, _, err := client.Gateways.Toggle(ctx, *createdGateway1.ID, false)
	if err != nil {
		log.Fatalf("Failed to toggle gateway: %v", err)
	}
	fmt.Printf("   ✓ Gateway is now enabled: %v\n\n", toggledGateway.Enabled)

	// Step 12: Toggle gateway (enable)
	fmt.Println("11. Toggling gateway (enabling)...")
	toggledGateway, _, err = client.Gateways.Toggle(ctx, *createdGateway1.ID, true)
	if err != nil {
		log.Fatalf("Failed to toggle gateway: %v", err)
	}
	fmt.Printf("   ✓ Gateway is now enabled: %v\n\n", toggledGateway.Enabled)

	// Step 13: Error handling example
	fmt.Println("12. Demonstrating error handling...")
	_, _, err = client.Gateways.Get(ctx, "non-existent-gateway-id")
	if err != nil {
		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
			fmt.Printf("   ✓ Caught expected error: HTTP %d - %s\n",
				apiErr.Response.StatusCode, apiErr.Message)
		} else {
			fmt.Printf("   ✓ Caught error: %v\n", err)
		}
	}
	fmt.Println()

	// Step 14: Delete all gateways
	fmt.Println("13. Deleting all gateways...")
	gatewayIDs := []*string{
		createdGateway1.ID,
		createdGateway2.ID,
		createdGateway3.ID,
		createdGateway4.ID,
		createdGateway5.ID,
	}
	for _, id := range gatewayIDs {
		if id != nil {
			_, err = client.Gateways.Delete(ctx, *id)
			if err != nil {
				log.Fatalf("Failed to delete gateway %s: %v", *id, err)
			}
			fmt.Printf("   ✓ Deleted gateway: %s\n", *id)
		}
	}
	fmt.Println()

	fmt.Println("=== Example completed successfully! ===")
	fmt.Println("\nAuthentication Types Demonstrated:")
	fmt.Println("• none - No authentication required")
	fmt.Println("• basic - HTTP Basic Authentication (username/password)")
	fmt.Println("• bearer - Bearer token authentication (JWT)")
	fmt.Println("• api_key - API key in custom headers")
	fmt.Println("• oauth - OAuth 2.0 with client credentials")
	fmt.Println("\nKey Differences:")
	fmt.Println("• Gateways use unwrapped request bodies (unlike Tools/Resources/Servers)")
	fmt.Println("• Single Gateway type for all operations (no separate Create/Update types)")
	fmt.Println("• Toggle returns nested response like Tools")
	fmt.Println("• Complex authentication configurations supported")
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
			"access_token": "mock-jwt-token-99999",
			"token_type":   "bearer",
		})
	})

	// Mock gateway storage (in-memory)
	gateways := make(map[string]*contextforge.Gateway)
	var gatewayCounter int

	// POST /gateways - Create gateway
	// GET /gateways - List gateways
	mux.HandleFunc("/gateways", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			// Note: Gateways use unwrapped request body (unlike Tools/Resources/Servers)
			var gateway contextforge.Gateway
			if err := json.NewDecoder(r.Body).Decode(&gateway); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			gatewayCounter++
			id := fmt.Sprintf("gateway-%d", gatewayCounter)
			now := time.Now()

			gateway.ID = &id
			gateway.Enabled = true
			gateway.CreatedAt = &contextforge.Timestamp{Time: now}
			gateway.UpdatedAt = &contextforge.Timestamp{Time: now}

			gateways[id] = &gateway

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-RateLimit-Limit", "1000")
			w.Header().Set("X-RateLimit-Remaining", "995")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", now.Add(time.Hour).Unix()))

			// Return gateway directly (not wrapped)
			json.NewEncoder(w).Encode(&gateway)

		case http.MethodGet:
			query := r.URL.Query()
			result := []*contextforge.Gateway{}

			for _, gateway := range gateways {
				// Apply filters
				if query.Get("include_inactive") != "true" && !gateway.Enabled {
					continue
				}
				result = append(result, gateway)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		}
	})

	// GET /gateways/{id} - Get gateway
	// PUT /gateways/{id} - Update gateway
	// DELETE /gateways/{id} - Delete gateway
	mux.HandleFunc("/gateways/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 3 {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		gatewayID := parts[2]

		// Handle toggle endpoint
		if strings.Contains(r.URL.Path, "/toggle") {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			gateway, exists := gateways[gatewayID]
			if !exists {
				http.Error(w, `{"message":"Gateway not found"}`, http.StatusNotFound)
				return
			}

			// Extract activate parameter from query string
			activate := r.URL.Query().Get("activate") == "true"
			gateway.Enabled = activate
			gateway.UpdatedAt = &contextforge.Timestamp{Time: time.Now()}

			// Gateways toggle returns wrapped response like Tools
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"status":  "success",
				"message": "Gateway toggled successfully",
				"gateway": gateway,
			})
			return
		}

		switch r.Method {
		case http.MethodGet:
			gateway, exists := gateways[gatewayID]
			if !exists {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]any{
					"message": "Gateway not found",
				})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(gateway)

		case http.MethodPut:
			gateway, exists := gateways[gatewayID]
			if !exists {
				http.Error(w, `{"message":"Gateway not found"}`, http.StatusNotFound)
				return
			}

			// Note: Update is also unwrapped (consistent with Create)
			var update contextforge.Gateway
			if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// Update fields
			if update.Description != nil {
				gateway.Description = update.Description
			}
			if update.Tags != nil {
				gateway.Tags = update.Tags
			}
			gateway.UpdatedAt = &contextforge.Timestamp{Time: time.Now()}

			w.Header().Set("Content-Type", "application/json")
			// Return gateway directly (not wrapped)
			json.NewEncoder(w).Encode(gateway)

		case http.MethodDelete:
			if _, exists := gateways[gatewayID]; !exists {
				http.Error(w, `{"message":"Gateway not found"}`, http.StatusNotFound)
				return
			}

			delete(gateways, gatewayID)
			w.WriteHeader(http.StatusNoContent)
		}
	})
}
