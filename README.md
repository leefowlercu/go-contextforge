# go-contextforge

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.25-blue.svg)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A Go SDK for the [IBM ContextForge MCP Gateway](https://github.com/IBM/mcp-context-forge) - a feature-rich gateway, proxy, and MCP Registry that federates MCP and REST services.

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Usage Guide](#usage-guide)
  - [Client Configuration](#client-configuration)
  - [Pointer Helpers](#pointer-helpers)
  - [Managing Tools](#managing-tools)
  - [Managing Resources](#managing-resources)
  - [Managing Gateways](#managing-gateways)
  - [Managing Servers](#managing-servers)
  - [Managing Prompts](#managing-prompts)
  - [Pagination](#pagination)
  - [Error Handling](#error-handling)
- [API Methods Reference](#api-methods-reference)
  - [Tools Service](#tools-service)
  - [Resources Service](#resources-service)
  - [Gateways Service](#gateways-service)
  - [Servers Service](#servers-service)
  - [Prompts Service](#prompts-service)
- [Development](#development)
- [Releasing](#releasing)
- [Architecture](#architecture)
- [Known Issues](#known-issues)
- [Links](#links)
- [License](#license)

## Overview

ContextForge is a unified endpoint for AI clients that consolidates discovery, authentication, rate-limiting, observability, and virtual server management. It's a fully compliant MCP server that supports multi-cluster Kubernetes environments with Redis-backed federation.

This Go SDK provides an idiomatic interface to the ContextForge API, allowing you to:

- **Manage tools** with create, update, delete, and toggle operations
- **Manage resources** with URI-based access and template support
- **Manage gateways** for MCP server federation and proxying
- **Manage servers** with CRUD operations and association endpoints for tools, resources, and prompts
- **Manage prompts** with template-based AI interactions and argument support
- **Handle pagination** with cursor-based navigation
- **Track rate limits** and handle API errors gracefully
- **Authenticate** using Bearer token (JWT) authentication

## Installation

```bash
go get github.com/leefowlercu/go-contextforge
```

**Requirements:** Go 1.25.3 or later

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/leefowlercu/go-contextforge/contextforge"
)

func main() {
    // Create a client with bearer token
    client := contextforge.NewClient(nil, "your-jwt-token")
    ctx := context.Background()

    // List tools
    tools, _, err := client.Tools.List(ctx, &contextforge.ToolListOptions{
        ListOptions: contextforge.ListOptions{Limit: 10},
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Found %d tools:\n", len(tools))
    for _, tool := range tools {
        fmt.Printf("- %s: %s\n", tool.Name, *tool.Description)
    }

    // Get a specific tool
    tool, _, err := client.Tools.Get(ctx, "tool-id")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("\nTool details: %s (Enabled: %v)\n", tool.Name, tool.Enabled)
}
```

## Usage Guide

### Client Configuration

```go
import (
    "net/http"
    "time"
    "github.com/leefowlercu/go-contextforge/contextforge"
)

// Client with default base URL (http://localhost:8000/)
client := contextforge.NewClient(nil, "your-jwt-token")

// Client with custom base URL
client, err := contextforge.NewClientWithBaseURL(nil, "https://contextforge.example.com/", "your-jwt-token")
if err != nil {
    log.Fatal(err)
}

// Custom HTTP client with timeout
httpClient := &http.Client{
    Timeout: 60 * time.Second,
}
client = contextforge.NewClient(httpClient, "your-jwt-token")

// Custom HTTP client with custom base URL
client, err = contextforge.NewClientWithBaseURL(httpClient, "https://contextforge.example.com/", "your-jwt-token")
if err != nil {
    log.Fatal(err)
}

// Note: NewClientWithBaseURL automatically adds trailing slash if missing
```

### Pointer Helpers

The SDK provides helper functions for working with optional fields:

```go
// Create pointers from values
name := contextforge.String("my-tool")
limit := contextforge.Int(10)
enabled := contextforge.Bool(true)

// Extract values from pointers (with zero-value fallback)
nameStr := contextforge.StringValue(name)      // "my-tool"
limitInt := contextforge.IntValue(nil)         // 0
enabledBool := contextforge.BoolValue(enabled) // true
```

### Managing Tools

```go
ctx := context.Background()

// List tools with filtering
opts := &contextforge.ToolListOptions{
    IncludeInactive: false,
    Tags: "automation,api",
    Visibility: "public",
    ListOptions: contextforge.ListOptions{
        Limit: 20,
    },
}
tools, resp, err := client.Tools.List(ctx, opts)

// Get tool by ID
tool, _, err := client.Tools.Get(ctx, "tool-id")

// Create a new tool
newTool := &contextforge.Tool{
    Name: "my-tool",
    Description: contextforge.String("A custom tool"),
    InputSchema: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "input": map[string]any{"type": "string"},
        },
    },
    Enabled: true,
}

// Create with optional team/visibility settings
createOpts := &contextforge.ToolCreateOptions{
    TeamID:     contextforge.String("team-123"),
    Visibility: contextforge.String("public"),
}
created, _, err := client.Tools.Create(ctx, newTool, createOpts)

// Create without options
created, _, err = client.Tools.Create(ctx, newTool, nil)

// Update tool
tool.Description = contextforge.String("Updated description")
updated, _, err := client.Tools.Update(ctx, "tool-id", tool)

// Toggle tool status
toggled, _, err := client.Tools.Toggle(ctx, "tool-id", true) // activate

// Delete tool
_, err = client.Tools.Delete(ctx, "tool-id")
```

### Managing Resources

Resources have different types for different operations due to API field naming conventions:

- **ResourceCreate**: For creating resources (uses snake_case: `mime_type`)
- **Resource**: For reading resources (uses camelCase: `mimeType`)
- **ResourceUpdate**: For updating resources (uses camelCase: `mimeType`)

```go
ctx := context.Background()

// List resources
resources, _, err := client.Resources.List(ctx, nil)

// Create a resource
newResource := &contextforge.ResourceCreate{
    URI:         "file:///path/to/resource",
    Name:        "my-resource",
    Content:     "Resource content here",
    Description: contextforge.String("A custom resource"),
    MimeType:    contextforge.String("text/plain"), // Note: snake_case for Create
    Tags:        []string{"documentation", "example"},
}

// Create with optional team/visibility settings
createOpts := &contextforge.ResourceCreateOptions{
    TeamID:     contextforge.String("team-123"),
    Visibility: contextforge.String("public"),
}
created, _, err := client.Resources.Create(ctx, newResource, createOpts)

// Create without options
created, _, err = client.Resources.Create(ctx, newResource, nil)

// Update resource (uses camelCase fields)
update := &contextforge.ResourceUpdate{
    Description: contextforge.String("Updated description"),
    MimeType:    contextforge.String("text/markdown"), // Note: camelCase for Update
    Tags:        []string{"updated", "documentation"},
}
updated, _, err := client.Resources.Update(ctx, "resource-id", update)

// Toggle resource status
toggled, _, err := client.Resources.Toggle(ctx, "resource-id", false) // deactivate

// List available templates
templates, _, err := client.Resources.ListTemplates(ctx)
for _, template := range templates.Templates {
    fmt.Printf("Template: %s\n", template.Name)
}

// Delete resource
_, err = client.Resources.Delete(ctx, "resource-id")
```

### Managing Gateways

Gateways enable federation and proxying of MCP servers:

```go
ctx := context.Background()

// List gateways
gateways, _, err := client.Gateways.List(ctx, nil)

// Create a gateway
newGateway := &contextforge.Gateway{
    Name:        "my-gateway",
    URL:         "http://mcp-server.example.com",
    Description: contextforge.String("Proxy to external MCP server"),
    Transport:   "STREAMABLEHTTP",
    AuthType:    contextforge.String("bearer"),
    AuthToken:   contextforge.String("server-token"),
}

// Create with optional team/visibility settings
createOpts := &contextforge.GatewayCreateOptions{
    TeamID:     contextforge.String("team-123"),
    Visibility: contextforge.String("public"),
}
created, _, err := client.Gateways.Create(ctx, newGateway, createOpts)

// Create without options
created, _, err = client.Gateways.Create(ctx, newGateway, nil)

// Get gateway by ID
gateway, _, err := client.Gateways.Get(ctx, "gateway-id")

// Update gateway
gateway.Description = contextforge.String("Updated gateway description")
updated, _, err := client.Gateways.Update(ctx, "gateway-id", gateway)

// Toggle gateway status
toggled, _, err := client.Gateways.Toggle(ctx, "gateway-id", true) // activate

// Delete gateway
_, err = client.Gateways.Delete(ctx, "gateway-id")
```

### Managing Servers

Servers represent MCP server instances managed by ContextForge:

```go
ctx := context.Background()

// List servers
servers, _, err := client.Servers.List(ctx, nil)

// Create a server
newServer := &contextforge.Server{
    Name:        "my-server",
    Command:     "/usr/bin/my-mcp-server",
    Description: contextforge.String("Custom MCP server"),
    Args:        []string{"--port", "8080"},
    Env: map[string]string{
        "API_KEY": "secret-key",
    },
    Enabled: true,
}

// Create with optional team/visibility settings
createOpts := &contextforge.ServerCreateOptions{
    TeamID:     contextforge.String("team-123"),
    Visibility: contextforge.String("public"),
}
created, _, err := client.Servers.Create(ctx, newServer, createOpts)

// Create without options
created, _, err = client.Servers.Create(ctx, newServer, nil)

// Get server by ID
server, _, err := client.Servers.Get(ctx, "server-id")

// Update server
update := &contextforge.ServerUpdate{
    Description: contextforge.String("Updated server description"),
    Enabled:     contextforge.Bool(true),
}
updated, _, err := client.Servers.Update(ctx, "server-id", update)

// Toggle server status
toggled, _, err := client.Servers.Toggle(ctx, "server-id", true) // activate

// List server's tools
tools, _, err := client.Servers.ListTools(ctx, "server-id", nil)

// List server's resources
resources, _, err := client.Servers.ListResources(ctx, "server-id", nil)

// List server's prompts
prompts, _, err := client.Servers.ListPrompts(ctx, "server-id", nil)

// Delete server
_, err = client.Servers.Delete(ctx, "server-id")
```

**Note:** The ServersService excludes MCP protocol communication endpoints (`GET /servers/{id}/sse` and `POST /servers/{id}/message`). These are for MCP protocol communication, not REST API management.

### Managing Prompts

Prompts provide templated interactions for AI models:

```go
ctx := context.Background()

// List prompts
prompts, _, err := client.Prompts.List(ctx, nil)

// List with filtering
opts := &contextforge.PromptListOptions{
    IncludeInactive: true,
    Tags:            "ai,code-review",
    TeamID:          "team-123",
}
prompts, _, err = client.Prompts.List(ctx, opts)

// Create a prompt
newPrompt := &contextforge.PromptCreate{
    Name:        "code-review",
    Description: contextforge.String("Code review prompt template"),
    Template:    "Please review this {{language}} code:\n\n{{code}}",
    Arguments: []contextforge.PromptArgument{
        {Name: "language", Description: contextforge.String("Programming language"), Required: true},
        {Name: "code", Description: contextforge.String("Code to review"), Required: true},
    },
    Tags: []string{"ai", "code-review"},
}

// Create with optional team/visibility settings
createOpts := &contextforge.PromptCreateOptions{
    TeamID:     contextforge.String("team-123"),
    Visibility: contextforge.String("public"),
}
created, _, err := client.Prompts.Create(ctx, newPrompt, createOpts)

// Update prompt
update := &contextforge.PromptUpdate{
    Description: contextforge.String("Updated description"),
    Template:    contextforge.String("Updated template: {{new_arg}}"),
}
updated, _, err := client.Prompts.Update(ctx, 123, update)

// Toggle prompt status
toggled, _, err := client.Prompts.Toggle(ctx, 123, true) // activate
toggled, _, err = client.Prompts.Toggle(ctx, 123, false) // deactivate

// Delete prompt
_, err = client.Prompts.Delete(ctx, 123)
```

**Note:** The PromptsService excludes MCP client endpoints (`POST /prompts/{id}` for rendered prompts). These are for MCP client communication, not REST API management.

### Pagination

```go
opts := &contextforge.ToolListOptions{
    ListOptions: contextforge.ListOptions{Limit: 50},
}

for {
    tools, resp, err := client.Tools.List(ctx, opts)
    if err != nil {
        break
    }

    // Process tools
    for _, tool := range tools {
        fmt.Printf("Tool: %s\n", tool.Name)
    }

    // Check for more pages
    if resp.NextCursor == "" {
        break
    }
    opts.Cursor = resp.NextCursor
}
```

### Error Handling

```go
tools, resp, err := client.Tools.List(ctx, nil)
if err != nil {
    // Check for rate limiting
    if rateLimitErr, ok := err.(*contextforge.RateLimitError); ok {
        fmt.Printf("Rate limited. Reset at: %v\n", rateLimitErr.Rate.Reset)
        return
    }

    // Check for API errors
    if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
        fmt.Printf("API error: %v\n", apiErr.Message)
        return
    }

    log.Fatal(err)
}

// Check rate limit info
if resp.Rate.Limit > 0 {
    fmt.Printf("Rate limit: %d/%d remaining\n", resp.Rate.Remaining, resp.Rate.Limit)
}
```

## API Methods Reference

### Tools Service

| Method | Description |
|--------|-------------|
| `List(ctx, opts)` | List tools with pagination and filtering |
| `Get(ctx, toolID)` | Get tool by ID |
| `Create(ctx, tool, opts)` | Create a new tool with optional settings |
| `Update(ctx, toolID, tool)` | Update tool |
| `Delete(ctx, toolID)` | Delete tool |
| `Toggle(ctx, toolID, activate)` | Toggle tool enabled status |

### Resources Service

| Method | Description |
|--------|-------------|
| `List(ctx, opts)` | List resources with pagination and filtering |
| `Create(ctx, resource, opts)` | Create a new resource with optional settings |
| `Update(ctx, resourceID, resource)` | Update resource |
| `Delete(ctx, resourceID)` | Delete resource |
| `Toggle(ctx, resourceID, activate)` | Toggle resource active status |
| `ListTemplates(ctx)` | List available resource templates |

### Gateways Service

| Method | Description |
|--------|-------------|
| `List(ctx, opts)` | List gateways with pagination and filtering |
| `Get(ctx, gatewayID)` | Get gateway by ID |
| `Create(ctx, gateway, opts)` | Create a new gateway with optional settings |
| `Update(ctx, gatewayID, gateway)` | Update gateway |
| `Delete(ctx, gatewayID)` | Delete gateway |
| `Toggle(ctx, gatewayID, activate)` | Toggle gateway active status |

### Servers Service

| Method | Description |
|--------|-------------|
| `List(ctx, opts)` | List servers with pagination and filtering |
| `Get(ctx, serverID)` | Get server by ID |
| `Create(ctx, server, opts)` | Create a new server with optional settings |
| `Update(ctx, serverID, server)` | Update server |
| `Delete(ctx, serverID)` | Delete server |
| `Toggle(ctx, serverID, activate)` | Toggle server enabled status |
| `ListTools(ctx, serverID, opts)` | List tools associated with a server |
| `ListResources(ctx, serverID, opts)` | List resources associated with a server |
| `ListPrompts(ctx, serverID, opts)` | List prompts associated with a server |

### Prompts Service

| Method | Description |
|--------|-------------|
| `List(ctx, opts)` | List prompts with pagination and filtering |
| `Create(ctx, prompt, opts)` | Create a new prompt with optional settings |
| `Update(ctx, promptID, prompt)` | Update prompt |
| `Delete(ctx, promptID)` | Delete prompt |
| `Toggle(ctx, promptID, activate)` | Toggle prompt active status |

## Development

### Running Tests

```bash
# Unit tests
make test
# or
go test ./...

# Unit tests with coverage
make test-cover

# Integration tests (requires ContextForge running)
make integration-test-setup  # Start ContextForge gateway
make integration-test        # Run integration tests
make integration-test-teardown  # Stop gateway

# Full integration test cycle
make integration-test-all

# Run both unit and integration tests
make test-all

# Generate HTML coverage report
make coverage
```

### Integration Test Configuration

Integration tests require environment variables:

```bash
# Required to enable integration tests
export INTEGRATION_TESTS=true

# Optional configuration (defaults shown)
export CONTEXTFORGE_BASE_URL="http://localhost:8000/"
export CONTEXTFORGE_ADMIN_EMAIL="admin@test.local"
export CONTEXTFORGE_ADMIN_PASSWORD="testpassword123"
```

### Building

```bash
# Build all packages
make build

# Build with formatting and linting
make check

# Format code
make fmt

# Lint
make vet

# Full CI pipeline
make ci
```

### Available Make Targets

**Development:**
- `make deps` - Download dependencies
- `make fmt` - Format code with gofmt
- `make vet` - Run go vet
- `make lint` - Format and vet
- `make test` - Run unit tests
- `make test-verbose` - Run unit tests with verbose output
- `make test-cover` - Run unit tests with coverage
- `make build` - Build all packages
- `make clean` - Clean build artifacts
- `make coverage` - Generate HTML coverage report
- `make ci` - Full CI pipeline (deps, lint, test, build)

**Testing:**
- `make integration-test-setup` - Start ContextForge gateway
- `make integration-test` - Run integration tests
- `make integration-test-teardown` - Stop gateway
- `make integration-test-all` - Full integration test cycle
- `make test-all` - Run both unit and integration tests

**Releasing:**
- `make release-patch` - Prepare patch release (auto-increment patch version)
- `make release-minor` - Prepare minor release (auto-increment minor version)
- `make release-major` - Prepare major release (auto-increment major version)
- `make release-prep VERSION=vX.Y.Z` - Prepare release with specific version
- `make release` - Full release preparation workflow
- `make release-check` - Verify release prerequisites

## Releasing

This project uses semantic versioning and includes automated release tooling to streamline the release process.

### Semantic Version Bumping

The release workflow supports automatic semantic version bumping:

```bash
# Patch release (0.1.0 → 0.1.1) - bug fixes, no API changes
make release-patch

# Minor release (0.1.0 → 0.2.0) - new features, backward compatible
make release-minor

# Major release (0.1.0 → 1.0.0) - breaking changes
make release-major
```

### Manual Version Override

You can also specify a version manually:

```bash
make release-prep VERSION=v0.2.5
```

### Release Workflow

Each release command performs the following steps:

1. **Prerequisites check**: Ensures git working directory is clean
2. **Version calculation**: Determines new version based on current version in `contextforge/version.go`
3. **Update version constant**: Updates `contextforge/version.go` with new version
4. **Update changelog**: Updates `CHANGELOG.md` with release date
5. **Create commit**: Commits changes with message `release: prepare vX.Y.Z`
6. **Create tag**: Creates annotated git tag for the release
7. **Display instructions**: Shows commands to push changes and create GitHub release

### Pushing Releases

After running a release command, push the changes:

```bash
# Push commit and tag
git push && git push --tags

# Then create a GitHub release at:
# https://github.com/leefowlercu/go-contextforge/releases/new?tag=vX.Y.Z
```

### Version Management

- **SDK Version**: Defined in `contextforge/version.go` as `Version` constant
- **User Agent**: Automatically includes SDK version (`go-contextforge/vX.Y.Z`)
- **Changelog**: Maintained in `CHANGELOG.md` following [Keep a Changelog](https://keepachangelog.com/) format
- **Git Tags**: Use format `vX.Y.Z` (semantic versioning with `v` prefix)

### Undoing a Release Preparation

If you need to undo a release preparation (before pushing):

```bash
# Remove the tag
git tag -d vX.Y.Z

# Reset the commit
git reset --hard HEAD~1
```

## Architecture

This SDK follows the service-oriented architecture pattern established by [google/go-github](https://github.com/google/go-github), organizing API endpoints into logical service groups:

- **Client** - Main entry point with HTTP client management and JWT authentication
- **ToolsService** - All tool-related operations
- **ResourcesService** - All resource-related operations
- **GatewaysService** - All gateway-related operations
- **ServersService** - All server-related operations and associations
- **PromptsService** - All prompt management operations

### Custom Types

- **FlexibleID** - Handles API inconsistencies where IDs may be returned as integers or strings
- **Timestamp** - Custom timestamp parsing for API responses without timezone information
- **Pointer helpers** - `String()`, `Int()`, `Bool()`, `Time()` for working with optional fields

## Links

- **ContextForge Repository:** https://github.com/IBM/mcp-context-forge
- **MCP Protocol:** https://modelcontextprotocol.io/

## Known Issues

### Upstream ContextForge API Bugs

The SDK integration tests have identified three bugs in ContextForge v0.8.0 that affect the Prompts API. These bugs are in the upstream API, not the SDK implementation. The affected integration tests are currently skipped and will be re-enabled once the upstream bugs are fixed.

**CONTEXTFORGE-001: Prompts Toggle Returns Stale State**
The `POST /prompts/{id}/toggle` endpoint returns stale `isActive` state in responses despite correctly updating the database. See [`docs/upstream-bugs/prompt-toggle.md`](docs/upstream-bugs/prompt-toggle.md) for details.

**CONTEXTFORGE-002: Prompts API Accepts Empty Template Field**
The `POST /prompts` endpoint accepts prompt creation without the `template` field, allowing semantically invalid prompts. See [`docs/upstream-bugs/prompt-validation-missing-template.md`](docs/upstream-bugs/prompt-validation-missing-template.md) for details.

**CONTEXTFORGE-003: Prompts Toggle Returns 400 Instead of 404**
The `POST /prompts/{id}/toggle` endpoint returns HTTP 400 for non-existent prompts instead of the expected 404. See [`docs/upstream-bugs/prompt-toggle-error-code.md`](docs/upstream-bugs/prompt-toggle-error-code.md) for details.

All bug reports include root cause analysis, proposed solutions, and workarounds for SDK users.

## License

MIT License - see [LICENSE](LICENSE) file for details.
