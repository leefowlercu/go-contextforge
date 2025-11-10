# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.5.0] - 2025-11-10

### Added
- Feat: move from "base url" to "address" in client configuration and codebase references

### Documentation
- Docs: add info about mcp and expand a2a section in readme
- Docs: update project documentation
- Docs: update toc in readme

### Build
- Build: update prepare release script to automatically merge goreleaser-created changelog contents into root changelog

## [0.4.0] - 2025-11-09

### Added
- Add teams service

### Documentation
- Add teams service example program
- Add upstream bug reports for teams service bugs
- Update in-code service documentation for gateways and tools services
- Update project documentation
- Update readme for accuracy regarding service toggle endpoints inconsistency

### Tests
- Add teams service unit tests and integration tests
- Fix integration tests after client creation functionality update
- Update integration test helpers
- Update integration test setup and teardown script to store helper files in project root tmp/ directory

## [0.3.0] - 2025-11-09

### Added
- Standardize on new sdk client baseurl parameter requirement

### Tests
- Skip resources service toggle unit test until upstream bug is fixed

## [0.2.1] - 2025-11-09

### Documentation
- Update contextforge package documentation

### Build
- Update goreleaser config
- Update release toolchain to use goreleaser for changelog generation and github release generation
- Update prepare release script to use project canonical annotated tag message format

## [0.2.0] - 2025-11-09

### Added
- Add agents service, associated unit tests, associated integration tests, associated example program, update readme

### Documentation
- Add example programs for the gateways, servers, resources, tools, and prompts services

## [0.1.0] - 2025-11-09

### Added
- Initial commit, add contextforge api client sdk, add gateways, servers, prompts, resources, and tools services, add integration test suite, add unit tests

[0.5.0]: https://github.com/leefowlercu/go-contextforge/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/leefowlercu/go-contextforge/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/leefowlercu/go-contextforge/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/leefowlercu/go-contextforge/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/leefowlercu/go-contextforge/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/leefowlercu/go-contextforge/releases/tag/v0.1.0
