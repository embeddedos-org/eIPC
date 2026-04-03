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
- Minimum key size: 32 bytes for HMAC-SHA256, 32 bytes for AES-256

---

## 2. Key Storage

### Environment Variables (Development)

```bash
export EIPC_HMAC_KEY="your-32-byte-hex-key-here"
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
