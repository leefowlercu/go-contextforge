# ContextForge Bug: Tag Filtering Returns Empty Results

**Bug ID:** CONTEXTFORGE-009
**Component:** ContextForge MCP Gateway
**Affected Version:** v1.0.0-BETA-1 (fixed in v1.0.0-BETA-2)
**Severity:** High
**Status:** FIXED in v1.0.0-BETA-2
**Reported:** 2026-01-12
**Last Validated:** 2026-02-06

## Summary

Filtering entities by tags using the `tags` query parameter returns empty results even when entities with matching tags exist. This affects tools, prompts, and servers list endpoints.

## Affected Endpoints

```
GET /tools?tags={tag_name}
GET /prompts?tags={tag_name}
GET /servers?tags={tag_name}
```

## Expected Behavior

When listing entities with a tag filter:
1. Create an entity with tags (e.g., `["filter-test", "integration"]`)
2. List entities with `?tags=filter-test`
3. The created entity should appear in the results

## Actual Behavior

When listing entities with a tag filter:
1. Create an entity with tags (tags are correctly stored)
2. List entities with `?tags=filter-test`
3. Empty results are returned, even though matching entities exist

## Reproduction Steps

### Tools Example

```bash
# Create a tool with tags
curl -X POST http://localhost:8000/tools \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "tool": {
      "name": "test-tool-tags",
      "description": "Test tool",
      "input_schema": {},
      "tags": ["filter-test", "integration"]
    }
  }'

# Response shows tags were saved:
# "tags": [{"id": "filter-test", "label": "filter-test"}, {"id": "integration", "label": "integration"}]

# List tools with tag filter - returns empty!
curl "http://localhost:8000/tools?tags=filter-test" \
  -H "Authorization: Bearer $TOKEN"

# Expected: List containing the created tool
# Actual: Empty array []
```

### Servers Example

```bash
# Create a server with tags
curl -X POST http://localhost:8000/servers \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "server": {
      "name": "test-server-tags",
      "tags": ["filter-test"]
    }
  }'

# List servers with tag filter - returns empty!
curl "http://localhost:8000/servers?tags=filter-test" \
  -H "Authorization: Bearer $TOKEN"

# Expected: List containing the created server
# Actual: Empty array []
```

## Evidence from Integration Tests

```
=== RUN   TestToolsService_Filtering/filter_by_tags
    tools_integration_test.go:342: Expected to find created tool in filtered results

=== RUN   TestPromptsService_Filtering/filter_by_tags
    prompts_integration_test.go:321: Expected to find created prompt in filtered results

=== RUN   TestServersService_Filtering/filter_by_tags
    servers_integration_test.go:388: Found 0 servers with tag filter

=== RUN   TestServersService_Filtering/combined_filters
    servers_integration_test.go:502: Found 0 servers with combined filters
```

## Affected SDK Tests

| Test File | Test Name | Status |
|-----------|-----------|--------|
| `tools_integration_test.go` | `filter_by_tags` | Skipped |
| `prompts_integration_test.go` | `filter_by_tags` | Skipped |
| `servers_integration_test.go` | `filter_by_tags` | Skipped |
| `servers_integration_test.go` | `combined_filters` | Skipped |

## Workaround

None available. Users must list all entities and filter client-side.

## Notes

- Tags ARE correctly stored when creating entities (verified by listing without filter)
- The issue is specifically with the tag filtering functionality
- This may be a regression in v1.0.0-BETA-1 as the feature likely worked in previous versions
- Combined filters (e.g., `?tags=x&visibility=public`) also fail when tags are involved
- Filtering by other parameters (e.g., `visibility`, `include_inactive`) works correctly

## Root Cause Analysis (Validated 2026-01-13)

This bug is **related to CONTEXTFORGE-007** (Gateway tags not persisted).

### The Connection

Both bugs share the same root cause: **type mismatch in tag validation/storage**.

**File:** `mcpgateway/validation/tags.py:218-265`

```python
def validate_tags_field(tags: Optional[List[str]]) -> List[str]:
    # Signature claims to return List[str]
    # Actually returns List[Dict[str, str]]: [{"id": "x", "label": "x"}, ...]
    return TagValidator.validate_list(expanded_tags)
```

### Why Filtering Fails

1. Tags are stored in DB as `[{"id": "tag", "label": "tag"}, ...]` (dict format)
2. Filter query uses `json_contains_expr("tag")` to find string "tag"
3. SQLite/PostgreSQL JSON comparison looks for exact string match
4. String "tag" doesn't match dict `{"id": "tag", ...}` in JSON column
5. No results returned

### Filtering Implementation

**File:** `mcpgateway/utils/sqlalchemy_modifier.py`

```python
def json_contains_expr(column: Column, search_value: Any) -> Any:
    # For SQLite: func.json_each(column).c.value == search_value
    # For PostgreSQL: column.op("@>")(...)
    # Both expect storage format to match search format
```

The `json_contains_expr` correctly implements JSON containment checks, but assumes tags are stored as strings, not dicts.

### Fix

Fix CONTEXTFORGE-007 (tag storage format) and this will be resolved automatically.

---

## v1.0.0-BETA-2 Revalidation Notes

**Validated:** 2026-02-06

- **Still Valid?** No. Tag filtering now returns matching results.
- **Is it actually a bug?** Yes. Returning empty lists for matching tag filters was a functional bug.

### Evidence

- Runtime checks for tools/prompts/servers with `?tags=` now return created entities.
- `json_contains_tag_expr` in `mcpgateway/utils/sqlalchemy_modifier.py` supports current tag formats.
