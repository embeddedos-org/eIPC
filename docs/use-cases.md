# EIPC Use Cases

This document provides concrete deployment scenarios for EIPC, including recommended transports, configuration, and security considerations.

---

## 1. Embedded Sensor ↔ Controller

**Scenario**: A BCI (Brain-Computer Interface) sensor board communicates neural intent signals to a local motor controller running on the same SoC.

**Architecture**:
```text
┌──────────────────┐     SHM Ring Buffer     ┌──────────────────┐
│  ENI Sensor       │ ════════════════════╗   │  EAI Controller  │
│  (goroutine)      │                     ╚══▶│  (goroutine)     │
│  - reads BCI data │                         │  - processes     │
│  - emits intents  │                         │    intent        │
│  - P0 priority    │                         │  - drives motors │
└──────────────────┘                          └──────────────────┘
```

**Recommended Transport**: `transport/shm` (Shared Memory Ring Buffer)

**Why**: Zero-copy, sub-microsecond latency for same-process communication. No network overhead.

**Configuration**:
```go
import "github.com/embeddedos-org/eipc/transport/shm"

txBuf := shm.NewRingBuffer(shm.Config{
    Name:       "eni-to-eai",
    BufferSize: 65536,  // 64KB ring buffer
    SlotCount:  256,    // 256 message slots
})
rxBuf := shm.NewRingBuffer(shm.Config{
    Name:       "eai-to-eni",
    BufferSize: 65536,
    SlotCount:  256,
})
conn := shm.NewConnection(txBuf, rxBuf, "eni.sensor")
```

**Security Considerations**:
- HMAC integrity still applies (protect against memory corruption)
- No network exposure — attack surface limited to process boundary
- Use P0 priority for safety-critical intent messages

---

## 2. Cross-Process on Same Board

**Scenario**: Multiple services run as separate processes on a Raspberry Pi — a sensor daemon, an AI inference service, and a policy engine. They communicate via Unix domain sockets.

**Architecture**:
```text
┌────────────────┐
│  ENI Daemon     │──┐
│  (process 1)    │  │
└────────────────┘  │
                    │  Unix Socket: /tmp/eipc.sock
┌────────────────┐  │  ┌──────────────────┐
│  EAI Agent      │──┼──│  EIPC Server     │
│  (process 2)    │  │  │  (process 0)     │
└────────────────┘  │  │  - auth gateway   │
                    │  │  - policy engine   │
┌────────────────┐  │  │  - message broker │
│  Tool Service   │──┘  └──────────────────┘
│  (process 3)    │
└────────────────┘
```

**Recommended Transport**: `transport/unix` (Unix Domain Sockets)

**Why**: Low-latency IPC without TCP overhead. File-system permissions add an extra security layer.

**Configuration**:
```bash
# Server
export EIPC_HMAC_KEY="your-32-byte-secret-key-here!!!"
export EIPC_LISTEN_ADDR="/tmp/eipc.sock"
export EIPC_SESSION_TTL="1h"
export EIPC_MAX_CONNECTIONS="16"
./eipc-server
```

```go
import "github.com/embeddedos-org/eipc/transport/unix"

server := unix.New()
server.Listen("/tmp/eipc.sock")
defer server.Close()

for {
    conn, _ := server.Accept()
    go handleConnection(conn)
}
```

**Security Considerations**:
- Set socket file permissions to restrict access (`chmod 660 /tmp/eipc.sock`)
- Use capability-based auth: sensor gets `device:read`, AI gets `ui:control`
- Enable replay protection (default sliding window of 128)
- Set reasonable session TTL (e.g., 1 hour with background cleanup)

---

## 3. Multi-Platform Deployment

**Scenario**: An industrial control system has a Linux-based edge device, a macOS development workstation, and a Windows HMI (Human-Machine Interface). All communicate over TCP with TLS.

**Architecture**:
```text
┌──────────────────┐         TLS/TCP          ┌──────────────────┐
│  Linux Edge       │ ══════════════════════╗  │  EIPC Server     │
│  (arm64)          │                       ╠═▶│  (linux/amd64)   │
│  - sensor data    │                       ║  │  - centralized   │
└──────────────────┘                        ║  │    broker        │
                                            ║  │  - policy engine │
┌──────────────────┐                        ║  │  - audit logging │
│  macOS Dev        │ ══════════════════════╣  └──────────────────┘
│  (arm64)          │                       ║
│  - monitoring     │                       ║
└──────────────────┘                        ║
                                            ║
┌──────────────────┐                        ║
│  Windows HMI      │ ═════════════════════╝
│  (amd64)          │
│  - operator UI    │
└──────────────────┘
```

**Recommended Transport**: `transport/tcp` with TLS enabled

**Why**: Cross-platform, encrypted transport over any network topology.

**Build & Deploy**:
```bash
# Cross-compile for all platforms
make build-all

# Resulting binaries:
# bin/eipc-server-linux-amd64    bin/eipc-client-linux-arm64
# bin/eipc-server-darwin-arm64   bin/eipc-client-darwin-arm64
# bin/eipc-server-windows-amd64  bin/eipc-client-windows-amd64
```

**TLS Configuration**:
```bash
# Option A: Auto-generated self-signed cert (development)
export EIPC_TLS_AUTO_CERT=true

# Option B: Provide your own certificates (production)
export EIPC_TLS_CERT=/etc/eipc/server.crt
export EIPC_TLS_KEY=/etc/eipc/server.key

export EIPC_HMAC_KEY="production-secret-key-32bytes!!"
export EIPC_LISTEN_ADDR="0.0.0.0:9090"
export EIPC_MAX_CONNECTIONS="128"
./eipc-server
```

**Security Considerations**:
- Always enable TLS for network-facing deployments
- Use proper CA-signed certificates in production (not `InsecureSkipVerify`)
- Rotate HMAC keys periodically using the `security/keyring` package
- Separate capability grants per platform role (edge: `device:read`, HMI: `ui:control`)
- Enable audit logging to persistent storage for compliance

---

## 4. eBot Chat Integration

**Scenario**: An AI chatbot (ebot) connects to EIPC to send chat messages and receive completions from the EAI layer.

**Architecture**:
```text
┌──────────────────┐     TCP      ┌──────────────────┐     Internal     ┌──────────────────┐
│  eBot Client      │ ══════════▶ │  EIPC Server     │ ═══════════════▶ │  EAI Agent       │
│  - sends prompts  │             │  - auth gateway   │                  │  - LLM inference │
│  - receives       │ ◀══════════ │  - chat routing   │ ◀═══════════════ │  - tool calls    │
│    completions    │             │  - audit trail    │                  │  - completions   │
└──────────────────┘              └──────────────────┘                  └──────────────────┘
```

**Message Flow**:
1. eBot authenticates with `ai:chat` capability
2. Sends `TypeChat` message with `ChatRequestEvent` payload
3. Server validates capability for `ai.chat.send` action
4. Server forwards to EAI agent
5. EAI responds with `ChatResponseEvent`

```go
chatReq := core.ChatRequestEvent{
    SessionID:  "session-123",
    UserPrompt: "Explain EIPC security model",
    Model:      "llama3",
    MaxTokens:  1024,
}
payload, _ := codec.Marshal(chatReq)
ep.Send(core.Message{
    Type:       core.TypeChat,
    Source:     "ebot.client",
    Capability: "ai:chat",
    Payload:    payload,
})
```

---

## Transport Selection Guide

| Scenario | Transport | Latency | Security | Platform |
|----------|-----------|---------|----------|----------|
| Same goroutine / thread | SHM | ~1μs | HMAC only | All |
| Same board, different process | Unix Socket | ~10μs | HMAC + file perms | Linux, macOS |
| Same board, Windows | Named Pipe (TCP) | ~100μs | HMAC + TLS | Windows |
| Cross-network | TCP + TLS | ~1ms | Full stack | All |
| Development / testing | TCP (plaintext) | ~100μs | HMAC only | All |
