# Zero-copy tensor descriptor transport (Track 1, phase 2)

**Status:** Implemented in `protocol/tensor.go` + `protocol/tensor_test.go` (landed 2026-10-11).
**Date:** 2026-10-11
**Cross-links:** `docs/track1/zero-copy-tensor-ipc.md` (the design sketch this
implements), eAI `accel/include/eai/accel.h` (`eai_dtype_t` values mirrored),
`transport/shm/shm.go` (segment named by the descriptor), `core/message.go`
(`Capability` binding), eos #162 (capability envelope claims).

## 1. What a descriptor is

`protocol.TensorDescriptor` is a small, copyable handle:

```
Dtype        TensorDtype        // mirrors eai_dtype_t (F32=0 … Q8_0=7)
Shape        []int64            // e.g. [1 3 224 224]
Strides      []int64            // e.g. [150528 50176 224 1]
ShmName      string             // transport/shm segment — the zero-copy handle
Offset, Size uint64             // byte span claimed inside that segment
Capabilities []string           // #162-envelope capability claims, e.g. "inference.local"
ModelDigest  string             // sha256 of the signed model producing/consuming
```

Only the descriptor crosses the IPC boundary — as a `protocol.Frame`
payload or inside `core.Message.Payload`. The tensor bytes never cross it:
the consumer maps the named segment and reads `[Offset, Offset+Size)`
directly. This is the data plane from the phase-1 sketch, pinned to the
existing shm transport instead of a new descriptor ring.

## 2. Lifecycle: descriptor → shm segment → Message{Capability} binding

```
publish                                        consume
───────                                        ───────
1. Producer writes tensor bytes into            1. Control channel delivers a
   the shm segment ("eai-infer-0")                 descriptor (Frame payload).
2. Producer builds the descriptor,             2. Consumer DecodeTensor +
   sets ShmName/Offset/Size,                     Validate BEFORE mapping.
   Capabilities from its granted set.          3. Consumer checks the
3. Producer Encode()s it onto the                 capability claims against
   existing control channel.                      its own grant; denial =>
                                                   drop, audit-log.
                                                4. Consumer maps the segment,
                                                   reads only [Offset, Size).
```

- **Descriptor transport.** `(*TensorDescriptor).Encode(io.Writer)` /
  `DecodeTensor(io.Reader)` use the same big-endian length-prefixed style as
  `protocol.Frame`. JSON tags are present so a descriptor can also ride the
  existing JSON header codec when convenient.
- **`core.Message{Capability}` binding.** The `Capability` field on
  `core.Message` (and `protocol.Header.Capability`) is the existing choke
  point for capability-scoped IPC. A tensor publish MUST set it to the
  narrowest capability covering the descriptor's claims (e.g.
  `Capability: "inference.local"`), so the transport's policy layer sees the
  same grant the descriptor asserts. The two cannot drift: `Validate` checks
  the descriptor's claims are non-empty-shaped, and the message layer
  enforces the envelope grant.
- **`transport/shm` as backing store.** `ShmName` names a segment managed
  like `transport/shm.RingBuffer` (see `shm.go`): the producer allocates,
  the consumer maps read-only. Offset/Size are the only window the consumer
  is entitled to.

## 3. Validate: consistency before mapping

`Validate()` is the mandatory pre-map gate — decode is not enough, because a
well-formed wire encoding can still describe an impossible or hostile
tensor:

- Dtype must be one of the transported values (rejects eAI-only Q4_1 and
  anything unknown — numeric drift with `eai_dtype_t` would mis-size reads).
- 1 ≤ ndim ≤ 8 (`EAI_TENSOR_MAX_DIMS`); no zero or negative dims.
- `len(Strides) == len(Shape)`; strides non-negative and overflow-guarded.
- **Footprint check:** `Size ≥ (1 + Σ (shape[i]-1)·strides[i]) · dtypeBytes`
  with element sizes from eAI's `eai_dtype_size` (INT4/Q dtypes count 1 byte
  per element at stride level). A descriptor claiming less than its footprint
  would let a reader walk off the end of its mapping — rejected.
- `Size > 0` requires a non-empty `ShmName` — an unnamed span is a handle to
  nothing, and mapping it would fault or, worse, resolve to whatever the
  platform's default segment is.
- `ModelDigest`, when present, must be 64 hex chars (optionally
  `sha256:`-prefixed) — it names the signed model, so it must actually name
  one.

Decode enforces its own wire bounds (ndim ≤ 8, string length caps) so a
truncated or hostile stream fails fast instead of allocating.

## 4. Threat note: a handle must not confer broader memory access

A tensor descriptor is a *capability-shaped* object: possession of it grants
access to shared memory. The threat model is therefore the bounded-handle
lesson from Track 1 (cf. the MaxKB-style reasoning): a handle that names a
span must never widen into access to the whole segment, let alone the whole
address space.

- **Narrow the window.** The consumer is entitled to exactly
  `[Offset, Offset+Size)` — never the full segment, never adjacent tensors.
  Map read-only where the platform allows (see `shm.go`: the in-process ring
  buffer is a placeholder; the mmap backends must map consumer views
  `PROT_READ`).
- **Capability-scoped, not identity-scoped.** `Capabilities` are claims from
  the #162 envelope, set at publish time from the producer's *granted* set
  and never widened in flight. A consumer outside the claim set gets a
  denial and an audit-log entry — not the data. Repeated violations revoke
  the session (fail closed), per the phase-1 sketch.
- **No ambient authority.** `ShmName` alone is not authorization; it is a
  name. Mapping requires the descriptor to have arrived over an
  authenticated channel with a matching `Capability` grant, and to have
  passed `Validate`. A forged descriptor pointing at another module's
  segment must fail at the grant check, not at the page fault.
- **Digest binds the model.** `ModelDigest` ties the tensor to the signed
  model that produced it; a consumer that only trusts model M must reject
  descriptors whose digest is not M's — otherwise a tensor from a
  compromised model can be laundered through a trusted producer's segment.
- **Lifetime.** The descriptor does not own the segment. Producer teardown
  must unmap/unlink; consumers must treat `Offset/Size` as invalid after
  the descriptor's session ends (refcount/timeout reclaim, per phase 1).

## 5. What's next (phase 3)

- mmap-backed consumer views in `transport/shm` (`shm_linux.go`) with
  read-only mappings, so the narrow-window rule is hardware-enforced.
- ~~`core.Message` helper to publish a descriptor with the matching
  `Capability` set atomically~~ — landed 2026-10-13, see section 6.
- Quantized-dtype packing contracts (INT4/Q4_0/Q8_0 element layout) so
  producer and consumer agree on byte packing inside the claimed span.


## 6. Message binding (landed 2026-10-13)

The phase-3 helper from section 5 now exists. Because `core` already
imports `protocol` (the endpoint codec), the binding is two-layered —
a `core.Message` cannot be named from inside `protocol` without an
import cycle:

- `protocol/tensor_message.go` — the capability contract, shared by
  every envelope: `(*TensorDescriptor).MessagePayload()` validates
  the descriptor, encodes it, and returns the primary capability (the
  first declared claim); `TensorFromPayload(payload, capability)`
  decodes, validates, and requires the asserted capability to be one
  of the descriptor's claims.
- `core/tensor_message.go` — `AttachTensor(d)` builds a `core.Message`
  with the descriptor as `Payload` and `Capability` set from the same
  validated descriptor, atomically; `ExtractTensor(m)` is the
  receiving half.

Fail-closed rules, enforced identically on both paths:

- A descriptor declaring **no capabilities is refused at attach** —
  a tensor that names no grant does not travel.
- Extraction **rejects a capability mismatch**: a `Capability`
  widened, swapped, or stripped in flight no longer matches the
  descriptor's claims, and the descriptor is not handed out.

**Zero-copy note.** What crosses a process boundary in this binding
is the descriptor only: it carries arena **handles** (`ShmName` +
`Offset` + `Size`) and declared capabilities — raw pointers never
cross. This follows the dma-buf/IIO handle pattern: the exporter
allocates the segment, importers map by handle, and the broker/kernel
validates the mapping against the declared window. Today's binding
transports the descriptor; shared-arena mapping by handle is the
phase-2 step, and the descriptor property/fuzz tests will guard it.
