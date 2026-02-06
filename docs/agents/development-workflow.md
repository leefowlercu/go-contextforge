# Development Workflow for Agents

## Primary Entry Point

Run:

```bash
make help
```

## Build and Test Commands

Unit tests:

```bash
make test
make test-verbose
make test-cover
make coverage
```

Integration tests:

```bash
make integration-test-setup
make integration-test
make integration-test-teardown
make integration-test-all
make test-all
```

Build:

```bash
make build
make build-all
make examples
```

Quality:

```bash
make fmt
make vet
make lint
make check
make ci
```

Dependencies:

```bash
make deps
make tidy
make update-deps
make clean
```

## Running SDK Examples

```bash
go run examples/tools/main.go
go run examples/prompts/main.go
go run examples/agents/main.go
```

## Release Commands

Prereqs:

- `goreleaser` installed
- `GITHUB_TOKEN` set

Commands:

```bash
make release-patch
make release-minor
make release-major
make release-prep VERSION=vX.Y.Z
make goreleaser-check
make goreleaser-snapshot
```
