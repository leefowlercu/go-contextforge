# Testing Guidance for Agents

## Unit Tests

Location: `/Users/lee/Dev/leefowlercu/go-contextforge/contextforge/*_test.go`

- Use `httptest.NewServer` for API mocking.
- Prefer table-driven tests.
- Cover happy paths, edge cases, and error behavior.
- Reuse existing test helper patterns in current test files.

## Integration Tests

Location: `/Users/lee/Dev/leefowlercu/go-contextforge/test/integration/*_integration_test.go`

- Must include build tag: `//go:build integration`.
- Run only when `INTEGRATION_TESTS=true`.
- Defaults:
  - `CONTEXTFORGE_ADDR=http://localhost:8000/`
  - `CONTEXTFORGE_ADMIN_EMAIL=admin@test.local`
  - `CONTEXTFORGE_ADMIN_PASSWORD=testpassword123`
- Use cleanup hooks (`t.Cleanup`) for created resources.

## Known Upstream Bug Impact

Some integration tests are intentionally skipped due to confirmed upstream API issues.  
See bug documents in:

- `/Users/lee/Dev/leefowlercu/go-contextforge/docs/upstream-bugs/`

Known skipped issue IDs:

- `CONTEXTFORGE-001`
- `CONTEXTFORGE-002`
- `CONTEXTFORGE-003`
- `CONTEXTFORGE-004`
- `CONTEXTFORGE-005`
- `CONTEXTFORGE-007`
- `CONTEXTFORGE-008`
- `CONTEXTFORGE-009`
- `CONTEXTFORGE-010`

Resolved issue:

- `CONTEXTFORGE-006` (validation error behavior is by design)

## Re-Enabling a Skipped Test

1. Confirm the upstream bug is fixed in the tested ContextForge version.
2. Remove the corresponding `t.Skip()` call.
3. Re-run the test and verify pass.
4. Update the related bug markdown document with resolution details.
