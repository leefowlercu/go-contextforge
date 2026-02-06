# Project Overview for Agents

## What This Repository Is

`go-contextforge` is a Go SDK for the IBM ContextForge MCP Gateway REST API.  
It provides CRUD and related operations for tools, resources, gateways, servers, prompts, agents, and teams.

## Core Design Principles

1. Follow `google/go-github` service-oriented SDK patterns.
2. Preserve three-state semantics for optional update fields.
3. Use context-first method signatures (`context.Context` first).
4. Prefer explicit types for API inconsistencies.
5. Keep test coverage strong with unit and integration tests.

## Important Scope Boundary

- In scope: REST management API endpoints.
- Out of scope: MCP JSON-RPC protocol methods and SSE transport endpoints.

## Key Files to Know

- `/Users/lee/Dev/leefowlercu/go-contextforge/contextforge/contextforge.go` (client, request execution)
- `/Users/lee/Dev/leefowlercu/go-contextforge/contextforge/types.go` (core SDK types)
- `/Users/lee/Dev/leefowlercu/go-contextforge/contextforge/pointers.go` (pointer helpers)
- `/Users/lee/Dev/leefowlercu/go-contextforge/contextforge/errors.go` (error types and response checks)
