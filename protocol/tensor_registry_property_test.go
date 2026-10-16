// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package protocol_test

import (
	"errors"
	"math/rand"
	"testing"

	"github.com/embeddedos-org/eipc/protocol"
)

// Property tests for TensorRegistry: the table-driven cases in
// tensor_registry_test.go pin individual overlaps, but overlap is a
// session property (eNI audit F2-1) — it must hold under arbitrary
// interleavings of Register/Deregister across segments, not just in
// the cases someone thought to enumerate. A simple model of the
// live spans is maintained alongside the registry; after every
// random operation the registry must agree with the model exactly,
// and its live spans must be pairwise disjoint and offset-sorted.

type modelSpan struct {
	offset uint64
	size   uint64
}

func spansOverlap(a, b modelSpan) bool {
	return a.offset < b.offset+b.size && b.offset < a.offset+a.size
}

func TestTensorRegistryPropertyRandomOps(t *testing.T) {
	rng := rand.New(rand.NewSource(20261016))
	segments := []string{"seg-a", "seg-b"}
	r := protocol.NewTensorRegistry()
	model := map[string][]modelSpan{"seg-a": nil, "seg-b": nil}

	checkInvariants := func(step int) {
		t.Helper()
		for _, seg := range segments {
			got := r.Spans(seg)
			if len(got) != len(model[seg]) {
				t.Fatalf("step %d: Spans(%q) = %d spans, model has %d", step, seg, len(got), len(model[seg]))
			}
			for i, d := range got {
				want := model[seg][i]
				if d.Offset != want.offset || d.Size != want.size {
					t.Fatalf("step %d: Spans(%q)[%d] = [%d, %d), model = [%d, %d)",
						step, seg, i, d.Offset, d.Offset+d.Size, want.offset, want.offset+want.size)
				}
				if i > 0 && got[i-1].Offset >= d.Offset {
					t.Fatalf("step %d: Spans(%q) not sorted by offset at %d", step, seg, i)
				}
				for j := i + 1; j < len(got); j++ {
					a := modelSpan{offset: got[i].Offset, size: got[i].Size}
					b := modelSpan{offset: got[j].Offset, size: got[j].Size}
					if spansOverlap(a, b) {
						t.Fatalf("step %d: live spans [%d, %d) and [%d, %d) overlap in %q",
							step, a.offset, a.offset+a.size, b.offset, b.offset+b.size, seg)
					}
				}
			}
		}
	}

	for step := 0; step < 2000; step++ {
		seg := segments[rng.Intn(len(segments))]
		if rng.Intn(3) == 0 {
			// Deregister: a random offset that may or may not be live.
			offset := uint64(rng.Intn(128)) * 4
			want := false
			for i, s := range model[seg] {
				if s.offset == offset {
					model[seg] = append(model[seg][:i], model[seg][i+1:]...)
					want = true
					break
				}
			}
			if got := r.Deregister(seg, offset); got != want {
				t.Fatalf("step %d: Deregister(%q, %d) = %v, want %v", step, seg, offset, got, want)
			}
		} else {
			// Register: a random aligned span in a bounded arena.
			offset := uint64(rng.Intn(128)) * 4
			size := uint64(rng.Intn(32)+1) * 4
			candidate := modelSpan{offset: offset, size: size}
			conflict := false
			for _, s := range model[seg] {
				if spansOverlap(candidate, s) {
					conflict = true
					break
				}
			}
			err := r.Register(registryDescriptor(seg, offset, size))
			if conflict {
				if !errors.Is(err, protocol.ErrTensorOverlap) {
					t.Fatalf("step %d: Register overlapping [%d, %d) in %q = %v, want ErrTensorOverlap",
						step, offset, offset+size, seg, err)
				}
			} else {
				if err != nil {
					t.Fatalf("step %d: Register disjoint [%d, %d) in %q = %v, want nil (adjacency must be allowed)",
						step, offset, offset+size, seg, err)
				}
				// Insert keeping the model sorted by offset, like Spans.
				pos := 0
				for pos < len(model[seg]) && model[seg][pos].offset < offset {
					pos++
				}
				model[seg] = append(model[seg], modelSpan{})
				copy(model[seg][pos+1:], model[seg][pos:])
				model[seg][pos] = candidate
			}
		}
		checkInvariants(step)
	}
}

// A span freed by Deregister must be claimable again even after a
// long random history — the property the table tests can only show
// for one hand-picked span.
func TestTensorRegistryPropertyFreedSpansReusable(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	r := protocol.NewTensorRegistry()
	live := map[uint64]uint64{} // offset -> size, single segment

	for step := 0; step < 500; step++ {
		offset := uint64(rng.Intn(64)) * 4
		if size, ok := live[offset]; ok {
			if !r.Deregister("seg", offset) {
				t.Fatalf("step %d: deregister of live span at %d = false", step, offset)
			}
			delete(live, offset)
			// Immediately re-register the exact freed span — same
			// offset AND size. (Registering a larger span here
			// would be a different claim and could legitimately
			// hit a neighbouring live span.)
			if err := r.Register(registryDescriptor("seg", offset, size)); err != nil {
				t.Fatalf("step %d: re-register freed span at %d: %v", step, offset, err)
			}
			live[offset] = size
			continue
		}
		size := uint64(rng.Intn(8)+1) * 4
		candidate := modelSpan{offset: offset, size: size}
		conflict := false
		for off, sz := range live {
			if spansOverlap(candidate, modelSpan{offset: off, size: sz}) {
				conflict = true
				break
			}
		}
		err := r.Register(registryDescriptor("seg", offset, size))
		if conflict && !errors.Is(err, protocol.ErrTensorOverlap) {
			t.Fatalf("step %d: register conflicting span at %d = %v, want ErrTensorOverlap", step, offset, err)
		}
		if !conflict {
			if err != nil {
				t.Fatalf("step %d: register span at %d: %v", step, offset, err)
			}
			live[offset] = size
		}
	}
}
