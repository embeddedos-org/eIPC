# EIPC Performance Guide

## Overview

EIPC is designed for real-time embedded IPC with bounded latency and predictable throughput. This document covers performance characteristics, tuning guidance, and benchmarking.

---

## Benchmark Suite

Run benchmarks with:

```bash
make bench
# Or directly:
go test -bench=. -benchmem ./...
```

### Core Benchmarks

| Benchmark | Description |
|-----------|-------------|
| `BenchmarkNewMessage` | Message creation with defaults |
| `BenchmarkRouterDispatch` | Single message dispatch to handler |
| `BenchmarkRouterDispatchBatch` | Priority-ordered batch dispatch (4 messages) |
| `BenchmarkMsgTypeToByte` | Wire byte conversion |

### Protocol Benchmarks

| Benchmark | Description |
|-----------|-------------|
| `BenchmarkFrameEncode` | Frame encoding to bytes |
| `BenchmarkFrameDecode` | Frame decoding from bytes |
| `BenchmarkSignableBytes` | HMAC signable byte extraction |

---

## Priority Lanes

The Router uses a heap-based priority queue (`container/heap`) to ensure P0 messages are dispatched before P1, P2, P3:

| Priority | Target Latency | Typical Use |
|----------|---------------|-------------|
| P0 | < 1ms | Motor commands, safety interlocks |
| P1 | < 10ms | UI events, chat messages |
| P2 | < 100ms | Sensor telemetry streams |
| P3 | Best-effort | Audit logs, debug bulk |

**Behavior**: `DispatchBatch()` sorts messages by priority before processing. Single `Dispatch()` calls bypass the queue for zero overhead.

---

## Backpressure Handling

### Connection Semaphore

The server limits concurrent connections using a buffered channel:

```go
connSem := make(chan struct{}, maxConns)
select {
case connSem <- struct{}{}:
    go handleConnection(conn)
default:
    conn.Close() // Reject: limit reached
}
```

**Default**: 64 concurrent connections (`EIPC_MAX_CONNECTIONS`).

### SHM Ring Buffer

The shared memory transport uses a fixed-size ring buffer. When full:

- **Writer blocks** until a slot is freed
- No data is dropped
- Ensures zero message loss at the cost of latency spikes under overload

### Queue Sizing Guidance

| Scenario | Buffer Size | Slot Count | Notes |
|----------|-------------|------------|-------|
| Low-latency control | 16KB | 64 | Minimize buffering |
| Sensor streaming | 64KB | 256 | Balance throughput/memory |
| Bulk data transfer | 256KB | 1024 | Maximize throughput |

---

## Memory Safety

### Race Detector

All tests run with `-race` flag:

```bash
go test -race -v ./...
```

### Stress Tests

The `tests/stress_test.go` suite validates:

- **Large payloads**: 512KB message integrity through encode/decode/HMAC pipeline
- **Concurrent clients**: 10 simultaneous connections, each sending and receiving
- **Message ordering**: 50 sequential messages maintain delivery order

### Memory Allocation

Key allocations per message cycle:

| Operation | Allocations | Notes |
|-----------|-------------|-------|
| Frame encode | 2-3 | Preamble buffer + write |
| Frame decode | 2-3 | Read buffers + frame struct |
| HMAC sign | 1 | MAC output (32 bytes) |
| JSON marshal | 1-2 | Header serialization |

---

## Transport Performance Comparison

| Transport | Round-Trip Latency | Throughput | Use Case |
|-----------|-------------------|------------|----------|
| SHM (ring buffer) | ~1 μs | Very high | Same-process goroutines |
| Unix socket | ~10 μs | High | Same-host processes |
| TCP (loopback) | ~100 μs | High | Cross-host or testing |
| TCP + TLS | ~500 μs | Moderate | Production network |

---

## Tuning Recommendations

### For Minimum Latency

- Use SHM transport for same-process communication
- Set priority to P0 for critical messages
- Use single `Dispatch()` instead of `DispatchBatch()` for individual messages
- Disable TLS for loopback-only deployments

### For Maximum Throughput

- Increase SHM ring buffer size and slot count
- Use `DispatchBatch()` to amortize lock overhead
- Increase `EIPC_MAX_CONNECTIONS` for high fan-in scenarios
- Consider disabling replay detection if sequence monotonicity is guaranteed

### For Production Safety

- Always enable HMAC (default)
- Enable TLS for any network transport
- Set appropriate session TTL
- Monitor audit logs for performance degradation signals
