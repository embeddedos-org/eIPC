# EIPC Architecture

## Overview

EIPC (Embedded Inter-Process Communication) is a layered, security-enhanced IPC framework designed for communication between ENI (Neural Interface) and EAI (AI Layer) in the EoS ecosystem. It features pluggable transports, a binary wire protocol, priority-aware message routing, and a comprehensive security pipeline.

---

## Component Architecture

```mermaid
graph TB
    subgraph Application["Application Layer"]
        CMD["cmd/<br/>eipc-server · eipc-client · eipc-cli"]
    end

    subgraph Services["Services Layer"]
        BRK["Broker<br/>pub/sub routing"]
        REG["Registry<br/>service discovery"]
        POL["Policy Engine<br/>3-tier authorization"]
        AUD["Audit Logger<br/>JSON-line tracing"]
        HLT["Health Monitor<br/>heartbeat tracking"]
    end

    subgraph Core["Core Layer"]
        MSG["Message<br/>canonical envelope"]
        EP["Endpoint<br/>Client · Server"]
        RTR["Router<br/>priority-lane dispatch"]
        EVT["Events<br/>Intent · Chat · Tool · Ack"]
    end

    subgraph Security["Security Layer"]
        AUTH["Authenticator<br/>challenge-response"]
        CAP["Capability Checker<br/>action allowlists"]
        HMAC["HMAC Integrity<br/>SHA-256 signing"]
        AES["AES-GCM Encryption<br/>payload confidentiality"]
        RPL["Replay Tracker<br/>sliding window"]
        KR["Keyring<br/>key lifecycle"]
    end

    subgraph Protocol["Protocol Layer"]
        FRM["Frame<br/>binary wire format"]
        HDR["Header<br/>routing metadata"]
        CDC["Codec<br/>JSON · Protobuf · CBOR"]
    end

    subgraph Transport["Transport Layer"]
        TCP["TCP<br/>+ optional TLS/mTLS"]
        UNX["Unix Socket<br/>Linux · macOS"]
        WIN["Named Pipe<br/>Windows"]
        SHM["Shared Memory<br/>ring buffer"]
    end

    CMD --> Services
    CMD --> Core
    Services --> Core
    Core --> Security
    Core --> Protocol
    Protocol --> Transport
    Security --> Protocol
```

---

## Message Flow

The following diagram shows the complete lifecycle of a message from client to server:

```mermaid
sequenceDiagram
    participant Client as Client Application
    participant CE as ClientEndpoint
    participant Codec as Codec (JSON)
    participant Frame as Frame Encoder
    participant HMAC as HMAC-SHA256
    participant Transport as Transport (TCP/Unix/SHM)
    participant SE as ServerEndpoint
    participant Replay as Replay Tracker
    participant Cap as Capability Checker
    participant Router as Router
    participant Handler as Message Handler

    Client->>CE: Send(Message)
    CE->>Codec: Marshal(Header)
    CE->>Frame: Build Frame (preamble + header + payload)
    CE->>HMAC: Sign(key, SignableBytes)
    Frame-->>CE: Frame with MAC appended
    CE->>Transport: Send(length-prefixed frame)

    Transport->>SE: Receive(length-prefixed frame)
    SE->>HMAC: Verify(key, SignableBytes, MAC)
    alt HMAC Invalid
        SE-->>Client: ErrIntegrity
    end
    SE->>Codec: Unmarshal(Header)
    SE->>Replay: Check(sequence)
    alt Replay Detected
        SE-->>Client: ErrReplay
    end
    SE->>Cap: ValidateCapability(msg.Capability)
    alt Capability Denied
        SE-->>Client: ErrCapability
    end
    SE->>Router: Dispatch(Message)
    Router->>Handler: handler(msg)
    Handler-->>Router: *Response
    Router-->>SE: Response
    SE->>Transport: Send(response)
    Transport->>CE: Receive(response)
```

---

## Authentication Handshake

EIPC uses a 3-round challenge-response protocol for peer authentication:

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server
    participant Auth as Authenticator

    Note over C,S: Round 1: Identity
    C->>S: {service_id: "eni.min"} (TypeAck)

    Note over S,Auth: Lookup service, generate nonce
    S->>Auth: CreateChallenge("eni.min")
    Auth-->>S: Challenge{Nonce: 32 bytes}

    Note over C,S: Round 2: Challenge
    S->>C: {status: "challenge", nonce: hex(nonce)} (TypeAck)

    Note over C: Compute HMAC-SHA256(shared_secret, nonce)
    C->>S: {service_id: "eni.min", response: hex(hmac)} (TypeAck)

    Note over S,Auth: Round 3: Verify
    S->>Auth: VerifyResponse("eni.min", hmac_bytes)
    Auth-->>S: PeerIdentity{SessionToken, Capabilities}

    S->>C: {status: "ok", session_token: "...", capabilities: [...]} (TypeAck)

    Note over C,S: Authenticated session established
    C->>S: Messages with capability-gated actions
```

---

## Wire Protocol Format

```text
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
├─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┤
│                         Magic (0x45495043)                         │  Bytes 0-3
├─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┤
│         Version (uint16)        │  MsgType  │    Flags    │         Bytes 4-7
├─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┤
│                       Header Length (uint32)                       │  Bytes 8-11
├─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┤
│                      Payload Length (uint32)                       │  Bytes 12-15
├─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┤
│                     Header (variable length)                      │
├─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┤
│                    Payload (variable length)                       │
├─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┼─┤
│                  MAC (32 bytes, if FlagHMAC set)                   │
└─┴─┴─┴─┴─┴─┴─┴─┴─┴─┴─┴─┴─┴─┴─┴─┴─┴─┴─┴─┴─┴─┴─┴─┴─┴─┴─┴─┴─┴─┴─┴─┘
```

**Preamble**: 16 bytes, big-endian.

| Field | Bytes | Description |
|-------|-------|-------------|
| Magic | 4 | `0x45495043` (ASCII "EIPC") |
| Version | 2 | Protocol version (currently `1`) |
| MsgType | 1 | Message type wire byte |
| Flags | 1 | Bitfield: `FlagHMAC` (0x01), `FlagCompress` (0x02), `FlagEncrypted` (0x04) |
| Header Length | 4 | Length of JSON header |
| Payload Length | 4 | Length of payload |

**Maximum frame size**: 1 MB (header + payload).

---

## Priority Lanes

The Router uses a heap-based priority queue to ensure critical messages are dispatched first:

| Priority | Value | Use Case | Latency Target |
|----------|-------|----------|----------------|
| P0 | 0 | Control-critical (motor commands, safety) | < 1ms |
| P1 | 1 | Interactive (UI events, chat) | < 10ms |
| P2 | 2 | Telemetry (sensor data streams) | < 100ms |
| P3 | 3 | Debug / audit bulk | Best-effort |

---

## Transport Architecture

```mermaid
graph LR
    subgraph Transports
        TCP["TCP Transport<br/>All platforms<br/>Optional TLS/mTLS"]
        Unix["Unix Socket<br/>Linux · macOS<br/>File-based addressing"]
        Pipe["Named Pipe<br/>Windows<br/>Over TCP fallback"]
        SHM["Shared Memory<br/>All platforms<br/>Ring buffer (in-process)"]
    end

    subgraph Common
        CW["ConnWrapper<br/>Length-prefixed framing<br/>4-byte big-endian prefix"]
    end

    TCP --> CW
    Unix --> CW
    Pipe --> CW
    SHM -.-> |"Direct ring buffer I/O"| SHM
```

All stream-based transports (TCP, Unix, Pipe) use `ConnWrapper` which adds a 4-byte big-endian length prefix before each encoded frame. The SHM transport uses direct ring buffer read/write for zero-copy performance.

---

## Security Pipeline

Every incoming message on the server passes through this pipeline:

```mermaid
graph TD
    A["Receive Frame"] --> B{"HMAC Flag Set?"}
    B -->|Yes| C["Verify HMAC-SHA256"]
    B -->|No| D["Skip HMAC"]
    C -->|Invalid| E["Reject: ErrIntegrity"]
    C -->|Valid| F["Decode Header"]
    D --> F
    F --> G["Check Replay (sequence)"]
    G -->|Duplicate| H["Reject: ErrReplay"]
    G -->|Valid| I["Check Session TTL"]
    I -->|Expired| J["Reject: Session Expired"]
    I -->|Valid| K["Validate Capability"]
    K -->|Denied| L["Reject: ErrCapability"]
    K -->|Allowed| M["Dispatch to Router"]
    M --> N["Execute Handler"]
    N --> O["Send Response"]
```

---

## Package Dependencies

```mermaid
graph TD
    CMD["cmd/"] --> CFG["config/"]
    CMD --> CORE["core/"]
    CMD --> PROTO["protocol/"]
    CMD --> SEC["security/"]
    CMD --> SVC["services/"]
    CMD --> TRANS["transport/"]

    CORE --> PROTO
    CORE --> SEC_INT["security/integrity"]
    CORE --> SEC_RPL["security/replay"]
    CORE --> TRANS

    SVC --> CORE
    SVC --> SEC

    TRANS --> PROTO
```

Zero external dependencies — the entire framework is built on Go's standard library.
