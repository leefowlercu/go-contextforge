# API Conventions for Agents

## Request Body Wrapping

Some endpoints require wrapped payloads (example: `{"tool": {...}}`) while others do not.
Verify wrapping from existing service implementations and the upstream API schema/routers for the target tag.
This repository no longer maintains local `reference/` OpenAPI snapshots.

## Field Naming Inconsistencies

- Create payloads may use snake_case fields.
- Read/update payloads may use camelCase fields.
- Keep behavior aligned with existing model structs and JSON tags.

## Prompt ID Rules

- Prompt IDs are `string` in this SDK.
- Prompt service methods should accept `promptID string`.

## Three-State Semantics (Critical)

For optional update fields:

1. `nil` pointer/slice: omit from request (leave existing value).
2. Pointer to zero value or empty slice: clear/reset field.
3. Pointer to non-zero value or populated slice: set/update field.

This behavior is required for safe partial updates (especially Terraform-provider use cases).

## REST vs MCP Protocol

Implement only REST management endpoints in this SDK.

- Implement: resource management endpoints (tools/resources/gateways/servers/prompts/agents/teams).
- Do not implement: JSON-RPC methods under `/rpc`.
- Do not implement: SSE endpoints (`/servers/{id}/sse`, `/resources/subscribe/{id}`).
