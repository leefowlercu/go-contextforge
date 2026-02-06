# Code Organization for Agents

## Repository Layout

- `/Users/lee/Dev/leefowlercu/go-contextforge/contextforge/`: main SDK package
- `/Users/lee/Dev/leefowlercu/go-contextforge/test/integration/`: integration tests
- `/Users/lee/Dev/leefowlercu/go-contextforge/examples/`: runnable usage examples
- `/Users/lee/Dev/leefowlercu/go-contextforge/docs/`: documentation and upstream bug notes
- `/Users/lee/Dev/leefowlercu/go-contextforge/scripts/`: build/release scripts

## Service Pattern

When adding a new service:

1. Add `contextforge/<service>.go`.
2. Define `type <Service>Service service`.
3. Add client field in `contextforge/types.go`.
4. Initialize in `newClient()` in `contextforge/contextforge.go`.
5. Implement methods with `(ctx context.Context, ...)`.
6. Add unit tests: `contextforge/<service>_test.go`.
7. Add integration tests: `test/integration/<service>_integration_test.go`.
8. Update integration helpers for data generation and cleanup as needed.

## Existing Services to Mirror

- `ToolsService`
- `ResourcesService`
- `GatewaysService`
- `ServersService`
- `PromptsService`
- `AgentsService`
- `TeamsService`

Use these existing files as behavioral and style references.
