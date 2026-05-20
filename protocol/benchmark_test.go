// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package protocol

import (
	"bytes"
	"testing"
)

func BenchmarkFrameEncode(b *testing.B) {
	frame := &Frame{
		Version: ProtocolVersion,
		MsgType: 'i',
		Flags:   FlagHMAC,
		Header:  []byte(`{"service_id":"eni.min","session_id":"sess-1","request_id":"req-1","sequence":1,"timestamp":"2026-01-01T00:00:00Z","priority":0}`),
		Payload: []byte(`{"intent":"move_left","confidence":0.91}`),
		MAC:     make([]byte, MACSize),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = frame.Encode(&buf)
	}
}

func BenchmarkFrameDecode(b *testing.B) {
	frame := &Frame{
		Version: ProtocolVersion,
		MsgType: 'i',
		Flags:   FlagHMAC,
		Header:  []byte(`{"service_id":"eni.min","session_id":"sess-1","request_id":"req-1","sequence":1,"timestamp":"2026-01-01T00:00:00Z","priority":0}`),
		Payload: []byte(`{"intent":"move_left","confidence":0.91}`),
		MAC:     make([]byte, MACSize),
	}

	var buf bytes.Buffer
	_ = frame.Encode(&buf)
	encoded := buf.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(encoded)
		Decode(r)
	}
}

func BenchmarkSignableBytes(b *testing.B) {
	frame := &Frame{
		Version: ProtocolVersion,
		MsgType: 'i',
		Flags:   FlagHMAC,
		Header:  []byte(`{"service_id":"eni.min"}`),
		Payload: []byte(`{"intent":"move"}`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		frame.SignableBytes()
	}
}
