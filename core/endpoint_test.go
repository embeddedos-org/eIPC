// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package core

import (
	"errors"
	"net"
	"testing"

	"github.com/embeddedos-org/eipc/protocol"
	"github.com/embeddedos-org/eipc/transport"
)

func setupTCPPair(t *testing.T) (transport.Connection, transport.Connection) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer ln.Close()

	acceptCh := make(chan net.Conn, 1)
	errCh := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			errCh <- err
			return
		}
		acceptCh <- conn
	}()

	clientRaw, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}

	var serverRaw net.Conn
	select {
	case serverRaw = <-acceptCh:
	case err := <-errCh:
		t.Fatalf("accept failed: %v", err)
	}

	return transport.NewConnWrapper(clientRaw), transport.NewConnWrapper(serverRaw)
}

func TestClientEndpointSendReceive(t *testing.T) {
	clientConn, serverConn := setupTCPPair(t)
	defer clientConn.Close()
	defer serverConn.Close()

	hmacKey := []byte("test-secret-key-for-hmac-256bit!")
	sessionID := "test-session-001"
	c := protocol.DefaultCodec()

	clientEP := NewClientEndpoint(clientConn, c, hmacKey, sessionID)
	serverEP := NewServerEndpoint(serverConn, c, hmacKey)

	msg := NewMessage(TypeIntent, "test-service", []byte(`{"action":"ping"}`))
	msg.Priority = PriorityP0

	errCh := make(chan error, 1)
	msgCh := make(chan Message, 1)
	go func() {
		received, err := serverEP.Receive()
		if err != nil {
			errCh <- err
			return
		}
		msgCh <- received
	}()

	if err := clientEP.Send(msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	select {
	case received := <-msgCh:
		if received.Type != TypeIntent {
			t.Errorf("type mismatch: got %v, want %v", received.Type, TypeIntent)
		}
		if received.Source != "test-service" {
			t.Errorf("source mismatch: got %q, want %q", received.Source, "test-service")
		}
		if string(received.Payload) != `{"action":"ping"}` {
			t.Errorf("payload mismatch: got %q", string(received.Payload))
		}
	case err := <-errCh:
		t.Fatalf("Receive failed: %v", err)
	}
}

func TestHMACVerificationFailure(t *testing.T) {
	clientConn, serverConn := setupTCPPair(t)
	defer clientConn.Close()
	defer serverConn.Close()

	c := protocol.DefaultCodec()

	clientEP := NewClientEndpoint(clientConn, c, []byte("client-key-32-bytes-long-aaaaaa!"), "session-hmac")
	serverEP := NewServerEndpoint(serverConn, c, []byte("different-key-32-bytes-long-bbb!"))

	msg := NewMessage(TypeHeartbeat, "svc", []byte("beat"))

	errCh := make(chan error, 1)
	go func() {
		_, err := serverEP.Receive()
		errCh <- err
	}()

	if err := clientEP.Send(msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	err := <-errCh
	if err == nil {
		t.Fatal("expected HMAC verification error with mismatched keys")
	}
}

func TestHeartbeatMessage(t *testing.T) {
	msg := NewMessage(TypeHeartbeat, "monitor", nil)
	if msg.Type != TypeHeartbeat {
		t.Errorf("expected heartbeat type, got %v", msg.Type)
	}
	if msg.Source != "monitor" {
		t.Errorf("expected source 'monitor', got %q", msg.Source)
	}
}

// sendUnsignedFrame transmits a raw frame with FlagHMAC clear, bypassing
// Endpoint.Send (which always signs). It lets tests prove what a secured
// endpoint does with a peer that simply does not sign.
func sendUnsignedFrame(t *testing.T, conn transport.Connection, c protocol.Codec) {
	t.Helper()
	hdr := protocol.Header{
		ServiceID:     "unsigned-peer",
		SessionID:     "session-unsigned",
		RequestID:     "req-unsigned",
		Sequence:      1,
		Timestamp:     "2026-01-01T00:00:00Z",  // fixed: TESTING.md forbids wall-clock dependence
		Priority:      uint8(PriorityP0),
		PayloadFormat: uint8(PayloadJSON),
	}
	hdrBytes, err := c.Marshal(hdr)
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	frame := &protocol.Frame{
		Version: ProtocolVersion,
		MsgType: MsgTypeToByte(TypeIntent),
		Flags:   0,
		Header:  hdrBytes,
		Payload: []byte(`{"action":"ping"}`),
	}
	if err := conn.Send(frame); err != nil {
		t.Fatalf("send unsigned frame: %v", err)
	}
}

func TestSecuredServerEndpointRejectsUnsignedFrame(t *testing.T) {
	clientConn, serverConn := setupTCPPair(t)
	defer clientConn.Close()
	defer serverConn.Close()

	c := protocol.DefaultCodec()
	serverEP := NewServerEndpoint(serverConn, c, []byte("server-key-32-bytes-long-cccccc!"))

	errCh := make(chan error, 1)
	go func() {
		_, err := serverEP.Receive()
		errCh <- err
	}()

	sendUnsignedFrame(t, clientConn, c)

	if err := <-errCh; !errors.Is(err, ErrIntegrity) {
		t.Fatalf("expected ErrIntegrity for unsigned frame on secured endpoint, got %v", err)
	}
}

func TestSecuredClientEndpointRejectsUnsignedFrame(t *testing.T) {
	clientConn, serverConn := setupTCPPair(t)
	defer clientConn.Close()
	defer serverConn.Close()

	c := protocol.DefaultCodec()
	clientEP := NewClientEndpoint(clientConn, c, []byte("client-key-32-bytes-long-dddddd!"), "session-unsigned")

	errCh := make(chan error, 1)
	go func() {
		_, err := clientEP.Receive()
		errCh <- err
	}()

	sendUnsignedFrame(t, serverConn, c)

	if err := <-errCh; !errors.Is(err, ErrIntegrity) {
		t.Fatalf("expected ErrIntegrity for unsigned frame on secured endpoint, got %v", err)
	}
}

func TestUnsecuredEndpointAcceptsUnsignedFrame(t *testing.T) {
	clientConn, serverConn := setupTCPPair(t)
	defer clientConn.Close()
	defer serverConn.Close()

	c := protocol.DefaultCodec()
	// Nil key = unsecured endpoint: today's behavior is preserved, and this
	// test pins it intentionally. Changing it is a separate API decision.
	serverEP := NewServerEndpoint(serverConn, c, nil)

	msgCh := make(chan Message, 1)
	errCh := make(chan error, 1)
	go func() {
		received, err := serverEP.Receive()
		if err != nil {
			errCh <- err
			return
		}
		msgCh <- received
	}()

	sendUnsignedFrame(t, clientConn, c)

	select {
	case received := <-msgCh:
		if string(received.Payload) != `{"action":"ping"}` {
			t.Errorf("payload mismatch: got %q", string(received.Payload))
		}
	case err := <-errCh:
		t.Fatalf("unsecured endpoint unexpectedly rejected unsigned frame: %v", err)
	}
}
