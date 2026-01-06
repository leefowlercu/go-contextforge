# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.8.0] - 2026-01-06

### Added
- Feat: add contextforge v1.0.0-beta-1 compatibility

### Build
- Build: update prepare-release script

## [0.7.0] - 2025-11-20

### Added
- Feat: add comprehensive three-state semantics for optional fields

### Documentation
- Docs: add comprehensive guides for three-state pattern and terraform provider usage
- Docs: update project documentation with three-state system references

### Build
- Build: add blank lines between changelog subsections
- Build: update gitignore and changelog


## [0.6.3] - 2025-11-20

### Fixed
- Fix: enable clearing array fields in update operations

### Documentation
- Docs: fix server creation example and add missing api methods
- Docs: update architecture documentation with correct file locations

### Build
- Build: update prepare release script and fix changelog

## [0.6.2] - 2025-11-19

### Fixed
- Fix(release): create tag after commit amend to prevent tag drift
- Fix(release): create temporary tag for goreleaser validation

## [0.6.1] - 2025-11-19

### Fixed
- Fix(tools): remove incorrect wrapper from update request body

### Tests
- Test(integration): add proper assertions to update operation tests
- Test(integration): improve token generation and output formatting

## [0.6.0] - 2025-11-13

### Added
- Feat: add hybrid rest endpoint methods for resources and prompts
- Feat: move from "base url" to "address" in client configuration and codebase references

### Documentation
- Docs: add info about mcp and expand a2a section in readme
- Docs: move claude instructions to project root
- Docs: update documentation for hybrid rest endpoints
- Docs: update project documentation
- Docs: update toc in readme

### Build
- Build: update prepare release script to automatically merge goreleaser-created changelog contents into root changelog

### Tests
- Test: add integration tests for hybrid endpoint methods
- Test: add unit tests for hybrid endpoint methods
- Test: update output text in integration test setup script

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

[0.8.0]: https://github.com/leefowlercu/go-contextforge/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/leefowlercu/go-contextforge/compare/v0.6.3...v0.7.0
[0.6.3]: https://github.com/leefowlercu/go-contextforge/compare/v0.6.2...v0.6.3
[0.6.2]: https://github.com/leefowlercu/go-contextforge/compare/v0.6.1...v0.6.2
[0.6.1]: https://github.com/leefowlercu/go-contextforge/compare/v0.4.0...v0.6.1
[0.6.0]: https://github.com/leefowlercu/go-contextforge/compare/v0.4.0...v0.6.0
[0.5.0]: https://github.com/leefowlercu/go-contextforge/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/leefowlercu/go-contextforge/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/leefowlercu/go-contextforge/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/leefowlercu/go-contextforge/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/leefowlercu/go-contextforge/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/leefowlercu/go-contextforge/releases/tag/v0.1.0
