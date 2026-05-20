// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package core

import (
	"testing"
	"time"
)

func BenchmarkNewMessage(b *testing.B) {
	payload := []byte(`{"intent":"move_left","confidence":0.91}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewMessage(TypeIntent, "eni.min", payload)
	}
}

func BenchmarkRouterDispatch(b *testing.B) {
	router := NewRouter()
	router.Handle(TypeIntent, func(msg Message) (*Message, error) {
		return nil, nil
	})

	msg := Message{
		Version:   ProtocolVersion,
		Type:      TypeIntent,
		Source:    "bench",
		Timestamp: time.Now().UTC(),
		Priority:  PriorityP0,
		Payload:   []byte(`{"intent":"test"}`),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = router.Dispatch(msg)
	}
}

func BenchmarkRouterDispatchBatch(b *testing.B) {
	router := NewRouter()
	router.Handle(TypeIntent, func(msg Message) (*Message, error) {
		return nil, nil
	})
	router.Handle(TypeHeartbeat, func(msg Message) (*Message, error) {
		return nil, nil
	})

	msgs := []Message{
		{Type: TypeIntent, Priority: PriorityP2, Payload: []byte(`{}`)},
		{Type: TypeHeartbeat, Priority: PriorityP0, Payload: []byte(`{}`)},
		{Type: TypeIntent, Priority: PriorityP1, Payload: []byte(`{}`)},
		{Type: TypeHeartbeat, Priority: PriorityP3, Payload: []byte(`{}`)},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.DispatchBatch(msgs)
	}
}

func BenchmarkMsgTypeToByte(b *testing.B) {
	for i := 0; i < b.N; i++ {
		MsgTypeToByte(TypeIntent)
		MsgTypeToByte(TypeChat)
		MsgTypeToByte(TypeAck)
	}
}
