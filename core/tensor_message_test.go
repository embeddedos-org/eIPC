// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package core

import (
	"errors"
	"reflect"
	"testing"

	"github.com/embeddedos-org/eipc/protocol"
)

func validCoreTensorDescriptor() *protocol.TensorDescriptor {
	return &protocol.TensorDescriptor{
		Dtype:        protocol.TensorF16,
		Shape:        []int64{1, 3, 224, 224},
		Strides:      []int64{3 * 224 * 224, 224 * 224, 224, 1},
		ShmName:      "eai-infer-0",
		Offset:       4096,
		Size:         301056,
		Capabilities: []string{"inference.local", "tensor.read"},
		ModelDigest:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	}
}

func TestAttachExtractTensorRoundTrip(t *testing.T) {
	original := validCoreTensorDescriptor()

	msg, err := AttachTensor(original)
	if err != nil {
		t.Fatalf("AttachTensor: %v", err)
	}
	if msg.Capability != original.Capabilities[0] {
		t.Fatalf("Message.Capability = %q, want primary claim %q", msg.Capability, original.Capabilities[0])
	}
	if len(msg.Payload) == 0 {
		t.Fatal("AttachTensor produced an empty payload")
	}

	decoded, err := ExtractTensor(msg)
	if err != nil {
		t.Fatalf("ExtractTensor: %v", err)
	}
	if !reflect.DeepEqual(original, decoded) {
		t.Fatalf("round trip mismatch:\n got %+v\nwant %+v", decoded, original)
	}
}

func TestAttachTensorRefusesNoCapabilities(t *testing.T) {
	d := validCoreTensorDescriptor()
	d.Capabilities = nil

	if _, err := AttachTensor(d); !errors.Is(err, protocol.ErrTensorNoCapability) {
		t.Fatalf("AttachTensor with no claims: err = %v, want ErrTensorNoCapability", err)
	}
}

func TestExtractTensorRejectsTamperedCapability(t *testing.T) {
	msg, err := AttachTensor(validCoreTensorDescriptor())
	if err != nil {
		t.Fatalf("AttachTensor: %v", err)
	}

	// A capability widened or swapped in flight must not extract.
	msg.Capability = "inference.admin"
	if _, err := ExtractTensor(msg); !errors.Is(err, protocol.ErrTensorCapabilityMismatch) {
		t.Fatalf("tampered capability: err = %v, want ErrTensorCapabilityMismatch", err)
	}

	// A capability stripped in flight must not extract either.
	msg.Capability = ""
	if _, err := ExtractTensor(msg); !errors.Is(err, protocol.ErrTensorCapabilityMismatch) {
		t.Fatalf("stripped capability: err = %v, want ErrTensorCapabilityMismatch", err)
	}
}

func TestExtractTensorRejectsNilAndGarbage(t *testing.T) {
	if _, err := ExtractTensor(nil); !errors.Is(err, ErrNilMessage) {
		t.Fatalf("nil message: err = %v, want ErrNilMessage", err)
	}
	msg := &Message{Capability: "inference.local", Payload: []byte{0xFF, 0x00}}
	if _, err := ExtractTensor(msg); err == nil {
		t.Fatal("garbage payload extracted without error")
	}
}
