// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package core

import (
	"errors"
	"testing"
)

func testPeer() *AuthenticatedPeer {
	return &AuthenticatedPeer{
		ServiceID:    "ebot.client",
		SessionToken: "tok-abc123",
		Capabilities: []string{"ai:chat", "ui:control"},
	}
}

func testMessage() Message {
	return Message{
		Type:       TypeChat,
		Source:     "ebot.client",
		SessionID:  "tok-abc123",
		Capability: "ai:chat",
	}
}

func TestAuthenticatedPeerValidateMessage_OK(t *testing.T) {
	if err := testPeer().ValidateMessage(testMessage()); err != nil {
		t.Fatalf("valid message rejected: %v", err)
	}
}

func TestAuthenticatedPeerValidateMessage_EmptyCapabilityOK(t *testing.T) {
	msg := testMessage()
	msg.Capability = ""
	if err := testPeer().ValidateMessage(msg); err != nil {
		t.Fatalf("message without capability rejected: %v", err)
	}
}

func TestAuthenticatedPeerValidateMessage_ForgedSource(t *testing.T) {
	msg := testMessage()
	msg.Source = "eipc-server" // impersonating another service
	if err := testPeer().ValidateMessage(msg); !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth for forged source, got %v", err)
	}
}

func TestAuthenticatedPeerValidateMessage_MissingSource(t *testing.T) {
	msg := testMessage()
	msg.Source = ""
	if err := testPeer().ValidateMessage(msg); !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth for missing source, got %v", err)
	}
}

func TestAuthenticatedPeerValidateMessage_WrongSession(t *testing.T) {
	msg := testMessage()
	msg.SessionID = "tok-stolen-from-another-session"
	if err := testPeer().ValidateMessage(msg); !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth for wrong session, got %v", err)
	}
}

func TestAuthenticatedPeerValidateMessage_MissingSession(t *testing.T) {
	msg := testMessage()
	msg.SessionID = ""
	if err := testPeer().ValidateMessage(msg); !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth for missing session, got %v", err)
	}
}

func TestAuthenticatedPeerValidateMessage_UngrantedCapability(t *testing.T) {
	msg := testMessage()
	msg.Capability = "device:write" // not granted to ebot.client
	if err := testPeer().ValidateMessage(msg); !errors.Is(err, ErrCapability) {
		t.Fatalf("expected ErrCapability for ungranted capability, got %v", err)
	}
}

func TestAuthenticatedPeerValidateMessage_NilPeerAllows(t *testing.T) {
	var peer *AuthenticatedPeer
	msg := testMessage()
	msg.Source = "anyone"
	msg.SessionID = "whatever"
	if err := peer.ValidateMessage(msg); err != nil {
		t.Fatalf("nil peer should pass every message, got %v", err)
	}
}

func TestAuthenticatedPeerValidateIdentity_IgnoresCapability(t *testing.T) {
	// Identity binding is source + session only; capability is a
	// separate dispatch-time check.
	msg := testMessage()
	msg.Capability = "device:write"
	if err := testPeer().ValidateIdentity(msg); err != nil {
		t.Fatalf("ValidateIdentity should ignore capability, got %v", err)
	}
}
