// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package tests

import (
	"crypto/hmac"
	"crypto/sha256"
	"testing"

	"github.com/embeddedos-org/eipc/security/auth"
)

// challengeAuth issues a session through the challenge-response flow, the
// way a real peer authenticates. The legacy no-secret Authenticate path is
// disabled by default and must not be used to obtain test sessions.
func challengeAuth(t *testing.T, a *auth.Authenticator, secret []byte, serviceID string) *auth.PeerIdentity {
	t.Helper()
	challenge, err := a.CreateChallenge(serviceID)
	if err != nil {
		t.Fatalf("CreateChallenge: %v", err)
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write(challenge.Nonce)
	peer, err := a.VerifyResponse(serviceID, mac.Sum(nil))
	if err != nil {
		t.Fatalf("VerifyResponse: %v", err)
	}
	return peer
}
