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
