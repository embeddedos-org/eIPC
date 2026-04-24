# eIPC — Embedded Inter-Process Communication

[![CI](https://github.com/embeddedos-org/eipc/actions/workflows/ci.yml/badge.svg)](https://github.com/embeddedos-org/eipc/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey)](https://github.com/embeddedos-org/eipc)

**Secure, real-time IPC framework for communication between ENI (Neural Interface) and EAI (AI Layer)**

```text
ENI ══▶ EIPC ══▶ EAI
```

EIPC is a **standalone, cross-platform, security-enhanced IPC framework** for embedded and industrial systems with pluggable transports and built-in security services.

---

## Features

- **Real-time capable** — bounded queues, priority lanes, timeout-aware delivery
- **Security-enhanced** — peer auth, capability authorization, HMAC integrity, replay protection
- **Cross-platform** — Linux (amd64/arm64/armv7), macOS (amd64/arm64), Windows (amd64/arm64)
- **Pluggable transports** — TCP, Unix domain sockets, Windows named pipes, shared memory
- **Auditable** — JSON-line audit logging with full request tracing
- **Policy engine** — three-tier action classification (safe/controlled/restricted)
- **Zero external dependencies** — pure Go standard library
- **LTS-friendly** — versioned protocol with compatibility guarantees

---

## Architecture

```text
┌──────────────────────────────────────────────┐
│                 Application                   │
├─────────────┬─────────────┬──────────────────┤
│  core/      │  services/  │  security/       │
│  Message    │  Broker     │  Authenticator   │
│  Router     │  Registry   │  Capability      │
│  Endpoint   │  Policy     │  HMAC Integrity  │
│             │  Audit      │  ReplayTracker   │
│             │  Health     │  Keyring         │
├─────────────┴─────────────┴──────────────────┤
│               protocol/                       │
│  Frame · Codec · Header                      │
├──────────────────────────────────────────────┤
│               transport/                      │
│  TCP · Unix · Windows Pipe · Shared Memory   │
└──────────────────────────────────────────────┘
```

---

## Documentation

| Document | Description |
|----------|-------------|
| [Architecture](docs/architecture.md) | Component diagrams, message flow, wire protocol |
| [API Reference](docs/api-reference.md) | Full godoc-style package documentation |
| [Security Model](docs/security-model.md) | Threat model, HMAC, auth, capability system |
| [Key Management](docs/key-management.md) | Key generation, rotation, storage guidance |
| [Use Cases](docs/use-cases.md) | Deployment scenarios with config examples |
| [Performance](docs/performance.md) | Benchmarks, tuning, backpressure |
| [Benchmarks](docs/benchmarks.md) | Comparisons vs gRPC, ZeroMQ, MQTT |

## Quick Start

### Server

```go
package main

import (
    "log"
    "github.com/embeddedos-org/eipc/core"
    "github.com/embeddedos-org/eipc/protocol"
    "github.com/embeddedos-org/eipc/transport/tcp"
)

func main() {
    t := tcp.New()
    t.Listen("127.0.0.1:9090")
    defer t.Close()
    for {
        conn, _ := t.Accept()
        ep := core.NewServerEndpoint(conn, protocol.DefaultCodec(), []byte("secret-key-32bytes!!"))
        msg, _ := ep.Receive()
        log.Printf("type=%s source=%s", msg.Type, msg.Source)
    }
}
```

### Client

```go
package main

import (
    "time"
    "github.com/embeddedos-org/eipc/core"
    "github.com/embeddedos-org/eipc/protocol"
    "github.com/embeddedos-org/eipc/transport/tcp"
)

func main() {
    t := tcp.New()
    conn, _ := t.Dial("127.0.0.1:9090")
    ep := core.NewClientEndpoint(conn, protocol.DefaultCodec(), []byte("secret-key-32bytes!!"), "")
    ep.Send(core.Message{
        Version: core.ProtocolVersion, Type: core.TypeIntent, Source: "eni.min",
        Timestamp: time.Now().UTC(), RequestID: "req-1", Priority: core.PriorityP0,
        Payload: []byte(`{"intent":"move_left","confidence":0.91}`),
    })
    ep.Close()
}
```

---

## Installation

```bash
go get github.com/embeddedos-org/eipc@latest
```

## Building

```bash
make build            # Current platform
make build-all        # All platforms (cross-compile)
make build-linux      # linux/amd64, linux/arm64, linux/armv7
make build-darwin     # darwin/amd64, darwin/arm64
make build-windows    # windows/amd64, windows/arm64
make release-binaries # Package release archives
make test             # Run all tests
```

### GitHub Release

```bash
gh release create v0.1.0 bin/release/* --title "EIPC v0.1.0" --notes-file CHANGELOG.md
```

---

## API Reference

### Core — `github.com/embeddedos-org/eipc/core`

#### Message

```go
type Message struct {
    Version    uint16      `json:"version"`
    Type       MessageType `json:"type"`
    Source     string      `json:"source"`
    Timestamp  time.Time   `json:"timestamp"`
    SessionID  string      `json:"session_id"`
    RequestID  string      `json:"request_id"`
    Priority   Priority    `json:"priority"`
    Capability string      `json:"capability"`
    Payload    []byte      `json:"payload"`
}
```

#### MessageType & Priority

```go
const (
    TypeIntent       MessageType = "intent"
    TypeFeatures     MessageType = "features"
    TypeToolRequest  MessageType = "tool_request"
    TypeAck          MessageType = "ack"
    TypePolicyResult MessageType = "policy_result"
    TypeHeartbeat    MessageType = "heartbeat"
    TypeAudit        MessageType = "audit"
)

const (
    PriorityP0 Priority = 0  // Control-critical
    PriorityP1 Priority = 1  // Interactive
    PriorityP2 Priority = 2  // Telemetry
    PriorityP3 Priority = 3  // Debug / audit bulk
)
```

#### Functions

| Function | Signature | Description |
|---|---|---|
| `NewMessage` | `(msgType, source, payload) Message` | Creates message with defaults |
| `MsgTypeToByte` | `(mt MessageType) uint8` | Converts to wire byte |

#### Endpoint Interface

```go
type Endpoint interface {
    Send(msg Message) error
    Receive() (Message, error)
    Close() error
}
```

**ClientEndpoint** — `NewClientEndpoint(conn, codec, hmacKey, sessionID)`

| Method | Description |
|---|---|
| `Send(msg) error` | Encodes, signs (HMAC), transmits |
| `Receive() (Message, error)` | Reads, verifies HMAC, decodes |
| `Close() error` | Closes connection |

**ServerEndpoint** — `NewServerEndpoint(conn, codec, hmacKey)`

| Method | Description |
|---|---|
| `Send(msg) error` | Encodes, signs, transmits |
| `Receive() (Message, error)` | Reads, verifies HMAC, checks replay |
| `Close() error` | Closes connection |
| `RemoteAddr() string` | Remote peer address |

#### Router

```go
r := core.NewRouter()
r.Handle(core.TypeIntent, func(msg Message) (*Message, error) { ... })
r.Dispatch(msg)                    // Single message
r.DispatchBatch(msgs)              // Priority-ordered batch (P0 first)
```

#### Event Types

| Type | Key Fields | Description |
|---|---|---|
| `IntentEvent` | Intent, Confidence, SessionID | Neural intent |
| `FeatureStreamEvent` | Features map | Real-time features |
| `ToolRequestEvent` | Tool, Args, Permission, AuditID | Tool request |
| `AckEvent` | RequestID, Status, Error | Acknowledgement |
| `PolicyResultEvent` | RequestID, Allowed, Reason | Auth decision |
| `HeartbeatEvent` | Service, Status | Liveness signal |
| `AuditEvent` | RequestID, Actor, Action, Target, Decision, Result | Audit record |

#### Errors

```go
var (
    ErrAuth         = errors.New("eipc: authentication failed")
    ErrCapability   = errors.New("eipc: capability check failed")
    ErrIntegrity    = errors.New("eipc: integrity verification failed")
    ErrReplay       = errors.New("eipc: replay detected")
    ErrTimeout      = errors.New("eipc: operation timed out")
    ErrBackpressure = errors.New("eipc: backpressure limit reached")
)
```

---

### Protocol — `github.com/embeddedos-org/eipc/protocol`

#### Frame

```go
type Frame struct {
    Version uint16; MsgType uint8; Flags uint8
    Header  []byte; Payload []byte; MAC []byte
}
```

| Constant | Value | Description |
|---|---|---|
| `MagicBytes` | `0x45495043` | ASCII "EIPC" |
| `MaxFrameSize` | 1 MB | Maximum frame size |
| `MACSize` | 32 | HMAC-SHA256 output |
| `ProtocolVersion` | 1 | Wire version |
| `FlagHMAC` | `1<<0` | Frame carries HMAC |
| `FlagCompress` | `1<<1` | Compressed (reserved) |

| Method | Description |
|---|---|
| `Encode(w io.Writer) error` | Write frame in wire format |
| `SignableBytes() []byte` | Bytes covered by MAC |
| `Decode(r io.Reader) (*Frame, error)` | Parse frame from reader |

#### Header

```go
type Header struct {
    ServiceID string; SessionID string; RequestID string
    Sequence uint64; Timestamp string; Priority uint8
    Capability string; Route string; PayloadFormat uint8
}
```

#### Codec

```go
type Codec interface {
    Marshal(v interface{}) ([]byte, error)
    Unmarshal(data []byte, v interface{}) error
}
func DefaultCodec() Codec  // JSONCodec
```

---

### Transport — `github.com/embeddedos-org/eipc/transport`

```go
type Transport interface {
    Listen(address string) error
    Dial(address string) (Connection, error)
    Accept() (Connection, error)
    Close() error
}
type Connection interface {
    Send(frame *protocol.Frame) error
    Receive() (*protocol.Frame, error)
    RemoteAddr() string
    Close() error
}
```

| Package | Platforms | Address Example |
|---|---|---|
| `transport/tcp` | All | `"127.0.0.1:9090"` |
| `transport/unix` | Linux, macOS | `"/tmp/eipc.sock"` |
| `transport/windows` | Windows | `"127.0.0.1:9090"` |
| `transport/shm` | All (in-process) | Ring buffer config |

#### Shared Memory

```go
rb := shm.NewRingBuffer(shm.Config{Name: "eipc", BufferSize: 65536, SlotCount: 256})
conn := shm.NewConnection(txBuf, rxBuf, "peer")
```

---

### Security

#### Auth — `security/auth`

```go
a := auth.NewAuthenticator(secret, map[string][]string{"eni.min": {"ui:control"}})
peer, _ := a.Authenticate("eni.min")      // → PeerIdentity with session token
a.ValidateSession(peer.SessionToken)       // Check session
a.RevokeSession(peer.SessionToken)         // Revoke
```

#### Capability — `security/capability`

```go
c := capability.NewChecker(map[string][]string{"ui:control": {"ui.cursor.move"}})
err := c.Check(peer.Capabilities, "ui.cursor.move")  // nil = allowed
c.Grant("ui:control", "ui.scroll")                    // Runtime grant
c.Revoke("ui:control", "ui.scroll")                   // Runtime revoke
```

#### Integrity — `security/integrity`

```go
mac := integrity.Sign(key, data)              // HMAC-SHA256
ok  := integrity.Verify(key, data, mac)       // Verify
```

#### Replay — `security/replay`

```go
t := replay.NewTracker(128)    // Sliding window
err := t.Check(seq)            // nil = valid
t.Reset()                      // Clear state
```

#### Keyring — `security/keyring`

```go
kr := keyring.New()
entry, _ := kr.Generate("id", 32, 1*time.Hour)
kr.Lookup("id")
kr.Rotate("id", 32, 1*time.Hour)
kr.Cleanup()
```

---

### Services

#### Broker — `services/broker`

```go
brk := broker.NewBroker(registry, auditLogger)
brk.Subscribe(&broker.Subscriber{ServiceID: "eai", Endpoint: ep, Priority: core.PriorityP1})
brk.AddRoute(core.TypeIntent, "eai")
results := brk.Route(msg)       // Priority-ordered delivery
results  = brk.Fanout(msg)      // All subscribers
```

#### Registry — `services/registry`

```go
reg := registry.NewRegistry()
reg.Register(registry.ServiceInfo{ServiceID: "eni.min", Capabilities: []string{"ui:control"}})
info, _ := reg.Lookup("eni.min")
svcs := reg.FindByCapability("ui:control")
```

#### Policy — `services/policy`

```go
pe := policy.NewEngine(true, auditLogger)  // default-deny
pe.LoadSafeDefaults()
result := pe.Evaluate(policy.Request{Source: "eni", Action: "ui.cursor.move"})
// ActionSafe → VerdictAllow | ActionControlled → capability check | ActionRestricted → VerdictConfirm
```

#### Audit — `services/audit`

```go
logger, _ := audit.NewFileLogger("/var/log/eipc-audit.jsonl")  // "" → stdout
logger.Log(audit.Entry{RequestID: "r1", Source: "eni", Action: "move", Decision: "allow"})
```

#### Health — `services/health`

```go
h := health.NewService(5*time.Second, 15*time.Second)
h.RecordHeartbeat("eni.min", "ready")
h.IsAlive("eni.min")   // true if within timeout
h.LivePeers()           // All alive peers
```

---

## Wire Protocol

```text
[magic:4][version:2][msg_type:1][flags:1][header_len:4][payload_len:4][header][payload][mac:32?]
```

Big-endian. Magic = `0x45495043`. Preamble = 16 bytes. MAC present when `FlagHMAC` set.

## Message Types

| Byte | Type | Direction |
|---|---|---|
| `'i'` | intent | ENI→EAI |
| `'f'` | features | ENI→EAI |
| `'t'` | tool_request | EAI→Tool |
| `'a'` | ack | Bidirectional |
| `'p'` | policy_result | EAI→ENI |
| `'h'` | heartbeat | Bidirectional |
| `'u'` | audit | Internal |
| `'c'` | chat | ebot→EAI |
| `'C'` | complete | ebot→EAI |

## Platform Support

| Feature | Linux | macOS | Windows |
|---|:---:|:---:|:---:|
| Core protocol | ✅ | ✅ | ✅ |
| TCP transport | ✅ | ✅ | ✅ |
| Unix sockets | ✅ | ✅ | — |
| Named pipes | — | — | ✅ |
| Shared memory | ✅ | ✅ | ✅ |
| Full security | ✅ | ✅ | ✅ |

## Testing

```bash
make test
```

## C SDK

Available under `sdk/c/`. Build: `cd sdk/c && mkdir build && cd build && cmake .. && make`

## Repository Layout

```text
eipc/
├── cmd/eipc-server/    Server binary
├── cmd/eipc-client/    Client binary
├── core/               Message, Router, Endpoint, Events
├── protocol/           Frame, Header, Codec
├── transport/          TCP, Unix, Windows, SHM
├── security/           Auth, Capability, Integrity, Replay, Keyring
├── services/           Broker, Registry, Policy, Audit, Health
├── sdk/c/              C SDK
├── tests/              Integration tests
└── Makefile            Cross-platform build
```

## Security

EIPC is designed for **security-critical embedded environments**. Key security properties:

- **Authentication** — challenge-response (HMAC-SHA256) with session tokens and TTL-based expiry
- **Authorization** — capability-based access control with runtime grant/revoke
- **Integrity** — HMAC-SHA256 on every frame; constant-time comparison via `crypto/hmac.Equal`
- **Replay protection** — sliding-window nonce tracker rejects duplicate sequence numbers
- **Key management** — in-memory keyring with generation, rotation, expiry, and revocation
- **Audit logging** — JSON-line logs with per-request tracing (file permissions `0600`)
- **Policy engine** — three-tier action classification (safe / controlled / restricted)
- **Zero external dependencies** — pure Go stdlib reduces supply-chain risk

### Reporting Vulnerabilities

If you discover a security vulnerability, please **do not** open a public issue. Instead, email **security@embeddedos.org** with:

1. Description of the vulnerability
2. Steps to reproduce
3. Affected versions

We aim to acknowledge reports within 48 hours and provide a fix within 7 days for critical issues.

See [SECURITY.md](SECURITY.md) for the full security policy.

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Related Projects

| Project | Description |
|---|---|
| [eos](https://github.com/embeddedos-org/eos) | Embedded OS — HAL, kernel, drivers |
| [eboot](https://github.com/embeddedos-org/eboot) | Bootloader — multicore, secure boot |
| [ebuild](https://github.com/embeddedos-org/ebuild) | Unified build system |
| [eai](https://github.com/embeddedos-org/eai) | AI layer — inference, tools, policy |
| [eni](https://github.com/embeddedos-org/eni) | Neural interface — BCI, Neuralink adapter |
| [eApps](https://github.com/embeddedos-org/eApps) | Cross-platform apps (C + LVGL) |
| [eosim](https://github.com/embeddedos-org/eosim) | Multi-architecture simulator |
| [EoStudio](https://github.com/embeddedos-org/EoStudio) | Design suite with LLM integration |

## Standards Compliance

This project is part of the EoS ecosystem and aligns with international standards including ISO/IEC/IEEE 15288:2023, ISO/IEC 12207, ISO/IEC/IEEE 42010, ISO/IEC 25000, ISO/IEC 25010, ISO/IEC 27001, ISO/IEC 15408, IEC 61508, ISO 26262, DO-178C, FIPS 140-3, POSIX (IEEE 1003), WCAG 2.1, and more. See the [EoS Compliance Documentation](https://github.com/embeddedos-org/.github/tree/master/docs/compliance) for full details including NTIA SBOM, SPDX, CycloneDX, and OpenChain compliance.

## License

MIT License — see [LICENSE](LICENSE) for details.
