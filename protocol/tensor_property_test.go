// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package protocol_test

// Property tests for the tensor descriptor bridge contract (audit
// phase 2, eNI #38): a descriptor crossing a process boundary is
// untrusted input, and the properties below are the ones a receiver
// relies on before mapping anything — footprints that cannot wrap,
// spans that cannot overflow the address space, offsets aligned to
// the element type, and capability claims that are at least
// well-formed. Everything fails closed.
//
// Two of these checks (offset alignment, span wrap) were missing
// from Validate until the tests demanded them; they are enforced in
// protocol/tensor.go and pinned here.

import (
	"errors"
	"strings"
	"testing"

	"github.com/embeddedos-org/eipc/protocol"
)

func mutateValid(fn func(*protocol.TensorDescriptor)) *protocol.TensorDescriptor {
	d := validTensorDescriptor()
	fn(d)
	return d
}

func TestTensorPropertyFootprintOverflowRejected(t *testing.T) {
	cases := map[string]*protocol.TensorDescriptor{
		"stride multiply overflows": mutateValid(func(d *protocol.TensorDescriptor) {
			d.Shape = []int64{1 << 40, 1 << 40}
			d.Strides = []int64{1 << 40, 1}
			d.Size = ^uint64(0)
		}),
		"extent overflows int64": mutateValid(func(d *protocol.TensorDescriptor) {
			d.Shape = []int64{1 << 62}
			d.Strides = []int64{1}
			d.Size = ^uint64(0)
		}),
		"negative dim": mutateValid(func(d *protocol.TensorDescriptor) {
			d.Shape = []int64{4, -1}
			d.Strides = []int64{4, 1}
		}),
		"negative stride": mutateValid(func(d *protocol.TensorDescriptor) {
			d.Shape = []int64{4, 4}
			d.Strides = []int64{4, -1}
		}),
		"size below footprint": mutateValid(func(d *protocol.TensorDescriptor) {
			d.Size = 100 // footprint is 301056
		}),
		"span wraps uint64": mutateValid(func(d *protocol.TensorDescriptor) {
			d.Offset = ^uint64(0) - 8
			// Size stays 301056, so Offset+Size wraps.
		}),
	}
	for name, d := range cases {
		if err := d.Validate(); err == nil {
			t.Errorf("%s: Validate accepted a hostile descriptor", name)
		}
	}
}

func TestTensorPropertyOffsetAlignment(t *testing.T) {
	// F32 (4-byte elements): odd offsets are lies about the layout.
	for _, off := range []uint64{1, 2, 3, 4098} {
		d := mutateValid(func(d *protocol.TensorDescriptor) {
			d.Dtype = protocol.TensorF32
			d.Offset = off
		})
		if err := d.Validate(); !errors.Is(err, protocol.ErrTensorAlign) {
			t.Errorf("F32 offset %d: got %v, want ErrTensorAlign", off, err)
		}
	}
	// F16: offset 1 misaligned, 4096 fine.
	d := mutateValid(func(d *protocol.TensorDescriptor) { d.Offset = 1 })
	if err := d.Validate(); !errors.Is(err, protocol.ErrTensorAlign) {
		t.Errorf("F16 offset 1: got %v, want ErrTensorAlign", err)
	}
	if err := validTensorDescriptor().Validate(); err != nil {
		t.Errorf("aligned control descriptor rejected: %v", err)
	}
	// INT8 (1-byte elements): every offset is aligned.
	d = mutateValid(func(d *protocol.TensorDescriptor) {
		d.Dtype = protocol.TensorINT8
		d.Offset = 4097
	})
	if err := d.Validate(); err != nil {
		t.Errorf("INT8 offset 4097 rejected: %v", err)
	}
}

func TestTensorPropertyCapabilityClaimsWellFormed(t *testing.T) {
	// A descriptor's capabilities are claims a receiver checks against
	// its grants (see core.ExtractTensor for the mismatch rule); the
	// claims themselves must be well-formed before that comparison.
	cases := map[string]*protocol.TensorDescriptor{
		"empty claim": mutateValid(func(d *protocol.TensorDescriptor) {
			d.Capabilities = []string{"inference.local", ""}
		}),
		"too many claims": mutateValid(func(d *protocol.TensorDescriptor) {
			d.Capabilities = make([]string, protocol.MaxTensorCaps+1)
			for i := range d.Capabilities {
				d.Capabilities[i] = "inference.local"
			}
		}),
		"oversized claim": mutateValid(func(d *protocol.TensorDescriptor) {
			d.Capabilities = []string{strings.Repeat("x", protocol.MaxTensorCapLen+1)}
		}),
	}
	for name, d := range cases {
		if err := d.Validate(); !errors.Is(err, protocol.ErrTensorCaps) {
			t.Errorf("%s: got %v, want ErrTensorCaps", name, err)
		}
	}
}
