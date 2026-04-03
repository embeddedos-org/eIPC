# EIPC Benchmarks & Comparisons

## How EIPC Compares

EIPC is purpose-built for embedded and industrial IPC where security, real-time performance, and zero external dependencies are critical. This document compares EIPC against common alternatives.

---

## EIPC vs gRPC

| Feature | EIPC | gRPC |
|---------|------|------|
| **Dependencies** | Zero (Go stdlib only) | Protobuf, gRPC runtime, HTTP/2 |
| **Binary size** | ~5 MB | ~15-30 MB |
| **Wire format** | Custom binary (16-byte preamble) | Protobuf over HTTP/2 |
| **Transport** | TCP, Unix, SHM, Named Pipe | HTTP/2 (TCP only) |
| **Local IPC** | SHM ring buffer (~1μs) | Over loopback (~200μs) |
| **Security** | Built-in HMAC, capability auth, replay protection | TLS + interceptors |
| **Priority lanes** | P0-P3 with heap dispatch | No built-in priority |
| **Code generation** | None required | Requires protoc |
| **Embedded targets** | ARM, RISC-V, cross-compiled | Limited embedded support |

**When to use EIPC**: Same-board IPC, embedded systems, when zero-dependency deployment matters, when you need built-in security beyond TLS.

**When to use gRPC**: Microservices, polyglot environments, when you need streaming, when ecosystem tooling (load balancers, service mesh) is important.

---

## EIPC vs ZeroMQ

| Feature | EIPC | ZeroMQ |
|---------|------|--------|
| **Dependencies** | Zero | libzmq (C library) |
| **Security** | HMAC, capability auth, replay, optional AES-GCM | CurveZMQ (optional) |
| **Auth model** | Challenge-response + capability-based | None built-in (app-level) |
| **Message types** | 9 typed messages | Raw bytes |
| **Routing** | Priority-aware router with policy engine | Patterns (pub/sub, push/pull) |
| **Audit trail** | Built-in JSON-line logging | None |
| **Wire format** | Typed, versioned frames | Raw frames |

**When to use EIPC**: When you need authentication, authorization, and audit out of the box. When message types and priority lanes matter.

**When to use ZeroMQ**: High-throughput pub/sub, fan-out patterns, when you need advanced messaging patterns like dealer/router.

---

## EIPC vs MQTT

| Feature | EIPC | MQTT |
|---------|------|------|
| **Broker** | Optional (direct client-server) | Mandatory broker |
| **QoS** | Priority lanes (P0-P3) | 3 QoS levels |
| **Security** | HMAC + capability + replay + encryption | Username/password + TLS |
| **Latency (local)** | ~1μs (SHM), ~10μs (Unix) | ~1ms (via broker) |
| **Protocol overhead** | 16-byte preamble | 2-byte fixed header |
| **Session management** | Built-in with TTL and cleanup | Broker-managed |
| **Topic routing** | Message type + capability-based | Topic string hierarchy |

**When to use EIPC**: Peer-to-peer communication, same-board IPC, when broker overhead is unacceptable, when you need fine-grained authorization.

**When to use MQTT**: IoT cloud connectivity, many-to-many pub/sub, when you need retained messages and last-will features.

---

## Running Benchmarks

```bash
# Run all benchmarks
make bench

# Run specific package benchmarks
go test -bench=. -benchmem ./core/
go test -bench=. -benchmem ./protocol/

# Run with CPU profiling
go test -bench=BenchmarkFrameEncode -cpuprofile=cpu.prof ./protocol/
go tool pprof cpu.prof

# Run with memory profiling
go test -bench=BenchmarkRouterDispatch -memprofile=mem.prof ./core/
go tool pprof mem.prof
```

---

## Stress Test Results

Run with:
```bash
go test -race -v -run TestStress ./tests/
```

| Test | Description | Validates |
|------|-------------|-----------|
| `TestStress_LargePayload` | 512KB message through full pipeline | Memory safety, HMAC correctness |
| `TestStress_ConcurrentClients` | 10 simultaneous connections | Race conditions, goroutine safety |
| `TestStress_MessageOrdering` | 50 sequential messages | Delivery order preservation |
