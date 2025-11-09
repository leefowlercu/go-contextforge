# ContextForge Bug: Prompts Toggle Endpoint Returns Stale State

**Bug ID:** CONTEXTFORGE-001
**Component:** ContextForge MCP Gateway
**Affected Version:** v0.8.0
**Severity:** Medium
**Status:** Confirmed
**Reported:** 2025-11-09

## Summary

The `POST /prompts/{prompt_id}/toggle` endpoint does not reliably return the updated `isActive` state after toggling a prompt. When toggling a prompt from inactive to active (`activate=true`), the response returns the old state (`isActive=false`) instead of the new state (`isActive=true`).

## Affected Endpoint

```
POST /prompts/{prompt_id}/toggle?activate={true|false}
```

## Expected Behavior

When calling the toggle endpoint with `activate=true`:
1. The prompt's `is_active` field should be updated in the database to `true`
2. The response should return the updated prompt with `isActive: true`

## Actual Behavior

When calling the toggle endpoint with `activate=true`:
1. The prompt's `is_active` field IS updated in the database to `true` ✅
2. The response returns the OLD state with `isActive: false` ❌

## Reproduction Steps

1. Create a new prompt (starts as `isActive=true`)
2. Toggle to inactive: `POST /prompts/{id}/toggle?activate=false`
   - Response correctly shows `isActive=false` ✅
3. Toggle back to active: `POST /prompts/{id}/toggle?activate=true`
   - Response INCORRECTLY shows `isActive=false` ❌
4. List prompts with `include_inactive=true`
   - Shows prompt with `isActive=true` ✅ (database state is correct)

**Conclusion:** The database state is correct, but the API response returns stale/cached data.

## Evidence from Integration Tests

Our SDK integration tests consistently fail on this scenario:

```
=== RUN   TestPromptsService_Toggle/toggle_inactive_to_active
    prompts_integration_test.go:220: Expected prompt to be active after toggle(true), got isActive=false
--- FAIL: TestPromptsService_Toggle/toggle_inactive_to_active (0.01s)
```

The test verifies the response from the toggle endpoint immediately after the call, confirming the bug.

## Root Cause Analysis

### Location
**File:** `mcpgateway/services/prompt_service.py`
**Method:** `toggle_prompt_status()`
**Lines:** 847-899

### Code Analysis

```python
async def toggle_prompt_status(self, db: Session, prompt_id: int, activate: bool) -> PromptRead:
    try:
        prompt = db.get(DbPrompt, prompt_id)
        if not prompt:
            raise PromptNotFoundError(f"Prompt not found: {prompt_id}")

        if prompt.is_active != activate:
            prompt.is_active = activate          # Update state
            prompt.updated_at = datetime.now(timezone.utc)
            db.commit()                          # Commit to database
            db.refresh(prompt)                   # Refresh object from DB
            # ... notifications ...

        prompt.team = self._get_team_name(db, prompt.team_id)
        return PromptRead.model_validate(self._convert_db_prompt(prompt))  # ⚠️ PROBLEM
```

### Issue: SQLAlchemy Session State

The `_convert_db_prompt()` method reads from the `db_prompt` object:

```python
def _convert_db_prompt(self, db_prompt: DbPrompt) -> Dict[str, Any]:
    # Line 240
    return {
        "is_active": db_prompt.is_active,  # Reads from potentially stale object
        # ...
    }
```

Despite calling `db.refresh(prompt)` on line 889, the object state may still be cached when `_convert_db_prompt()` is called on line 896. This is a known SQLAlchemy behavior where:

1. `db.refresh()` updates the object's state
2. BUT subsequent attribute accesses may hit the SQLAlchemy identity map cache
3. The cache may not be properly expired after the team name lookup on line 895

### Comparison: Working Implementation (Servers)

The **servers toggle** endpoint works correctly because it manually constructs the response dict AFTER the refresh:

**File:** `mcpgateway/services/server_service.py:863-929`

```python
async def toggle_server_status(self, db: Session, server_id: str, activate: bool) -> ServerRead:
    # ... same pattern: get, check, update, commit, refresh ...

    if server.is_active != activate:
        server.is_active = activate
        server.updated_at = datetime.now(timezone.utc)
        db.commit()
        db.refresh(server)
        # ...

    # ✅ Manually builds dict AFTER refresh, ensuring fresh state
    server_data = {
        "id": server.id,
        "name": server.name,
        # ...
        "is_active": server.is_active,  # Guaranteed fresh from refresh
        # ...
    }
    return ServerRead.model_validate(server_data)
```

The key difference: servers builds the response dictionary inline AFTER the refresh, while prompts calls a helper method that may read cached state.

## Proposed Solutions

### Solution 1: Expire Session State (Minimal Change)

Add explicit session expiry before reading the prompt state:

```python
# Line 895 - BEFORE _convert_db_prompt
db.expire(prompt)  # Force SQLAlchemy to reload from DB
prompt.team = self._get_team_name(db, prompt.team_id)
return PromptRead.model_validate(self._convert_db_prompt(prompt))
```

### Solution 2: Additional Refresh (Safe)

Add another `db.refresh()` right before the return:

```python
# Line 895
prompt.team = self._get_team_name(db, prompt.team_id)
db.refresh(prompt)  # Refresh again to ensure latest state
return PromptRead.model_validate(self._convert_db_prompt(prompt))
```

### Solution 3: Manual Dict Construction (Best)

Follow the servers pattern and build the dict manually:

```python
# Replace _convert_db_prompt with inline dict construction
# (Similar to server_service.py:912-929)
prompt_data = {
    "id": prompt.id,
    "name": prompt.name,
    "description": prompt.description,
    "template": prompt.template,
    "is_active": prompt.is_active,  # Fresh from refresh
    # ... other fields ...
}
return PromptRead.model_validate(prompt_data)
```

This ensures the state is read immediately after refresh without any intervening operations.

## API Inconsistency Note

There's also an inconsistency in toggle endpoint response formats across the API:

- **Servers:** Returns `ServerRead` directly (unwrapped)
- **Tools, Resources, Prompts:** Return wrapped format `{"status": "success", "message": "...", "<entity>": {...}}`

While not the cause of this bug, this inconsistency should be addressed for API uniformity.

## Impact

**Severity: Medium**

- ✅ Database state is correctly updated
- ✅ Subsequent API calls return correct state
- ❌ Toggle endpoint response shows stale state
- ❌ Clients relying on immediate response will see incorrect state

**Affected Clients:**
- Any SDK or client that relies on the toggle endpoint response to update UI state
- Automation scripts that chain operations based on toggle response

## Workaround

SDK users can work around this by:

1. **Option 1:** Ignore the toggle response and immediately fetch the prompt:
   ```go
   client.Prompts.Toggle(ctx, promptID, true)
   prompt, _, _ := client.Prompts.List(ctx, &contextforge.PromptListOptions{
       IncludeInactive: true,
   })
   // Find prompt in list to get fresh state
   ```

2. **Option 2:** Use the Update endpoint instead:
   ```go
   client.Prompts.Update(ctx, promptID, &contextforge.PromptUpdate{
       // No need to set isActive - it's read-only in update
   })
   ```

3. **Option 3:** Trust that database state is correct and assume success if no error returned

## SDK Implementation Status

**Status:** ✅ SDK implementation is correct

Our go-contextforge SDK correctly:
- Sends the toggle request with proper `activate` parameter
- Parses the wrapped response format `{"status": "success", "prompt": {...}}`
- Extracts the prompt data from the nested structure

The SDK integration test failure is expected given the ContextForge bug. All SDK unit tests pass.

## Related Issues

- None known

## References

- ContextForge Source: `mcpgateway/services/prompt_service.py:847-899`
- Working Implementation: `mcpgateway/services/server_service.py:863-929`
- SDK Integration Test: `test/integration/prompts_integration_test.go:196-223`
- OpenAPI Spec: `reference/contextforge-openapi-v0.8.0.json`

## Next Steps

1. Report this issue to the ContextForge team
2. Request fix in next release
3. Update SDK tests to document expected behavior once fixed
4. Consider adding workaround documentation to SDK README

---

**Report Generated:** 2025-11-09
**Tested Against:** ContextForge v0.8.0
**Reporter:** go-contextforge SDK Team
