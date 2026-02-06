# ContextForge Bug: Agent Bearer Auth Requires auth_token Field

**Bug ID:** CONTEXTFORGE-008
**Component:** ContextForge MCP Gateway
**Affected Version:** v1.0.0-BETA-1, v1.0.0-BETA-2
**Severity:** Medium
**Status:** Confirmed in v1.0.0-BETA-2 (still valid)
**Reported:** 2026-01-12
**Last Validated:** 2026-02-06

## Summary

When creating an agent with `auth_type: "bearer"`, the API requires the `auth_token` field instead of `auth_value`. The OpenAPI specification documents `auth_value` as the field for authentication credentials, but the API validation rejects this and requires `auth_token` specifically for bearer authentication.

## Affected Endpoint

```
POST /a2a (create agent)
```

## Expected Behavior

Based on the OpenAPI specification, creating an agent with bearer authentication should work with:
```json
{
  "agent": {
    "name": "my-agent",
    "endpoint_url": "https://example.com/a2a",
    "auth_type": "bearer",
    "auth_value": "my-secret-token"
  }
}
```

## Actual Behavior

The API returns a 422 validation error:
```json
{
  "detail": [
    {
      "type": "value_error",
      "loc": ["body", "agent", "auth_value"],
      "msg": "Value error, For 'bearer' auth, 'auth_token' must be provided.",
      "input": "my-secret-token",
      "ctx": {"error": {}}
    }
  ]
}
```

## Reproduction Steps

```bash
curl -X POST http://localhost:8000/a2a \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "agent": {
      "name": "test-agent",
      "endpoint_url": "https://example.com/a2a",
      "auth_type": "bearer",
      "auth_value": "test-token"
    }
  }'
```

**Expected:** Agent created successfully
**Actual:** 422 Validation Error

### Workaround

Use `auth_token` instead of `auth_value` for bearer authentication:
```bash
curl -X POST http://localhost:8000/a2a \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "agent": {
      "name": "test-agent",
      "endpoint_url": "https://example.com/a2a",
      "auth_type": "bearer",
      "auth_token": "test-token"
    }
  }'
```

## SDK Impact

The SDK's `AgentCreate` struct uses `auth_value` field based on the OpenAPI specification:
```go
type AgentCreate struct {
    AuthType  *string `json:"auth_type,omitempty"`
    AuthValue *string `json:"auth_value,omitempty"` // API rejects this for bearer auth
}
```

To fix, the SDK would need to add `auth_token` field or the API would need to accept `auth_value` for bearer auth.

## Evidence from Integration Tests

```
=== RUN   TestAgentsService_EdgeCases/create_agent_with_authentication
    agents_integration_test.go:580: Failed to create agent with auth: POST http://localhost:8000/a2a; 422
```

## Affected SDK Tests

| Test File | Test Name | Status |
|-----------|-----------|--------|
| `agents_integration_test.go` | `create_agent_with_authentication` | Skipped |

## Notes

- The API appears to have different field requirements for different auth types
- The response object includes both `authValue` and `authToken` fields, suggesting the API supports multiple auth mechanisms
- This may be intentional API design, but it contradicts the OpenAPI specification for `A2AAgentCreate`

## Root Cause Analysis (Validated 2026-01-13)

Source code confirms this is intentional but poorly documented.

### Evidence

**File:** `mcpgateway/schemas.py:4104-4107`

```python
# In A2AAgentCreate model_validator
token = data.get("auth_token")
if not token:
    raise ValueError("For 'bearer' auth, 'auth_token' must be provided.")
```

### Schema Definition

The schema has BOTH fields defined:

**File:** `mcpgateway/schemas.py:3911 and 3920`

```python
auth_token: Optional[str] = Field(None, description="Token for bearer authentication")
# ...
auth_value: Optional[str] = Field(None, description="Alias for authentication value")
```

### Analysis

1. The schema accepts both `auth_token` and `auth_value`
2. The model validator ONLY checks `auth_token` for bearer auth
3. `auth_value` is described as an "alias" but isn't used in bearer auth validation
4. This is a validation logic inconsistency, not a schema error

### SDK Impact

The SDK uses `auth_value` based on OpenAPI spec. Options:
1. Add `auth_token` field to SDK's `AgentCreate` struct
2. Use `auth_token` instead of `auth_value` for bearer auth
3. Wait for upstream fix to accept either field

---

## v1.0.0-BETA-2 Revalidation Notes

**Validated:** 2026-02-06

- **Still Valid?** Yes. `auth_value` alone is still rejected for bearer auth while `auth_token` works.
- **Is it actually a bug?** Yes. It is an API contract mismatch between documented/typed fields and runtime validation.

### Evidence

- `POST /a2a` with `auth_type=bearer` + `auth_value` returns validation error.
- Same request with `auth_token` succeeds.
- Validator in `mcpgateway/schemas.py` still checks only `auth_token` for bearer mode.
