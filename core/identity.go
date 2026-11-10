// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package core

import "fmt"

// AuthenticatedPeer is the core-level view of a peer that has completed
// challenge-response authentication (see security/auth.Authenticator).
// It is the identity that dispatch binds a message to: a message is only
// dispatched for the peer it claims to come from when its envelope
// (Source + SessionID) matches this peer and its Capability is granted.
//
// The type lives in core rather than security/auth so the router, broker,
// health service and endpoints can enforce the binding without an import
// cycle (security/auth already imports core).
type AuthenticatedPeer struct {
	// ServiceID is the authenticated service identity (e.g. "ebot.client").
	ServiceID string
	// SessionToken is the session token issued at authentication time.
	// Senders carry it as Message.SessionID.
	SessionToken string
	// Capabilities granted to this peer at authentication time.
	Capabilities []string
}

// ValidateIdentity checks the envelope identity of msg against the
// authenticated peer. A nil peer means "no peer bound" (unsecured/local
// path) and passes every message; callers that require authentication
// must not pass nil.
//
// When a peer is bound the check is fail-closed:
//   - msg.Source must be present and equal the peer's ServiceID, so a
//     peer cannot impersonate another service by forging Source;
//   - msg.SessionID must be present and equal the peer's SessionToken,
//     so an envelope from another session is rejected.
func (p *AuthenticatedPeer) ValidateIdentity(msg Message) error {
	if p == nil {
		return nil
	}
	if msg.Source == "" {
		return fmt.Errorf("%w: message has no source", ErrAuth)
	}
	if msg.Source != p.ServiceID {
		return fmt.Errorf("%w: source %q is not the authenticated peer %q", ErrAuth, msg.Source, p.ServiceID)
	}
	if msg.SessionID == "" {
		return fmt.Errorf("%w: message from %q carries no session", ErrAuth, msg.Source)
	}
	if msg.SessionID != p.SessionToken {
		return fmt.Errorf("%w: session mismatch for %q", ErrAuth, msg.Source)
	}
	return nil
}

// ValidateMessage is ValidateIdentity plus capability validation: when
// msg.Capability is set it must be one of the peer's granted
// capabilities. An empty Capability passes (the message claims no
// capability-gated action).
func (p *AuthenticatedPeer) ValidateMessage(msg Message) error {
	if err := p.ValidateIdentity(msg); err != nil {
		return err
	}
	if p == nil || msg.Capability == "" {
		return nil
	}
	for _, c := range p.Capabilities {
		if c == msg.Capability {
			return nil
		}
	}
	return fmt.Errorf("%w: peer %q lacks capability %q", ErrCapability, p.ServiceID, msg.Capability)
}
