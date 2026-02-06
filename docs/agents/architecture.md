# Architecture Reference for Agents

## SDK Shape

The SDK uses a service-oriented architecture:

- One `Client` owns HTTP behavior, auth, and rate-limit tracking.
- Resource-specific services are attached once and reused for client lifetime.
- Service methods are context-first and return typed values plus `*Response`.

## Client Responsibilities

`/Users/lee/Dev/leefowlercu/go-contextforge/contextforge/contextforge.go`:

- `NewClient(httpClient, address, bearerToken)` validates and normalizes base URL.
- `NewRequest(...)` builds authenticated requests.
- `Do(...)` executes requests, checks errors, and tracks rate limits.

## Shared Types and Helpers

- `FlexibleID`: handles numeric-or-string ID responses.
- `Timestamp`: parses API timestamps that may be non-standard.
- `Tag`: marshaling/unmarshaling compatibility for tag payload differences.
- Pointer helpers enable three-state update semantics.

Files:

- `/Users/lee/Dev/leefowlercu/go-contextforge/contextforge/types.go`
- `/Users/lee/Dev/leefowlercu/go-contextforge/contextforge/pointers.go`

## Pagination Patterns

- Cursor-based list endpoints: `ListOptions` (`Limit`, `Cursor`), next cursor from `X-Next-Cursor`.
- Offset-based list endpoints (agents/teams): endpoint-specific options (`Skip`, `Limit`).

## Error and Rate-Limit Handling

- Standard API failures return `ErrorResponse`.
- HTTP `429` maps to `RateLimitError`.
- Rate-limit headers parsed: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`.
- URL sanitization avoids leaking tokens in error paths.
