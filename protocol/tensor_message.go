// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

// Binding between TensorDescriptor and the message path (Track 1).
//
// The descriptor lives in package protocol and core.Message lives in
// package core, which already imports protocol — so the Message-level
// helpers (AttachTensor / ExtractTensor) live in package core and are
// built on the two primitives here. Keeping the capability rules in
// this file means the Frame-payload path and the core.Message path
// enforce exactly the same fail-closed contract:
//
//   - A descriptor with no capability claims cannot be attached. A
//     tensor that names no grant must not travel (fail closed).
//   - The message's primary capability is the descriptor's first
//     claim, and extraction requires the message capability to be one
//     of the descriptor's claims. A capability rewritten in flight —
//     widened or swapped — is a mismatch and the descriptor is refused.
package protocol

import (
	"bytes"
	"errors"
	"fmt"
)

var (
	// ErrTensorNoCapability is returned when a descriptor declares no
	// capability claims and so cannot be bound to a message.
	ErrTensorNoCapability = errors.New("eipc: tensor descriptor declares no capabilities")
	// ErrTensorCapabilityMismatch is returned when a message's
	// capability is not one of the claims of the descriptor it carries.
	ErrTensorCapabilityMismatch = errors.New("eipc: message capability does not match tensor descriptor claims")
)

// MessagePayload validates d and returns its wire encoding plus the
// primary capability a message carrying it must assert (the first
// declared claim). It is the attach half of the binding for callers
// that build their own envelope; core.AttachTensor wraps it for
// core.Message.
func (d *TensorDescriptor) MessagePayload() (payload []byte, capability string, err error) {
	if d == nil {
		return nil, "", ErrTensorNoCapability
	}
	if err := d.Validate(); err != nil {
		return nil, "", err
	}
	if len(d.Capabilities) == 0 {
		return nil, "", ErrTensorNoCapability
	}
	var buf bytes.Buffer
	if err := d.Encode(&buf); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), d.Capabilities[0], nil
}

// TensorFromPayload is the extract half of the binding: it decodes a
// descriptor from a message payload, validates it, and requires the
// message's asserted capability to be one of the descriptor's claims.
// A tampered or widened message capability fails the last check.
func TensorFromPayload(payload []byte, capability string) (*TensorDescriptor, error) {
	d, err := DecodeTensor(bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	if err := d.Validate(); err != nil {
		return nil, err
	}
	for _, claim := range d.Capabilities {
		if claim == capability {
			return d, nil
		}
	}
	return nil, fmt.Errorf("%w: message asserts %q, descriptor claims %v",
		ErrTensorCapabilityMismatch, capability, d.Capabilities)
}
