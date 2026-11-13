// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package core

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/embeddedos-org/eipc/protocol"
)

func TestServerEndpointSetReadDeadline(t *testing.T) {
	clientConn, serverConn := setupTCPPair(t)
	defer clientConn.Close()
	defer serverConn.Close()

	serverEP := NewServerEndpoint(serverConn, protocol.DefaultCodec(), []byte("test-secret-key-for-hmac-256bit!"))

	// A deadline already in the past: Receive must fail fast with a
	// timeout instead of blocking until the peer speaks, so an idle
	// authenticated peer cannot hold its connection slot forever.
	if err := serverEP.SetReadDeadline(time.Now().Add(-time.Second)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}
	start := time.Now()
	_, err := serverEP.Receive()
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
