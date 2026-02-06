# ContextForge Bug: Prompts Toggle Returns 400 Instead of 404 for Non-Existent Prompts

**Bug ID:** CONTEXTFORGE-003
**Component:** ContextForge MCP Gateway - Prompts API
**Affected Version:** v0.8.0, v1.0.0-BETA-1 (fixed in v1.0.0-BETA-2)
**Severity:** Low
**Status:** FIXED in v1.0.0-BETA-2
**Reported:** 2025-11-09
**Last Validated:** 2026-02-06

## Summary

The `POST /prompts/{prompt_id}/toggle` endpoint returns HTTP 400 Bad Request when attempting to toggle a non-existent prompt, instead of the expected 404 Not Found. This is inconsistent with other prompts endpoints and standard REST conventions.

## Affected Endpoint

```
POST /prompts/{prompt_id}/toggle?activate={true|false}
```

## Expected Behavior

When calling the toggle endpoint with a non-existent prompt ID:
- The API should return **404 Not Found**
- The response should indicate the prompt was not found
- Consistent with other endpoints (GET, PUT, DELETE)

This follows standard REST conventions where:
- 400 = Client sent invalid/malformed request
- 404 = Resource not found

## Actual Behavior

When calling the toggle endpoint with a non-existent prompt ID:
- The API returns **400 Bad Request** instead of 404
- The error indicates the prompt was not found (correct error, wrong code)

## Reproduction Steps

1. Call toggle endpoint with non-existent ID:
   ```
   POST /prompts/99999999/toggle?activate=true
   ```
2. Receive 400 Bad Request response

**Expected:** 404 Not Found
**Actual:** 400 Bad Request

## Evidence from Integration Tests

```
=== RUN   TestPromptsService_ErrorHandling/toggle_non-existent_prompt_returns_404
    prompts_integration_test.go:556: Expected 404 Not Found, got 400
    prompts_integration_test.go:558: Correctly received 404 for non-existent prompt
--- FAIL: TestPromptsService_ErrorHandling/toggle_non-existent_prompt_returns_404 (0.00s)
```

The test expects 404 but receives 400, confirming the inconsistency.

## API Inconsistency Analysis

### Prompts Endpoints Comparison

| Endpoint | Non-Existent Resource | Status Code | Consistent? |
|----------|----------------------|-------------|-------------|
| `GET /prompts/{id}` | N/A (MCP endpoint) | N/A | N/A |
| `PUT /prompts/{id}` | Update non-existent | **404** ✓ | Yes |
| `DELETE /prompts/{id}` | Delete non-existent | **404** ✓ | Yes |
| `POST /prompts/{id}/toggle` | Toggle non-existent | **400** ✗ | **No** |

The toggle endpoint is inconsistent with other prompts endpoints.

### Cross-Resource Comparison

| Resource | Toggle Non-Existent | Status Code |
|----------|---------------------|-------------|
| Tools | `POST /tools/{id}/toggle` | Unknown |
| Resources | `POST /resources/{id}/toggle` | Unknown |
| Servers | `POST /servers/{id}/toggle` | Unknown |
| Prompts | `POST /prompts/{id}/toggle` | **400** |

## Root Cause Analysis

### Location
**File:** `mcpgateway/main.py`
**Endpoint:** `@prompt_router.post("/{prompt_id}/toggle")`
**Lines:** ~2690-2723

### Error Handling

The toggle endpoint likely catches `PromptNotFoundError` but returns it as a 400 instead of 404:

```python
@prompt_router.post("/{prompt_id}/toggle")
async def toggle_prompt_status(
    prompt_id: int,
    activate: bool = True,
    db: Session = Depends(get_db),
    user=Depends(get_current_user_with_permissions),
) -> Dict[str, Any]:
    try:
        prompt = await prompt_service.toggle_prompt_status(db, prompt_id, activate)
        return {
            "status": "success",
            "message": f"Prompt {prompt_id} {'activated' if activate else 'deactivated'}",
            "prompt": prompt.model_dump(),
        }
    except PromptNotFoundError as e:
        # ⚠️ Likely raises HTTPException(400) instead of HTTPException(404)
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
```

### Service Layer

The service correctly raises `PromptNotFoundError`:

**File:** `mcpgateway/services/prompt_service.py:847-899`

```python
async def toggle_prompt_status(self, db: Session, prompt_id: int, activate: bool) -> PromptRead:
    try:
        prompt = db.get(DbPrompt, prompt_id)
        if not prompt:
            raise PromptNotFoundError(f"Prompt not found: {prompt_id}")  # ✓ Correct
        # ...
```

The issue is in how the endpoint handles this exception.

### Comparison: Working Endpoints

Other prompts endpoints correctly return 404:

**DELETE endpoint (presumed):**
```python
@prompt_router.delete("/{prompt_id}")
async def delete_prompt(...):
    try:
        await prompt_service.delete_prompt(db, prompt_id)
    except PromptNotFoundError as e:
        raise HTTPException(status_code=404, detail=str(e))  # ✓ Correct
```

**UPDATE endpoint (presumed):**
```python
@prompt_router.put("/{prompt_id}")
async def update_prompt(...):
    try:
        prompt = await prompt_service.update_prompt(...)
    except PromptNotFoundError as e:
        raise HTTPException(status_code=404, detail=str(e))  # ✓ Correct
```

## HTTP Status Code Standards

According to RFC 9110 (HTTP Semantics):

**400 Bad Request:**
> The server cannot or will not process the request due to something that is perceived to be a **client error** (e.g., malformed request syntax, invalid request message framing, or deceptive request routing).

Use cases:
- Invalid JSON syntax
- Missing required fields
- Invalid parameter format
- Constraint violations

**404 Not Found:**
> The origin server did not find a current representation for the **target resource** or is not willing to disclose that one exists.

Use cases:
- Resource ID does not exist
- Resource has been deleted
- Resource never existed

**Correct Usage:**
- Prompt ID 99999999 is a valid integer (not a client error)
- The request is well-formed (not a client error)
- The resource simply doesn't exist (should be 404)

## Impact

**Severity: Low**

**Positive Impacts:**
- None (incorrect status code provides no benefit)

**Negative Impacts:**
- Inconsistent with REST conventions
- Inconsistent with other prompts endpoints
- Confusing for API consumers
- Makes client-side error handling more complex
- Violates principle of least surprise

**Client Code Impact:**

Without this bug, clients could handle errors uniformly:
```go
_, _, err := client.Prompts.Toggle(ctx, promptID, true)
if err != nil {
    if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
        switch apiErr.Response.StatusCode {
        case 404:
            return fmt.Errorf("prompt not found")
        case 400:
            return fmt.Errorf("invalid request")
        }
    }
}
```

With this bug, clients must special-case toggle:
```go
// Special handling for toggle endpoint
if strings.Contains(err.Error(), "not found") {
    // It's really a 404, even though API says 400
}
```

## Recommended Solution

Update the toggle endpoint error handling to return 404:

```python
@prompt_router.post("/{prompt_id}/toggle")
async def toggle_prompt_status(
    prompt_id: int,
    activate: bool = True,
    db: Session = Depends(get_db),
    user=Depends(get_current_user_with_permissions),
) -> Dict[str, Any]:
    try:
        prompt = await prompt_service.toggle_prompt_status(db, prompt_id, activate)
        return {
            "status": "success",
            "message": f"Prompt {prompt_id} {'activated' if activate else 'deactivated'}",
            "prompt": prompt.model_dump(),
        }
    except PromptNotFoundError as e:
        # ✓ FIXED: Return 404 instead of 400
        raise HTTPException(status_code=404, detail=str(e))
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
```

This makes the endpoint consistent with:
- Other prompts endpoints (PUT, DELETE)
- HTTP status code standards
- REST API best practices

## SDK Implementation Status

**Status:** ✅ SDK implementation is correct

Our go-contextforge SDK correctly:
- Sends the toggle request with proper prompt ID
- Handles error responses appropriately
- Returns `ErrorResponse` with actual status code
- Does not assume specific error codes

The SDK integration test documents the expected (404) vs actual (400) behavior for contract testing purposes.

## Workaround

SDK users can handle both error codes:

```go
_, _, err := client.Prompts.Toggle(ctx, promptID, true)
if err != nil {
    if apiErr, ok := err.(*contextforge.ErrorResponse); ok {
        // Accept both 400 and 404 as "not found"
        if apiErr.Response.StatusCode == 404 ||
           apiErr.Response.StatusCode == 400 {
            return fmt.Errorf("prompt not found: %w", err)
        }
    }
}
```

Or check error message content:
```go
if err != nil && strings.Contains(err.Error(), "not found") {
    return fmt.Errorf("prompt not found: %w", err)
}
```

## Related Issues

- CONTEXTFORGE-001: Prompts toggle returns stale state
- CONTEXTFORGE-002: Prompts API accepts empty template field

## References

- RFC 9110 (HTTP Semantics): https://www.rfc-editor.org/rfc/rfc9110.html
- REST API Design Best Practices: Status codes for resource not found
- SDK Integration Test: `test/integration/prompts_integration_test.go:548-560`
- ContextForge Source: `mcpgateway/main.py:2690-2723`

## Next Steps

1. Report this issue to the ContextForge team
2. Request fix to return 404 for consistency
3. Update SDK tests once fixed
4. Verify other resource toggle endpoints return correct codes

---

## v1.0.0-BETA-1 Validation Notes

**Validated:** 2026-01-13

The bug is **confirmed still present** in v1.0.0-BETA-1.

### Source Code Evidence

**File:** `mcpgateway/main.py:3270-3271`

```python
except Exception as e:
    raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail=str(e))
```

This catch-all exception handler returns 400 for ALL exceptions, including `PromptNotFoundError` which should return 404.

### Documentation Confirms Behavior

The docstring at line 3257 even documents this as intended behavior:

```
"emitted with *400 Bad Request* status"
```

### Code Location

The toggle endpoint is now at `main.py:3236-3271` (was ~2690-2723).

---

**Report Generated:** 2025-11-09
**Tested Against:** ContextForge v0.8.0
**Validated Against:** ContextForge v1.0.0-BETA-2
**Reporter:** go-contextforge SDK Team

---

## v1.0.0-BETA-2 Revalidation Notes

**Validated:** 2026-02-06

- **Still Valid?** No. The endpoint now returns `404` for non-existent prompts.
- **Is it actually a bug?** Yes. Returning `400` for missing resources was an HTTP contract bug.

### Evidence

- Runtime check on `POST /prompts/99999999/toggle?activate=true` now returns `404`.
- Current handler in `mcpgateway/main.py` maps this path to not-found behavior correctly.
