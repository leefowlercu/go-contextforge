# AGENTS.md

Use this file as a task router. It points to focused agent docs derived from `CLAUDE.md`.

## Start Here

- Read `/Users/lee/Dev/leefowlercu/go-contextforge/docs/agents/project-overview.md` for SDK scope, principles, and non-goals.

## Task Router

- Implementing or changing a service:
  - `/Users/lee/Dev/leefowlercu/go-contextforge/docs/agents/architecture.md`
  - `/Users/lee/Dev/leefowlercu/go-contextforge/docs/agents/code-organization.md`
  - `/Users/lee/Dev/leefowlercu/go-contextforge/docs/agents/api-conventions.md`
- Working with request/response shapes, updates, or partial-field semantics:
  - `/Users/lee/Dev/leefowlercu/go-contextforge/docs/agents/api-conventions.md`
- Adding or fixing tests:
  - `/Users/lee/Dev/leefowlercu/go-contextforge/docs/agents/testing.md`
- Running build/test/lint/release commands:
  - `/Users/lee/Dev/leefowlercu/go-contextforge/docs/agents/development-workflow.md`
- Investigating known upstream API bugs:
  - `/Users/lee/Dev/leefowlercu/go-contextforge/docs/agents/testing.md`
  - `/Users/lee/Dev/leefowlercu/go-contextforge/docs/upstream-bugs/`

## Guardrails

- Implement only ContextForge REST management endpoints in this SDK.
- Do not implement MCP JSON-RPC (`/rpc`) methods or SSE streaming endpoints.
- Preserve three-state update semantics (`nil` omit, empty clear, value set).
- Follow existing `google/go-github`-style service patterns.
