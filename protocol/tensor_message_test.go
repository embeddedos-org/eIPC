// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package protocol_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/embeddedos-org/eipc/protocol"
)

func TestTensorMessagePayloadRoundTrip(t *testing.T) {
	original := validTensorDescriptor()

	payload, capability, err := original.MessagePayload()
	if err != nil {
		t.Fatalf("MessagePayload: %v", err)
	}
	if capability != original.Capabilities[0] {
		t.Fatalf("primary capability = %q, want first claim %q", capability, original.Capabilities[0])
	}

	decoded, err := protocol.TensorFromPayload(payload, capability)
	if err != nil {
		t.Fatalf("TensorFromPayload: %v", err)
	}
	if !reflect.DeepEqual(original, decoded) {
		t.Fatalf("round trip mismatch:\n got %+v\nwant %+v", decoded, original)
	}
}

func TestTensorMessagePayloadRefusesNoCapabilities(t *testing.T) {
	d := validTensorDescriptor()
	d.Capabilities = nil

	if _, _, err := d.MessagePayload(); !errors.Is(err, protocol.ErrTensorNoCapability) {
		t.Fatalf("MessagePayload with no claims: err = %v, want ErrTensorNoCapability", err)
	}
}

func TestTensorFromPayloadRejectsCapabilityMismatch(t *testing.T) {
	d := validTensorDescriptor()
	payload, _, err := d.MessagePayload()
	if err != nil {
		t.Fatalf("MessagePayload: %v", err)
	}

	for _, bogus := range []string{"inference.admin", "tensor.write", ""} {
		if _, err := protocol.TensorFromPayload(payload, bogus); !errors.Is(err, protocol.ErrTensorCapabilityMismatch) {
			t.Errorf("capability %q: err = %v, want ErrTensorCapabilityMismatch", bogus, err)
		}
	}

	// Any declared claim — not only the primary — is a valid assertion.
	if _, err := protocol.TensorFromPayload(payload, d.Capabilities[1]); err != nil {
		t.Errorf("secondary claim should extract: %v", err)
	}
}
