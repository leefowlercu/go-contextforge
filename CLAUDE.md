# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Table of Contents

- [Project Overview](#project-overview)
- [Project Principles](#project-principles)
- [High-Level Architecture](#high-level-architecture)
  - [Key Components](#key-components)
- [Subsystems Reference](#subsystems-reference)
- [Conventions & Patterns](#conventions--patterns)
  - [API Patterns](#api-patterns)
  - [Three-State System for Optional Fields](#three-state-system-for-optional-fields)
  - [MCP Protocol vs REST API Endpoints](#mcp-protocol-vs-rest-api-endpoints)
- [Code Organization Principles](#code-organization-principles)
  - [Adding New Services](#adding-new-services)
- [Testing Approach](#testing-approach)
  - [Unit Tests](#unit-tests)
  - [Integration Tests](#integration-tests)
  - [Skipped Integration Tests](#skipped-integration-tests)
- [Development Commands](#development-commands)
  - [Building & Testing](#building--testing)
  - [Running the Application](#running-the-application)
  - [Releasing](#releasing)

## Project Overview

Go SDK for the IBM ContextForge MCP Gateway API, providing an idiomatic client library for programmatically managing MCP resources and A2A agents.

**Key Features:**
- Service-oriented architecture following `google/go-github` patterns
- Full CRUD operations for Tools, Resources, Gateways, Servers, Prompts, Agents, and Teams
- JWT bearer token authentication
- Cursor-based and offset-based pagination support
- Rate limit tracking and error handling
- Three-state semantics for partial updates (essential for Terraform providers)

## Project Principles

This SDK follows these guiding principles:

1. **google/go-github Pattern Adherence**: Follow the service-oriented architecture established by google/go-github for consistent, idiomatic Go SDK design
2. **Three-State Semantics**: Support nil (omit), empty (clear), and value (set) states for all optional fields to enable partial updates
3. **Context-First API**: All API methods accept `context.Context` as first parameter for cancellation and timeout support
4. **Explicit Over Implicit**: Prefer explicit type definitions over interface{} where possible; use custom types for API inconsistencies
5. **Test-Driven Development**: Unit tests with HTTP mocking for all service methods; integration tests against real API

## High-Level Architecture

The SDK follows the **service-oriented architecture pattern** established by `google/go-github`:

- **Single `Client` struct**: Manages HTTP communication, JWT authentication, and rate limit tracking
- **Service structs**: `ToolsService`, `ResourcesService`, `GatewaysService`, `ServersService`, `PromptsService`, `AgentsService`, `TeamsService` (each embeds shared `service` struct)
- **Context support**: All API methods accept `context.Context` for cancellation and timeouts

### Key Components

**Client Management** (`contextforge/contextforge.go`):
- `NewClient(httpClient, address, bearerToken)`: Factory for authenticated clients (returns `*Client, error`)
  - Accepts address as string parameter (e.g., `"https://api.example.com/"`)
  - Automatically appends trailing slash if missing
  - Returns error for invalid URL formats
- `NewRequest()`: Creates HTTP requests with proper headers and authentication
- `Do()`: Executes requests with context support, error handling, and rate limit tracking
- Thread-safe rate limit tracking using `sync.Mutex` and `rateLimits` map

**Service Pattern**:
- Each service embeds `service` struct with pointer to parent `Client`
- Services instantiated once during client creation via `common` service pattern
- All services reused throughout client lifetime

**Custom Types** (`contextforge/types.go`):
- `FlexibleID`: Handles API inconsistencies where IDs may be integers or strings
- `Timestamp`: Custom timestamp parsing for API responses without timezone information
- `Tag`: Handles tag objects with `ID` and `Label` fields; custom JSON marshal/unmarshal for API compatibility (accepts strings, returns objects)

**Pointer Helpers** (`contextforge/pointers.go`):
- Creating pointers: `String()`, `Int()`, `Int64()`, `Bool()`, `Float64()`, `Time()`
- Safe dereferencing: `StringValue()`, `IntValue()`, `Int64Value()`, `BoolValue()`, `Float64Value()`, `TimeValue()`
- Tag helpers: `NewTag()`, `NewTags()`, `TagNames()` for converting between `[]string` and `[]Tag`
- Enables three-state semantics for optional fields

**Rate Limiting**:
- Tracked per-endpoint path in `Client.rateLimits`
- Parses headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`
- Custom `RateLimitError` type for 429 responses

**Pagination Patterns**:
The SDK supports two pagination patterns based on API endpoint design:
- **Cursor-based** (Tools, Resources, Gateways, Servers, Prompts): Uses `ListOptions` with `Limit` and `Cursor` fields. Next cursor from `X-Next-Cursor` header.
- **Offset-based** (Agents, Teams): Uses service-specific options with `Skip` and `Limit` fields for offset pagination.

**Error Handling** (`contextforge/errors.go`):
- `ErrorResponse`: Standard API error with message and error details
- `RateLimitError`: Specialized error for rate limiting
- `CheckResponse()`: Validates HTTP responses and returns typed errors
- URL sanitization to prevent token leakage in error messages

**Additional Core Files**:
- `contextforge/doc.go` - Package-level documentation
- `contextforge/version.go` - SDK version constant used in User-Agent header
- `docs/three-state-system.md` - Comprehensive guide to three-state semantics pattern
- `docs/terraform-provider-usage.md` - Guide for building Terraform providers with the SDK

## Subsystems Reference

This SDK does not have separate subsystems. All functionality is contained within the `contextforge/` package.

**Related Documentation:**
- `docs/three-state-system.md` - Guide to three-state semantics for optional fields
- `docs/terraform-provider-usage.md` - Building Terraform providers with this SDK
- `docs/upstream-bugs/` - Documentation of known upstream API bugs

## Conventions & Patterns

### API Patterns

Some API endpoints require request body wrapping (e.g., `{"tool": {...}}`). Check upstream schema/router definitions for the target tag, or existing service implementations, for wrapping requirements.

**Field Naming Conventions:**
- Create types use snake_case for some fields (e.g., `mime_type`, `endpoint_url`)
- Read/Update types use camelCase (e.g., `mimeType`, `endpointUrl`)
- The SDK handles these inconsistencies internally

**Prompt ID Type:**
- Prompt IDs are `string` type (changed from `int` in v1.0.0)
- All PromptService methods accept `promptID string`

### Three-State System for Optional Fields

The SDK uses a **three-state semantics pattern** for optional fields in update operations:

1. **nil pointer/slice** - Field omitted from request (don't update existing value)
2. **Pointer to zero value or empty slice** - Field cleared/set to empty
3. **Pointer to value or populated slice** - Field set to specific value

```go
// Update only name, leave other fields unchanged
update := &contextforge.ResourceUpdate{
    Name: contextforge.String("new-name"),
    // Description is nil - won't be sent to API
}

// Clear all tags vs don't update tags
update1 := &contextforge.ResourceUpdate{Tags: nil}        // Tags unchanged
update2 := &contextforge.ResourceUpdate{Tags: []string{}} // Clears all tags
```

**Documentation:**
- [docs/three-state-system.md](docs/three-state-system.md) - Comprehensive guide
- [docs/terraform-provider-usage.md](docs/terraform-provider-usage.md) - Terraform provider integration

### MCP Protocol vs REST API Endpoints

ContextForge implements **TWO SEPARATE APIs**:

**1. REST API (Management Operations) - IMPLEMENT IN SDK**
- Purpose: Manage tools, resources, servers, gateways, prompts, agents, teams
- Authentication: JWT bearer tokens
- Examples: `GET /tools`, `POST /resources`, `PUT /servers/{id}`

**2. JSON-RPC API (MCP Protocol) - DO NOT IMPLEMENT**
- Purpose: MCP protocol compliance for client communication
- Endpoint: `POST /rpc` (single endpoint for all JSON-RPC methods)
- Methods: `initialize`, `tools/list`, `tools/call`, `resources/read`, etc.

**3. SSE/Streaming Endpoints - DO NOT IMPLEMENT**
- `GET /servers/{id}/sse` - SSE connection for MCP protocol transport
- `POST /resources/subscribe/{id}` - SSE streaming for change notifications

## Code Organization Principles

```
go-contextforge/
├── contextforge/           # Main SDK package
│   ├── contextforge.go     # Client, NewClient(), request handling
│   ├── types.go            # All type definitions
│   ├── pointers.go         # Pointer helper functions
│   ├── errors.go           # Error types and handling
│   ├── version.go          # SDK version constant
│   ├── doc.go              # Package documentation
│   ├── tools.go            # ToolsService
│   ├── resources.go        # ResourcesService
│   ├── gateways.go         # GatewaysService
│   ├── servers.go          # ServersService
│   ├── prompts.go          # PromptsService
│   ├── agents.go           # AgentsService
│   ├── teams.go            # TeamsService
│   └── *_test.go           # Unit tests for each file
├── test/integration/       # Integration tests
│   ├── helpers.go          # Test utilities and cleanup
│   └── *_integration_test.go
├── examples/               # Working examples for each service
├── docs/                   # Additional documentation
│   ├── upstream-bugs/      # Known API bug reports
│   └── *.md                # Guides and references
└── scripts/                # Build and release scripts
```

### Adding New Services

When implementing new services for additional ContextForge API resources:

1. Create service file: `contextforge/<service>.go`
2. Define service struct: `type <Service>Service service`
3. Add service field to `Client` struct in `types.go`
4. Initialize service in `newClient()` in `contextforge.go`: `c.<Service> = (*<Service>Service)(&c.common)`
5. Implement service methods with signature: `(ctx context.Context, ...) (*ReturnType, *Response, error)`
6. Add unit tests in `contextforge/<service>_test.go`
7. Add integration tests in `test/integration/<service>_integration_test.go`
8. Add helper functions to `test/integration/helpers.go` for test data generation and cleanup

Follow existing service implementations as patterns:
- **ToolsService** - Wrapped create/update, nested toggle response
- **ResourcesService** - Wrapped create, unwrapped update
- **GatewaysService** - Complex types with authentication fields
- **ServersService** - Direct toggle response, association endpoints
- **PromptsService** - String IDs, API case inconsistencies
- **AgentsService** - Offset-based pagination, A2A protocol
- **TeamsService** - Offset-based pagination, wrapped list response

## Testing Approach

### Unit Tests

Located in `contextforge/*_test.go`:
- Use `httptest.NewServer` for HTTP mocking
- Table-driven tests for most functions
- Setup/teardown helpers: `setup()`, `testMethod()`, `testURLParseError()`, `testJSONMarshal()`
- Test edge cases, error scenarios, and happy paths

### Integration Tests

Located in `test/integration/*_test.go`:
- Build tag: `//go:build integration` (required for all integration test files)
- Run against real ContextForge gateway
- Environment configuration:
  - `INTEGRATION_TESTS=true` (required to run)
  - `CONTEXTFORGE_ADDR` (default: `http://localhost:8000/`)
  - `CONTEXTFORGE_ADMIN_EMAIL` (default: `admin@test.local`)
  - `CONTEXTFORGE_ADMIN_PASSWORD` (default: `testpassword123`)
- JWT authentication via login endpoint
- Automatic cleanup using `t.Cleanup()`

### Skipped Integration Tests

Some integration tests are skipped due to confirmed bugs in the upstream ContextForge API (confirmed in both v0.8.0 and v1.0.0-BETA-1). See `docs/upstream-bugs/` for detailed bug reports.

| Bug ID | Summary | Skipped Tests |
|--------|---------|---------------|
| CONTEXTFORGE-001 | Toggle endpoints return stale state | 4 tests |
| CONTEXTFORGE-002 | API accepts empty template string | 1 test |
| CONTEXTFORGE-003 | Toggle returns 400 instead of 404 | 1 test |
| CONTEXTFORGE-004 | Teams endpoints reject valid auth | 12 tests |
| CONTEXTFORGE-005 | Teams slug field ignored | 2 tests |
| CONTEXTFORGE-007 | Gateway tags not persisted | 2 tests |
| CONTEXTFORGE-008 | Agent bearer auth requires auth_token | 1 test |
| CONTEXTFORGE-009 | Tag filtering returns empty results | 5 tests |
| CONTEXTFORGE-010 | Team ID filter returns 403 | 1 test |

**Resolved Issues (tests re-enabled):**
| Bug ID | Summary | Resolution |
|--------|---------|------------|
| CONTEXTFORGE-006 | 422 for validation errors | By design (FastAPI standard)

To re-enable a skipped test once the upstream bug is fixed:
1. Verify the bug is fixed in the ContextForge version you're testing against
2. Remove the `t.Skip()` line from the test
3. Run the test to confirm it passes
4. Update the bug report in `docs/upstream-bugs/` with resolution details

## Development Commands

Run `make help` to see all available targets.

### Building & Testing

**Unit Tests:**
```bash
make test              # Run unit tests
make test-verbose      # Run unit tests with verbose output
make test-cover        # Run unit tests with coverage
make coverage          # Generate HTML coverage report
```

**Integration Tests:**
```bash
make integration-test-all      # Full cycle: setup, test, teardown
make integration-test-setup    # Start ContextForge gateway
make integration-test          # Run integration tests
make integration-test-teardown # Stop gateway
make test-all                  # Run both unit and integration tests
```

**Building:**
```bash
make build      # Build all packages
make build-all  # Build packages + examples
make examples   # Build all example programs
```

**Code Quality:**
```bash
make fmt    # Format code with gofmt
make vet    # Run go vet
make lint   # Format + vet
make check  # Lint + test
make ci     # Full CI pipeline (deps, lint, test, build)
```

**Dependencies:**
```bash
make deps        # Download dependencies
make tidy        # Tidy go.mod and go.sum
make update-deps # Update dependencies to latest versions
make clean       # Clean build artifacts and test cache
```

### Running the Application

This is an SDK library, not a standalone application. To use it:

```go
import "github.com/leefowlercu/go-contextforge/contextforge"

client, err := contextforge.NewClient(nil, "http://localhost:8000/", "your-jwt-token")
```

**Examples:** Run any example program to see SDK usage:
```bash
go run examples/tools/main.go
go run examples/prompts/main.go
go run examples/agents/main.go
```

### Releasing

The project uses semantic versioning with automated release tooling.

**Prerequisites:**
- GoReleaser: `go install github.com/goreleaser/goreleaser/v2@latest`
- GitHub Token: Set `GITHUB_TOKEN` environment variable

**Release Commands:**
```bash
make release-patch              # 0.1.0 → 0.1.1 (bug fixes)
make release-minor              # 0.1.0 → 0.2.0 (new features)
make release-major              # 0.1.0 → 1.0.0 (breaking changes)
make release-prep VERSION=vX.Y.Z # Manual version
make goreleaser-check           # Validate GoReleaser config
make goreleaser-snapshot        # Test release locally
```

**Release Process:**
1. Checks git working directory is clean
2. Updates `contextforge/version.go` with new version
3. Creates commit: `release: prepare vX.Y.Z`
4. Creates annotated git tag
5. Runs GoReleaser to update CHANGELOG.md and create draft GitHub release
6. Review and publish the draft release

**Undoing a release (before pushing):**
```bash
git tag -d vX.Y.Z
git reset --hard HEAD~1
```
