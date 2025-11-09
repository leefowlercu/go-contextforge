//go:build integration
// +build integration

package integration

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestMockMCPServer_GETEndpoint(t *testing.T) {
	skipIfNotIntegration(t)

	// Get mock server URL
	url := GetMockMCPServerURL()
	if url == "" {
		t.Fatal("Mock server URL is empty")
	}

	t.Logf("Testing mock server at: %s", url)

	// Test GET request
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to GET mock server: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("Response status: %d", resp.StatusCode)
	t.Logf("Response headers: %+v", resp.Header)

	// Check for Mcp-Session-Id header
	sessionID := resp.Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		t.Error("Missing Mcp-Session-Id header")
	} else {
		t.Logf("Mcp-Session-Id: %s", sessionID)
	}

	// Check for application/json content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected application/json content type, got: %s", contentType)
	}

	// Read and log response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	t.Logf("Response body: %s", string(body))

	// Check status code is 200
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", resp.StatusCode)
	}
}
