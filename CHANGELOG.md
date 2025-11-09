# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2025-11-09

### Added
- Initial SDK implementation with service-oriented architecture
- `ToolsService` for managing MCP tools
- `ResourcesService` for managing MCP resources
- `GatewaysService` for managing MCP gateways
- `ServersService` for managing MCP servers with full CRUD operations
- `ServersService.ListTools()`, `ListResources()`, `ListPrompts()` for server associations
- `PromptsService` for managing MCP prompts with full CRUD operations
- Support for prompt templates with arguments
- Comprehensive integration tests for servers and prompts
- Upstream bug documentation in `docs/upstream-bugs/`
- Semi-automated release workflow with version management
- `contextforge/version.go` with `Version` constant
- CHANGELOG.md for tracking releases
- `Client` with JWT authentication support
- Rate limit tracking per endpoint
- Cursor-based pagination support
- Custom types: `FlexibleID`, `Timestamp` with pointer helpers
- Comprehensive error handling with `ErrorResponse` and `RateLimitError`
- `NewClient()` factory function for default configuration
- `NewClientWithBaseURL()` factory function for custom base URLs
- Complete unit test suite
- Integration test suite with build tag
- Makefile with build, test, and CI targets
