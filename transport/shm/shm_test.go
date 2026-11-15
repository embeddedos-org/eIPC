// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package shm

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/embeddedos-org/eipc/protocol"
)

func TestRingBufferWriteRead(t *testing.T) {
	rb := NewRingBuffer(Config{Name: "test", BufferSize: 8192, SlotCount: 8})

	frame := &protocol.Frame{
		Version: 1,
		MsgType: 'i',
		Flags:   0,
		Header:  []byte(`{"service_id":"test"}`),
		Payload: []byte(`{"intent":"move_left"}`),
	}

	if err := rb.Write(frame); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if rb.Len() != 1 {
		t.Errorf("expected Len()=1, got %d", rb.Len())
	}

	readFrame, err := rb.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if readFrame == nil {
		t.Fatal("Read returned nil frame")
	}
	if readFrame.MsgType != 'i' {
		t.Errorf("expected msg_type 'i', got %c", readFrame.MsgType)
	}
}

func TestRingBufferEmpty(t *testing.T) {
	rb := NewRingBuffer(Config{Name: "empty", BufferSize: 4096, SlotCount: 4})

	frame, err := rb.Read()
	if err != nil {
		t.Fatalf("Read from empty buffer should not error: %v", err)
	}
	if frame != nil {
		t.Error("Read from empty buffer should return nil")
	}
}

func TestRingBufferFull(t *testing.T) {
	rb := NewRingBuffer(Config{Name: "full", BufferSize: 4096, SlotCount: 2})

	frame := &protocol.Frame{
		Version: 1,
		MsgType: 'h',
		Header:  []byte(`{}`),
		Payload: []byte(`{}`),
	}

	_ = rb.Write(frame)
	_ = rb.Write(frame)

	err := rb.Write(frame)
	if err == nil {
		t.Error("expected backpressure error when buffer is full")
	}
}

func TestConnectionInterface(t *testing.T) {
	txBuf := NewRingBuffer(Config{Name: "tx", BufferSize: 8192, SlotCount: 8})
	rxBuf := NewRingBuffer(Config{Name: "rx", BufferSize: 8192, SlotCount: 8})

	conn := NewConnection(txBuf, rxBuf, "shm://test")

	if conn.RemoteAddr() != "shm://test" {
		t.Errorf("expected remote addr shm://test, got %s", conn.RemoteAddr())
	}

	if err := conn.Close(); err != nil {
		t.Errorf("Close should not error: %v", err)
	}
}

func TestRingBufferName(t *testing.T) {
	rb := NewRingBuffer(Config{Name: "my-region"})
	if rb.Name() != "my-region" {
		t.Errorf("expected name my-region, got %s", rb.Name())
	}
}

func TestConnectionCloseUnblocksReceive(t *testing.T) {
	txBuf := NewRingBuffer(Config{Name: "tx", BufferSize: 8192, SlotCount: 8})
	rxBuf := NewRingBuffer(Config{Name: "rx", BufferSize: 8192, SlotCount: 8})
	conn := NewConnection(txBuf, rxBuf, "shm://test")

	// Nothing is ever written to rx, so Receive waits until Close.
	done := make(chan error, 1)
	go func() {
		_, err := conn.Receive()
		done <- err
	}()

	time.Sleep(20 * time.Millisecond)
	if err := conn.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	select {
	case err := <-done:
		if !errors.Is(err, net.ErrClosed) {
			t.Fatalf("Receive after Close: got %v, want net.ErrClosed", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Receive still blocked 2s after Close")
	}

	if _, err := conn.Receive(); !errors.Is(err, net.ErrClosed) {
		t.Errorf("Receive on a closed connection: got %v, want net.ErrClosed", err)
	}
	frame := &protocol.Frame{Version: 1, MsgType: 'h', Header: []byte(`{}`), Payload: []byte(`{}`)}
	if err := conn.Send(frame); !errors.Is(err, net.ErrClosed) {
		t.Errorf("Send on a closed connection: got %v, want net.ErrClosed", err)
	}
	if txBuf.Len() != 0 {
		t.Errorf("Send on a closed connection wrote %d frame(s)", txBuf.Len())
	}
}
