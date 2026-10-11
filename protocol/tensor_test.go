// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package protocol_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/embeddedos-org/eipc/protocol"
)

func validTensorDescriptor() *protocol.TensorDescriptor {
	return &protocol.TensorDescriptor{
		Dtype:   protocol.TensorF16,
		Shape:   []int64{1, 3, 224, 224},
		Strides: []int64{3 * 224 * 224, 224 * 224, 224, 1}, // contiguous row-major
		ShmName: "eai-infer-0",
		Offset:  4096,
		// footprint = (1 + 0*… + 2*224*224 + 223*224 + 223*1 + 223*… ) * 2
		//         = 1*3*224*224 elements * 2 bytes = 301056
		Size:         301056,
		Capabilities: []string{"inference.local", "tensor.read"},
		ModelDigest:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	}
}

func TestTensorDescriptorRoundTrip(t *testing.T) {
	original := validTensorDescriptor()

	var buf bytes.Buffer
	if err := original.Encode(&buf); err != nil {
		t.Fatalf("encode: %v", err)
	}

	decoded, err := protocol.DecodeTensor(&buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if !reflect.DeepEqual(original, decoded) {
		t.Fatalf("round trip mismatch:\n got %+v\nwant %+v", decoded, original)
	}

	if err := decoded.Validate(); err != nil {
		t.Fatalf("validate decoded: %v", err)
	}
}

func TestTensorDescriptorDtypeMirrorsEAI(t *testing.T) {
	// Numeric values must stay aligned with eAI's eai_dtype_t
	// (eAI accel/include/eai/accel.h).
	want := map[protocol.TensorDtype]uint8{
		protocol.TensorF32:  0,
		protocol.TensorF16:  1,
		protocol.TensorBF16: 2,
		protocol.TensorINT8: 3,
		protocol.TensorINT4: 4,
		protocol.TensorQ4_0: 5,
		protocol.TensorQ8_0: 7,
	}
	for dt, v := range want {
		if uint8(dt) != v {
			t.Errorf("dtype %d: got value %d, want %d (eai_dtype_t drift)", dt, uint8(dt), v)
		}
	}
}

func TestTensorDescriptorValidate(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*protocol.TensorDescriptor)
		wantErr bool
	}{
		{"valid", func(d *protocol.TensorDescriptor) {}, false},
		{"zero dim", func(d *protocol.TensorDescriptor) { d.Shape[1] = 0 }, true},
		{"negative dim", func(d *protocol.TensorDescriptor) { d.Shape[0] = -2 }, true},
		{"no dims", func(d *protocol.TensorDescriptor) { d.Shape, d.Strides = nil, nil }, true},
		{"strides/shape length mismatch", func(d *protocol.TensorDescriptor) { d.Strides = d.Strides[:2] }, true},
		{"negative stride", func(d *protocol.TensorDescriptor) { d.Strides[3] = -1 }, true},
		{"size smaller than footprint", func(d *protocol.TensorDescriptor) { d.Size = 1024 }, true},
		{"empty shm name with nonzero size", func(d *protocol.TensorDescriptor) { d.ShmName = "" }, true},
		{"unknown dtype", func(d *protocol.TensorDescriptor) { d.Dtype = protocol.TensorDtype(42) }, true},
		{"eai-only Q4_1 not transported", func(d *protocol.TensorDescriptor) { d.Dtype = protocol.TensorDtype(6) }, true},
		{"bad model digest", func(d *protocol.TensorDescriptor) { d.ModelDigest = "not-a-digest" }, true},
		{"empty digest ok", func(d *protocol.TensorDescriptor) { d.ModelDigest = "" }, false},
		{"empty capabilities ok", func(d *protocol.TensorDescriptor) { d.Capabilities = nil }, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := validTensorDescriptor()
			tc.mutate(d)
			err := d.Validate()
			if tc.wantErr && err == nil {
				t.Errorf("Validate() = nil, want error")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Validate() = %v, want nil", err)
			}
		})
	}
}

func TestTensorDescriptorDecodeHostile(t *testing.T) {
	// Truncated stream.
	var buf bytes.Buffer
	if err := validTensorDescriptor().Encode(&buf); err != nil {
		t.Fatalf("encode: %v", err)
	}
	if _, err := protocol.DecodeTensor(bytes.NewReader(buf.Bytes()[:7])); err == nil {
		t.Errorf("DecodeTensor(truncated) = nil, want error")
	}

	// Bogus ndim must not allocate.
	if _, err := protocol.DecodeTensor(bytes.NewReader([]byte{0, 255})); err == nil {
		t.Errorf("DecodeTensor(ndim=255) = nil, want error")
	}

	// Garbage entirely.
	if _, err := protocol.DecodeTensor(strings.NewReader("")); err == nil {
		t.Errorf("DecodeTensor(empty) = nil, want error")
	}
}

func TestTensorDescriptorValidateAfterDecode(t *testing.T) {
	// A descriptor that decodes but is semantically inconsistent must fail
	// Validate before anything is mapped.
	d := validTensorDescriptor()
	d.Size = 16 // smaller than footprint

	var buf bytes.Buffer
	if err := d.Encode(&buf); err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := protocol.DecodeTensor(&buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := decoded.Validate(); err == nil {
		t.Errorf("Validate() = nil for undersized span, want error")
	}
}
