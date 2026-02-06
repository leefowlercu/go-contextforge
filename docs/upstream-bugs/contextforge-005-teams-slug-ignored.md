# ContextForge Bug: Teams API Ignores User-Provided Slug Field

**Bug ID:** CONTEXTFORGE-005
**Component:** ContextForge MCP Gateway
**Affected Version:** v0.8.0, v1.0.0-BETA-1, v1.0.0-BETA-2
**Severity:** Low
**Status:** Confirmed in v1.0.0-BETA-2 (still valid)
**Reported:** 2025-11-09
**Last Validated:** 2026-02-06

## Summary

The `POST /teams` endpoint ignores the user-provided `slug` field in team creation requests and instead auto-generates the slug from the team name. This prevents users from creating teams with custom slugs and makes the `slug` field effectively read-only despite being documented as an input field in the API schema.

## Affected Endpoint

```
POST /teams
```

## Expected Behavior

When creating a team with a custom slug:
1. The API should accept the `slug` field in the request body
2. The created team should use the provided slug value
3. The slug should be validated (lowercase, alphanumeric with hyphens)
4. Duplicate slugs should be rejected with an appropriate error

Example request:
```json
{
  "name": "My Team",
  "slug": "custom-slug-123"
}
```

Expected response:
```json
{
  "id": "abc123...",
  "name": "My Team",
  "slug": "custom-slug-123",
  ...
}
```

## Actual Behavior

When creating a team with a custom slug:
1. The API accepts the `slug` field without error
2. The created team ignores the provided slug
3. The slug is auto-generated from the team name
4. No validation error or warning is returned

Example request:
```json
{
  "name": "test-team-1762712080059212000",
  "slug": "test-team-1762712080059212000-slug"
}
```

Actual response:
```json
{
  "id": "5f3c153ce9aa404398fc517d177cdd37",
  "name": "test-team-1762712080059212000",
  "slug": "test-team-1762712080059212000",  // ← Ignores provided value
  ...
}
```

The slug is generated from the name, completely ignoring the user-provided value.

## Reproduction Steps

1. Create a team with both `name` and `slug` fields:
   ```bash
   curl -X POST http://localhost:8000/teams/ \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
       "name": "My Awesome Team",
       "slug": "custom-slug"
     }'
   ```

2. Observe the response - the slug will be `"my-awesome-team"` (generated) instead of `"custom-slug"` (provided)

## Evidence from Integration Tests

SDK integration tests consistently show the slug being ignored:

```
=== RUN   TestTeamsService_BasicCRUD/create_team_with_all_optional_fields
    teams_integration_test.go:71: Expected slug "test-team-1762712080059212000-slug",
                                    got "test-team-1762712080059212000"

=== RUN   TestTeamsService_Validation/create_team_with_valid_slug_pattern
    teams_integration_test.go:541: Expected slug "valid-slug-123",
                                     got "test-team-1762710811262903000"
```

## Root Cause Analysis

### Location
Likely in `mcpgateway/services/team_management_service.py` or `mcpgateway/routers/teams.py`

### Hypothesis

The team creation logic likely:
1. Accepts the `slug` field in the Pydantic schema (no validation error)
2. Ignores the provided value during team creation
3. Auto-generates the slug from `team.name` using a slugify function

**Possible code pattern:**
```python
async def create_team(self, name: str, slug: Optional[str] = None, ...):
    # Provided slug is ignored
    generated_slug = slugify(name)  # Always uses name, ignores slug parameter

    new_team = EmailTeam(
        name=name,
        slug=generated_slug,  # Uses generated value, not user input
        ...
    )
```

### Comparison: Expected Pattern

The service should check if slug is provided and validate it:

```python
async def create_team(self, name: str, slug: Optional[str] = None, ...):
    if slug:
        # Validate slug format
        if not is_valid_slug(slug):
            raise InvalidSlugError(...)
        # Check for duplicates
        if await self.slug_exists(slug):
            raise DuplicateSlugError(...)
        team_slug = slug
    else:
        # Auto-generate from name
        team_slug = slugify(name)

    new_team = EmailTeam(
        name=name,
        slug=team_slug,  # Uses provided or generated
        ...
    )
```

## API Schema Inconsistency

The OpenAPI schema documents `slug` as an input field in `TeamCreateRequest`:

```json
{
  "TeamCreateRequest": {
    "type": "object",
    "properties": {
      "name": {"type": "string"},
      "slug": {"type": "string"},  // ← Documented but ignored
      "description": {"type": "string"},
      ...
    },
    "required": ["name"]
  }
}
```

This creates a misleading API contract where:
- The schema says slug can be provided
- The implementation ignores it
- No error or warning is returned

## Impact

**Severity: Low**

- ❌ Cannot create teams with custom slugs
- ❌ API behavior doesn't match schema documentation
- ❌ No validation or error message to alert users
- ✅ Auto-generated slugs work correctly
- ✅ Workaround exists (accept auto-generated slugs)

**Affected Users:**
- API consumers who want consistent slug naming
- Organizations with slug naming conventions
- Systems that need predictable slug values for URLs/references

**Business Impact:**
Low - Auto-generated slugs are functional, but custom slug support would improve user control and naming consistency.

## Workaround

Accept the auto-generated slug and don't rely on the `slug` field in creation requests:

```go
// Instead of:
team := &contextforge.TeamCreate{
    Name: "My Team",
    Slug: contextforge.String("my-custom-slug"),  // This will be ignored
}

// Use:
team := &contextforge.TeamCreate{
    Name: "my-custom-slug",  // Put desired slug as name
}
// Result: slug will be "my-custom-slug" (auto-generated from name)
```

Alternatively, if the API later supports updates to the slug field, update it after creation.

## Proposed Solutions

### Solution 1: Honor User-Provided Slugs (Recommended)

Modify the team creation logic to use provided slugs:

```python
async def create_team(self, name: str, slug: Optional[str] = None, ...):
    if slug:
        # Validate slug format (lowercase, alphanumeric, hyphens)
        if not re.match(r'^[a-z0-9-]+$', slug):
            raise ValueError("Invalid slug format")
        # Check uniqueness
        if await self._slug_exists(slug):
            raise ValueError("Slug already exists")
        team_slug = slug
    else:
        team_slug = self._generate_slug_from_name(name)

    new_team = EmailTeam(name=name, slug=team_slug, ...)
```

### Solution 2: Remove Slug from Input Schema

If auto-generation is intentional, remove `slug` from the input schema:

```python
class TeamCreateRequest(BaseModel):
    name: str
    # slug: str  ← Remove from input (make it response-only)
    description: Optional[str] = None
    ...
```

This makes the API contract accurate (slug is read-only, auto-generated).

### Solution 3: Add Warning Documentation

If the current behavior is intentional, document it clearly:

```python
class TeamCreateRequest(BaseModel):
    name: str
    slug: Optional[str] = Field(
        None,
        description="IGNORED: Slug is auto-generated from name. This field has no effect."
    )
```

## SDK Implementation Status

**Status:** ✅ SDK implementation is correct

The SDK correctly:
- Sends the `slug` field in creation requests
- Parses the response and extracts the actual slug value
- Does not assume the provided slug matches the returned slug

The test failures correctly identify the API behavior mismatch.

## Related Issues

- May affect other resources that accept slug fields (servers, tools, etc.)
- Should verify if `PUT /teams/{id}` allows slug updates (workaround possibility)

## References

- ContextForge Source: `mcpgateway/services/team_management_service.py`
- API Router: `mcpgateway/routers/teams.py`
- SDK Integration Tests: `test/integration/teams_integration_test.go:52-84, 513-544`
- OpenAPI Spec: upstream `mcp-context-forge` tag schema (no local snapshot maintained in this repo)

## Next Steps

1. Verify if this is intentional design or a bug
2. If intentional, update API documentation and schema
3. If a bug, implement slug validation and usage
4. Consider if slug updates are supported via PUT endpoint
5. Update SDK tests based on final API behavior

---

## v1.0.0-BETA-1 Validation Notes

**Validated:** 2026-01-13

Source code analysis **confirms the root cause** - the slug field is accepted by the schema but never passed to the service.

### Evidence

**File:** `mcpgateway/routers/teams.py:88`

```python
async def create_team(request: TeamCreateRequest, current_user_ctx: dict = Depends(get_current_user_with_permissions)) -> TeamResponse:
    try:
        db = current_user_ctx["db"]
        service = TeamManagementService(db)
        team = await service.create_team(
            name=request.name,
            description=request.description,
            created_by=current_user_ctx["email"],
            visibility=request.visibility,
            max_members=request.max_members
        )
        # Note: request.slug is NOT passed to create_team()
```

**File:** `mcpgateway/services/team_management_service.py:100`

The `create_team` method signature has NO slug parameter:

```python
async def create_team(self, name: str, description: str = None, created_by: str = None,
                      visibility: str = "private", max_members: int = None):
    # No slug parameter - slug is always auto-generated
```

### Schema vs Implementation Mismatch

**Schema accepts slug** (`TeamCreateRequest`):
```python
slug: Optional[str] = Field(
    None,
    description="(optional, auto-generated if not provided)"  # Line 5121
)
```

**But implementation ignores it** - the router doesn't pass `request.slug` to the service.

### Confirmation

This is definitively a bug, not intentional behavior. The schema documents slug as optional (implying it can be provided), but the implementation completely ignores any provided value.

---

**Report Generated:** 2025-11-09
**Tested Against:** ContextForge v0.8.0
**Validated Against:** ContextForge v1.0.0-BETA-2
**Reporter:** go-contextforge SDK Team

---

## v1.0.0-BETA-2 Revalidation Notes

**Validated:** 2026-02-06

- **Still Valid?** Yes. User-provided team slug is still ignored.
- **Is it actually a bug?** Yes. Request schema advertises `slug` input, but implementation discards it.

### Evidence

- `TeamCreateRequest` still includes `slug`.
- Router/service flow still does not pass user `slug` into creation logic.
- Created team slug continues to be auto-generated from name.
