// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package transport

import (
	"encoding/binary"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/embeddedos-org/eipc/protocol"
)

func TestConnWrapperSetReadDeadline(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	serverWrapper := NewConnWrapper(server)
	clientWrapper := NewConnWrapper(client)

	// A live deadline still lets a frame through.
	if err := serverWrapper.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}
	sent := &protocol.Frame{
		Version: protocol.ProtocolVersion,
		MsgType: 'a',
		Header:  []byte("{}"),
		Payload: []byte("hello"),
	}
	errCh := make(chan error, 1)
	go func() { errCh <- clientWrapper.Send(sent) }()
	recv, err := serverWrapper.Receive()
	if err != nil {
		t.Fatalf("Receive with live deadline: %v", err)
	}
	if string(recv.Payload) != "hello" {
		t.Errorf("expected payload %q, got %q", "hello", recv.Payload)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("Send: %v", err)
	}

	// A deadline already in the past: Receive must fail fast with a
	// timeout instead of blocking until the peer speaks.
	if err := serverWrapper.SetReadDeadline(time.Now().Add(-time.Second)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}
	start := time.Now()
	_, err = serverWrapper.Receive()
	if err == nil {
		t.Fatal("expected Receive to fail past its read deadline")
	}
	if time.Since(start) > 5*time.Second {
		t.Errorf("Receive blocked %v past an expired deadline", time.Since(start))
	}
	var ne net.Error
	if !errors.As(err, &ne) || !ne.Timeout() {
		t.Errorf("expected a timeout error, got %T: %v", err, err)
	}
}

// A frame whose header and payload together are exactly
// protocol.MaxFrameSize is valid: Frame.Encode accepts it and
// protocol.Decode accepts it. On the wire it is that plus the 16-byte
// preamble and the 32-byte MAC, so the length-prefix check in Receive
// must allow for both or the peer can never read a frame Send wrote.
func TestConnWrapperMaxSizeFrameRoundTrip(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	serverWrapper := NewConnWrapper(server)
	clientWrapper := NewConnWrapper(client)

	header := []byte("{}")
	mac := make([]byte, protocol.MACSize)
	for i := range mac {
		mac[i] = byte(i)
	}
	sent := &protocol.Frame{
		Version: protocol.ProtocolVersion,
		MsgType: 'c',
		Flags:   protocol.FlagHMAC,
		Header:  header,
		Payload: make([]byte, int(protocol.MaxFrameSize)-len(header)),
		MAC:     mac,
	}

	errCh := make(chan error, 1)
	go func() { errCh <- clientWrapper.Send(sent) }()

	if err := serverWrapper.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}
	recv, err := serverWrapper.Receive()
	if err != nil {
		// Unblock the sender, which is still writing into the pipe.
		client.Close()
		<-errCh
		t.Fatalf("Receive of a max-size frame: %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(recv.Payload) != len(sent.Payload) {
		t.Errorf("payload length = %d, want %d", len(recv.Payload), len(sent.Payload))
	}
	if string(recv.MAC) != string(mac) {
		t.Errorf("MAC not carried through")
	}
}

// A length prefix larger than the biggest encodable frame is still
// refused before anything is allocated for it.
func TestConnWrapperRejectsOversizedLengthPrefix(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	serverWrapper := NewConnWrapper(server)

	maxWire := uint32(protocol.FrameFixedSize) + protocol.MaxFrameSize + uint32(protocol.MACSize)
	go func() {
		var lenBuf [4]byte
		binary.BigEndian.PutUint32(lenBuf[:], maxWire+1)
		_, _ = client.Write(lenBuf[:])
	}()

	if err := serverWrapper.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}
	_, err := serverWrapper.Receive()
	if err == nil || !strings.Contains(err.Error(), "frame too large") {
		t.Fatalf("expected a frame too large error, got %v", err)
	}
}
