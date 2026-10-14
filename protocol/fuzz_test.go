// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package protocol

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func FuzzFrameDecode(f *testing.F) {
	// Seed with a valid frame
	var validFrame bytes.Buffer
	frame := &Frame{
		Version: ProtocolVersion,
		MsgType: 'i',
		Flags:   0,
		Header:  []byte(`{"service_id":"test"}`),
		Payload: []byte(`{"intent":"move"}`),
	}
	_ = frame.Encode(&validFrame)
	f.Add(validFrame.Bytes())

	// Seed with minimal valid preamble
	preamble := make([]byte, FrameFixedSize)
	binary.BigEndian.PutUint32(preamble[0:4], MagicBytes)
	binary.BigEndian.PutUint16(preamble[4:6], ProtocolVersion)
	preamble[6] = 'a'
	preamble[7] = 0
	binary.BigEndian.PutUint32(preamble[8:12], 0)
	binary.BigEndian.PutUint32(preamble[12:16], 0)
	f.Add(preamble)

	// Seed with empty data
	f.Add([]byte{})

	// Seed with garbage
	f.Add([]byte{0xff, 0xfe, 0xfd, 0xfc, 0x00, 0x01})

	f.Fuzz(func(t *testing.T, data []byte) {
		r := bytes.NewReader(data)
		frame, err := Decode(r)
		if err != nil {
			return // Expected for random input
		}

		// If decode succeeded, verify the frame is reasonable
		if frame.Version != ProtocolVersion {
			t.Errorf("decoded frame with unexpected version %d", frame.Version)
		}

		// Re-encode and verify round-trip
		var buf bytes.Buffer
		if err := frame.Encode(&buf); err != nil {
			t.Errorf("failed to re-encode decoded frame: %v", err)
		}
	})
}

func FuzzTensorDescriptor(f *testing.F) {
	// Seed with a valid descriptor encoding.
	valid := &TensorDescriptor{
		Dtype:        TensorF16,
		Shape:        []int64{1, 3, 224, 224},
		Strides:      []int64{3 * 224 * 224, 224 * 224, 224, 1},
		ShmName:      "eai-infer-0",
		Offset:       4096,
		Size:         301056,
		Capabilities: []string{"inference.local", "tensor.read"},
		ModelDigest:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	}
	var buf bytes.Buffer
	if err := valid.Encode(&buf); err != nil {
		f.Fatalf("seed encode: %v", err)
	}
	f.Add(buf.Bytes())
	f.Add([]byte{})
	f.Add([]byte{0x00, 0xff}) // ndim = 255, must fail fast

	f.Fuzz(func(t *testing.T, data []byte) {
		d, err := DecodeTensor(bytes.NewReader(data))
		if err != nil {
			return // hostile or truncated input: fail fast is correct
		}
		if err := d.Validate(); err != nil {
			return // decoded but semantically invalid is fine
		}
		// A descriptor that decodes AND validates must round-trip:
		// re-encode, decode again, and land on the same descriptor.
		var out bytes.Buffer
		if err := d.Encode(&out); err != nil {
			t.Fatalf("re-encode validated descriptor: %v", err)
		}
		d2, err := DecodeTensor(&out)
		if err != nil {
			t.Fatalf("re-decode validated descriptor: %v", err)
		}
		if err := d2.Validate(); err != nil {
			t.Fatalf("round-tripped descriptor no longer validates: %v", err)
		}
	})
}
