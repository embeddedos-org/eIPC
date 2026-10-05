# Zero-copy tensor transport for eIPC

**Status:** Design sketch (Track 1 — tightly-coupled AI).
**Date:** 2026-10-05
**Cross-links:** `eAI` `docs/track1/runtime-api.md` (capability argument),
`eos` `docs/track1/accelerator-hal-profiles.md` (backend tiers),
`eApps` manifest `ai_capabilities` (declarations).

## 1. Transport shape

Models are loadable modules with declared capabilities. Inference moves
tensors between processes — serializing them per call would dominate
latency on MCU-class targets. The data plane is therefore:

- **Shared-memory tensor buffers.** The producer allocates from a shared
  pool; the consumer maps the same pages. No copy on the hot path.
- **Descriptor ring.** Each descriptor carries: buffer handle, shape/dtype,
  owner process, reference count, and a **capability mask**. The ring itself
  is the only thing that crosses the process boundary as a message.
- **Control plane stays on existing eIPC channels.** Session setup,
  capability negotiation, and teardown use the current IPC; only tensor
  payloads take the zero-copy path.

## 2. Capability declarations per module

A module declares, at load time, the capabilities its tensors may carry —
e.g. `camera_frames: read`, `mic_frames: none`. Declarations originate in
the app manifest (`eApps` `ai_capabilities`) and the envelope's capability
claims (eos #162); the transport enforces them per descriptor:

- A descriptor's capability mask is set at publish time from the
  publisher's granted set — never widened in flight.
- A consumer requesting a tensor outside its mask gets a denial, not the
  data. Denials are logged; repeated denials revoke the publisher's
  session (fail closed).

## 3. Lifecycle

```
load → attest → map → infer → unload
```

1. **Load:** module registered, capabilities declared.
2. **Attest:** envelope signature verified (eos #162); claims checked
   against the manifest.
3. **Map:** shared buffers mapped into the consumer; descriptors published
   to the ring.
4. **Infer:** zero-copy dispatch; refcount guards use-after-free across the
   boundary.
5. **Unload:** buffers unmapped, refcounts drained; revocation on
   capability violation is immediate (no grace period).

**Error paths:** publisher crash → ring entries reclaimed by refcount
timeout; capability violation → session revoked + audit log entry;
attestation failure → module never maps.

## 4. Relationship to existing eIPC

This transport does not replace eIPC's message channels — it is a data
plane beside them. Small control messages, RPC, and capability negotiation
stay on the existing paths; only bulk tensor payloads move through shared
memory. Backends that cannot share memory (e.g. across a UART link) fall
back to serialized transport automatically — the API is identical, the
performance is not, and the caller can query which path was taken.
