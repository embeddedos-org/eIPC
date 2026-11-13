// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package transport

import (
	"errors"
	"net"
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
