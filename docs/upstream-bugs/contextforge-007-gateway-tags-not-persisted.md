# ContextForge Bug: Gateway Tags Not Persisted

**Bug ID:** CONTEXTFORGE-007
**Component:** ContextForge MCP Gateway
**Affected Version:** v1.0.0-BETA-1
**Severity:** Medium
**Status:** Confirmed
**Reported:** 2026-01-12

## Summary

Tags provided when creating or updating a gateway are not persisted. The API accepts the request without error but returns the gateway with an empty tags array.

## Affected Endpoints

```
POST /gateways (create)
PUT /gateways/{gateway_id} (update)
```

## Expected Behavior

When creating or updating a gateway with tags:
1. The tags should be stored in the database
2. The response should include the provided tags
3. Subsequent GET requests should return the gateway with its tags

## Actual Behavior

When creating or updating a gateway with tags:
1. The API accepts the request without error
2. The response returns an empty tags array `[]`
3. Tags are not persisted in the database

## Reproduction Steps

### Create with Tags

```bash
curl -X POST http://localhost:8000/gateways \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-gateway",
    "url": "https://example.com/gateway",
    "tags": [{"id": "test", "label": "test"}]
  }'
```

**Expected Response (tags included):**
```json
{
  "id": "...",
  "name": "test-gateway",
  "tags": [{"id": "test", "label": "test"}]
}
```

**Actual Response (tags empty):**
```json
{
  "id": "...",
  "name": "test-gateway",
  "tags": []
}
```

### Update with Tags

```bash
curl -X PUT http://localhost:8000/gateways/{id} \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-gateway",
    "url": "https://example.com/gateway",
    "tags": [{"id": "updated", "label": "updated"}]
  }'
```

**Expected:** Tags should be updated
**Actual:** Tags remain empty

## Evidence from Integration Tests

```
=== RUN   TestGatewaysService_BasicCRUD/create_gateway_with_all_optional_fields
    gateways_integration_test.go:161: Expected 2 tags, got 0

=== RUN   TestGatewaysService_BasicCRUD/update_gateway
    gateways_integration_test.go:224: Expected tags [updated integration-test], got []
```

## Affected SDK Tests

| Test File | Test Name | Status |
|-----------|-----------|--------|
| `gateways_integration_test.go` | `create_gateway_with_all_optional_fields` | Skipped |
| `gateways_integration_test.go` | `update_gateway` | Skipped |

## Workaround

None available. Tags cannot be assigned to gateways in v1.0.0-BETA-1.

## Notes

- Other entity types (tools, prompts, servers, resources) correctly persist tags
- This appears to be specific to the gateways endpoint
- The API does not return an error, making this a silent failure

## Root Cause Analysis (Validated 2026-01-13)

Source code analysis reveals a **type mismatch bug** in the Pydantic schema validator.

### Evidence

**File:** `mcpgateway/schemas.py:2440-2451`

```python
@field_validator("tags")
@classmethod
def validate_tags(cls, v: Optional[List[str]]) -> List[str]:  # ← Wrong return type
    """Validate and normalize tags."""
    return validate_tags_field(v)  # Returns List[Dict[str, str]], not List[str]!
```

**File:** `mcpgateway/validation/tags.py:218-265`

```python
def validate_tags_field(tags: Optional[List[str]]) -> List[str]:  # Signature says List[str]
    # ...
    return TagValidator.validate_list(expanded_tags)  # Returns List[Dict[str, str]]
```

### The Bug

1. User submits `tags: ["test", "integration"]` (list of strings)
2. `validate_tags` calls `validate_tags_field()`
3. `validate_tags_field()` returns `[{"id": "test", "label": "test"}, ...]` (list of dicts)
4. Return type annotation lies - says `List[str]` but returns `List[Dict[str, str]]`
5. Database column expects `List[str]` (defined as `tags: Mapped[List[str]]`)
6. Storing dict objects in a string column causes data loss or silent failure

### Related Issue

This is also the root cause of **CONTEXTFORGE-009** (tag filtering empty results):
- If tags ARE somehow stored as dicts `[{"id": "x", ...}]`, the filter function looks for strings
- `json_contains_expr("x")` won't match `{"id": "x", "label": "x"}` in JSON column

### Fix Recommendation

Either:
1. Fix `validate_tags_field()` to return `List[str]` (just the tag IDs)
2. Or update the database schema to expect `List[Dict]`
3. Or update the gateway service to extract IDs before storage
