# EIPC Security Model

## Overview

EIPC implements a defense-in-depth security architecture with five layers: authentication, integrity, replay protection, capability-based authorization, and policy enforcement. This document describes each mechanism, the threat model, and known limitations.

---

## Security Pipeline

Every connection passes through the following security stages:

```text
Connection → Authentication → Session Management → Per-Message Security
                                                        ├── HMAC Integrity
                                                        ├── Replay Detection
                                                        ├── Capability Check
                                                        ├── Policy Evaluation
                                                        └── Audit Logging
```

---

## 1. Authentication: Challenge-Response

EIPC uses a 3-round HMAC-based challenge-response protocol.

### Protocol

1. **Client → Server**: `{service_id: "eni.min"}` — Identify the connecting service
2. **Server → Client**: `{nonce: hex(32 random bytes)}` — Server generates cryptographic challenge
3. **Client → Server**: `{response: hex(HMAC-SHA256(shared_secret, nonce))}` — Client proves knowledge of secret

### Security Properties

- **Mutual knowledge proof**: Both parties must possess the shared secret
- **Freshness**: 32-byte random nonce prevents replay of auth handshake
- **No plaintext secret**: The secret is never transmitted — only its HMAC output
- **Timeout**: Server enforces 10-second auth deadline; unauthenticated connections are dropped

### Implementation

```go
authenticator := auth.NewAuthenticator(secret, map[string][]string{
    "eni.min": {"ui:control", "device:read"},
})
challenge, _ := authenticator.CreateChallenge("eni.min")  // 32-byte nonce
peer, _ := authenticator.VerifyResponse("eni.min", clientHMAC)
```

---

## 2. HMAC Integrity

Every frame is signed with HMAC-SHA256 to detect tampering.

### What Is Signed

```text
HMAC-SHA256(key, preamble || header || payload)
```

Where preamble is the 16-byte fixed header:
```text
[magic:4][version:2][msg_type:1][flags:1][header_len:4][payload_len:4]
```

### Verification Flow

1. Sender computes `MAC = HMAC-SHA256(key, SignableBytes())`
2. MAC is appended to the frame (32 bytes) when `FlagHMAC` is set
3. Receiver recomputes HMAC over received `SignableBytes()`
4. Receiver uses `hmac.Equal()` for constant-time comparison
5. Mismatch → `ErrIntegrity` and message is rejected

### Key Properties

- **Constant-time comparison**: Prevents timing side-channel attacks
- **Covers entire frame**: Magic, version, type, flags, header, and payload are all signed
- **Always-on by default**: Both `ClientEndpoint` and `ServerEndpoint` always set `FlagHMAC`

---

## 3. Replay Protection

The server uses a sliding-window algorithm to detect replayed messages.

### Algorithm

- Each message carries a monotonically increasing `Sequence` number in the header
- The `ReplayTracker` maintains a window of the last N sequence numbers (default: 128)
- A sequence number is rejected if:
  - It has been seen before (duplicate)
  - It falls below the window's lower bound (too old)

### Configuration

```go
tracker := replay.NewTracker(256)  // Custom window size
err := tracker.Check(seq)          // nil = valid, error = replay
tracker.Reset()                    // Clear state
```

### Limitations

- **Client-side**: Replay detection is only enforced on `ServerEndpoint.Receive()`. The client does not check replay (by design — responses are correlated by `RequestID`).
- **Window size tradeoff**: Larger windows consume more memory but catch delayed replays. Smaller windows are more memory-efficient but may miss out-of-order legitimate messages.

---

## 4. Capability-Based Authorization

EIPC uses a capability-action model where each authenticated peer receives a set of capabilities, and each action requires a specific capability.

### Model

```text
Peer "eni.min" → capabilities: ["ui:control", "device:read"]

Capability "ui:control" → actions: ["ui.cursor.move", "ui.click", "ui.scroll"]
Capability "device:read" → actions: ["device.sensor.read", "device.status"]
```

### Enforcement

1. On authentication, the server assigns capabilities to the peer based on service ID
2. The `ServerEndpoint` stores the peer's capability list via `SetPeerCapabilities()`
3. Each incoming message declares its required capability in the `Capability` field
4. `ValidateCapability()` checks if the peer possesses the required capability
5. The `CapabilityChecker` validates specific actions against capability rules

### Runtime Modification

```go
checker.Grant("ui:control", "ui.pan")      // Add action at runtime
checker.Revoke("ui:control", "ui.scroll")  // Remove action at runtime
```

---

## 5. Policy Engine

The three-tier policy engine classifies actions and determines authorization verdicts.

| Tier | Action Level | Verdict | Example |
|------|-------------|---------|---------|
| Safe | `ActionSafe` | `VerdictAllow` | `ui.cursor.move`, `device.status` |
| Controlled | `ActionControlled` | Capability check | `device.sensor.read`, `ai.chat.send` |
| Restricted | `ActionRestricted` | `VerdictConfirm` | `system.reboot`, `system.update` |

---

## 6. Encryption (Optional)

EIPC supports optional AES-256-GCM encryption for payload confidentiality.

### Usage

```go
import "github.com/embeddedos-org/eipc/security/encryption"

ciphertext, err := encryption.Encrypt(key32, plaintext)
plaintext, err := encryption.Decrypt(key32, ciphertext)
```

### Wire Format

When `FlagEncrypted` (0x04) is set on the frame:
- The payload field contains `[nonce:12][ciphertext][tag:16]`
- Encryption is applied before HMAC signing
- The recipient decrypts after HMAC verification

### Properties

- AES-256-GCM provides both confidentiality and authenticity
- Random 12-byte nonce per message (never reused with same key)
- Zero external dependencies — uses Go stdlib `crypto/aes` + `crypto/cipher`

---

## 7. Session Management

### Session Tokens

- Generated as hex-encoded 32-byte random values
- Bound to a `PeerIdentity` with service ID, capabilities, and expiration time
- Validated on every message via `peer.IsExpired()`

### Cleanup

- Background goroutine runs every 5 minutes
- Calls `authenticator.CleanupExpired()` to remove stale sessions
- Configurable via `EIPC_SESSION_TTL` environment variable (default: 1 hour)

---

## 8. Connection Limits

- Server enforces maximum concurrent connections via buffered channel semaphore
- Default: 64 (configurable via `EIPC_MAX_CONNECTIONS`)
- Connections exceeding the limit are immediately rejected and audit-logged

---

## Threat Model

### Attacks Mitigated

| Attack | Mitigation | Layer |
|--------|-----------|-------|
| **Message tampering** | HMAC-SHA256 integrity check on every frame | Integrity |
| **Replay attack** | Sliding-window sequence number tracking | Replay |
| **Unauthorized access** | Challenge-response authentication | Auth |
| **Privilege escalation** | Capability-based authorization per action | Capability |
| **Eavesdropping (with TLS)** | TLS 1.3 transport encryption | Transport |
| **Eavesdropping (without TLS)** | Optional AES-256-GCM payload encryption | Encryption |
| **Connection flooding** | Semaphore-based connection limit | Server |
| **Unauthenticated probing** | 10-second auth timeout with forced disconnect | Auth |
| **Brute-force auth** | Audit logging of all failed auth attempts | Audit |

### Attacks Partially Mitigated

| Attack | Current State | Recommendation |
|--------|--------------|----------------|
| **DoS (resource exhaustion)** | Connection limit only; no rate limiting | Add per-IP rate limiting |
| **Key compromise** | Keyring supports rotation but no automatic rotation | Implement scheduled key rotation |
| **Insider threat** | Audit logging captures actions but no anomaly detection | Add anomaly detection hooks |

### Attacks Not Addressed

| Attack | Notes |
|--------|-------|
| **Side-channel attacks** | HMAC uses constant-time comparison, but no broader side-channel hardening |
| **Physical access** | No hardware security module (HSM) integration |
| **Supply chain** | No code signing for binaries (planned for v0.3.0) |
| **Advanced persistent threats** | No intrusion detection system integration |

---

## Key Management Recommendations

See [Key Management Guide](key-management.md) for detailed guidance on:
- Key generation using `security/keyring`
- Key rotation strategies
- Secure storage options
- Key lifecycle (generation → distribution → rotation → revocation → cleanup)

---

## Audit Trail

All security-relevant events are logged as JSON lines via the audit service:

```json
{"timestamp":"2026-04-03T12:00:00Z","request_id":"req-1","source":"eni.min","target":"eipc-server","action":"authenticate","decision":"allowed","result":"session created"}
{"timestamp":"2026-04-03T12:00:01Z","request_id":"req-2","source":"eni.min","target":"eipc-server","action":"ui.cursor.move","decision":"allowed","result":"success"}
{"timestamp":"2026-04-03T12:00:02Z","request_id":"req-3","source":"eni.min","target":"eipc-server","action":"system.reboot","decision":"denied","result":"capability violation"}
```

Events captured:
- Authentication success/failure
- Session creation/expiration/cleanup
- Capability violations
- Policy decisions (allow/deny/confirm)
- Connection limit rejections
- Message dispatch results

---

## Configuration Reference

| Environment Variable | Purpose | Default |
|---------------------|---------|---------|
| `EIPC_HMAC_KEY` | Shared HMAC secret (plaintext) | (required) |
| `EIPC_KEY_FILE` | Path to HMAC key file (alternative) | — |
| `EIPC_SESSION_TTL` | Session lifetime (Go duration) | `1h` |
| `EIPC_MAX_CONNECTIONS` | Max concurrent connections | `64` |
| `EIPC_TLS_CERT` | TLS certificate path | — |
| `EIPC_TLS_KEY` | TLS private key path | — |
| `EIPC_TLS_AUTO_CERT` | Auto-generate self-signed cert | `false` |
