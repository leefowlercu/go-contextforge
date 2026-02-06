# ContextForge Upstream Reconciliation Report

- Generated: 2026-02-05T23:02:44-05:00
- SDK root: `/Users/lee/Dev/leefowlercu/go-contextforge`
- Upstream: `https://github.com/IBM/mcp-context-forge.git`
- Channel: `all semver tags`

## Version Discovery

- README tested-against: `v1.0.0-BETA-2` (high)
- CLAUDE.md mention: `v0.8.0` (low)
- CLAUDE.md mention: `v1.0.0` (low)
- CLAUDE.md mention: `v0.8.0` (low)
- CLAUDE.md mention: `v1.0.0-BETA-1` (low)

- Selected current tag: `v1.0.0-BETA-2`
- Selected from: README tested-against (`v1.0.0-BETA-2`)
- Selected target tag: `v1.0.0-BETA-2`
- Latest upstream semver tag: `v1.0.0-BETA-2`
- Latest upstream stable tag: `v0.9.0`

## Newer Tags

- None

## Service Delta

- Added services: none
- Removed services: none
- Changed existing services: 0

## SDK Mapping Impact

- Local SDK service roots: `a2a`, `cancellation`, `gateways`, `list_decode`, `prompts`, `resources`, `servers`, `teams`, `tools`
- Added upstream services already mapped: none
- Added upstream services not mapped in SDK: none
- Removed upstream services still mapped in SDK: none

## Reconciliation Checklist

1. Update target-version metadata (README tested-against text and any maintained metadata snapshots).
2. For each changed mapped service, reconcile request/response structs and service methods in `contextforge/<service>.go`.
3. For added upstream services not mapped in SDK, decide whether to add a new service file and client field wiring.
4. For removed upstream services, deprecate/remove SDK service coverage and adjust tests/examples.
5. Update/expand unit tests and integration tests for endpoint or schema changes.
6. Update docs (`README.md`, `CLAUDE.md`/`AGENTS.md`, bug notes) to match the new target tag.
7. Run `make fmt`, `make vet`, `make test`, and relevant integration targets before finalizing.
