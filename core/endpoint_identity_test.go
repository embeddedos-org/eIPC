// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package core

import (
	"errors"
	"testing"

	"github.com/embeddedos-org/eipc/protocol"
)

// TestServerEndpointBindPeer_AcceptsGenuineMessage proves a bound endpoint
// accepts a message whose Source and SessionID match the peer, including
// the client-side session stamping (ClientEndpoint stamps its session
// token when the message carries none).
func TestServerEndpointBindPeer_AcceptsGenuineMessage(t *testing.T) {
	clientConn, serverConn := setupTCPPair(t)
	defer clientConn.Close()
	defer serverConn.Close()

	hmacKey := []byte("test-secret-key-for-hmac-256bit!")
	c := protocol.DefaultCodec()

	clientEP := NewClientEndpoint(clientConn, c, hmacKey, "tok-abc123")
	serverEP := NewServerEndpoint(serverConn, c, hmacKey)
	serverEP.BindPeer(&AuthenticatedPeer{
		ServiceID:    "ebot.client",
		SessionToken: "tok-abc123",
		Capabilities: []string{"ai:chat"},
	})

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

	// No SessionID set on the message: the client endpoint stamps it.
	msg := NewMessage(TypeChat, "ebot.client", []byte(`{"prompt":"hi"}`))
	if err := clientEP.Send(msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	select {
	case received := <-msgCh:
		if received.Source != "ebot.client" {
			t.Errorf("source mismatch: got %q", received.Source)
		}
		if received.SessionID != "tok-abc123" {
			t.Errorf("session was not stamped/bound: got %q", received.SessionID)
		}
	case err := <-errCh:
		t.Fatalf("Receive rejected a genuine message: %v", err)
	}
}

// TestServerEndpointBindPeer_RejectsForgedSource proves a bound endpoint
// drops a message that claims another service's identity, even when the
// session token is correct.
func TestServerEndpointBindPeer_RejectsForgedSource(t *testing.T) {
	clientConn, serverConn := setupTCPPair(t)
	defer clientConn.Close()
	defer serverConn.Close()

	hmacKey := []byte("test-secret-key-for-hmac-256bit!")
	c := protocol.DefaultCodec()

	clientEP := NewClientEndpoint(clientConn, c, hmacKey, "tok-abc123")
	serverEP := NewServerEndpoint(serverConn, c, hmacKey)
	serverEP.BindPeer(&AuthenticatedPeer{
		ServiceID:    "ebot.client",
		SessionToken: "tok-abc123",
	})

	errCh := make(chan error, 1)
	go func() {
		_, err := serverEP.Receive()
		errCh <- err
	}()

	msg := NewMessage(TypeChat, "eipc-server", []byte(`{}`))
	msg.SessionID = "tok-abc123"
	if err := clientEP.Send(msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if err := <-errCh; !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth for forged source, got %v", err)
	}
}

// TestServerEndpointBindPeer_RejectsWrongSession proves a bound endpoint
// drops a message from the right service carrying another session's token.
func TestServerEndpointBindPeer_RejectsWrongSession(t *testing.T) {
	clientConn, serverConn := setupTCPPair(t)
	defer clientConn.Close()
	defer serverConn.Close()

	hmacKey := []byte("test-secret-key-for-hmac-256bit!")
	c := protocol.DefaultCodec()

	clientEP := NewClientEndpoint(clientConn, c, hmacKey, "tok-other-session")
	serverEP := NewServerEndpoint(serverConn, c, hmacKey)
	serverEP.BindPeer(&AuthenticatedPeer{
		ServiceID:    "ebot.client",
		SessionToken: "tok-abc123",
	})

	errCh := make(chan error, 1)
	go func() {
		_, err := serverEP.Receive()
		errCh <- err
	}()

	msg := NewMessage(TypeChat, "ebot.client", []byte(`{}`))
	msg.SessionID = "tok-other-session"
	if err := clientEP.Send(msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if err := <-errCh; !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth for wrong session, got %v", err)
	}
}

// TestServerEndpointBindPeer_ClearsOnNil proves BindPeer(nil) removes the
// binding and restores the previous unbound behavior.
func TestServerEndpointBindPeer_ClearsOnNil(t *testing.T) {
	clientConn, serverConn := setupTCPPair(t)
	defer clientConn.Close()
	defer serverConn.Close()

	hmacKey := []byte("test-secret-key-for-hmac-256bit!")
	c := protocol.DefaultCodec()

	serverEP := NewServerEndpoint(serverConn, c, hmacKey)
	serverEP.BindPeer(&AuthenticatedPeer{ServiceID: "ebot.client", SessionToken: "tok-abc123"})
	if serverEP.BoundPeer() == nil {
		t.Fatal("expected a bound peer")
	}
	serverEP.BindPeer(nil)
	if serverEP.BoundPeer() != nil {
		t.Fatal("expected the binding to be cleared")
	}

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

	clientEP := NewClientEndpoint(clientConn, c, hmacKey, "tok-abc123")
	msg := NewMessage(TypeChat, "anyone", []byte(`{}`))
	if err := clientEP.Send(msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	select {
	case <-msgCh:
	case err := <-errCh:
		t.Fatalf("unbound endpoint rejected a message: %v", err)
	}
}
