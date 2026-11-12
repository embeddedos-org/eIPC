// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package auth

import (
	"testing"

	"github.com/embeddedos-org/eipc/core"
)

func TestPeerIdentityBoundPeer_RoundTrip(t *testing.T) {
	secret := []byte("test-key-32-bytes-long-enough!!")
	authenticator := NewAuthenticator(secret, map[string][]string{
		"ebot.client": {"ai:chat"},
	})

	peer := challengeAuth(t, authenticator, secret, "ebot.client")

	bound := peer.BoundPeer()
	if bound == nil {
		t.Fatal("expected a bound peer")
	}
	if bound.ServiceID != "ebot.client" {
		t.Errorf("ServiceID mismatch: got %q", bound.ServiceID)
	}
	if bound.SessionToken != peer.SessionToken {
		t.Error("SessionToken was not carried over")
	}
	if len(bound.Capabilities) != 1 || bound.Capabilities[0] != "ai:chat" {
		t.Errorf("Capabilities mismatch: got %v", bound.Capabilities)
	}

	// The bound peer validates a genuine message from this identity.
	msg := core.Message{
		Type:       core.TypeChat,
		Source:     "ebot.client",
		SessionID:  peer.SessionToken,
		Capability: "ai:chat",
	}
	if err := bound.ValidateMessage(msg); err != nil {
		t.Fatalf("genuine message rejected: %v", err)
	}

	// ... and rejects a message impersonating another service.
	msg.Source = "eipc-server"
	if err := bound.ValidateMessage(msg); err == nil {
		t.Fatal("expected rejection for forged source")
	}
}

func TestPeerIdentityBoundPeer_NilIdentity(t *testing.T) {
	var peer *PeerIdentity
	if peer.BoundPeer() != nil {
		t.Fatal("expected nil for nil identity")
	}
}
