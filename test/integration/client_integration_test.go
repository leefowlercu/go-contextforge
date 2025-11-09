//go:build integration
// +build integration

package integration

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/leefowlercu/go-contextforge/contextforge"
)

func TestClient_Authentication(t *testing.T) {
	skipIfNotIntegration(t)

	t.Run("successful login and token usage", func(t *testing.T) {
		token := getTestToken(t)
		if token == "" {
			t.Fatal("Expected non-empty JWT token")
		}

		client, err := contextforge.NewClient(nil, getBaseURL(), token)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		// Test that the token works by making a simple API call
		ctx := context.Background()
		_, _, err = client.Tools.List(ctx, nil)
		if err != nil {
			t.Errorf("Expected successful API call with valid token, got error: %v", err)
		}
	})

	t.Run("invalid credentials", func(t *testing.T) {
		// This test verifies that invalid credentials are rejected
		// We won't test this extensively as it's covered by the auth system
		t.Skip("Skipping invalid credentials test - covered by auth system tests")
	})

	t.Run("request without token", func(t *testing.T) {
		client, err := contextforge.NewClient(nil, getBaseURL(), "") // No token
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		ctx := context.Background()
		_, _, err = client.Tools.List(ctx, nil)
		if err == nil {
			t.Error("Expected error when making request without token")
		}

		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
			if apiErr.Response.StatusCode != http.StatusUnauthorized {
				t.Errorf("Expected 401 Unauthorized, got %d", apiErr.Response.StatusCode)
			}
			t.Logf("Correctly received error: %v", err)
		} else {
			t.Errorf("Expected ErrorResponse, got %T: %v", err, err)
		}
	})

	t.Run("request with invalid token", func(t *testing.T) {
		client, err := contextforge.NewClient(nil, getBaseURL(), "invalid-token-12345")
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		ctx := context.Background()
		_, _, err = client.Tools.List(ctx, nil)
		if err == nil {
			t.Error("Expected error when making request with invalid token")
		}

		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
			if apiErr.Response.StatusCode != http.StatusUnauthorized {
				t.Errorf("Expected 401 Unauthorized, got %d", apiErr.Response.StatusCode)
			}
			t.Logf("Correctly received error: %v", err)
		} else {
			t.Errorf("Expected ErrorResponse, got %T: %v", err, err)
		}
	})
}

func TestClient_RequestResponse(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)

	t.Run("bearer token injection", func(t *testing.T) {
		// Create a custom request and verify Authorization header
		req, err := client.NewRequest("GET", "tools", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		authHeader := req.Header.Get("Authorization")
		if authHeader == "" {
			t.Error("Expected Authorization header to be set")
		}

		if len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			t.Errorf("Expected Authorization header to start with 'Bearer ', got: %s", authHeader)
		}

		t.Logf("Authorization header correctly set: %s...", authHeader[:20])
	})

	t.Run("user agent header", func(t *testing.T) {
		req, err := client.NewRequest("GET", "tools", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		userAgent := req.Header.Get("User-Agent")
		if userAgent == "" {
			t.Error("Expected User-Agent header to be set")
		}

		if userAgent != client.UserAgent {
			t.Errorf("Expected User-Agent %q, got %q", client.UserAgent, userAgent)
		}

		t.Logf("User-Agent header correctly set: %s", userAgent)
	})

	t.Run("json content type", func(t *testing.T) {
		tool := minimalToolInput()
		req, err := client.NewRequest("POST", "tools", tool)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		contentType := req.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got %q", contentType)
		}

		accept := req.Header.Get("Accept")
		if accept != "application/json" {
			t.Errorf("Expected Accept 'application/json', got %q", accept)
		}

		t.Logf("Content-Type and Accept headers correctly set")
	})
}

func TestClient_Pagination(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	// Create multiple test tools to ensure pagination
	t.Logf("Creating test tools for pagination test...")
	for i := 0; i < 5; i++ {
		createTestTool(t, client, randomToolName())
	}

	t.Run("cursor extraction from response", func(t *testing.T) {
		opts := &contextforge.ToolListOptions{
			ListOptions: contextforge.ListOptions{
				Limit: 2, // Small limit to force pagination
			},
		}

		tools, resp, err := client.Tools.List(ctx, opts)
		if err != nil {
			t.Fatalf("Failed to list tools: %v", err)
		}

		t.Logf("First page: got %d tools", len(tools))

		// If there are more than 2 tools total, we should have a cursor
		if len(tools) == 2 && resp.NextCursor == "" {
			t.Log("Warning: Expected NextCursor but got empty string (may indicate < 3 total tools)")
		}

		if resp.NextCursor != "" {
			t.Logf("NextCursor correctly extracted: %s", resp.NextCursor)

			// Try to fetch next page
			opts.Cursor = resp.NextCursor
			tools2, resp2, err := client.Tools.List(ctx, opts)
			if err != nil {
				t.Fatalf("Failed to fetch next page: %v", err)
			}

			t.Logf("Second page: got %d tools", len(tools2))

			// Verify we got different tools
			if len(tools) > 0 && len(tools2) > 0 {
				if tools[0].ID == tools2[0].ID {
					t.Error("Expected different tools on second page")
				}
			}

			t.Logf("Pagination working correctly, NextCursor for page 2: %s", resp2.NextCursor)
		}
	})
}

func TestClient_RateLimiting(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("rate limit header parsing", func(t *testing.T) {
		_, resp, err := client.Tools.List(ctx, nil)
		if err != nil {
			t.Fatalf("Failed to list tools: %v", err)
		}

		// Check if rate limit headers are present and parsed
		if resp.Rate.Limit > 0 {
			t.Logf("Rate limit detected: %d/%d remaining, resets at %v",
				resp.Rate.Remaining, resp.Rate.Limit, resp.Rate.Reset)

			if resp.Rate.Remaining > resp.Rate.Limit {
				t.Errorf("Remaining (%d) should not exceed Limit (%d)",
					resp.Rate.Remaining, resp.Rate.Limit)
			}

			if !resp.Rate.Reset.IsZero() && resp.Rate.Reset.Before(time.Now()) {
				t.Error("Rate limit reset time should be in the future")
			}
		} else {
			t.Log("No rate limiting detected (headers not present or rate limiting disabled)")
		}
	})

	t.Run("rate limit tracking per endpoint", func(t *testing.T) {
		// Make multiple requests to track rate limit changes
		var rates []contextforge.Rate

		for i := 0; i < 3; i++ {
			_, resp, err := client.Tools.List(ctx, nil)
			if err != nil {
				t.Fatalf("Request %d failed: %v", i+1, err)
			}

			if resp.Rate.Limit > 0 {
				rates = append(rates, resp.Rate)
				t.Logf("Request %d: %d/%d remaining",
					i+1, resp.Rate.Remaining, resp.Rate.Limit)
			}
		}

		if len(rates) >= 2 {
			// Verify remaining count is decreasing (or staying same if unlimited)
			if rates[0].Remaining > 0 && rates[1].Remaining > rates[0].Remaining {
				t.Error("Rate limit remaining should decrease or stay the same with each request")
			}
		} else {
			t.Log("Rate limiting not enabled for testing")
		}
	})
}

func TestClient_ErrorHandling(t *testing.T) {
	skipIfNotIntegration(t)

	client := setupClient(t)
	ctx := context.Background()

	t.Run("404 not found", func(t *testing.T) {
		nonExistentID := "non-existent-tool-id-12345"
		_, _, err := client.Tools.Get(ctx, nonExistentID)
		if err == nil {
			t.Error("Expected error for non-existent tool")
		}

		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
			if apiErr.Response.StatusCode != http.StatusNotFound {
				t.Errorf("Expected 404 Not Found, got %d", apiErr.Response.StatusCode)
			}
			t.Logf("Correctly received 404 error: %v", err)
		} else {
			t.Errorf("Expected ErrorResponse, got %T: %v", err, err)
		}
	})

	t.Run("validation error", func(t *testing.T) {
		invalidTool := &contextforge.Tool{
			// Missing required fields
			Name:        "",
			Description: contextforge.String(""),
			InputSchema: nil,
		}

		_, _, err := client.Tools.Create(ctx, invalidTool, nil)
		if err == nil {
			t.Error("Expected validation error for invalid tool")
		}

		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
			// Typically 400 or 422 for validation errors
			if apiErr.Response.StatusCode != http.StatusBadRequest &&
				apiErr.Response.StatusCode != http.StatusUnprocessableEntity {
				t.Logf("Got status %d for validation error (may vary by implementation)",
					apiErr.Response.StatusCode)
			}
			t.Logf("Correctly received validation error: %v", err)
		} else {
			t.Logf("Got error (type %T): %v", err, err)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, _, err := client.Tools.List(cancelCtx, nil)
		if err == nil {
			t.Error("Expected error for cancelled context")
		}

		if err != context.Canceled {
			t.Logf("Got error (expected context.Canceled): %v", err)
		} else {
			t.Logf("Correctly received context cancellation error")
		}
	})

	t.Run("context timeout", func(t *testing.T) {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		time.Sleep(10 * time.Millisecond) // Ensure timeout occurs

		_, _, err := client.Tools.List(timeoutCtx, nil)
		if err == nil {
			t.Error("Expected error for timed out context")
		}

		t.Logf("Correctly received timeout error: %v", err)
	})
}
