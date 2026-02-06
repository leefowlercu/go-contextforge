# go-contextforge

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.25.3-blue.svg)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A Go SDK for the [IBM ContextForge MCP Gateway](https://github.com/IBM/mcp-context-forge) - a feature-rich gateway, proxy, and MCP Registry that federates MCP and REST services.

## Table of Contents

- [Overview](#overview)
  - [Compatibility](#compatibility)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Usage Guide](#usage-guide)
  - [Client Configuration](#client-configuration)
  - [Pointer Helpers and Tags](#pointer-helpers-and-tags)
  - [Managing Tools](#managing-tools)
  - [Managing Resources](#managing-resources)
  - [Managing Gateways](#managing-gateways)
  - [Managing Servers](#managing-servers)
  - [Managing Prompts](#managing-prompts)
  - [Managing Agents](#managing-agents)
  - [Managing Teams](#managing-teams)
  - [Pagination](#pagination)
  - [Error Handling](#error-handling)
- [API Methods Reference](#api-methods-reference)
  - [Tools Service](#tools-service)
  - [Resources Service](#resources-service)
  - [Gateways Service](#gateways-service)
  - [Servers Service](#servers-service)
  - [Prompts Service](#prompts-service)
  - [Agents Service](#agents-service)
  - [Teams Service](#teams-service)
- [Examples](#examples)
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
- **Manage agents** with agent-to-agent (A2A) protocol support, invocation, and performance tracking
- **Handle pagination** with cursor-based or offset-based (skip/limit) navigation
- **Track rate limits** and handle API errors gracefully
- **Authenticate** using Bearer token (JWT) authentication

### MCP (Model Context Protocol)

The Model Context Protocol (MCP) is an open protocol that standardizes how AI systems access external data sources, tools, and resources. MCP enables AI models to securely interact with databases, APIs, file systems, and other services through a unified interface, making it easier to build context-aware AI applications.

ContextForge implements the MCP protocol as both a compliant MCP server and a federation gateway. It consolidates multiple MCP servers and REST services into a single endpoint, providing centralized authentication, rate limiting, observability, and resource management. This allows AI clients to discover and access tools, resources, and prompts from multiple sources through a single, unified API.

For more information about the MCP protocol, see the [official documentation](https://modelcontextprotocol.io/).

### A2A Protocol

The A2A (Agent-to-Agent) protocol is an open standard that enables communication and interoperability between independent AI agent systems. While MCP focuses on connecting AI models to tools and resources, A2A enables agents to discover, invoke, and collaborate with other agents, regardless of their underlying framework or implementation.

Through ContextForge, agents advertise their capabilities using Agent Cards (JSON-formatted capability descriptions), allowing other agents to discover and invoke the best agent for a specific task. A2A agents can receive invocations with parameters, return structured responses, and track performance metrics across interactions.

The SDK provides full management of A2A agents including creation, configuration, endpoint registration, invocation, and performance metrics tracking. This enables building multi-agent systems where specialized agents can collaborate through a standardized protocol.

For more information about the A2A protocol, see the [official specification](https://a2a-protocol.org/latest/specification/) and [project website](https://a2aprotocol.ai/).

### Compatibility

This SDK is tested against **ContextForge v1.0.0-BETA-2** (PyPI: `mcpgateway==1.0.0b2`).

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
    // Create a client with address and bearer token
    client, err := contextforge.NewClient(nil, "http://localhost:8000/", "your-jwt-token")
    if err != nil {
        log.Fatal(err)
    }
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
        desc := contextforge.StringValue(tool.Description) // Safe nil handling
        fmt.Printf("- %s: %s\n", tool.Name, desc)
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

// Create a client with locally hosted ContextForge instance
client, err := contextforge.NewClient(nil, "http://localhost:8000/", "your-jwt-token")
if err != nil {
    log.Fatal(err)
}

// Create a client with remote ContextForge instance
client, err := contextforge.NewClient(nil, "https://contextforge.example.com/", "your-jwt-token")
if err != nil {
    log.Fatal(err)
}

// Custom HTTP client with timeout
httpClient := &http.Client{
    Timeout: 60 * time.Second,
}
client, err = contextforge.NewClient(httpClient, "https://contextforge.example.com/", "your-jwt-token")
if err != nil {
    log.Fatal(err)
}

// Note: NewClient automatically adds trailing slash if missing
```

### Pointer Helpers and Tags

The SDK uses pointers and slices to distinguish between three states for optional fields:

1. **nil** - field not set (omitted from API request)
2. **pointer to zero value or empty slice** - field explicitly cleared
3. **pointer to value or populated slice** - field set to that value

This pattern (used by google/go-github, hashicorp/go-tfe, AWS SDK) enables partial updates where only changed fields are sent to the API.

**Helper functions:**

```go
// Creating pointers (for setting values)
name := contextforge.String("my-tool")
limit := contextforge.Int(10)
enabled := contextforge.Bool(true)
timeout := contextforge.Int64(3000)
weight := contextforge.Float64(0.95)
timestamp := contextforge.Time(time.Now())

// Extracting values (with zero-value fallback for nil)
nameStr := contextforge.StringValue(name)      // "my-tool"
limitInt := contextforge.IntValue(nil)         // 0
enabledBool := contextforge.BoolValue(enabled) // true
```

**Partial update examples:**

```go
// Update only the name (other fields unchanged)
update := &contextforge.ResourceUpdate{
    Name: contextforge.String("new-name"),
    // Description, Tags, etc. are nil and won't be sent
}

// Clear the description (set to empty string)
update := &contextforge.ResourceUpdate{
    Description: contextforge.String(""),
}

// Don't update tags vs clear all tags
update1 := &contextforge.ResourceUpdate{
    Tags: nil,        // Tags field omitted - existing tags unchanged
}
update2 := &contextforge.ResourceUpdate{
    Tags: []string{}, // Empty array sent - clears all tags
}
update3 := &contextforge.ResourceUpdate{
    Tags: []string{"new-tag"}, // Sets new tags
}
```

**Tag type handling:**

Tags have different types for input vs output due to API response format changes in v1.0.0:

- **Create/Update types** use `[]string` for input (e.g., `ResourceCreate.Tags`, `PromptCreate.Tags`)
- **Read types** return `[]Tag` structs with `ID` and `Label` fields (e.g., `Tool.Tags`, `Prompt.Tags`)

Helper functions for conversion:

```go
// Convert strings to Tag structs (for update operations on read types)
tags := contextforge.NewTags([]string{"tag1", "tag2"})

// Extract tag names from Tag structs
names := contextforge.TagNames(tool.Tags) // Returns []string{"tag1", "tag2"}
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

// Get resource content (hybrid REST endpoint returns MCP-compatible format)
content, _, err := client.Resources.Get(ctx, "resource-id")
if err == nil {
    fmt.Printf("Resource type: %s\n", content.Type)
    fmt.Printf("URI: %s\n", content.URI)
    if content.Text != nil {
        fmt.Printf("Text content: %s\n", *content.Text)
    }
    if content.Blob != nil {
        fmt.Printf("Blob content (base64): %s\n", *content.Blob)
    }
}

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
newServer := &contextforge.ServerCreate{
    Name:        "my-server",
    Description: contextforge.String("Custom MCP server"),
    Icon:        contextforge.String("server"),
    Tags:        []string{"mcp", "production"},

    // Optional: Associate with existing resources
    AssociatedTools:     []string{"tool-id-1", "tool-id-2"},
    AssociatedResources: []string{"resource-id-1"},
    AssociatedPrompts:   []string{"prompt-id-1"},
    AssociatedA2aAgents: []string{"agent-id-1"},
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
    Description:         contextforge.String("Updated server description"),
    Tags:                []string{"mcp", "production", "updated"},
    AssociatedTools:     []string{"tool-id-3"}, // Replace associations
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

// Get rendered prompt with arguments (hybrid REST endpoint)
args := map[string]string{
    "language": "Go",
    "code":     "func main() { fmt.Println(\"Hello\") }",
}
result, _, err := client.Prompts.Get(ctx, "code-review", args)
if err == nil {
    fmt.Printf("Description: %s\n", *result.Description)
    for _, msg := range result.Messages {
        fmt.Printf("Role: %s, Content: %s\n", msg.Role, *msg.Content.Text)
    }
}

// Get prompt without arguments
result, _, err = client.Prompts.GetNoArgs(ctx, "simple-prompt")

// Update prompt (promptID is a string)
update := &contextforge.PromptUpdate{
    Description: contextforge.String("Updated description"),
    Template:    contextforge.String("Updated template: {{new_arg}}"),
}
updated, _, err := client.Prompts.Update(ctx, "prompt-id", update)

// Toggle prompt status
toggled, _, err := client.Prompts.Toggle(ctx, "prompt-id", true) // activate
toggled, _, err = client.Prompts.Toggle(ctx, "prompt-id", false) // deactivate

// Delete prompt
_, err = client.Prompts.Delete(ctx, "prompt-id")
```

### Managing Agents

A2A (Agent-to-Agent) agents enable inter-agent communication through ContextForge. Agents have different types for different operations due to API field naming conventions:

- **AgentCreate**: For creating agents (uses snake_case: `endpoint_url`, `agent_type`)
- **Agent**: For reading agents (uses camelCase: `endpointUrl`, `agentType`)
- **AgentUpdate**: For updating agents (uses camelCase: `endpointUrl`, `agentType`)

```go
ctx := context.Background()

// List agents with skip/limit pagination (not cursor-based)
agents, _, err := client.Agents.List(ctx, &contextforge.AgentListOptions{
    Skip:  0,
    Limit: 20,
})

// List with filtering
opts := &contextforge.AgentListOptions{
    Skip:            10,
    Limit:           50,
    IncludeInactive: true,
    Tags:            "automation,integration",
    TeamID:          "team-123",
    Visibility:      "public",
}
agents, _, err = client.Agents.List(ctx, opts)

// Get agent by ID
agent, _, err := client.Agents.Get(ctx, "agent-id")

// Create a new agent
newAgent := &contextforge.AgentCreate{
    Name:            "data-processor",
    EndpointURL:     "https://agent.example.com/a2a",
    Description:     contextforge.String("Processes data records"),
    AgentType:       "generic",        // Note: snake_case for Create
    ProtocolVersion: "1.0",
    Capabilities: map[string]any{
        "streaming": true,
        "batch":     true,
    },
    Config: map[string]any{
        "timeout": 30,
        "retries": 3,
    },
    AuthType:  contextforge.String("bearer"),
    AuthValue: contextforge.String("secret-token"), // Encrypted by API
    Tags:      []string{"data", "processing"},
}

// Create with optional team/visibility settings
createOpts := &contextforge.AgentCreateOptions{
    TeamID:     contextforge.String("team-123"),
    Visibility: contextforge.String("public"),
}
created, _, err := client.Agents.Create(ctx, newAgent, createOpts)

// Create without options
created, _, err = client.Agents.Create(ctx, newAgent, nil)

// Update agent (uses camelCase fields)
update := &contextforge.AgentUpdate{
    Description:     contextforge.String("Updated description"),
    AgentType:       contextforge.String("specialized"), // Note: camelCase for Update
    ProtocolVersion: contextforge.String("2.0"),
    Tags:            []string{"updated", "enhanced"},
}
updated, _, err := client.Agents.Update(ctx, "agent-id", update)

// Toggle agent status
toggled, _, err := client.Agents.Toggle(ctx, "agent-id", true) // enable
toggled, _, err = client.Agents.Toggle(ctx, "agent-id", false) // disable

// Invoke an agent by name (not ID!)
invokeReq := &contextforge.AgentInvokeRequest{
    Parameters: map[string]any{
        "input": "data to process",
        "options": map[string]any{
            "format":   "json",
            "validate": true,
        },
    },
    InteractionType: "query", // default: "query"
}
result, _, err := client.Agents.Invoke(ctx, created.Name, invokeReq)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Result: %v\n", result)

// Delete agent
_, err = client.Agents.Delete(ctx, "agent-id")
```

**Important Notes:**

- **Pagination**: Agents use skip/limit (offset-based) pagination instead of cursor-based pagination used by other services
- **Invoke endpoint**: Uses agent name (not ID) as the identifier
- **Field naming**: AgentCreate uses snake_case, while AgentUpdate uses camelCase
- **Authentication**: The `AuthValue` field is encrypted by the API when stored
- **Dual states**: Agents have both `Enabled` (user-controlled) and `Reachable` (system status) states
- **Performance metrics**: Agents track execution metrics including success/failure rates and response times

### Managing Teams

Teams enable collaborative resource management and access control. Team operations support member management, invitations, and team discovery:

```go
ctx := context.Background()

// List teams with skip/limit pagination (not cursor-based)
teams, _, err := client.Teams.List(ctx, &contextforge.TeamListOptions{
    Skip:  0,
    Limit: 20,
})

// Get team by ID
team, _, err := client.Teams.Get(ctx, "team-id")

// Create a basic team
newTeam := &contextforge.TeamCreate{
    Name:        "engineering",
    Description: contextforge.String("Engineering team for product development"),
}
created, _, err := client.Teams.Create(ctx, newTeam)

// Create a team with all options
completeTeam := &contextforge.TeamCreate{
    Name:        "design",
    Slug:        contextforge.String("design-team"), // Always auto-generated from name (API ignores user value)
    Description: contextforge.String("Design team for UI/UX"),
    Visibility:  contextforge.String("public"),      // "private" (default) | "public"
    MaxMembers:  contextforge.Int(50),
}
created, _, err = client.Teams.Create(ctx, completeTeam)

// Update team
update := &contextforge.TeamUpdate{
    Description: contextforge.String("Updated description"),
    Visibility:  contextforge.String("public"),
    MaxMembers:  contextforge.Int(100),
}
updated, _, err := client.Teams.Update(ctx, "team-id", update)

// Delete team
_, err = client.Teams.Delete(ctx, "team-id")

// List team members
members, _, err := client.Teams.ListMembers(ctx, "team-id")

// Update member role (uses email as identifier)
memberUpdate := &contextforge.TeamMemberUpdate{
    Role: "owner", // "owner" | "member"
}
member, _, err := client.Teams.UpdateMember(ctx, "team-id", "user@example.com", memberUpdate)

// Remove member (uses email as identifier)
_, err = client.Teams.RemoveMember(ctx, "team-id", "user@example.com")

// Invite a new member
invite := &contextforge.TeamInvite{
    Email: "newuser@example.com",
    Role:  contextforge.String("member"), // Optional, defaults to "member"
}
invitation, _, err := client.Teams.InviteMember(ctx, "team-id", invite)

// List team invitations
invitations, _, err := client.Teams.ListInvitations(ctx, "team-id")

// Accept invitation (using token)
member, _, err := client.Teams.AcceptInvitation(ctx, "invitation-token")

// Cancel invitation
_, err = client.Teams.CancelInvitation(ctx, "invitation-id")

// Discover public teams
discoveredTeams, _, err := client.Teams.Discover(ctx, &contextforge.TeamDiscoverOptions{
    Limit: 10,
})

// Request to join a public team
joinRequest, _, err := client.Teams.Join(ctx, "team-id", &contextforge.TeamJoinRequest{
    Message: contextforge.String("I'd like to contribute to this team"),
})

// Leave a team
_, err = client.Teams.Leave(ctx, "team-id")

// List join requests (owners only)
joinRequests, _, err := client.Teams.ListJoinRequests(ctx, "team-id")

// Approve join request
member, _, err = client.Teams.ApproveJoinRequest(ctx, "team-id", "request-id")

// Reject join request
_, err = client.Teams.RejectJoinRequest(ctx, "team-id", "request-id")
```

**Important Notes:**

- **Pagination**: Teams use skip/limit (offset-based) pagination like Agents
- **List response**: Returns structured response `{teams: [], total: N}` not just array
- **No request wrapping**: Unlike tools/resources, team create/update are not wrapped
- **Slug generation**: Automatically generated from team name if not provided
- **Member identification**: Member endpoints use email (not ID) as identifier
- **Invitation tokens**: One-time use tokens with expiration for accepting invitations
- **Personal teams**: Cannot be deleted or left; special restrictions apply
- **Last owner protection**: Cannot leave or be demoted if last owner

### Pagination

ContextForge supports two pagination patterns:

**Cursor-based pagination** (Tools, Resources, Gateways, Servers, Prompts):

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

**Skip/limit (offset-based) pagination** (Agents, Teams):

```go
opts := &contextforge.AgentListOptions{
    Limit: 50,
}

for skip := 0; ; skip += opts.Limit {
    opts.Skip = skip
    agents, _, err := client.Agents.List(ctx, opts)
    if err != nil {
        break
    }

    // Process agents
    for _, agent := range agents {
        fmt.Printf("Agent: %s\n", agent.Name)
    }

    // Check if we've reached the end
    if len(agents) < opts.Limit {
        break
    }
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
| `Get(ctx, resourceID)` | Get resource content (returns MCP-compatible `ResourceContent`) |
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
| `Get(ctx, promptID, args)` | Get rendered prompt with arguments (returns MCP-compatible `PromptResult`) |
| `GetNoArgs(ctx, promptID)` | Get rendered prompt without arguments (returns MCP-compatible `PromptResult`) |
| `Create(ctx, prompt, opts)` | Create a new prompt with optional settings |
| `Update(ctx, promptID, prompt)` | Update prompt |
| `Delete(ctx, promptID)` | Delete prompt |
| `Toggle(ctx, promptID, activate)` | Toggle prompt active status |

### Agents Service

| Method | Description |
|--------|-------------|
| `List(ctx, opts)` | List agents with skip/limit pagination and filtering |
| `Get(ctx, agentID)` | Get agent by ID |
| `Create(ctx, agent, opts)` | Create a new agent with optional settings |
| `Update(ctx, agentID, agent)` | Update agent |
| `Delete(ctx, agentID)` | Delete agent |
| `Toggle(ctx, agentID, activate)` | Toggle agent enabled status |
| `Invoke(ctx, agentName, req)` | Invoke agent by name with parameters |

**Note:** Agents use skip/limit (offset-based) pagination instead of cursor-based pagination. The Invoke method uses agent name (not ID) as the identifier.

### Teams Service

| Method | Description |
|--------|-------------|
| `List(ctx, opts)` | List teams with skip/limit pagination |
| `Get(ctx, teamID)` | Get team by ID |
| `Create(ctx, team)` | Create a new team |
| `Update(ctx, teamID, team)` | Update team |
| `Delete(ctx, teamID)` | Delete team |
| `ListMembers(ctx, teamID)` | List team members |
| `UpdateMember(ctx, teamID, email, update)` | Update member role (uses email) |
| `RemoveMember(ctx, teamID, email)` | Remove member (uses email) |
| `InviteMember(ctx, teamID, invite)` | Invite user to team |
| `ListInvitations(ctx, teamID)` | List team invitations |
| `AcceptInvitation(ctx, token)` | Accept invitation (uses token) |
| `CancelInvitation(ctx, invitationID)` | Cancel invitation |
| `Discover(ctx, opts)` | Discover public teams |
| `Join(ctx, teamID, req)` | Request to join public team |
| `Leave(ctx, teamID)` | Leave team |
| `ListJoinRequests(ctx, teamID)` | List join requests (owners only) |
| `ApproveJoinRequest(ctx, teamID, reqID)` | Approve join request |
| `RejectJoinRequest(ctx, teamID, reqID)` | Reject join request |

**Note:** Teams use skip/limit (offset-based) pagination like Agents. List returns structured response `{teams: [], total: N}`. Member operations use email as identifier, not ID.

## Examples

The SDK includes working example programs demonstrating all service features:

- **[tools/](examples/tools/)** - Tools service CRUD operations and filtering
- **[resources/](examples/resources/)** - Resources service with templates
- **[gateways/](examples/gateways/)** - Gateway federation and proxying
- **[servers/](examples/servers/)** - Server management and associations
- **[prompts/](examples/prompts/)** - Prompt templates and arguments
- **[agents/](examples/agents/)** - A2A agents, invocation, and skip/limit pagination
- **[teams/](examples/teams/)** - Team management, members, invitations, and discovery

Each example includes a mock HTTP server and demonstrates:
- Authentication flow
- CRUD operations
- Pagination patterns (cursor or skip/limit)
- Filtering and querying
- Error handling
- Service-specific features

Run any example:
```bash
go run examples/tools/main.go
go run examples/agents/main.go
```

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
export CONTEXTFORGE_ADDR="http://localhost:8000/"
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
- `make goreleaser-check` - Validate GoReleaser configuration
- `make goreleaser-snapshot` - Test release locally without publishing
- `make release-check` - Verify release prerequisites
- `make release-patch` - Prepare patch release (auto-increment patch version)
- `make release-minor` - Prepare minor release (auto-increment minor version)
- `make release-major` - Prepare major release (auto-increment major version)
- `make release-prep VERSION=vX.Y.Z` - Prepare release with specific version
- `make release` - Full release preparation workflow

## Releasing

This project uses semantic versioning and includes automated release tooling to streamline the release process.

**Prerequisites:**
- [GoReleaser](https://goreleaser.com/install/) - Required for automated release management
  ```bash
  go install github.com/goreleaser/goreleaser/v2@latest
  ```
- **GitHub Token** - Set `GITHUB_TOKEN` environment variable for GitHub release creation
  - Create token at: https://github.com/settings/tokens/new
  - Required scopes: `repo` (full repository access)
  - Add to your shell profile: `export GITHUB_TOKEN=your_token_here`

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

1. **Prerequisites check**: Ensures git working directory is clean and goreleaser is installed
2. **Version calculation**: Determines new version based on current version in `contextforge/version.go`
3. **Update version constant**: Updates `contextforge/version.go` with new version
4. **Create commit**: Commits version change with message `release: prepare vX.Y.Z`
5. **Create tag**: Creates annotated git tag for the release
6. **Run GoReleaser**: Executes `goreleaser release --clean` which:
   - Updates CHANGELOG.md from conventional commits
   - Creates draft GitHub release with release notes
7. **Manual review**: Review the draft release on GitHub and CHANGELOG.md changes locally
8. **Publish**: When ready, push commit and tags, then publish the draft release on GitHub

**Note:** GoReleaser creates a DRAFT release for review before publishing. The changelog is auto-generated from commit messages using conventional commits format.

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
- **Changelog**: Auto-generated from conventional commits using GoReleaser, following [Keep a Changelog](https://keepachangelog.com/) format
- **Git Tags**: Use format `vX.Y.Z` (semantic versioning with `v` prefix)
- **Commit Format**: All commits should use [conventional commits](https://www.conventionalcommits.org/) format (e.g., `feat:`, `fix:`, `docs:`)

### Changelog Generation

The project uses [GoReleaser](https://goreleaser.com/) to automatically generate the changelog from commit messages. Changelog generation happens automatically during the release workflow and creates entries in both CHANGELOG.md and the GitHub release notes.

**Testing GoReleaser Configuration:**
```bash
# Validate GoReleaser configuration
make goreleaser-check

# Test release locally without publishing
make goreleaser-snapshot
```

**Configuration** (`.goreleaser.yaml`):
- Uses GitHub's native changelog generation
- Groups commits by conventional commit type
- Excludes merge commits and release preparation commits
- Creates draft GitHub releases for manual review

**Commit types map to changelog sections:**
- `feat:` → Added
- `fix:`, `bug:` → Fixed
- `refactor:` → Changed
- `docs:` → Documentation
- `build:`, `chore:` → Build
- `test:`, `style:` → Tests

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
- **AgentsService** - All A2A agent operations, invocation, and performance tracking
- **TeamsService** - Team management, members, invitations, and discovery
- **CancellationService** - Request cancellation and cancellation status checks for in-flight runs

### Custom Types

- **FlexibleID** - Handles API inconsistencies where IDs may be returned as integers or strings
- **Timestamp** - Custom timestamp parsing for API responses without timezone information
- **Tag** - Handles tag objects with `ID` and `Label` fields; custom JSON marshal/unmarshal for API compatibility
- **Pointer helpers** - `String()`, `Int()`, `Bool()`, `Time()` for working with optional fields
- **Tag helpers** - `NewTags()`, `NewTag()`, `TagNames()` for converting between `[]string` and `[]Tag`

## Links

- **ContextForge Repository:** https://github.com/IBM/mcp-context-forge
- **MCP Protocol:** https://modelcontextprotocol.io/
- **A2A Protocol Specification:** https://a2a-protocol.org/latest/specification/
- **A2A Protocol Website:** https://a2aprotocol.ai/

## Known Issues

### Upstream ContextForge API Bugs

The SDK integration tests have identified six bugs in ContextForge (confirmed in both v0.8.0 and v1.0.0-BETA-1; revalidation against v1.0.0-BETA-2 is pending in this repository). These bugs are in the upstream API, not the SDK implementation. Affected tests are skipped and will be re-enabled once upstream bugs are fixed.

**CONTEXTFORGE-001: Toggle Endpoints Return Stale State**
The `POST /prompts/{id}/toggle` and `POST /resources/{id}/toggle` endpoints return stale `isActive` state despite correctly updating the database. See [`docs/upstream-bugs/contextforge-001-prompt-toggle.md`](docs/upstream-bugs/contextforge-001-prompt-toggle.md).

**CONTEXTFORGE-002: Prompts API Accepts Empty Template Field**
The `POST /prompts` endpoint accepts prompt creation without the `template` field, allowing semantically invalid prompts. See [`docs/upstream-bugs/contextforge-002-prompt-validation-missing-template.md`](docs/upstream-bugs/contextforge-002-prompt-validation-missing-template.md).

**CONTEXTFORGE-003: Prompts Toggle Returns 400 Instead of 404**
The `POST /prompts/{id}/toggle` endpoint returns HTTP 400 for non-existent prompts instead of 404. See [`docs/upstream-bugs/contextforge-003-prompt-toggle-error-code.md`](docs/upstream-bugs/contextforge-003-prompt-toggle-error-code.md).

**CONTEXTFORGE-004: Teams Individual Resource Endpoints Reject Valid Authentication**
Individual team endpoints (`GET/PUT/DELETE /teams/{id}/*`) reject valid JWT tokens with 401 Unauthorized, despite list/create working correctly. See [`docs/upstream-bugs/contextforge-004-teams-auth-individual-endpoints.md`](docs/upstream-bugs/contextforge-004-teams-auth-individual-endpoints.md).

**CONTEXTFORGE-005: Teams API Ignores User-Provided Slug Field**
The `POST /teams` endpoint ignores the `slug` field and always auto-generates from team name. See [`docs/upstream-bugs/contextforge-005-teams-slug-ignored.md`](docs/upstream-bugs/contextforge-005-teams-slug-ignored.md).

**CONTEXTFORGE-006: Teams API Returns 422 Instead of 400 for Validation Errors**
The `POST /teams` endpoint returns HTTP 422 (Unprocessable Entity) for validation errors instead of 400 (Bad Request). See [`docs/upstream-bugs/contextforge-006-teams-validation-error-code.md`](docs/upstream-bugs/contextforge-006-teams-validation-error-code.md).

All bug reports include root cause analysis, proposed solutions, and workarounds.

## License

MIT License - see [LICENSE](LICENSE) file for details.
