//go:build integration
// +build integration

package integration

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
)

// mockMCPServer is the global mock MCP server instance
var mockMCPServer *MockMCPServer

// GetMockMCPServerURL returns the URL of the mock MCP server
func GetMockMCPServerURL() string {
	if mockMCPServer == nil {
		return ""
	}
	return mockMCPServer.URL
}

// InitMockServer initializes the global mock MCP server
func InitMockServer() {
	if mockMCPServer == nil {
		mockMCPServer = NewMockMCPServer()
	}
}

// CloseMockServer closes the global mock MCP server
func CloseMockServer() {
	if mockMCPServer != nil {
		mockMCPServer.Close()
		mockMCPServer = nil
	}
}

// generateRandomID generates a random hex ID for session tracking
func generateRandomID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// MockMCPServer represents a minimal MCP server conforming to spec 2025-06-18
type MockMCPServer struct {
	server *httptest.Server
	URL    string
}

// MCPRequest represents an incoming JSON-RPC request
type MCPRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id,omitempty"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params,omitempty"`
}

// MCPResponse represents a JSON-RPC response
type MCPResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

// MCPInitializeResult represents the result field in initialize response per spec 2025-06-18
type MCPInitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    MCPServerCapabilities  `json:"capabilities"`
	ServerInfo      MCPServerInfo          `json:"serverInfo"`
	Instructions    string                 `json:"instructions,omitempty"`
}

// MCPServerCapabilities represents server capabilities per spec
type MCPServerCapabilities struct {
	Logging     map[string]any `json:"logging,omitempty"`
	Prompts     map[string]any `json:"prompts,omitempty"`
	Resources   map[string]any `json:"resources,omitempty"`
	Tools       map[string]any `json:"tools,omitempty"`
	Completions map[string]any `json:"completions,omitempty"`
}

// MCPServerInfo represents server information
type MCPServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// NewMockMCPServer creates and starts a new mock MCP server
func NewMockMCPServer() *MockMCPServer {
	mock := &MockMCPServer{}

	mux := http.NewServeMux()

	// Main MCP endpoint
	mux.HandleFunc("/", mock.handleMCPRequest)

	// Optional health endpoint
	mux.HandleFunc("/health", mock.handleHealth)

	// Use httptest.NewServer which handles all the startup synchronization
	mock.server = httptest.NewServer(mux)
	mock.URL = mock.server.URL

	return mock
}

// handleMCPRequest handles MCP JSON-RPC requests per spec 2025-06-18
func (m *MockMCPServer) handleMCPRequest(w http.ResponseWriter, r *http.Request) {
	// Per spec: Single endpoint supports both POST (for messages) and GET (for SSE stream)

	if r.Method == http.MethodPost {
		// Handle JSON-RPC messages
		m.handlePOSTRequest(w, r)
		return
	}

	if r.Method == http.MethodGet {
		// Handle GET requests for session establishment and SSE
		m.handleGETRequest(w, r)
		return
	}

	// Other methods not allowed
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handlePOSTRequest handles POST requests containing JSON-RPC messages
func (m *MockMCPServer) handlePOSTRequest(w http.ResponseWriter, r *http.Request) {
	// Read and decode the JSON-RPC request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var req MCPRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON-RPC request", http.StatusBadRequest)
		return
	}

	// Handle the initialize method
	if req.Method == "initialize" {
		m.handleInitialize(w, req)
		return
	}

	// Handle initialized notification (no response needed per JSON-RPC)
	if req.Method == "initialized" {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	// Handle tools/list method
	if req.Method == "tools/list" {
		m.handleListTools(w, req)
		return
	}

	// Handle resources/list method
	if req.Method == "resources/list" {
		m.handleListResources(w, req)
		return
	}

	// Handle prompts/list method
	if req.Method == "prompts/list" {
		m.handleListPrompts(w, req)
		return
	}

	// For other methods, return a simple success response
	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]any{},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGETRequest handles GET requests for session establishment and optional SSE
func (m *MockMCPServer) handleGETRequest(w http.ResponseWriter, r *http.Request) {
	// Generate a session ID for this connection
	sessionID := "mock-session-" + generateRandomID()
	acceptHeader := r.Header.Get("Accept")

	// Support both SSE and regular JSON responses based on Accept header
	if strings.Contains(acceptHeader, "text/event-stream") {
		// SSE mode for server-initiated messages
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Mcp-Session-Id", sessionID)

		// Send initial connection message
		fmt.Fprintf(w, ": connected\n\n")

		// Flush the response
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		return
	}

	// Default JSON response for session establishment
	// This satisfies ContextForge gateway pre-flight validation
	w.Header().Set("Mcp-Session-Id", sessionID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "ready",
		"sessionId": sessionID,
	})
}

// handleInitialize handles the initialize request per spec 2025-06-18
func (m *MockMCPServer) handleInitialize(w http.ResponseWriter, req MCPRequest) {
	// Extract requested protocol version from params
	requestedVersion := "2025-06-18" // Default
	if params, ok := req.Params["protocolVersion"].(string); ok && params != "" {
		requestedVersion = params
	}

	// Create initialize result per spec
	result := MCPInitializeResult{
		ProtocolVersion: requestedVersion, // Echo back the version (or provide alternative if unsupported)
		Capabilities: MCPServerCapabilities{
			Logging: map[string]any{},
			Prompts: map[string]any{
				"listChanged": true,
			},
			Resources: map[string]any{
				"subscribe":   true,
				"listChanged": true,
			},
			Tools: map[string]any{
				"listChanged": true,
			},
		},
		ServerInfo: MCPServerInfo{
			Name:    "MockMCPServer",
			Version: "1.0.0",
		},
	}

	// Create JSON-RPC response
	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleListTools handles the tools/list request
func (m *MockMCPServer) handleListTools(w http.ResponseWriter, req MCPRequest) {
	result := map[string]any{
		"tools": []any{}, // Return empty tools list
	}

	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleListResources handles the resources/list request
func (m *MockMCPServer) handleListResources(w http.ResponseWriter, req MCPRequest) {
	result := map[string]any{
		"resources": []any{}, // Return empty resources list
	}

	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleListPrompts handles the prompts/list request
func (m *MockMCPServer) handleListPrompts(w http.ResponseWriter, req MCPRequest) {
	result := map[string]any{
		"prompts": []any{}, // Return empty prompts list
	}

	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleHealth handles health check requests (optional, not part of MCP spec)
func (m *MockMCPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Close stops the mock server
func (m *MockMCPServer) Close() {
	if m.server != nil {
		m.server.Close()
	}
}
