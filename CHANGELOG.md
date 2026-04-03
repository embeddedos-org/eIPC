# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-04-03

### Added
- **Documentation:**
  - Architecture diagram with Mermaid component, message flow, and auth sequence diagrams (`docs/architecture.md`)
  - Use case guide with 4 deployment scenarios and transport selection matrix (`docs/use-cases.md`)
  - Complete API reference for all public Go packages (`docs/api-reference.md`)
  - Security model documentation with formal threat model table (`docs/security-model.md`)
  - Key management guide with rotation strategy and secrets manager integration (`docs/key-management.md`)
  - Performance guide with tuning recommendations and backpressure docs (`docs/performance.md`)
  - Benchmark comparisons vs gRPC, ZeroMQ, and MQTT (`docs/benchmarks.md`)
  - Documentation index added to README.md

- **Testing & CI/CD:**
  - GitHub Actions CI workflow: matrix build (Linux/macOS/Windows), format check, vet, race-detected tests, coverage upload (`.github/workflows/ci.yml`)
  - GitHub Actions Release workflow: tag-triggered cross-platform build + GitHub Release (`.github/workflows/release.yml`)
  - Unit tests for `security/integrity` (7 tests), `services/audit` (5 tests), `services/health` (8 tests), `services/registry` (9 tests), `config` (13 tests)
  - Fuzz tests for protocol frame decoder and HMAC verifier
  - Stress tests: large payload (512KB), concurrent clients (10), message ordering (50 messages)
  - Benchmark suite for core (4 benchmarks) and protocol (3 benchmarks)

- **Security:**
  - AES-256-GCM encryption package (`security/encryption/`) with Encrypt/Decrypt functions
  - `FlagEncrypted` (0x04) frame flag for encrypted payload indication
  - AES encryption tests (8 tests): round-trip, wrong key, tampered, empty, large payload, unique nonces

- **Usability:**
  - Connection lifecycle management (`core/lifecycle.go`): `ReconnectPolicy` with exponential backoff, `HeartbeatSender`, `GracefulShutdown`
  - `eipc-cli` debugging tool (`cmd/eipc-cli/`): send, listen, ping commands
  - `make bench` and `make build-cli` Makefile targets

- **Community:**
  - Hello EIPC tutorial (`examples/hello-eipc/`): minimal server + client + README walkthrough

[0.2.0]: https://github.com/embeddedos-org/eipc/compare/v0.1.0...v0.2.0

## [0.1.0] - 2026-03-31

### Added
- Initial release of eipc
- **Core:** Message types, endpoint API (client/server), priority-lane router with batch dispatch
- **Protocol:** Frame encoding/decoding, HMAC integrity, codec abstraction
- **Security:** Authentication (session tokens), capability-based authorization, HMAC integrity verification, replay detection (sliding window)
- **Transports:** TCP transport with length-prefixed framing, Unix domain socket transport, Windows named pipe transport, shared memory ring buffer
- **Services:** Service registry and discovery, message broker, policy engine, audit logging, health monitoring
- **CLI:** `eipc-server` and `eipc-client` demo binaries
- **C SDK:** Frame codec, HMAC, transport, JSON helpers under `sdk/c/`
- Cross-platform builds for Linux (amd64/arm64/armv7), macOS (amd64/arm64), Windows (amd64/arm64)
- Release packaging with `.tar.gz` (Linux/macOS) and `.zip` (Windows) archives
- Complete CI/CD pipeline with nightly, weekly, and QEMU sanity runs
- Full cross-platform support (Linux, Windows, macOS)
- ISO/IEC standards compliance documentation
- MIT license

[0.1.0]: https://github.com/embeddedos-org/eipc/releases/tag/v0.1.0
