// Package contextforge provides a Go client library for the IBM ContextForge MCP Gateway API.
//
// ContextForge is a feature-rich gateway, proxy, and MCP Registry that federates
// MCP and REST services. It acts as a unified endpoint for AI clients, consolidating
// discovery, authentication, rate-limiting, observability, and virtual server management.
// This client library provides an idiomatic Go interface to the ContextForge API,
// following architectural patterns established by popular Go libraries like google/go-github.
//
// # Features
//
// The SDK provides full CRUD operations for ContextForge resources:
//
//   - Manage tools with create, update, delete, and toggle operations
//   - Manage resources with URI-based access and template support
//   - Manage servers including virtual MCP servers
//   - Cursor-based pagination support
//   - Rate limit tracking from response headers
//   - Context support for all API calls
//   - Bearer token (JWT) authentication
//   - Comprehensive error handling
//
// # Authentication
//
// The ContextForge API uses Bearer token (JWT) authentication. You must provide
// a valid JWT token when creating the client:
//
//	client := contextforge.NewClient(nil, "your-jwt-token")
//
// # Usage
//
// Import the package:
//
//	import "github.com/leefowlercu/go-contextforge/contextforge"
//
// Create a new client:
//
//	client := contextforge.NewClient(nil, "your-jwt-token")
//
// You can provide a custom HTTP client for advanced configuration:
//
//	httpClient := &http.Client{
//		Timeout: 60 * time.Second,
//	}
//	client := contextforge.NewClient(httpClient, "your-jwt-token")
//
// List tools:
//
//	tools, resp, err := client.Tools.List(context.Background(), nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("Found %d tools\n", len(tools))
//
// List tools with filtering:
//
//	opts := &contextforge.ToolListOptions{
//		IncludeInactive: false,
//		Tags: "automation,api",
//		Visibility: "public",
//		ListOptions: contextforge.ListOptions{
//			Limit: 20,
//		},
//	}
//	tools, resp, err := client.Tools.List(context.Background(), opts)
//
// Get a specific tool:
//
//	tool, resp, err := client.Tools.Get(context.Background(), "tool-id")
//
// Create a new tool:
//
//	newTool := &contextforge.Tool{
//		Name: "my-tool",
//		Description: "A custom tool",
//		InputSchema: map[string]any{
//			"type": "object",
//			"properties": map[string]any{
//				"input": map[string]any{"type": "string"},
//			},
//		},
//		Active: true,
//	}
//	created, resp, err := client.Tools.Create(context.Background(), newTool)
//
// Update a tool:
//
//	tool.Description = "Updated description"
//	updated, resp, err := client.Tools.Update(context.Background(), "tool-id", tool)
//
// Toggle a tool's status:
//
//	toggled, resp, err := client.Tools.Toggle(context.Background(), "tool-id", true)
//
// Delete a tool:
//
//	resp, err := client.Tools.Delete(context.Background(), "tool-id")
//
// # Pagination
//
// The API uses cursor-based pagination. Use ListOptions to control pagination:
//
//	var allTools []*contextforge.Tool
//	opts := &contextforge.ToolListOptions{
//		ListOptions: contextforge.ListOptions{Limit: 50},
//	}
//
//	for {
//		tools, resp, err := client.Tools.List(context.Background(), opts)
//		if err != nil {
//			break
//		}
//		allTools = append(allTools, tools...)
//
//		if resp.NextCursor == "" {
//			break
//		}
//		opts.Cursor = resp.NextCursor
//	}
//
// # Error Handling
//
// The library provides structured error handling with custom error types:
//
//	tools, resp, err := client.Tools.List(context.Background(), nil)
//	if err != nil {
//		if rateLimitErr, ok := err.(*contextforge.RateLimitError); ok {
//			fmt.Printf("Rate limited. Reset at: %v\n", rateLimitErr.Rate.Reset)
//			return
//		}
//		if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
//			fmt.Printf("API error: %v\n", apiErr.Message)
//			return
//		}
//		log.Fatal(err)
//	}
//
// # Rate Limiting
//
// Rate limit information is tracked and available in response objects:
//
//	tools, resp, err := client.Tools.List(context.Background(), nil)
//	if err == nil && resp.Rate.Limit > 0 {
//		fmt.Printf("Rate Limit: %d/%d remaining\n",
//			resp.Rate.Remaining, resp.Rate.Limit)
//		fmt.Printf("Reset at: %v\n", resp.Rate.Reset)
//	}
//
// # Service Architecture
//
// The client follows a service-oriented architecture where different API
// endpoints are organized into service structs:
//
//	// Available services
//	client.Tools      // Tool-related operations
//	client.Resources  // Resource-related operations
//	client.Servers    // Server-related operations (planned)
//	client.Gateways   // Gateway-related operations (planned)
//	client.Prompts    // Prompt-related operations (planned)
//
// Each service provides methods for different operations:
//
//	// ToolsService methods
//	List(ctx, opts) ([]*Tool, *Response, error)
//	Get(ctx, toolID) (*Tool, *Response, error)
//	Create(ctx, tool) (*Tool, *Response, error)
//	Update(ctx, toolID, tool) (*Tool, *Response, error)
//	Delete(ctx, toolID) (*Response, error)
//	Toggle(ctx, toolID, activate) (*Tool, *Response, error)
//
// # Helper Functions
//
// The package provides helper functions for working with pointer types,
// following Go API conventions:
//
//	contextforge.String("value")    // Returns *string
//	contextforge.Int(42)            // Returns *int
//	contextforge.Bool(true)         // Returns *bool
//	contextforge.Time(t)            // Returns *time.Time
//
//	contextforge.StringValue(ptr)   // Returns string value or ""
//	contextforge.IntValue(ptr)      // Returns int value or 0
//	contextforge.BoolValue(ptr)     // Returns bool value or false
//	contextforge.TimeValue(ptr)     // Returns time.Time value or zero time
//
// # See Also
//
// Related resources:
//
//   - ContextForge Repository: https://github.com/IBM/mcp-context-forge
//   - MCP Protocol: https://modelcontextprotocol.io/specification
package contextforge
