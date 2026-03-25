# EIPC

**Embedded IPC for secure, real-time communication between ENI and EAI**

```text
ENI ==>> EIPC ==>> EAI
```

EIPC is a **standalone, cross-platform, security-enhanced IPC framework** designed for:

* **ENI** — Neural Interface Adapter
* **EAI** — AI Layer
* embedded and industrial systems
* real-time local communication
* long-term support (LTS) deployments

It provides a **portable core**, **OS-specific transports**, and **security services** for safe, deterministic, low-latency system communication.

---

## Overview

EIPC exists to solve the core integration problem in intelligent embedded systems:

> **How do ENI and EAI communicate securely, predictably, and efficiently across device classes and operating systems?**

EIPC provides a layered answer:

* **IPC for speed**
* **eventing for scale**
* **APIs for external integration**
* **security services for trust, policy, and auditability**

---

## Design Goals

EIPC is built to be:

* **real-time capable**
* **OS-agnostic at the core**
* **embedded-friendly**
* **secure by design**
* **scalable from minimal to framework systems**
* **stable for LTS deployments**

---

# Architecture

## Core communication model

```text
ENI
 ↓
[ Event / Intent Layer ]
 ↓
[ EIPC ]
 ↓
EAI (Agent / Runtime / Tools)
```

EIPC is the secure communication layer between ENI and EAI.

---

## Expanded secure architecture

```text
ENI Service
   ↓
EIPC Client
   ↓
Secure Local Transport
   ↓
EIPC Router / Broker (optional)
   ↓
EAI Service / Policy Engine / Tool Executor
```

For minimal deployments, EIPC can be direct client/server.

For framework or industrial deployments, EIPC can run with a local secure broker and service registry.

---

# Why EIPC

General IPC mechanisms such as raw sockets, DBus, or generic RPC frameworks are useful, but EIPC is built specifically for:

* low-latency embedded communication
* deterministic control paths
* strong local trust boundaries
* capability-based authorization
* policy-controlled tool execution
* long-lived internal protocol compatibility

---

# Key Benefits

## 1. Stable Internal Contract

EIPC defines a canonical message model for:

* intents
* feature streams
* tool requests
* acknowledgements
* policy results
* audit events
* service health and heartbeat

## 2. Real-Time Control

EIPC is designed for:

* bounded queues
* local-only transport
* priority lanes
* timeout-aware delivery
* minimal protocol overhead

## 3. Security-Enhanced Services

EIPC includes:

* authenticated peers
* capability-scoped authorization
* message integrity
* replay protection
* service registration
* audit logging
* policy enforcement

## 4. Cross-Platform Portability

EIPC is designed with:

* **portable core**
* **pluggable transport backends**
* **OS-specific optimizations**

## 5. LTS-Friendly Evolution

EIPC controls:

* protocol versioning
* compatibility guarantees
* deprecation policy
* security model evolution
* audit semantics

---

# Platform Strategy

## Core + Adapters

EIPC is split into:

* **portable core**
* **platform-specific transport and runtime adapters**

```text
eipc/
├── core/
├── protocol/
├── security/
├── runtime/
├── transport/
│   ├── unix/
│   ├── windows/
│   ├── tcp/
│   └── shm/
├── services/
└── sdk/
```

---

## Supported Platforms

EIPC can be designed to run on:

* **Linux** — Ubuntu, embedded Linux, EoS
* **Windows**
* **macOS** for development and testing
* **containers** such as Docker or Kubernetes
* other platforms via transport adapters

---

## Platform Capability Matrix

| Feature             | Linux | Windows | macOS | Containers |
| ------------------- | ----: | ------: | ----: | ---------: |
| Core protocol       |     ✅ |       ✅ |     ✅ |          ✅ |
| Local IPC           |     ✅ |       ✅ |     ✅ |         ⚠️ |
| Shared memory       |     ✅ |      ⚠️ |    ⚠️ |          ❌ |
| Real-time tuning    |     ✅ |      ⚠️ |     ❌ |          ❌ |
| Full security model |     ✅ |       ✅ |     ✅ |          ✅ |

---

## Performance Reality

Cross-platform does not mean equal performance.

| Platform        | Expected performance                        |
| --------------- | ------------------------------------------- |
| EoS / Linux | Best                                        |
| Ubuntu / Linux  | Very strong                                 |
| Windows         | Good                                        |
| Containers      | Good for development and distributed setups |

> EIPC is portable by design, but Linux-class systems remain the strongest target for real-time embedded deployments.

---

# Communication Model

EIPC supports a layered communication strategy.

## Primary: Local IPC

Best for:

* embedded systems
* low-latency communication
* ENI-Min ↔ EAI-Min
* deterministic real-time paths

Supported implementations:

* Unix domain sockets
* named pipes
* loopback gRPC where appropriate
* shared memory for advanced cases

---

## Secondary: Event Bus

Best for:

* framework systems
* multiple consumers
* observability and analytics
* industrial fan-out

Supported styles:

* internal lightweight event bus
* MQTT integration
* NATS integration

---

## External / Distributed: APIs

Best for:

* cross-device communication
* remote orchestration
* cloud integration
* external tools

Supported styles:

* gRPC
* REST APIs

---

# Recommended Default Usage

## Minimal deployments

```text
ENI → EIPC → EAI
```

Use:

* local IPC
* structured messages
* synchronous control pipeline

## Framework deployments

```text
ENI → EIPC bus/router → EAI + services
                      → observability
                      → storage
                      → connectors
```

Use:

* brokered routing
* policy services
* audit services
* optional event fan-out

## Distributed deployments

```text
ENI → EIPC gateway/API → EAI remote services
```

Use:

* gRPC
* REST
* secure remote integration patterns

---

# Message Model

EIPC defines one shared schema between ENI and EAI.

## Standard event

```json
{
  "version": "v1",
  "type": "intent",
  "source": "eni",
  "timestamp": "2026-03-24T10:15:00Z",
  "payload": {
    "intent": "move_left",
    "confidence": 0.91
  }
}
```

---

## Message classes

### Intent message

```json
{
  "type": "intent",
  "intent": "move_left",
  "confidence": 0.91,
  "session_id": "sess-1"
}
```

### Feature stream message

```json
{
  "type": "features",
  "features": {
    "attention": 0.72,
    "motor_imagery": "left"
  }
}
```

### Tool request message

```json
{
  "type": "tool_request",
  "tool": "ui.cursor.move",
  "args": {
    "direction": "left",
    "step": 1
  }
}
```

### Acknowledgement message

```json
{
  "type": "ack",
  "request_id": "req-44",
  "status": "ok"
}
```

### Policy result message

```json
{
  "type": "policy_result",
  "request_id": "req-44",
  "allowed": true
}
```

### Heartbeat message

```json
{
  "type": "heartbeat",
  "service": "eni-min",
  "status": "ready"
}
```

### Audit event

```json
{
  "type": "audit",
  "request_id": "req-3001",
  "payload": {
    "actor": "eni.min",
    "action": "ui.cursor.move",
    "result": "success"
  }
}
```

---

# Protocol Design

## Frame structure

EIPC uses a versioned frame format.

```text
[magic][version][msg_type][flags][header_len][payload_len][header][payload][mac]
```

## Suggested header fields

* `service_id`
* `session_id`
* `request_id`
* `sequence`
* `timestamp`
* `priority`
* `capability`
* `route`
* `payload_format`

This design supports:

* stable parsing
* compatibility evolution
* integrity checks
* routing and priority handling

---

# Payload Formats

## Development mode

Use **JSON**

Pros:

* easy to inspect
* easy to debug
* easy to log
* ideal for early development

## Production mode

Use **MessagePack**

Pros:

* compact
* faster
* lower overhead
* still structured

## Recommendation

* JSON for development and debugging
* MessagePack for production
* same schema across both

---

# Transport Modes

EIPC supports multiple transport modes under one common API.

## 1. Control Channel

Use:

* Unix domain sockets on Linux / EoS
* named pipes on Windows

Best for:

* intents
* commands
* acks
* policy decisions

## 2. Stream Channel

Use:

* shared memory ring buffer later
* low-copy local channels

Best for:

* high-rate feature streams
* robotics
* real-time workloads

## 3. Event Channel

Use:

* internal pub/sub
* observability/event sinks

Best for:

* logs
* telemetry
* audit fan-out
* non-critical status traffic

---

# Security Model

EIPC is security-enhanced from day one.

## Security principles

* local-only by default
* explicit service identity
* capability-based authorization
* tamper-evident messaging
* replay protection
* policy-controlled execution
* auditable actions

---

## Minimum security requirements

* peer identity validation
* permission tags on messages
* allowlist of callable tools
* audit IDs for controlled actions
* session binding
* integrity protection

Example:

```json
{
  "type": "tool_request",
  "tool": "actuator.write",
  "permission": "device:write",
  "audit_id": "audit-00912"
}
```

---

## Security-enhanced services

### Peer Authentication Service

Every ENI/EAI service must prove identity.

Examples:

* `eni.min`
* `eni.framework`
* `eai.min.agent`
* `eai.framework.policy`

Possible mechanisms:

* local service identity
* signed service manifests
* startup-issued session token
* OS credential checks

---

### Authorization / Capability Service

Every request must carry capability scope.

Examples:

* `ui:control`
* `device:read`
* `device:write`
* `iot:publish`
* `system:restricted`

---

### Integrity Protection

Every message should be tamper-evident.

Recommended minimum:

* session MAC / HMAC
* sequence number
* timestamp
* request ID

---

### Replay Protection

Recommended protections:

* monotonic counters
* nonce/session binding
* time-window checks

---

### Audit Service

Every controlled action should be auditable.

Audit fields should include:

* request ID
* source identity
* target service
* action
* policy decision
* timestamp
* result

---

### Policy Enforcement Service

Policy should exist in the EIPC layer, not only in app logic.

Required flow:

```text
ENI request
→ EIPC identity check
→ capability check
→ EAI policy check
→ tool execution
→ audit log
```

Never allow:

```text
ENI → direct unsafe system control
```

---

### Secure Discovery / Registration

Services must register using:

* declared capabilities
* supported versions
* message types
* priority classes

This prevents untrusted local processes from impersonating valid components.

---

# Encryption Guidance

## Mandatory

* peer authentication
* message integrity

## Recommended

* encryption for framework/industrial/high-sensitivity deployments

For local-only hardened systems, integrity + auth may be enough for some minimal deployments.

For industrial and cognitive-sensitive systems, per-session encryption is recommended.

### Minimal mode

* authentication mandatory
* integrity mandatory
* encryption optional

### Framework mode

* authentication mandatory
* integrity mandatory
* encryption recommended

---

# Real-Time Design Rules

To keep EIPC suitable for ENI and EAI, define these rules early.

## Rules

* no unbounded queues
* no blocking writes on critical paths
* timeout-aware sends
* explicit backpressure handling
* defined drop policy for non-critical traffic

## Priority lanes

* `P0` — control critical
* `P1` — interactive
* `P2` — telemetry
* `P3` — debug / audit bulk

Priority behavior may vary by policy:

* `P0` can disallow dynamic routing
* `P2` and `P3` can allow buffered delivery

---

# Deployment Profiles

## EIPC-Min

Best for:

* ENI-Min ↔ EAI-Min
* assistive UI
* edge control
* handheld/mobile-class embedded systems

Recommended stack:

* Unix domain sockets
* OS credential check
* session token
* HMAC-protected messages
* strict local-only mode
* allowlist capabilities

---

## EIPC-Framework

Best for:

* ENI-Framework ↔ EAI-Framework
* industrial gateways
* robotics
* larger embedded systems

Recommended stack:

* broker/router
* service registration
* session key establishment
* signed manifests
* policy engine
* audit service
* priority lanes
* optional encrypted local transport

---

# Example Flows

## ENI-Min + EAI-Min

```text
ENI-Min
  └── EIPC client
        ↓
     EIPC socket
        ↓
EAI-Min
  └── EIPC server
```

Flow:

1. ENI receives decoded intent
2. ENI normalizes it
3. ENI sends intent over EIPC
4. EAI validates and maps tool
5. EAI returns ack/result

---

## ENI-Framework + EAI-Framework

```text
ENI-Framework
  └── EIPC producer(s)
        ↓
   EIPC routing/runtime
        ↓
EAI-Framework services
  ├── orchestrator
  ├── policy engine
  ├── observability
  └── connector manager
```

---

## End-to-end secure minimal flow

```text
1. eni.min connects to EIPC socket
2. EIPC validates peer identity
3. EIPC issues session token
4. eni.min sends MAC-protected intent
5. eai.min.agent receives request
6. capability ui:control is checked
7. action allowed
8. tool executes: ui.cursor.move
9. audit event recorded
10. ack returned
```

---

## End-to-end secure framework flow

```text
1. eni.framework registers with broker
2. provider stream enters secure routing lane
3. message normalized and classified
4. policy requires confirmation
5. operator approval granted
6. orchestrator invokes allowed connector
7. audit and metrics recorded
8. result returned
```

---

# Example API Surface

## Go interface

```go
type Message struct {
    Version   uint16
    Type      string
    SessionID string
    Priority  uint8
    Payload   []byte
}

type Endpoint interface {
    Send(msg Message) error
    Receive() (Message, error)
    Close() error
}
```

## Higher-level helper

```go
type IntentEvent struct {
    Intent     string  `json:"intent"`
    Confidence float64 `json:"confidence"`
    SessionID  string  `json:"session_id"`
}
```

---

# Repository Layout

```text
eipc/
├── core/
├── protocol/
├── security/
│   ├── auth/
│   ├── capability/
│   ├── integrity/
│   ├── replay/
│   └── keyring/
├── runtime/
├── transport/
│   ├── unix/
│   ├── windows/
│   ├── tcp/
│   └── shm/
├── services/
│   ├── broker/
│   ├── policy/
│   ├── audit/
│   ├── registry/
│   └── health/
├── sdk/
│   ├── go/
│   ├── c/
│   └── cpp/
└── tests/
```

---

# Service Set

## Core services

* `eipc-authd` — identity and session service
* `eipc-policyd` — authorization service
* `eipc-auditd` — audit sink
* `eipc-regd` — service registry and discovery
* `eipc-healthd` — health and heartbeat service

For minimal deployments, several services can be embedded into one daemon.

For framework deployments, keep them separate.

---

# Build Roadmap

## Phase 1

Build:

* protocol schema
* Unix socket transport
* Go SDK
* JSON payload support
* simple client/server demo

## Phase 2

Integrate with:

* `eni-min-service`
* `eai-min-agent`

## Phase 3

Add:

* MessagePack mode
* priorities
* backpressure handling
* observability hooks

## Phase 4

Add advanced transports and services:

* shared memory ring buffer
* Windows adapter
* broker/runtime for framework systems
* encryption support

## Phase 5

LTS hardening:

* protocol freeze for v1
* compatibility guarantees
* security review
* audit format stabilization
* deterministic deployment profiles

---

# Recommended Starting Point

For a practical v1, start with:

* Unix domain sockets
* structured messages
* versioned protocol
* JSON payloads
* service identity
* HMAC-protected messages
* sequence numbers
* capability policy
* audit logs

This is enough to deliver:

* speed
* portability
* debuggability
* security
* LTS stability

# Recommended EIPC variants

## 1. **EIPC-Lite**

Smallest footprint.

Best for:

* simple local IPC
* dev/test
* tiny embedded systems
* single producer ↔ single consumer

Characteristics:

* no broker
* no heavy security services
* direct point-to-point transport
* basic auth/integrity only
* lowest memory use

Use cases:

* `ENI-Min ↔ EAI-Min`
* simulator ↔ agent
* handheld/mobile-class devices

---

## 2. **EIPC-Min**

Secure minimal production profile.

Best for:

* real embedded deployments
* assistive systems
* low-latency control
* bounded service count

Characteristics:

* local IPC transport
* service identity
* capability checks
* HMAC/integrity
* audit hooks
* strict local-only mode

Use cases:

* production `ENI-Min ↔ EAI-Min`
* edge controllers
* compact AI appliances

---

## 3. **EIPC-Framework**

Full scalable profile.

Best for:

* industrial systems
* robotics
* multi-service orchestration
* observability and policy-heavy environments

Characteristics:

* optional broker/router
* service registry
* policy daemon
* audit daemon
* priority lanes
* richer routing
* optional encryption
* multi-client and multi-subscriber support

Use cases:

* `ENI-Framework ↔ EAI-Framework`
* industrial gateways
* large embedded AI systems

---

# Best way to think about them

Not:

* three separate repos
* three incompatible protocols

Instead:

> **one EIPC protocol and core**
> with **three deployment profiles**

That is much cleaner for LTS.

---

# Recommended relationship

```text
EIPC
├── core
├── protocol
├── transport
├── security
├── runtime
└── profiles
    ├── lite
    ├── min
    └── framework
```

---

# What changes between Lite, Min, and Framework

| Feature              |      EIPC-Lite | EIPC-Min | EIPC-Framework |
| -------------------- | -------------: | -------: | -------------: |
| Core protocol        |              ✅ |        ✅ |              ✅ |
| Direct local IPC     |              ✅ |        ✅ |              ✅ |
| Broker/router        |              ❌ |        ❌ |              ✅ |
| Capability auth      |          basic |        ✅ |              ✅ |
| Message integrity    | optional/basic |        ✅ |              ✅ |
| Audit logging        |        minimal |        ✅ |     ✅ advanced |
| Policy engine        |              ❌ |    basic |              ✅ |
| Priority lanes       |          basic |        ✅ |     ✅ advanced |
| Shared memory stream |              ❌ | optional |       optional |
| Encryption           |              ❌ | optional |    recommended |
| Observability        |        minimal |    basic |           full |

---

# My naming recommendation

Use these exact names:

* **EIPC-Lite**
* **EIPC-Min**
* **EIPC-Framework**

This reads well and is easy to explain.

---

# Mapping to ENI / EAI

## Smallest setup

* `ENI-Min + EAI-Min` → **EIPC-Lite** or **EIPC-Min**

## Real production embedded setup

* `ENI-Min + EAI-Min` → **EIPC-Min**

## Industrial / large setup

* `ENI-Framework + EAI-Framework` → **EIPC-Framework**

---

# Practical recommendation

If you want the cleanest product story:

* **EIPC-Lite** = developer/lightweight mode
* **EIPC-Min** = embedded production mode
* **EIPC-Framework** = industrial/full mode

That is a strong architecture and branding model.

# Summary

**EIPC** is a secure, standalone, cross-platform IPC framework for ENI and EAI.

It is designed to provide:

* real-time local communication
* security-enhanced services
* stable internal contracts
* scalable architecture from minimal to industrial systems
* portability across Linux, Windows, macOS, and containers

> Use **IPC for speed**,
> **eventing for scale**,
> **APIs for integration**,
> and **EIPC for trust, control, and long-term stability**.
