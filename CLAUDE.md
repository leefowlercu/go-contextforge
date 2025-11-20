# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Table of Contents

- [Project Overview](#project-overview)
- [Architecture](#architecture)
  - [Key Components](#key-components)
- [Development Commands](#development-commands)
- [Release Workflow](#release-workflow)
  - [Version Management](#version-management)
  - [Release Commands](#release-commands)
  - [Release Process](#release-process)
  - [Release Scripts](#release-scripts)
  - [Making Releases](#making-releases)
  - [Undoing Release Preparation](#undoing-release-preparation)
- [Testing Strategy](#testing-strategy)
  - [Unit Tests](#unit-tests)
  - [Integration Tests](#integration-tests)
  - [Integration Test Setup](#integration-test-setup)
  - [Running Single Tests](#running-single-tests)
  - [Skipped Integration Tests](#skipped-integration-tests)
- [Adding New Services](#adding-new-services)
- [API Patterns](#api-patterns)
- [Three-State System for Optional Fields](#three-state-system-for-optional-fields)
- [MCP Protocol vs REST API Endpoints](#mcp-protocol-vs-rest-api-endpoints)
  - [1. REST API Management Endpoints (IMPLEMENT IN SDK)](#1-rest-api-management-endpoints-implement-in-sdk)
  - [2. MCP Protocol/Client Endpoints (DO NOT IMPLEMENT)](#2-mcp-protocolclient-endpoints-do-not-implement)
  - [Identifying MCP Protocol Endpoints](#identifying-mcp-protocol-endpoints)
  - [Examples](#examples)
  - [Service Documentation Pattern](#service-documentation-pattern)

## Project Overview

Go SDK for the IBM ContextForge MCP Gateway API, providing idiomatic client library for programmatically managing MCP resources and A2A agents.

## Architecture

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

**Pointer Helpers** (`contextforge/pointers.go`):
- Creating pointers: `String()`, `Int()`, `Int64()`, `Bool()`, `Float64()`, `Time()`
- Safe dereferencing: `StringValue()`, `IntValue()`, `Int64Value()`, `BoolValue()`, `Float64Value()`, `TimeValue()`
- Enables three-state semantics for optional fields (see [Three-State System](#three-state-system-for-optional-fields))

**Rate Limiting**:
- Tracked per-endpoint path in `Client.rateLimits`
- Parses headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`
- Custom `RateLimitError` type for 429 responses

**Pagination**:
- Most services use cursor-based pagination (Tools, Resources, Gateways, Servers, Prompts)
- Cursor extracted from `X-Next-Cursor` response header
- `ListOptions` struct embedded in service-specific list options
- Agents and Teams services use offset-based pagination (skip/limit) instead

**Pagination Patterns**:
The SDK supports two pagination patterns based on API endpoint design:
- **Cursor-based** (Tools, Resources, Gateways, Servers, Prompts): Uses `ListOptions` with `Limit` and `Cursor` fields. Next cursor from `X-Next-Cursor` header.
- **Offset-based** (Agents, Teams): Uses service-specific options with `Skip` and `Limit` fields for offset pagination.

**Error Handling**:
- `ErrorResponse`: Standard API error with message and error details
- `RateLimitError`: Specialized error for rate limiting
- `CheckResponse()`: Validates HTTP responses and returns typed errors
- URL sanitization to prevent token leakage in error messages

**Additional Core Files**:
- `contextforge/pointers.go` - Pointer helper utilities for working with optional fields
- `contextforge/errors.go` - Error types: `ErrorResponse`, `RateLimitError`, `CheckResponse()`
- `contextforge/doc.go` - Package-level documentation
- `contextforge/version.go` - SDK version constant used in User-Agent header
- `docs/three-state-system.md` - Comprehensive guide to three-state semantics pattern
- `docs/terraform-provider-usage.md` - Guide for building Terraform providers with the SDK

## Development Commands

Run `make help` to see all available targets. Key workflows:

**Testing**:
- `make test` - Unit tests
- `make integration-test-all` - Full integration test cycle (setup, test, teardown)
- `make test-all` - Run both unit and integration tests
- `make coverage` - Generate HTML coverage report

**Building**:
- `make build` - Build all packages
- `make build-all` - Build packages + examples

**Code Quality**:
- `make fmt` - Format code with gofmt
- `make vet` - Run go vet
- `make lint` - Format + vet
- `make check` - Lint + test
- `make ci` - Full CI pipeline (deps, lint, test, build)

**Releasing**:
- `make goreleaser-check` - Validate GoReleaser configuration
- `make goreleaser-snapshot` - Test release locally without publishing
- `make release-check` - Verify release prerequisites
- `make release-patch` - Prepare patch release (auto-increment patch version)
- `make release-minor` - Prepare minor release (auto-increment minor version)
- `make release-major` - Prepare major release (auto-increment major version)
- `make release-prep VERSION=vX.Y.Z` - Prepare release with specific version

## Release Workflow

The project uses semantic versioning with automated release tooling.

### Version Management

- **Version constant**: Defined in `contextforge/version.go` as `const Version = "X.Y.Z"`
- **User agent**: Automatically constructed as `"go-contextforge/v" + Version`
- **Changelog**: Maintained in `CHANGELOG.md` following Keep a Changelog format
- **Git tags**: Use format `vX.Y.Z` (semantic versioning with `v` prefix)
- **GoReleaser**: Required for release automation - install with `go install github.com/goreleaser/goreleaser/v2@latest`
- **GitHub Token**: Set `GITHUB_TOKEN` environment variable for GitHub release creation

### Release Commands

**Automated semantic version bumping (recommended):**
```bash
make release-patch  # 0.1.0 → 0.1.1 (bug fixes)
make release-minor  # 0.1.0 → 0.2.0 (new features, backward compatible)
make release-major  # 0.1.0 → 1.0.0 (breaking changes)
```

**Manual version override:**
```bash
make release-prep VERSION=v0.2.5
```

### Release Process

Each release command performs:
1. Checks git working directory is clean (via `release-check` target)
2. Checks that `goreleaser` is installed
3. Calculates new version by parsing current version from `contextforge/version.go`
4. Updates `contextforge/version.go` with new version constant
5. Creates commit with message `release: prepare vX.Y.Z`
6. Creates annotated git tag `vX.Y.Z`
7. Runs `goreleaser release --clean` which:
   - Updates CHANGELOG.md from conventional commits
   - Creates draft GitHub release with release notes
8. Displays instructions for reviewing and publishing

### Release Scripts

**scripts/bump-version.sh:**
- Parses current version from `contextforge/version.go`
- Implements semver bumping logic (major/minor/patch)
- Validates version format and bump type
- Writes new version to `.next-version` temporary file
- Used by `release-patch`, `release-minor`, `release-major` targets

**scripts/prepare-release.sh:**
- Validates version format (vX.Y.Z)
- Checks for `goreleaser` installation
- Checks for `GITHUB_TOKEN` environment variable (warns if missing)
- Updates version constant using sed
- Commits version change
- Creates annotated git tag
- Runs `goreleaser release --clean` to update CHANGELOG.md and create draft GitHub release
- Includes error handling to rollback commit and tag if goreleaser fails
- Includes cleanup trap to remove `.next-version` on exit

### Making Releases

When assisting with releases:
1. **Never commit release changes directly** - only prepare them
2. **Always verify prerequisites** - ensure clean git status, goreleaser installed, and GITHUB_TOKEN set
3. **Use semantic versioning** - choose appropriate bump type based on changes
4. **Write good commit messages** - changelog is auto-generated from conventional commits, so descriptive subject lines are critical
5. **Review generated content** - verify draft GitHub release and CHANGELOG.md changes are accurate and complete
6. **Follow conventional commits** - use `release: prepare vX.Y.Z` format for release commits

**Important for Changelog Quality:**
- Since changelog entries are generated from commit messages, all commits should use conventional commit format
- Subject lines should be clear, descriptive, and user-facing when appropriate
- Use correct commit type prefixes (feat, fix, docs, etc.) as they determine changelog section grouping
- Commit types map to Keep a Changelog sections: feat→Added, fix/bug→Fixed, docs→Documentation, refactor→Changed, etc.
- GoReleaser creates DRAFT releases - always review before publishing

### Undoing Release Preparation

If a release needs to be undone (before pushing):
```bash
git tag -d vX.Y.Z          # Remove tag
git reset --hard HEAD~1    # Reset commit
```

## Testing Strategy

### Unit Tests

Located in `contextforge/*_test.go`:
- Use `httptest.NewServer` for HTTP mocking
- Table-driven tests for most functions
- Setup/teardown helpers: `setup()`, `testMethod()`, `testURLParseError()`, `testJSONMarshal()`
- Test edge cases, error scenarios, and happy paths

### Integration Tests

Located in `test/integration/*_test.go`:
- Build tag: `//go:build integration` (required for all integration test files)
- Run against real ContextForge gateway (or mock MCP server for gateway tests)
- Environment configuration:
  - `INTEGRATION_TESTS=true` (required to run)
  - `CONTEXTFORGE_ADDR` (default: `http://localhost:8000/`)
  - `CONTEXTFORGE_ADMIN_EMAIL` (default: `admin@test.local`)
  - `CONTEXTFORGE_ADMIN_PASSWORD` (default: `testpassword123`)
- JWT authentication via login endpoint
- Automatic cleanup using `t.Cleanup()`
- Tests organized by functionality: CRUD, filtering, pagination, validation, errors, edge cases

### Integration Test Setup

Automated setup (recommended): `make integration-test-all`

Manual setup:
```bash
./scripts/integration-test-setup.sh  # Start gateway
INTEGRATION_TESTS=true go test -v ./test/integration/...
./scripts/integration-test-teardown.sh  # Stop gateway
```

### Running Single Tests

```bash
# Unit test
go test -v -run TestToolsService_List ./contextforge/

# Integration test (requires gateway running)
INTEGRATION_TESTS=true go test -v -run TestToolsService ./test/integration/
```

### Skipped Integration Tests

Some integration tests are currently skipped due to confirmed bugs in the upstream ContextForge API (v0.8.0). These tests document expected behavior and will be re-enabled once the upstream bugs are fixed.

#### CONTEXTFORGE-001: Prompts Toggle Returns Stale State

**Bug Report:** `docs/upstream-bugs/prompt-toggle.md`

**Skipped Test:** `TestPromptsService_Toggle/toggle_inactive_to_active`

**Issue:** The `POST /prompts/{id}/toggle` endpoint returns stale `isActive` state in the response despite correctly updating the database. When toggling from inactive→active, the response shows the old `isActive=false` instead of new `isActive=true`.

**Root Cause:** SQLAlchemy session caching issue in `prompt_service.py` where `_convert_db_prompt()` reads cached attribute values despite calling `db.refresh()`.

**SDK Status:** ✅ SDK implementation is correct. The test failure is expected given the API bug.

**Workaround:** Query the prompt list after toggling to get fresh state, or use the Update endpoint instead.

#### CONTEXTFORGE-002: Prompts API Accepts Empty Template Field

**Bug Report:** `docs/upstream-bugs/prompt-validation-missing-template.md`

**Skipped Test:** `TestPromptsService_InputValidation/create_prompt_without_required_template`

**Issue:** The `POST /prompts` endpoint accepts prompt creation requests without a `template` field, allowing prompts to be created with empty/missing templates. A prompt without a template is semantically invalid since the template defines what the prompt renders.

**Root Cause:** Missing validation constraint in Pydantic model/OpenAPI spec. The `template` field is not marked as required.

**SDK Status:** ✅ SDK implementation is correct. This may be intentional to support draft prompts.

**Workaround:** Always provide the `template` field when creating prompts. Add client-side validation if needed.

#### CONTEXTFORGE-003: Prompts Toggle Returns 400 Instead of 404

**Bug Report:** `docs/upstream-bugs/prompt-toggle-error-code.md`

**Skipped Test:** `TestPromptsService_ErrorHandling/toggle_non-existent_prompt_returns_404`

**Issue:** The `POST /prompts/{id}/toggle` endpoint returns HTTP 400 Bad Request when attempting to toggle a non-existent prompt, instead of the expected 404 Not Found. This is inconsistent with other prompts endpoints (PUT, DELETE) which correctly return 404.

**Root Cause:** Incorrect error handling in the endpoint. The `PromptNotFoundError` is caught but returned as `HTTPException(400)` instead of `HTTPException(404)`.

**SDK Status:** ✅ SDK implementation is correct. The SDK properly handles both error codes.

**Workaround:** Accept both 400 and 404 as "not found" errors, or check error message content for "not found" string.

#### CONTEXTFORGE-004: Teams Individual Resource Endpoints Reject Valid Authentication

**Bug Report:** `docs/upstream-bugs/teams-auth-individual-endpoints.md`

**Skipped Tests:** 12 tests in `test/integration/teams_integration_test.go`:
- `TestTeamsService_BasicCRUD/get_team_by_ID`
- `TestTeamsService_BasicCRUD/update_team`
- `TestTeamsService_BasicCRUD/delete_team`
- `TestTeamsService_Members/list_team_members`
- `TestTeamsService_Invitations/create_invitation`
- `TestTeamsService_Invitations/list_team_invitations`
- `TestTeamsService_Invitations/cancel_invitation`
- `TestTeamsService_Discovery/discover_public_teams`
- `TestTeamsService_Discovery/discover_teams_with_pagination`
- `TestTeamsService_ErrorHandling/get_non-existent_team_returns_404`
- `TestTeamsService_ErrorHandling/update_non-existent_team_returns_404`
- `TestTeamsService_ErrorHandling/delete_non-existent_team_returns_404`

**Issue:** Individual team resource endpoints (`GET/PUT/DELETE /teams/{id}/*`) reject valid JWT authentication tokens with "Invalid token" (401 Unauthorized), despite the same token working correctly for team list and create operations. This affects all operations on individual teams including member management, invitations, join requests, and team discovery.

**Root Cause:** Suspected FastAPI route registration or middleware application issue with parameterized paths. Collection endpoints (`GET /teams/`, `POST /teams/`) work correctly, but all individual resource endpoints with path parameters fail authentication.

**SDK Status:** ✅ SDK implementation is correct. Test failures are expected given the API bug.

**Workaround:** For read operations, use `GET /teams/` and filter client-side. No workaround exists for update, delete, or member management operations.

#### CONTEXTFORGE-005: Teams API Ignores User-Provided Slug Field

**Bug Report:** `docs/upstream-bugs/teams-slug-ignored.md`

**Skipped Tests:** 2 tests in `test/integration/teams_integration_test.go`:
- `TestTeamsService_BasicCRUD/create_team_with_all_optional_fields`
- `TestTeamsService_Validation/create_team_with_valid_slug_pattern`

**Issue:** The `POST /teams` endpoint ignores the user-provided `slug` field in team creation requests and instead auto-generates the slug from the team name. This prevents users from creating teams with custom slugs and makes the `slug` field effectively read-only despite being documented as an input field.

**Root Cause:** Team creation logic accepts the `slug` field in the Pydantic schema but ignores the provided value during team creation, always auto-generating the slug from the team name using a slugify function.

**SDK Status:** ✅ SDK implementation is correct. The SDK correctly sends the slug field and parses the response.

**Workaround:** Use the desired slug as the team name - auto-generation will create a matching slug (e.g., name: `"my-custom-slug"` → slug: `"my-custom-slug"`).

#### CONTEXTFORGE-006: Teams API Returns 422 Instead of 400 for Validation Errors

**Bug Report:** `docs/upstream-bugs/teams-validation-error-code.md`

**Skipped Tests:** 1 test in `test/integration/teams_integration_test.go`:
- `TestTeamsService_ErrorHandling/create_team_without_required_name_returns_400`

**Issue:** The `POST /teams` endpoint returns HTTP 422 (Unprocessable Entity) for request validation errors (like missing required fields) instead of the more standard HTTP 400 (Bad Request). Additionally, the SDK's Response object shows `StatusCode: 0` for 422 responses, indicating potential response construction issues.

**Root Cause:** FastAPI automatically validates request bodies against Pydantic models and returns 422 for validation errors by default. This is FastAPI's standard behavior, though 400 is more semantically correct for missing required fields.

**SDK Status:** ⚠️ SDK correctly returns errors for validation failures, but the Response object may not be properly populated for 422 status codes.

**Workaround:** Accept both 400 and 422 as validation errors, or check for any 4xx error rather than specific status code.

#### Re-enabling Skipped Tests

To re-enable a skipped test once the upstream bug is fixed:

1. Verify the bug is fixed in the ContextForge version you're testing against
2. Remove the `t.Skip()` line from the test
3. Run the test to confirm it passes
4. Update the bug report in `docs/upstream-bugs/` with resolution details
5. Update this documentation to remove the entry

#### Testing Against Different ContextForge Versions

The skipped tests assume ContextForge v0.8.0. When testing against newer versions:

```bash
# Check ContextForge version
curl http://localhost:8000/health | jq '.app.version'

# Run integration tests
INTEGRATION_TESTS=true go test -v -tags=integration ./test/integration/
```

If tests that were previously skipped now pass, follow the re-enabling process above.

## Adding New Services

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
- **ToolsService** (`contextforge/tools.go`) - Wrapped create/update, nested toggle response
- **ResourcesService** (`contextforge/resources.go`) - Wrapped create, unwrapped update
- **GatewaysService** (`contextforge/gateways.go`) - Complex types with authentication fields
- **ServersService** (`contextforge/servers.go`) - Direct toggle response, association endpoints
- **PromptsService** (`contextforge/prompts.go`) - API case inconsistencies (snake_case create, camelCase update)
- **AgentsService** (`contextforge/agents.go`) - Offset-based pagination (skip/limit), A2A protocol management
- **TeamsService** (`contextforge/teams.go`) - Offset-based pagination, wrapped list response

## API Patterns

Some API endpoints require request body wrapping (e.g., `{"tool": {...}}`). Check OpenAPI spec (`reference/contextforge-openapi-v0.8.0.json`) or existing service implementations for wrapping requirements.

## Three-State System for Optional Fields

The SDK uses a **three-state semantics pattern** for optional fields in update operations, enabling precise control over which fields are included in API requests:

1. **nil pointer/slice** - Field omitted from request (don't update existing value)
2. **Pointer to zero value or empty slice** - Field cleared/set to empty
3. **Pointer to value or populated slice** - Field set to specific value

This pattern is the **industry standard** used by major Go SDKs (google/go-github, hashicorp/go-tfe, AWS SDK) and is essential for:
- **Partial updates** - Only changed fields sent to API
- **Supporting Terraform's `ignore_changes`** - Critical for Terraform provider development
- **Clear semantics** - Explicit distinction between clearing vs not updating array fields
- **Minimal API calls** - Efficient resource management

### Quick Examples

```go
// Update only name, leave other fields unchanged
update := &contextforge.ResourceUpdate{
    Name: contextforge.String("new-name"),
    // Description is nil - won't be sent to API
    // Tags is nil - won't be sent to API
}

// Clear description and all tags
update := &contextforge.ResourceUpdate{
    Description: contextforge.String(""),  // Sends empty string - clears description
    Tags: []string{},                       // Sends empty array - clears all tags
}

// Don't update tags vs clear all tags (critical distinction)
update1 := &contextforge.ResourceUpdate{
    Name: contextforge.String("new-name"),
    Tags: nil,        // Tags field omitted - existing tags unchanged
}

update2 := &contextforge.ResourceUpdate{
    Name: contextforge.String("new-name"),
    Tags: []string{}, // Tags field sent as [] - clears all tags
}
```

### Implementation Details

All Update structs (`ResourceUpdate`, `ServerUpdate`, `PromptUpdate`, `AgentUpdate`, `TeamUpdate`) follow this pattern:

- **Scalar optional fields** - Use pointers with `omitempty` JSON tag
  - `nil` → field omitted from JSON
  - `&""` → field included as `""` (clears field)
  - `&"value"` → field included as `"value"`

- **Array optional fields** - Use direct slices with `omitempty` JSON tag
  - `nil` → field omitted from JSON (don't update)
  - `[]string{}` → field included as `[]` (clear all items)
  - `[]string{"a", "b"}` → field included with values (replace items)

- **Map optional fields** - Use direct maps with `omitempty` JSON tag
  - Same semantics as arrays

**Critical Fix in v0.6.3:** Restored `omitempty` to array fields in Update structs, enabling the ability to clear arrays by sending empty slices. Prior to v0.6.3, empty arrays were incorrectly omitted from requests.

### Documentation

- **[docs/three-state-system.md](docs/three-state-system.md)** - Comprehensive guide with examples, common pitfalls, testing strategies
- **[docs/terraform-provider-usage.md](docs/terraform-provider-usage.md)** - How to use SDK in Terraform providers with Plugin Framework
- **[contextforge/pointers.go](contextforge/pointers.go)** - Package-level documentation and helper functions

### Why This Matters

When building a Terraform provider (or any declarative tool), you need to distinguish between:

1. **User removed field from config** → Don't update (send nil)
2. **User set field to empty** → Clear field (send empty value)
3. **User set field to value** → Update field (send value)

The three-state system provides the **mechanism** at the SDK level. Your provider code implements the **policy** by conditionally setting fields to non-nil only when changed. This is how terraform-provider-github, terraform-provider-aws, and other major providers handle partial updates.

## MCP Protocol vs REST API Endpoints

ContextForge implements **TWO SEPARATE APIs** with different purposes:

### 1. REST API (Management Operations) - IMPLEMENT IN SDK

Traditional REST endpoints for CRUD operations on ContextForge entities:
- **Purpose**: Manage tools, resources, servers, gateways, prompts, agents, teams
- **Authentication**: JWT bearer tokens
- **Format**: Standard REST with JSON request/response bodies
- **Examples**: `GET /tools`, `POST /resources`, `PUT /servers/{id}`, `DELETE /gateways/{id}`
- **Response format**: Management schemas with metadata (`createdAt`, `updatedAt`, `team`, `visibility`)
- **These should be implemented in the SDK**

### 2. JSON-RPC API (MCP Protocol) - DO NOT IMPLEMENT

Single `/rpc` endpoint implementing MCP spec JSON-RPC protocol:
- **Purpose**: MCP protocol compliance for client communication
- **Endpoint**: `POST /rpc` (single endpoint for all JSON-RPC methods)
- **Format**: JSON-RPC 2.0 messages
- **Methods**: `initialize`, `tools/list`, `tools/call`, `resources/list`, `resources/read`, `prompts/list`, `prompts/get`, `ping`
- **Example request**:
  ```json
  POST /rpc
  {"jsonrpc": "2.0", "id": 1, "method": "resources/read", "params": {"uri": "..."}}
  ```
- **This should NOT be implemented in the SDK**

### 3. Hybrid REST Endpoints - IMPLEMENT IN SDK

Some REST endpoints return MCP-compatible data formats while using standard REST patterns:
- **Purpose**: Provide MCP functionality via REST for convenience
- **Access pattern**: Standard HTTP GET/POST on resource paths (NOT JSON-RPC)
- **Response format**: MCP-compatible (e.g., `{type: "resource", uri: "...", text: "..."}`)
- **Examples**:
  - `GET /resources/{id}` - Returns resource content in MCP format
  - `POST /prompts/{id}` - Returns rendered prompt (MCP prompts/get semantics via REST)
  - `GET /prompts/{id}` - Returns prompt without arguments
- **These ARE REST endpoints and SHOULD be implemented in the SDK**

### Identifying Endpoint Types

**REST Management Endpoints (Implement in SDK):**
- Standard HTTP methods (GET, POST, PUT, DELETE) on resource paths
- Returns typed schemas: `ToolRead`, `ResourceRead`, `ServerRead`, etc.
- Includes management metadata: `createdAt`, `updatedAt`, `team`, `visibility`
- CRUD operations described in endpoint documentation

**JSON-RPC Protocol Endpoints (Do NOT implement):**
- Single `POST /rpc` endpoint
- Request body contains `{"jsonrpc": "2.0", "method": "...", "params": {...}}`
- Implements MCP spec methods (e.g., `resources/read`, `tools/call`)

**SSE/Event Streaming Endpoints (Do NOT implement):**
- Server-Sent Events for real-time notifications
- Endpoint paths containing `/sse`, `/subscribe/`, `/message`
- Long-lived connections, not request/response
- Examples:
  - `POST /resources/subscribe/{id}` - SSE streaming for resource change notifications
  - `GET /servers/{id}/sse` - SSE connection for MCP protocol proxying
  - `POST /servers/{id}/message` - JSON-RPC message relay for SSE sessions

### Examples

**REST Management Endpoints (Implemented in SDK):**
```
GET /resources              - List resources with metadata
GET /resources/{id}         - Read resource content (hybrid endpoint)
POST /resources             - Create resource
PUT /resources/{id}         - Update resource
DELETE /resources/{id}      - Delete resource
POST /resources/{id}/toggle - Toggle active status
GET /prompts/{id}           - Get prompt without arguments (hybrid endpoint)
POST /prompts/{id}          - Get prompt with arguments (hybrid endpoint)
```

**JSON-RPC Protocol Endpoints (NOT in SDK):**
```
POST /rpc                   - MCP JSON-RPC protocol endpoint
  Methods: resources/read, resources/list, tools/call, prompts/get, etc.
```

**SSE/Streaming Endpoints (NOT in SDK):**
```
POST /resources/subscribe/{id} - SSE streaming for change notifications
GET /servers/{id}/sse          - SSE connection for MCP protocol transport
POST /servers/{id}/message     - JSON-RPC message relay
```

### Service Documentation Pattern

Each service file should document excluded endpoints clearly:

```go
// ResourcesService handles communication with the resource-related
// methods of the ContextForge API.
//
// Note: This service intentionally excludes certain endpoints:
// - POST /resources/subscribe/{id} - SSE streaming for real-time change notifications
// The SSE endpoint is for event streaming, not REST API management.
//
// The /rpc endpoint handles MCP JSON-RPC protocol (resources/read, etc.)
// which is separate from these REST management endpoints.
```
