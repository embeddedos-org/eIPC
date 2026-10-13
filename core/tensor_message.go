// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package core

import (
	"errors"

	"github.com/embeddedos-org/eipc/protocol"
)

// Tensor binding for core.Message (Track 1 — capability-scoped
// inference). The descriptor itself is defined in package protocol
// (protocol/tensor.go); this file binds it to the canonical message
// envelope so a tensor publish carries its descriptor and its
// capability grant atomically, in one constructor — the two cannot
// drift apart the way a hand-assembled Message would let them.
//
// Zero-copy note: only the descriptor crosses the boundary, as the
// message Payload. It carries the shared-memory arena HANDLE
// (ShmName + Offset + Size) and the declared capabilities — never a
// raw pointer. Shared-arena mapping by handle is the phase-2 step;
// see docs/track1/tensor-transport.md, "Message binding".

// ErrNilMessage is returned by ExtractTensor for a nil message.
var ErrNilMessage = errors.New("eipc: nil message")

// AttachTensor validates d, encodes it as the payload of a new
// Message, and sets Message.Capability to the descriptor's primary
// (first) capability claim. A descriptor declaring no capabilities is
// refused: a tensor that names no grant must not travel.
func AttachTensor(d *protocol.TensorDescriptor) (*Message, error) {
	payload, capability, err := d.MessagePayload()
	if err != nil {
		return nil, err
	}
	msg := NewMessage(TypeFeatures, "", payload)
	msg.Capability = capability
	return &msg, nil
}

// ExtractTensor decodes and validates the tensor descriptor carried
// by m, and requires m.Capability to be one of the descriptor's
// declared claims. A message whose Capability was tampered with after
// AttachTensor — widened, swapped, or stripped — is rejected.
func ExtractTensor(m *Message) (*protocol.TensorDescriptor, error) {
	if m == nil {
		return nil, ErrNilMessage
	}
	return protocol.TensorFromPayload(m.Payload, m.Capability)
}
