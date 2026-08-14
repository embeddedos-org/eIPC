# eIPC — Secure Inter-Process Communication

[![CI](https://github.com/embeddedos-org/eIPC/actions/workflows/ci.yml/badge.svg)](https://github.com/embeddedos-org/eIPC/actions/workflows/ci.yml)
[![CodeQL](https://github.com/embeddedos-org/eIPC/actions/workflows/codeql.yml/badge.svg)](https://github.com/embeddedos-org/eIPC/actions/workflows/codeql.yml)
[![Scorecard](https://github.com/embeddedos-org/eIPC/actions/workflows/scorecard.yml/badge.svg)](https://github.com/embeddedos-org/eIPC/actions/workflows/scorecard.yml)
[![Release](https://github.com/embeddedos-org/eIPC/actions/workflows/release.yml/badge.svg)](https://github.com/embeddedos-org/eIPC/actions/workflows/release.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

eIPC is a transport-agnostic inter-process communication library written in Go
(module `github.com/embeddedos-org/eipc`). It exchanges messages as framed
envelopes over pluggable transports and signs every frame for integrity. It is
part of the [EmbeddedOS (EoS)](https://github.com/embeddedos-org) ecosystem,
where it provides the messaging layer between services.

## Features

Observed in the source tree:

- **Pluggable transports** — TCP, Unix domain sockets, shared memory, and
  Windows named pipes (`transport/`).
- **Framed binary protocol** — length-prefixed frames with a header/payload
  codec (`protocol/`).
- **Frame integrity** — HMAC-SHA256 signing of frames (`security/integrity`)
  and sequence-based replay protection (`security/replay`).
- **Security building blocks** — auth, capability tokens, encryption, and a
  keyring (`security/`), plus optional TLS on the TCP transport
  (`EIPC_TLS_AUTO_CERT`).
- **Services** — broker, registry, health, policy, and audit (`services/`).
- **C SDK** — a C binding under `sdk/c`.

## What's inside

| Path | Contents |
|------|----------|
| `core/` | `Endpoint` API, message envelope, router, lifecycle |
| `protocol/` | Frame/header definitions and codec |
| `transport/` | `tcp`, `unix`, `shm`, `windows` transports |
| `security/` | `auth`, `capability`, `encryption`, `integrity`, `keyring`, `replay` |
| `services/` | `broker`, `registry`, `health`, `policy`, `audit` |
| `config/` | Runtime configuration (e.g. listen address, TLS) |
| `cmd/` | `eipc-server`, `eipc-client`, `eipc-cli` binaries |
| `sdk/c/` | C SDK |
| `examples/` | `hello-eipc` minimal client/server |
| `tests/` | unit, functional, integration, performance, simulation |
| `docs/` | Architecture, API reference, security model |

## Requirements

- Go 1.22+

## Build

```bash
make build          # builds bin/eipc-server and bin/eipc-client
make build-cli      # builds bin/eipc-cli
make build-all      # cross-compiles for linux/darwin/windows (amd64/arm64/armv7)
```

## Run the example

From two terminals (see `examples/hello-eipc/README.md`):

```bash
# Terminal 1 — server
cd examples/hello-eipc && go run ./server/

# Terminal 2 — client
cd examples/hello-eipc && go run ./client/
```

The `eipc-cli` debugging tool can send, listen for, and ping messages against a
running server:

```bash
eipc-cli send   --addr 127.0.0.1:9090 --type chat --payload '{"text":"hello"}'
eipc-cli listen --addr 127.0.0.1:9090
eipc-cli ping   --addr 127.0.0.1:9090
```

## Test

```bash
make test           # go test -race ./...
make bench          # go test -bench=. -benchmem ./core/ ./protocol/
make vet            # go vet ./...
```

## Documentation

Docs live under `docs/` and are published with MkDocs (`mkdocs.yml`):
<https://embeddedos-org.github.io/eIPC/>.

## License

MIT — see [LICENSE](LICENSE).

Part of [embeddedos-org](https://github.com/embeddedos-org).
