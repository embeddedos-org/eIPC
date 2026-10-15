// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package protocol_test

import (
	"errors"
	"testing"

	"github.com/embeddedos-org/eipc/protocol"
)

// registryDescriptor builds a small valid descriptor claiming
// [offset, offset+size) in segment name. F32 with one element per
// 4 bytes, so size must be a multiple of 4 and offset 4-aligned.
func registryDescriptor(name string, offset, size uint64) protocol.TensorDescriptor {
	return protocol.TensorDescriptor{
		Dtype:   protocol.TensorF32,
		Shape:   []int64{int64(size / 4)},
		Strides: []int64{1},
		ShmName: name,
		Offset:  offset,
		Size:    size,
	}
}

func TestTensorRegistryOverlap(t *testing.T) {
	cases := []struct {
		name    string
		first   protocol.TensorDescriptor
		second  protocol.TensorDescriptor
		wantErr error // nil if the second register must succeed
	}{
		{"identical span refused",
			registryDescriptor("seg", 0, 64), registryDescriptor("seg", 0, 64),
			protocol.ErrTensorOverlap},
		{"partial overlap, second starts inside",
			registryDescriptor("seg", 0, 64), registryDescriptor("seg", 32, 64),
			protocol.ErrTensorOverlap},
		{"partial overlap, second ends inside",
			registryDescriptor("seg", 32, 64), registryDescriptor("seg", 0, 64),
			protocol.ErrTensorOverlap},
		{"contained span refused",
			registryDescriptor("seg", 0, 128), registryDescriptor("seg", 32, 32),
			protocol.ErrTensorOverlap},
		{"containing span refused",
			registryDescriptor("seg", 32, 32), registryDescriptor("seg", 0, 128),
			protocol.ErrTensorOverlap},
		{"adjacent spans allowed",
			registryDescriptor("seg", 0, 64), registryDescriptor("seg", 64, 64),
			nil},
		{"adjacent before allowed",
			registryDescriptor("seg", 64, 64), registryDescriptor("seg", 0, 64),
			nil},
		{"same span, different segment allowed",
			registryDescriptor("seg-a", 0, 64), registryDescriptor("seg-b", 0, 64),
			nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := protocol.NewTensorRegistry()
			if err := r.Register(tc.first); err != nil {
				t.Fatalf("register first: %v", err)
			}
			err := r.Register(tc.second)
			if tc.wantErr == nil && err != nil {
				t.Errorf("register second = %v, want nil", err)
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Errorf("register second = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestTensorRegistryDeregister(t *testing.T) {
	r := protocol.NewTensorRegistry()
	d := registryDescriptor("seg", 0, 64)
	if err := r.Register(d); err != nil {
		t.Fatalf("register: %v", err)
	}
	if !r.Deregister("seg", 0) {
		t.Fatal("deregister of a live span = false, want true")
	}
	if r.Deregister("seg", 0) {
		t.Fatal("second deregister = true, want false")
	}
	if r.Deregister("seg", 4096) {
		t.Fatal("deregister of unknown offset = true, want false")
	}
	if r.Deregister("other", 0) {
		t.Fatal("deregister in unknown segment = true, want false")
	}
	// The freed span can be claimed again.
	if err := r.Register(d); err != nil {
		t.Fatalf("re-register after deregister: %v", err)
	}
}

func TestTensorRegistryRefusesInvalid(t *testing.T) {
	r := protocol.NewTensorRegistry()

	misaligned := registryDescriptor("seg", 2, 64) // F32 offset must be 4-aligned
	if err := r.Register(misaligned); !errors.Is(err, protocol.ErrTensorAlign) {
		t.Fatalf("register misaligned = %v, want ErrTensorAlign", err)
	}
	// The refused descriptor must not occupy its span: a valid
	// descriptor covering the same bytes registers cleanly.
	if err := r.Register(registryDescriptor("seg", 0, 64)); err != nil {
		t.Fatalf("register after refused descriptor: %v", err)
	}

	zero := registryDescriptor("seg", 128, 0)
	zero.Shape = []int64{1} // footprint 4 > size 0: invalid either way
	if err := r.Register(zero); err == nil {
		t.Fatal("register zero-size = nil, want error")
	}
}

func TestTensorRegistrySpans(t *testing.T) {
	r := protocol.NewTensorRegistry()
	if got := r.Spans("seg"); len(got) != 0 {
		t.Fatalf("Spans on empty registry = %d, want 0", len(got))
	}
	for _, d := range []protocol.TensorDescriptor{
		registryDescriptor("seg", 128, 64),
		registryDescriptor("seg", 0, 64),
		registryDescriptor("seg", 64, 64),
		registryDescriptor("other", 0, 64),
	} {
		if err := r.Register(d); err != nil {
			t.Fatalf("register %+v: %v", d, err)
		}
	}
	got := r.Spans("seg")
	if len(got) != 3 {
		t.Fatalf("Spans(seg) = %d, want 3", len(got))
	}
	for i, want := range []uint64{0, 64, 128} {
		if got[i].Offset != want {
			t.Errorf("Spans(seg)[%d].Offset = %d, want %d (sorted)", i, got[i].Offset, want)
		}
	}
	if got := r.Spans("other"); len(got) != 1 {
		t.Errorf("Spans(other) = %d, want 1", len(got))
	}
}
