# ContextForge Bug: Teams API Returns 422 Instead of 400 for Validation Errors

**Bug ID:** CONTEXTFORGE-006
**Component:** ContextForge MCP Gateway
**Affected Version:** v0.8.0
**Severity:** Low
**Status:** Confirmed
**Reported:** 2025-11-09

## Summary

The `POST /teams` endpoint returns HTTP 422 (Unprocessable Entity) for request validation errors (like missing required fields) instead of the more standard HTTP 400 (Bad Request). While both status codes indicate client errors, 422 is semantically meant for well-formed requests with semantic errors, whereas 400 is more appropriate for malformed requests or missing required fields.

## Affected Endpoint

```
POST /teams
```

## Expected Behavior

When creating a team without a required field (e.g., `name`):
1. The API should return HTTP 400 Bad Request
2. The response should include error details about the missing field
3. This matches HTTP semantics where 400 = malformed request

Example:
```bash
POST /teams
{
  "description": "A team without a name"
}

Response: 400 Bad Request
{
  "detail": "name field is required"
}
```

## Actual Behavior

When creating a team without a required field:
1. The API returns HTTP 422 Unprocessable Entity
2. The response includes validation error details (FastAPI/Pydantic default)
3. The SDK's error handling doesn't capture the Response object correctly

Example:
```bash
POST /teams
{
  "description": "A team without a name"
}

Response: 422 Unprocessable Entity
{
  "detail": [
    {
      "type": "missing",
      "loc": ["body", "name"],
      "msg": "Field required",
      "input": {...}
    }
  ]
}
```

## Reproduction Steps

1. Create a team without the required `name` field:
   ```bash
   curl -X POST http://localhost:8000/teams/ \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"description": "No name provided"}'
   ```

2. Observe response status code is 422, not 400

3. SDK test expects 400:
   ```go
   _, resp, err := client.Teams.Create(ctx, team)
   if resp.StatusCode != http.StatusBadRequest {  // Expects 400
       t.Errorf("Expected 400, got %d", resp.StatusCode)  // Gets 422
   }
   ```

## Evidence from Integration Tests

SDK integration test shows the mismatch:

```
=== RUN   TestTeamsService_ErrorHandling/create_team_without_required_name_returns_400
    teams_integration_test.go:511: Expected status 400, got &{0x140003dc2d0  {0 0 0001-01-01 00:00:00 +0000 UTC}}
```

The Response object shows StatusCode of 0, which indicates the SDK's error handling might not be parsing 422 responses into the Response object correctly.

## Root Cause Analysis

### Primary Cause: FastAPI Default Behavior

**Location:** FastAPI/Pydantic validation in `mcpgateway/routers/teams.py`

FastAPI automatically validates request bodies against Pydantic models and returns **422 Unprocessable Entity** for validation errors by default. This is FastAPI's standard behavior:

```python
@teams_router.post("/", response_model=TeamResponse)
async def create_team(request: TeamCreateRequest, ...):
    # If TeamCreateRequest validation fails (missing required fields),
    # FastAPI returns 422 automatically before this function is called
    ...
```

**Pydantic Model:**
```python
class TeamCreateRequest(BaseModel):
    name: str  # Required field
    description: Optional[str] = None
    ...
```

When `name` is missing, Pydantic validation fails and FastAPI returns 422 with detailed validation error information.

### Secondary Issue: SDK Response Handling

The SDK's `Do()` method in `contextforge.go` may not be properly populating the Response object for 422 errors. The test output shows `StatusCode: 0` which suggests the Response might be nil or not fully constructed.

**Location:** `contextforge/contextforge.go` - `Do()` method

The SDK needs to ensure that even for validation errors (422), the Response object is properly constructed with the status code.

## HTTP Status Code Semantics

**422 Unprocessable Entity (RFC 4918):**
- Originally defined for WebDAV
- Means the request was well-formed but contains semantic errors
- FastAPI uses this for Pydantic validation failures
- Example: All fields present but value constraints violated

**400 Bad Request (RFC 7231):**
- General client error for malformed requests
- More appropriate for missing required fields
- Indicates the request itself is invalid
- Example: Missing required fields, invalid JSON syntax

For missing required fields, **400 is more semantically correct** than 422.

## API Consistency Check

Need to verify if other ContextForge endpoints use the same pattern:
- Do Tools, Resources, Servers, Prompts, Agents also return 422?
- Is this a consistent API-wide pattern?
- If consistent, should SDK accept both 400 and 422?

## Impact

**Severity: Low**

- ❌ API uses less common 422 status code
- ❌ Test expectations don't match actual behavior
- ❌ SDK Response object may not be populated correctly for 422
- ✅ Error is still properly returned (validation works)
- ✅ Error details are provided in response
- ✅ Workaround exists (check for 422 or any 4xx error)

**Affected Users:**
- SDK users expecting standard HTTP 400 for validation errors
- Automated tests checking specific status codes
- API consumers following strict HTTP semantics

**Business Impact:**
Minimal - The error is still returned and caught, just with a different status code than expected.

## Workaround

**Option 1: Accept Both 400 and 422**

Update SDK tests to accept either status code:

```go
if resp.StatusCode != http.StatusBadRequest &&
   resp.StatusCode != http.StatusUnprocessableEntity {
    t.Errorf("Expected 400 or 422, got %d", resp.StatusCode)
}
```

**Option 2: Just Check for Error**

Don't verify specific status codes for validation errors:

```go
if err == nil {
    t.Error("Expected error when creating team without name")
}
// Don't check specific status code
```

**Option 3: Check for Any 4xx Error**

```go
if resp.StatusCode < 400 || resp.StatusCode >= 500 {
    t.Errorf("Expected 4xx error, got %d", resp.StatusCode)
}
```

## Proposed Solutions

### Solution 1: Change API to Return 400 (Recommended)

Override FastAPI's default to return 400 for validation errors:

```python
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse

@app.exception_handler(RequestValidationError)
async def validation_exception_handler(request, exc):
    return JSONResponse(
        status_code=400,  # Instead of 422
        content={"detail": exc.errors()},
    )
```

This makes the API more consistent with HTTP semantics.

### Solution 2: Document 422 as Standard

If 422 is intentional, document it clearly in the API specification:

```yaml
/teams:
  post:
    responses:
      201:
        description: Team created successfully
      422:  # Not 400
        description: Validation error (missing required fields)
```

And update SDK tests to expect 422.

### Solution 3: Fix SDK Response Handling

Ensure the SDK properly constructs Response objects for 422 errors:

```go
// In Do() method
resp := newResponse(httpResp)
if httpResp.StatusCode == 422 {
    // Ensure Response object is populated
    // Parse validation error details
}
```

## SDK Implementation Status

**Status:** ⚠️ SDK Response handling may need improvement

The SDK correctly returns errors for validation failures, but the Response object appears to have StatusCode = 0 for 422 responses. This should be investigated and fixed.

**Required SDK Changes:**
1. Verify Response object is populated for 422 status codes
2. Update tests to accept 422 for validation errors (or wait for API fix)
3. Document that validation errors return 422, not 400

## Related Issues

- Should check if this affects other resources (Tools, Servers, etc.)
- May be related to how FastAPI is configured globally
- SDK Response construction may have issues with non-standard status codes

## References

- FastAPI Documentation: https://fastapi.tiangolo.com/tutorial/handling-errors/
- RFC 4918 (422): https://tools.ietf.org/html/rfc4918#section-11.2
- RFC 7231 (400): https://tools.ietf.org/html/rfc7231#section-6.5.1
- SDK Integration Test: `test/integration/teams_integration_test.go:499-515`
- SDK Do Method: `contextforge/contextforge.go`

## Next Steps

1. Verify if 422 is used consistently across all ContextForge endpoints
2. Decide if API should be changed to return 400 (recommended)
3. Fix SDK Response construction for 422 status codes
4. Update SDK tests based on final decision
5. Document the expected behavior clearly

---

**Report Generated:** 2025-11-09
**Tested Against:** ContextForge v0.8.0
**Reporter:** go-contextforge SDK Team
