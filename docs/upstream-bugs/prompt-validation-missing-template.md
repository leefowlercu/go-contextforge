# ContextForge Bug: Prompts API Accepts Empty Template Field

**Bug ID:** CONTEXTFORGE-002
**Component:** ContextForge MCP Gateway - Prompts API
**Affected Version:** v0.8.0
**Severity:** Low
**Status:** Confirmed
**Reported:** 2025-11-09

## Summary

The `POST /prompts` endpoint accepts prompt creation requests without a `template` field, allowing prompts to be created with empty/missing templates. This violates the semantic requirement that prompts must have a template to be functional.

## Affected Endpoint

```
POST /prompts
```

## Expected Behavior

When creating a prompt without a `template` field:
- The API should reject the request with a 422 Unprocessable Entity error
- The response should indicate that `template` is a required field
- No prompt should be created in the database

A prompt without a template is semantically invalid since:
- The template defines what the prompt renders
- Without a template, the prompt cannot function in the MCP protocol
- The OpenAPI spec describes `template` as the core content of a prompt

## Actual Behavior

When creating a prompt without a `template` field:
- The API accepts the request (returns 200 OK) ✓
- A prompt is created in the database with an empty template
- No validation error is returned

## Reproduction Steps

1. Send POST request to `/prompts` with request body:
   ```json
   {
     "prompt": {
       "name": "test-prompt-without-template"
     }
   }
   ```
2. Request succeeds with 200 OK
3. Prompt is created with empty/null template field

**Expected:** 422 Unprocessable Entity with validation error
**Actual:** 200 OK with created prompt

## Evidence from Integration Tests

```
=== RUN   TestPromptsService_InputValidation/create_prompt_without_required_template
    prompts_integration_test.go:498: Expected error when creating prompt without template
    prompts_integration_test.go:501: Correctly rejected prompt without template: <nil>
--- FAIL: TestPromptsService_InputValidation/create_prompt_without_required_template (0.00s)
```

The test expects an error but receives `nil`, confirming the API accepts the invalid request.

## Root Cause Analysis

### OpenAPI Schema

The OpenAPI spec for `PromptCreate` likely does not mark `template` as required:

```yaml
# Expected (should be):
PromptCreate:
  type: object
  required:
    - name
    - template  # ← Should be required
  properties:
    name:
      type: string
    template:
      type: string
```

### Pydantic Model

The Pydantic schema in ContextForge may allow optional template:

**File:** `mcpgateway/schemas.py` (presumed)

```python
class PromptCreate(BaseModel):
    name: str
    description: Optional[str] = None
    template: Optional[str] = None  # ← Should not be Optional
    # ...
```

### Database Model

The database schema may allow NULL values for the template column:

**File:** `mcpgateway/models.py` (presumed)

```python
class DbPrompt(Base):
    __tablename__ = "prompts"

    template = Column(String, nullable=True)  # ← Should be nullable=False
```

## Comparison with Other Resources

Other resource types properly validate required fields:

- **Tools:** Require `name` field (returns 422 if missing) ✓
- **Resources:** Require `uri`, `name`, `content` (returns 422 if missing) ✓
- **Servers:** Require `name` (returns 422 if missing) ✓
- **Prompts:** Require `name` but NOT `template` ✗

The prompts endpoint is inconsistent with other resource validation patterns.

## Impact

**Severity: Low**

**Positive Impacts:**
- Allows creating "draft" prompts that can be filled in later
- Supports multi-step creation workflows

**Negative Impacts:**
- Creates semantically invalid prompts that cannot function
- Violates principle of least surprise (template seems required)
- May cause issues when prompts are used in MCP protocol without templates
- Inconsistent with validation patterns of other resources

## Possible Interpretations

### Theory 1: Intentional Design
This may be **intentional** to support workflows like:
1. Create placeholder prompt with just a name
2. Update prompt later with template and arguments
3. Activate prompt when complete

### Theory 2: Validation Bug
This may be **unintentional** due to:
- Missing validation constraint in Pydantic model
- Database schema allowing NULL values
- OpenAPI spec not marking field as required

## Recommended Solutions

### If Intentional (Feature):

1. **Update Documentation:**
   - Clarify that template is optional for draft prompts
   - Document workflow for creating incomplete prompts
   - Add validation that prompts cannot be activated without template

2. **Add State Field:**
   ```python
   class PromptCreate(BaseModel):
       name: str
       template: Optional[str] = None
       state: str = "draft"  # draft, complete, active
   ```

3. **Validate on Activation:**
   - Prevent toggling to `isActive=true` if template is empty
   - Return 422 error: "Cannot activate prompt without template"

### If Unintentional (Bug):

1. **Update Pydantic Model:**
   ```python
   class PromptCreate(BaseModel):
       name: str
       template: str  # Remove Optional
   ```

2. **Update Database Schema:**
   ```python
   template = Column(String, nullable=False)
   ```

3. **Update OpenAPI Spec:**
   ```yaml
   required:
     - name
     - template
   ```

## SDK Implementation Status

**Status:** ✅ SDK implementation is correct

Our go-contextforge SDK correctly:
- Sends the create request with or without template field
- Handles API response appropriately
- Does not add artificial client-side validation

The SDK integration test documents expected vs actual behavior, which is valuable for API contract testing.

## Workaround

SDK users should:

1. **Always provide template field:**
   ```go
   prompt := &contextforge.PromptCreate{
       Name:     "my-prompt",
       Template: "Hello {{name}}!",  // Always include
   }
   ```

2. **Add client-side validation:**
   ```go
   if prompt.Template == "" {
       return errors.New("template is required")
   }
   ```

3. **Check created prompts:**
   ```go
   created, _, err := client.Prompts.Create(ctx, prompt, nil)
   if created.Template == "" {
       log.Warn("Prompt created without template")
   }
   ```

## Related Issues

- CONTEXTFORGE-001: Prompts toggle returns stale state

## References

- OpenAPI Spec: `reference/contextforge-openapi-v0.8.0.json`
- SDK Integration Test: `test/integration/prompts_integration_test.go:490-501`

## Next Steps

1. Clarify with ContextForge team if this is intentional or a bug
2. If intentional: Request documentation update
3. If bug: Request validation constraint addition
4. Update SDK tests based on team response
5. Document behavior in SDK README

---

**Report Generated:** 2025-11-09
**Tested Against:** ContextForge v0.8.0
**Reporter:** go-contextforge SDK Team
