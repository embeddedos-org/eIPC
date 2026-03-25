# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [0.1.0] — 2026-03-27

### Added
- **Core:** Message types, endpoint API (client/server), priority-lane router with batch dispatch
- **Protocol:** Frame encoding/decoding, HMAC integrity, codec abstraction
- **Security:** Authentication (session tokens), capability-based authorization, HMAC integrity verification, replay detection (sliding window)
- **Transports:** TCP transport with length-prefixed framing, Unix domain socket transport, Windows named pipe transport
- **Services:** Service registry and discovery
- **CLI:** Command-line tools for EIPC management
- **CI/CD:** GitHub Actions workflows for CI, nightly regression, and release automation
- **Project infrastructure:** LICENSE, CONTRIBUTING.md, Makefile
