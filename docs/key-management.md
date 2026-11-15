# EIPC Key Management Guide

## Overview

This guide covers secure key management for EIPC deployments, including generation, distribution, rotation, and revocation using the `security/keyring` package.

---

## Key Types in EIPC

| Key | Purpose | Size | Lifetime |
|-----|---------|------|----------|
| HMAC Key | Message integrity (HMAC-SHA256) | 32 bytes | Long-lived (rotate monthly) |
| Encryption Key | Payload encryption (AES-256-GCM) | 32 bytes | Long-lived (rotate monthly) |
| Session Token | Session identification | 32 bytes | Short-lived (1h default) |
| Auth Nonce | Challenge-response freshness | 32 bytes | Single-use |

---

## Per-service keys

The global key above (`EIPC_HMAC_KEY` / `EIPC_KEY_FILE`) is one secret
shared by every service: a leak of that one value compromises every
service's frames. Per-service keys give each service its own
purpose-separated secret, so a leak is contained to the service it
belongs to. They are loaded by `config.LoadHMACKeyForService(service)`
and `config.LoadHMACKeys(services...)`.

| Variable | Purpose |
|----------|---------|
| `EIPC_HMAC_KEY_<SERVICE>` | HMAC key for one service (env var) |
| `EIPC_KEY_FILE_<SERVICE>` | Path to one service's key file |
| `EIPC_HMAC_KEY` | Global key — fallback when a service has no per-service key |
| `EIPC_KEY_FILE` | Global key file — fallback |

`<SERVICE>` is the service name upper-cased with `.`, `-`, and `_`
mapped to `_`: service `eipc-server` reads
`EIPC_HMAC_KEY_EIPC_SERVER`; service `ebot.client` reads
`EIPC_HMAC_KEY_EBOT_CLIENT`. Lookup order for a service is
per-service env var → per-service key file → global key. Keys from any
source must be at least 32 bytes — `LoadHMACKey` and
`LoadHMACKeyForService` reject shorter keys at startup instead of
handing a weak key to the endpoints. Key files may end with a newline;
it is stripped on load.

```bash
# One key per service; the global key stays only as a fallback.
export EIPC_HMAC_KEY_EIPC_SERVER="$(openssl rand -hex 32)"
export EIPC_HMAC_KEY_EBOT_CLIENT="$(openssl rand -hex 32)"
export EIPC_HMAC_KEY="$(openssl rand -hex 32)"  # fallback until migration completes
```

### Migration from the global key

1. **Generate per-service keys now.** Set `EIPC_HMAC_KEY_<SERVICE>`
   (or `EIPC_KEY_FILE_<SERVICE>`) for every service. The loaders
   already prefer them; nothing else changes yet.
2. **Keep the global key as fallback** while the endpoints are still
   wired to the single global key (the follow-up wiring PR). A service
   with no per-service key keeps working on the global key.
3. **Remove the global key** once the endpoints load per-service keys:
   unset `EIPC_HMAC_KEY` / `EIPC_KEY_FILE`. A service that then has no
   key fails to start with an explicit error instead of silently
   running unsecured.

   Upgrade server and client together: a key file's trailing newline is
   stripped on load, so the effective key changes when either side
   upgrades — a mixed-version fleet will fail HMAC verification.

---

## 1. Key Generation

### Using the Keyring API

```go
import "github.com/embeddedos-org/eipc/security/keyring"

kr := keyring.New()

// Generate a new HMAC key with 1-hour TTL
entry, err := kr.Generate("hmac-primary", 32, 24*time.Hour)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Key ID: %s, Expires: %s\n", entry.ID, entry.ExpiresAt)
```

### Using Command Line

```bash
# Generate a 32-byte random key
openssl rand -hex 32

# Or using Go
go run -e 'import "crypto/rand"; import "encoding/hex"; import "fmt"; b := make([]byte, 32); rand.Read(b); fmt.Println(hex.EncodeToString(b))'
```

### Best Practices

- Always use cryptographically secure random number generators (`crypto/rand`)
- Never use predictable or human-readable passwords as HMAC keys
- Minimum key size: 32 bytes for HMAC-SHA256, 32 bytes for AES-256.
  The 32-byte HMAC minimum is enforced by `config.LoadHMACKey` /
  `config.LoadHMACKeyForService`: a shorter key fails startup with an
  error naming the length, instead of silently weakening the MAC.

---

## 2. Key Storage

### Environment Variables (Development)

```bash
export EIPC_HMAC_KEY="your-32-byte-hex-key-here-0000000"
```

**Pros**: Simple, no file I/O.
**Cons**: Visible in process listings, shell history.

### Key Files (Production)

```bash
# Create key file with restricted permissions
openssl rand -hex 32 > /etc/eipc/hmac.key
chmod 600 /etc/eipc/hmac.key
chown eipc:eipc /etc/eipc/hmac.key

export EIPC_KEY_FILE=/etc/eipc/hmac.key
```

**Pros**: File permissions, auditable access.
**Cons**: Key at rest on disk.

### Secrets Managers (Enterprise)

For production deployments, integrate with your organization's secrets manager:

| Platform | Integration |
|----------|-------------|
| HashiCorp Vault | `vault kv get -field=key secret/eipc/hmac` |
| AWS Secrets Manager | `aws secretsmanager get-secret-value --secret-id eipc/hmac` |
| GCP Secret Manager | `gcloud secrets versions access latest --secret=eipc-hmac` |
| Kubernetes | `kubectl get secret eipc-hmac -o jsonpath='{.data.key}'` |

Example wrapper script:
```bash
#!/bin/bash
export EIPC_HMAC_KEY=$(vault kv get -field=key secret/eipc/hmac)
exec ./eipc-server "$@"
```

---

## 3. Key Rotation

### Using the Keyring API

```go
kr := keyring.New()

// Store the current key
kr.Store("hmac-v1", currentKey, 0)

// Generate a new key
newEntry, _ := kr.Rotate("hmac-v1", 32, 24*time.Hour)

// The old key is automatically revoked
// Use newEntry.Key for all new operations
```

### Rotation Strategy

```text
Day 0:  Generate Key V1, deploy to all nodes
Day 30: Generate Key V2, deploy to all nodes
        - Server accepts both V1 and V2 (grace period)
Day 31: Remove Key V1 from all nodes
        - Server only accepts V2
```

### Zero-Downtime Rotation

1. **Generate new key**: `kr.Generate("hmac-v2", 32, 30*24*time.Hour)`
2. **Deploy new key**: Push to all server and client nodes
3. **Grace period**: Accept both old and new keys (24 hours)
4. **Revoke old key**: `kr.Revoke("hmac-v1")`
5. **Cleanup**: `kr.Cleanup()` removes expired/revoked keys

---

## 4. Key Revocation

```go
kr.Revoke("hmac-compromised")  // Mark as revoked (keep for audit)
kr.Delete("hmac-old")           // Permanently remove
```

### When to Revoke

- Suspected key compromise
- Employee departure who had key access
- After scheduled rotation
- Security incident response

---

## 5. Key Lifecycle Summary

```text
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│ Generate  │───▶│  Store    │───▶│  Active   │───▶│  Rotate   │───▶│  Revoke  │
│           │    │          │    │          │    │          │    │          │
│ crypto/   │    │ env var  │    │ signing  │    │ new key  │    │ mark     │
│ rand      │    │ file     │    │ verifying│    │ generated│    │ inactive │
│ 32 bytes  │    │ vault    │    │          │    │ old key  │    │          │
│           │    │          │    │          │    │ revoked  │    │          │
└──────────┘    └──────────┘    └──────────┘    └──────────┘    └──────────┘
                                                                      │
                                                                      ▼
                                                                ┌──────────┐
                                                                │  Cleanup  │
                                                                │          │
                                                                │ permanent│
                                                                │ deletion │
                                                                └──────────┘
```

---

## 6. Security Checklist

- [ ] Keys are at least 32 bytes, generated from `crypto/rand`
- [ ] Keys are never hardcoded in source code
- [ ] Key files have restricted permissions (600 or 400)
- [ ] Keys are rotated at least every 30 days
- [ ] Compromised keys are immediately revoked
- [ ] Key rotation is tested in staging before production
- [ ] All key access is audit-logged
- [ ] Backup keys are stored in a separate secure location
- [ ] Key distribution uses encrypted channels (TLS, VPN, or secrets manager)
