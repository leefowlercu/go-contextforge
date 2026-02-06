# ContextForge Bug: Team ID Filter Returns Permission Error

**Bug ID:** CONTEXTFORGE-010
**Component:** ContextForge MCP Gateway
**Affected Version:** v1.0.0-BETA-1, v1.0.0-BETA-2 (partial)
**Severity:** Medium
**Status:** PARTIALLY VALID in v1.0.0-BETA-2
**Reported:** 2026-01-12
**Last Validated:** 2026-02-06

## Summary

Filtering tools by `team_id` returns a 403 permission error instead of an empty result set or the expected filtered results. The API should either return matching tools or an empty array, not a permission error for a valid filter query.

## Affected Endpoint

```
GET /tools?team_id={team_id}
```

## Expected Behavior

When listing tools filtered by team_id:
1. If the user has access to the team: return tools belonging to that team
2. If the user doesn't have access to the team: return empty results or 403
3. If the team doesn't exist: return empty results

## Actual Behavior

When listing tools filtered by team_id:
- Returns 403 with message: "Access issue: This API token does not have the required permissions for this team."
- This occurs even when testing with an admin token
- This occurs even for non-existent team IDs

## Reproduction Steps

```bash
# Create a tool with a team_id
curl -X POST http://localhost:8000/tools \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "tool": {
      "name": "test-tool",
      "description": "Test tool",
      "input_schema": {},
      "team_id": "test-team-integration"
    }
  }'

# List tools filtered by team_id - returns 403!
curl "http://localhost:8000/tools?team_id=test-team-integration" \
  -H "Authorization: Bearer $TOKEN"
```

**Expected:** List of tools or empty array
**Actual:**
```json
{
  "message": "Access issue: This API token does not have the required permissions for this team."
}
```

## Evidence from Integration Tests

```
=== RUN   TestToolsService_Filtering/filter_by_team_id
    tools_integration_test.go:408: Failed to list tools: GET http://localhost:8000/tools?team_id=test-team-integration; 403 Access issue: This API token does not have the required permissions for this team.
```

## Affected SDK Tests

| Test File | Test Name | Status |
|-----------|-----------|--------|
| `tools_integration_test.go` | `filter_by_team_id` | Skipped |

## Workaround

None available. Users cannot filter tools by team_id.

## Notes

- The error message suggests this is intentional permission checking
- However, the same token can create tools with team_id without issues
- This may be a misconfiguration in the test environment or an API bug
- Other filter parameters (visibility, include_inactive) work correctly
- The team_id used in the filter may need to be a valid, existing team ID

## Validation Notes (2026-01-13)

Source code analysis did not find definitive evidence of the root cause.

### Hypothesis

The error message indicates permission service logic is rejecting the request. Possible causes:

1. **Invalid team_id format**: The filter may require a valid UUID format
2. **RBAC team scoping**: The permission service may be checking team membership before returning results
3. **Test environment issue**: The test may be using a non-existent or invalid team ID

### Further Investigation Needed

This bug requires runtime debugging to identify the exact code path that generates the 403 error. The permission checking logic in `mcpgateway/services/permission_service.py` handles team scoping but the specific error path was not identified in static analysis.

---

## v1.0.0-BETA-2 Revalidation Notes

**Validated:** 2026-02-06

- **Still Valid?** Partially. Some `403` responses are expected for unauthorized team filters, but there is still an upstream authorization bug for multi-team tokens.
- **Is it actually a bug?** Partially. The blanket claim is too broad, but there is a real defect in team selection logic.

### Evidence

- `main.list_tools` still compares requested `team_id` to singular `request.state.team_id`, which can reject valid team_ids present in token team scopes.
- Unauthorized team filters returning `403` remains expected behavior.
- This report should track the narrower multi-team authorization defect rather than all `403` outcomes.
