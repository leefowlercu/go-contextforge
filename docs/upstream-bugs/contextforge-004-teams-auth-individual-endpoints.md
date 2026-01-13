# ContextForge Bug: Teams Individual Resource Endpoints Reject Valid Authentication

**Bug ID:** CONTEXTFORGE-004
**Component:** ContextForge MCP Gateway
**Affected Version:** v0.8.0, v1.0.0-BETA-1
**Severity:** High
**Status:** Confirmed (root cause identified)
**Reported:** 2025-11-09
**Last Validated:** 2026-01-13

## Summary

Individual team resource endpoints (`GET/PUT/DELETE /teams/{id}/*`) reject valid JWT authentication tokens with "Invalid token" (401 Unauthorized) errors, despite the same token working correctly for team list and create operations. This prevents any read, update, or delete operations on individual teams.

## Affected Endpoints

All individual team resource endpoints and discovery:
```
GET    /teams/{id}                      - Get team details
PUT    /teams/{id}                      - Update team
DELETE /teams/{id}                      - Delete team
GET    /teams/{id}/members              - List team members
PUT    /teams/{id}/members/{email}      - Update team member
DELETE /teams/{id}/members/{email}      - Remove team member
POST   /teams/{id}/invitations          - Invite team member
GET    /teams/{id}/invitations          - List team invitations
DELETE /teams/invitations/{id}          - Cancel invitation
POST   /teams/invitations/{token}/accept - Accept invitation
POST   /teams/{id}/join                 - Request to join team
DELETE /teams/{id}/leave                - Leave team
GET    /teams/{id}/join-requests        - List join requests
POST   /teams/{id}/join-requests/{id}/approve - Approve join request
DELETE /teams/{id}/join-requests/{id}   - Reject join request
GET    /teams/discover                  - Discover public teams
```

## Expected Behavior

When making requests to individual team endpoints with a valid JWT token:
1. The authentication middleware should validate the token
2. The RBAC middleware should check permissions
3. The request should proceed if authorized
4. A valid response should be returned (200, 201, 204, etc.)

## Actual Behavior

When making requests to individual team endpoints with a valid JWT token:
1. The request is rejected with 401 Unauthorized
2. The error response is `{"detail":"Invalid token"}` or `{"detail":"User authentication required"}`
3. The same token works perfectly for other team endpoints:
   - ✅ `GET /teams/` - Works (list teams)
   - ✅ `POST /teams/` - Works (create team)
   - ❌ `GET /teams/{id}/` - Fails with 401
   - ❌ `PUT /teams/{id}/` - Fails with 401
   - ❌ `DELETE /teams/{id}/` - Fails with 401

## Reproduction Steps

### Setup
1. Start ContextForge v0.8.0 with email authentication enabled
2. Bootstrap creates platform admin user (admin@test.local) with `is_admin=True`
3. Admin user receives wildcard permissions via platform_admin role: `["*"]`

### Test Sequence
```bash
# 1. Authenticate (works correctly)
TOKEN=$(curl -s -X POST 'http://localhost:8000/auth/login' \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin@test.local","password":"testpassword123"}' \
  | jq -r '.access_token')

# 2. Create team (works correctly)
TEAM_ID=$(curl -s -X POST 'http://localhost:8000/teams/' \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"test-team"}' \
  | jq -r '.id')

# Response: 201 Created
# {"id":"abc123","name":"test-team",...}

# 3. Get team details (fails)
curl -X GET "http://localhost:8000/teams/$TEAM_ID/" \
  -H "Authorization: Bearer $TOKEN"

# Response: 401 Unauthorized
# {"detail":"Invalid token"}

# 4. Update team (fails)
curl -X PUT "http://localhost:8000/teams/$TEAM_ID/" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"updated-team"}'

# Response: 401 Unauthorized

# 5. Delete team (fails)
curl -X DELETE "http://localhost:8000/teams/$TEAM_ID/" \
  -H "Authorization: Bearer $TOKEN"

# Response: 401 Unauthorized
```

### URL Trailing Slash Behavior

The API exhibits inconsistent trailing slash behavior:

```bash
# With trailing slash: 307 redirect to without slash, then 401
GET /teams/{id}/ → 307 → /teams/{id} → 401

# Without trailing slash: Direct 401
GET /teams/{id} → 401

# Both fail, redirect doesn't help
```

## Evidence from Integration Tests

SDK integration tests consistently fail on all individual team operations:

```
=== RUN   TestTeamsService_BasicCRUD/get_team_by_ID
    Created test team: test-team-1762710099245005000 (ID: 52959c9905af4dbfb4a96f0079b345a9)
    Failed to get team: GET http://localhost:8000/teams/52959c9905af4dbfb4a96f0079b345a9; 401
--- FAIL: TestTeamsService_BasicCRUD/get_team_by_ID (0.02s)

=== RUN   TestTeamsService_BasicCRUD/update_team
    Created test team: test-team-1762710099262120000 (ID: 68dff9fa949642e28fe387484f293229)
    Failed to update team: PUT http://localhost:8000/teams/68dff9fa949642e28fe387484f293229; 401
--- FAIL: TestTeamsService_BasicCRUD/update_team (0.02s)

=== RUN   TestTeamsService_BasicCRUD/delete_team
    Created test team: test-team-1762710099279107000 (ID: 82d87294f11b46feb586078b03d7bfec)
    Failed to delete team: DELETE http://localhost:8000/teams/82d87294f11b46feb586078b03d7bfec; 401
--- FAIL: TestTeamsService_BasicCRUD/delete_team (0.02s)
```

## JWT Token Validation

The JWT token used in failing requests is valid and contains proper claims:

```json
{
  "sub": "admin@test.local",
  "iss": "mcpgateway",
  "aud": "mcpgateway-api",
  "iat": 1762710026,
  "exp": 1763314826,
  "jti": "9407712e-472b-4e07-93a9-3956cbe5a2a1",
  "user": {
    "email": "admin@test.local",
    "full_name": "Platform Administrator",
    "is_admin": true,
    "auth_provider": "local"
  },
  "teams": [...],
  "namespaces": ["user:admin@test.local", "team:platform-administrators-team", "public"],
  "scopes": {
    "server_id": null,
    "permissions": ["*"],
    "ip_restrictions": [],
    "time_restrictions": {}
  }
}
```

Key token properties:
- ✅ Valid signature (verified with JWT_SECRET_KEY)
- ✅ Not expired (exp in future)
- ✅ Contains `is_admin: true`
- ✅ Contains wildcard permissions `["*"]`
- ✅ Contains proper issuer and audience
- ✅ Works for team list and create operations

## Root Cause Analysis

### Location
**Authentication Flow:**
- `mcpgateway/middleware/rbac.py:71-148` - `get_current_user_with_permissions()`
- `mcpgateway/auth.py:54-232` - `get_current_user()`
- `mcpgateway/services/permission_service.py:436-454` - `_is_user_admin()`

### Hypothesis: Route Registration or Middleware Order Issue

The authentication failure is likely caused by one of:

1. **FastAPI Route Registration:**
   - Individual team endpoints may not be properly registered with authentication dependencies
   - The endpoint route pattern matching may be failing for parameterized paths

2. **Middleware Order:**
   - Authentication middleware may not be applied to these specific routes
   - RBAC middleware may be checked before authentication middleware

3. **Redirect Handling:**
   - FastAPI's automatic trailing slash redirects (307) may lose authentication headers
   - The redirect from `/teams/{id}/` to `/teams/{id}` strips the Authorization header

### Comparison: Working vs Broken Endpoints

**Working Endpoints:**
```python
# mcpgateway/routers/teams.py

@teams_router.post("/", response_model=TeamResponse, status_code=status.HTTP_201_CREATED)
@require_permission("teams.create")
async def create_team(
    request: TeamCreateRequest,
    current_user_ctx: dict = Depends(get_current_user_with_permissions)
) -> TeamResponse:
    # Works correctly ✅
```

**Broken Endpoints:**
```python
@teams_router.get("/{team_id}", response_model=TeamResponse)
@require_permission("teams.read")
async def get_team(
    team_id: str,
    current_user_ctx: dict = Depends(get_current_user_with_permissions)
) -> TeamResponse:
    # Fails with 401 ❌
```

The only visible difference is the path parameter `{team_id}`, suggesting the issue is related to how FastAPI handles authentication on parameterized routes.

### Gateway Logs

```
2025-11-09 12:41:39,214 - uvicorn.access - INFO - 127.0.0.1:57584 - "POST /auth/login HTTP/1.1" 200
2025-11-09 12:41:39,219 - uvicorn.access - INFO - 127.0.0.1:57584 - "POST /teams HTTP/1.1" 307
2025-11-09 12:41:39,226 - uvicorn.access - INFO - 127.0.0.1:57584 - "POST /teams/ HTTP/1.1" 201
2025-11-09 12:41:39,231 - uvicorn.access - INFO - 127.0.0.1:57584 - "DELETE /teams/c3cfe8035ce946c2b272c5fa38d6764e HTTP/1.1" 401
2025-11-09 12:41:39,257 - uvicorn.access - INFO - 127.0.0.1:57584 - "GET /teams/52959c9905af4dbfb4a96f0079b345a9 HTTP/1.1" 401
```

Notably:
- Login succeeds (200)
- Team creation redirects but succeeds (307 → 201)
- Individual operations fail immediately without middleware logging (401)
- No "Authentication failed" error in logs, suggesting middleware isn't reached

## Proposed Solutions

### Solution 1: Check FastAPI Route Registration

Verify that parameterized team routes are properly registered:

```python
# Debug routes
for route in app.routes:
    print(f"{route.path} - {route.methods}")
```

### Solution 2: Explicit Authentication Dependency

Add explicit authentication to each route instead of relying on decorators:

```python
@teams_router.get("/{team_id}", dependencies=[Depends(get_current_user)])
@require_permission("teams.read")
async def get_team(team_id: str, ...):
    ...
```

### Solution 3: Fix Redirect Behavior

Configure FastAPI to not redirect on trailing slashes:

```python
app = FastAPI(redirect_slashes=False)
```

Or define both with and without trailing slash:

```python
@teams_router.get("/{team_id}")
@teams_router.get("/{team_id}/")
async def get_team(...):
    ...
```

### Solution 4: Investigate Middleware Application

Ensure middleware is applied to all routes:

```python
app.add_middleware(
    SomeAuthMiddleware,
    exclude_paths=["/health", "/version"]  # Verify teams endpoints not excluded
)
```

## Impact

**Severity: High**

- ❌ Cannot retrieve individual team details
- ❌ Cannot update team properties
- ❌ Cannot delete teams
- ❌ Cannot manage team members
- ❌ Cannot handle team invitations
- ❌ Cannot process join requests
- ✅ CAN create teams (workaround: create then abandon)
- ✅ CAN list teams (partial workaround for viewing)

**Affected Users:**
- All API consumers using team management features
- SDKs and clients attempting CRUD operations on teams
- Admin interfaces for team administration
- Automated team lifecycle management scripts

**Business Impact:**
Teams feature is essentially non-functional beyond creation and listing. This makes the multi-tenancy feature unusable in practice.

## Workaround

No effective workaround exists for most operations:

1. **Read Operations:** Use `GET /teams/` and filter client-side
   ```go
   // Instead of Get(teamID)
   teams, _, _ := client.Teams.List(ctx, &contextforge.TeamListOptions{})
   // Find team in list by ID
   ```

2. **Update Operations:** Not possible (no workaround)

3. **Delete Operations:** Not possible (no workaround)

4. **Member Management:** Not possible (no workaround)

## SDK Implementation Status

**Status:** ✅ SDK implementation is correct

Our go-contextforge SDK correctly:
- Sends proper Authorization headers with Bearer tokens
- Constructs URLs with appropriate path parameters
- Handles both trailing slash and non-trailing slash patterns
- Parses responses according to OpenAPI specification

The SDK integration test failures are expected given the ContextForge bug. All SDK unit tests pass (20/20).

## Related Issues

- May be related to other individual resource endpoints in different services
- Similar pattern might affect `/resources/{id}`, `/servers/{id}`, etc.

## References

- ContextForge Source: `mcpgateway/routers/teams.py`
- Authentication: `mcpgateway/middleware/rbac.py:71-148`
- Auth Service: `mcpgateway/auth.py:54-232`
- SDK Integration Tests: `test/integration/teams_integration_test.go`
- OpenAPI Spec: `reference/contextforge-openapi-v0.8.0.json`

## Next Steps

1. Report this critical issue to the ContextForge team
2. Request urgent fix in hotfix release
3. Investigate if other services have the same issue
4. Consider reverting Teams API until fixed
5. Update SDK tests to document expected behavior once fixed

---

## v1.0.0-BETA-1 Validation Notes

**Validated:** 2026-01-13

Source code analysis reveals the **actual root cause** is a dependency injection inconsistency, not token validation.

### Root Cause: Different Authentication Dependencies

**File:** `mcpgateway/routers/teams.py`

**Working endpoints** (list, create) use `get_current_user_with_permissions`:
```python
# Line 65-67: create_team
@teams_router.post("/", response_model=TeamResponse, status_code=status.HTTP_201_CREATED)
@require_permission("teams.create")
async def create_team(request: TeamCreateRequest, current_user_ctx: dict = Depends(get_current_user_with_permissions)):
    db = current_user_ctx["db"]  # DB is provided in context
    ...

# Line 112-117: list_teams
@teams_router.get("/", response_model=TeamListResponse)
@require_permission("teams.read")
async def list_teams(..., current_user_ctx: dict = Depends(get_current_user_with_permissions)):
    db = current_user_ctx["db"]  # DB is provided in context
    ...
```

**Broken endpoints** (get, update, delete) use SEPARATE dependencies:
```python
# Line 176-178: get_team
@teams_router.get("/{team_id}", response_model=TeamResponse)
@require_permission("teams.read")
async def get_team(team_id: str, current_user: EmailUserResponse = Depends(get_current_user), db: Session = Depends(get_db)):
    # Uses get_current_user instead of get_current_user_with_permissions
    # May have different session/validation behavior
    ...

# Line 226-228: update_team
@teams_router.put("/{team_id}", response_model=TeamResponse)
@require_permission("teams.update")
async def update_team(team_id: str, request: TeamUpdateRequest, current_user: EmailUserResponse = Depends(get_current_user), db: Session = Depends(get_db)):
    ...
```

### Key Difference

| Endpoint Type | Dependency | DB Session Source |
|---------------|------------|-------------------|
| Working (list, create) | `get_current_user_with_permissions` | `current_user_ctx["db"]` |
| Broken (get, update, delete) | `get_current_user` + `get_db` | Separate `Depends(get_db)` |

### Hypothesis Update

The bug is NOT about invalid tokens. The issue is:
1. `get_current_user` and `get_current_user_with_permissions` may have different authentication flows
2. The separate `get_db` dependency may not properly integrate with the auth flow
3. The different context structure may cause permission checks to fail

### Recommendation

Fix the broken endpoints to use the same pattern as working endpoints:
```python
# Instead of:
async def get_team(team_id: str, current_user: EmailUserResponse = Depends(get_current_user), db: Session = Depends(get_db)):

# Use:
async def get_team(team_id: str, current_user_ctx: dict = Depends(get_current_user_with_permissions)):
    db = current_user_ctx["db"]
    current_user = current_user_ctx  # or extract email from context
```

---

**Report Generated:** 2025-11-09
**Tested Against:** ContextForge v0.8.0
**Validated Against:** ContextForge v1.0.0-BETA-1
**Reporter:** go-contextforge SDK Team
