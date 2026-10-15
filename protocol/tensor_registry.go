// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package protocol

import (
	"fmt"
	"sort"
	"sync"
)

// TensorRegistry tracks the live tensor spans of a session, keyed by
// shm segment name. It is the registration-side check the eNI
// security audit's phase 2 (finding F2-1) calls for: a single
// TensorDescriptor can be validated in isolation by Validate(), but
// overlap is a *session* property — only a registry that sees every
// span in a segment can refuse two live descriptors claiming the
// same bytes.
//
// The registry owns no memory and maps nothing; it is bookkeeping
// for the session layer that hands descriptors to consumers, so a
// descriptor that was never registered (or was deregistered) is
// not a live span, and a span that is still registered cannot be
// handed out twice.
type TensorRegistry struct {
	mu    sync.Mutex
	spans map[string][]registeredSpan
}

type registeredSpan struct {
	offset uint64
	size   uint64
	desc   TensorDescriptor
}

// NewTensorRegistry returns an empty registry.
func NewTensorRegistry() *TensorRegistry {
	return &TensorRegistry{spans: make(map[string][]registeredSpan)}
}

// Register validates d and records its span as live in its segment.
//
// A descriptor that fails Validate never enters the registry. A
// zero-size descriptor is refused outright: it claims no bytes, so
// registering it could only mask a bookkeeping bug. Within one
// segment (one ShmName), the half-open span [Offset, Offset+Size)
// must not intersect any live span; adjacency (end == start) is
// allowed. Spans in different segments are independent and never
// conflict. Validate() has already established that Offset+Size
// does not wrap, so the intersection test cannot overflow.
func (r *TensorRegistry) Register(d TensorDescriptor) error {
	if err := d.Validate(); err != nil {
		return err
	}
	if d.Size == 0 {
		return fmt.Errorf("%w: zero-size span", ErrTensorDims)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	live := r.spans[d.ShmName]
	for _, s := range live {
		if d.Offset < s.offset+s.size && s.offset < d.Offset+d.Size {
			return fmt.Errorf("%w: [%d, %d) intersects live span [%d, %d) in segment %q",
				ErrTensorOverlap, d.Offset, d.Offset+d.Size, s.offset, s.offset+s.size, d.ShmName)
		}
	}
	r.spans[d.ShmName] = append(live, registeredSpan{offset: d.Offset, size: d.Size, desc: d})
	return nil
}

// Deregister removes the live span starting at offset in the named
// segment, reporting whether such a span existed. Removing a span
// frees its bytes for a later Register.
func (r *TensorRegistry) Deregister(shmName string, offset uint64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	live := r.spans[shmName]
	for i, s := range live {
		if s.offset == offset {
			r.spans[shmName] = append(live[:i], live[i+1:]...)
			return true
		}
	}
	return false
}

// Spans returns the live descriptors registered in the named
// segment, sorted by offset. The result is a copy; mutating it
// does not affect the registry.
func (r *TensorRegistry) Spans(shmName string) []TensorDescriptor {
	r.mu.Lock()
	defer r.mu.Unlock()

	live := r.spans[shmName]
	out := make([]TensorDescriptor, len(live))
	for i, s := range live {
		out[i] = s.desc
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Offset < out[j].Offset })
	return out
}
