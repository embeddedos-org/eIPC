// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"strings"
	"testing"
	"time"
)

// challengeAuth issues a session through the challenge-response flow, the
// way a real peer authenticates. Tests that need a session use this; the
// legacy Authenticate path is exercised only by the opt-in tests below.
func challengeAuth(t *testing.T, a *Authenticator, secret []byte, serviceID string) *PeerIdentity {
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

func TestChallengeAuthKnownService(t *testing.T) {
	secret := []byte("shared-secret")
	knownServices := map[string][]string{
		"service-a": {"cap:read", "cap:write"},
		"service-b": {"cap:read"},
	}

	a := NewAuthenticator(secret, knownServices)

	identity := challengeAuth(t, a, secret, "service-a")
	if identity == nil {
		t.Fatal("expected identity, got nil")
	}
	if identity.ServiceID != "service-a" {
		t.Errorf("expected service-a, got %q", identity.ServiceID)
	}
	if len(identity.Capabilities) != 2 {
		t.Errorf("expected 2 capabilities, got %d", len(identity.Capabilities))
	}
}

func TestChallengeAuthUnknownService(t *testing.T) {
	knownServices := map[string][]string{
		"service-a": {"cap:read"},
	}

	a := NewAuthenticator([]byte("shared-secret"), knownServices)

	_, err := a.CreateChallenge("unknown-service")
	if err == nil {
		t.Fatal("expected error for unknown service")
	}
}

func TestLegacyAuthenticateDisabledByDefault(t *testing.T) {
	knownServices := map[string][]string{
		"service-a": {"cap:read"},
	}

	a := NewAuthenticator([]byte("shared-secret"), knownServices)

	_, err := a.Authenticate("service-a")
	if err == nil {
		t.Fatal("expected legacy Authenticate to be refused by default")
	}
	if !strings.Contains(err.Error(), "disabled") {
		t.Errorf("expected a 'disabled' error, got: %v", err)
	}
}

func TestLegacyAuthenticateOptIn(t *testing.T) {
	knownServices := map[string][]string{
		"service-a": {"cap:read"},
	}

	a := NewAuthenticator([]byte("shared-secret"), knownServices)
	a.SetAllowLegacyAuth(true)

	identity, err := a.Authenticate("service-a")
	if err != nil {
		t.Fatalf("opted-in Authenticate failed: %v", err)
	}
	if identity.ServiceID != "service-a" {
		t.Errorf("expected service-a, got %q", identity.ServiceID)
	}

	_, err = a.Authenticate("unknown-service")
	if err == nil {
		t.Fatal("expected error for unknown service even with legacy auth on")
	}
}

func TestVerifyResponseExpiredChallenge(t *testing.T) {
	secret := []byte("shared-secret")
	a := NewAuthenticator(secret, map[string][]string{
		"service-a": {"cap:read"},
	})
	a.SetChallengeTTL(1 * time.Millisecond)

	challenge, err := a.CreateChallenge("service-a")
	if err != nil {
		t.Fatalf("CreateChallenge: %v", err)
	}

	time.Sleep(5 * time.Millisecond)

	mac := hmac.New(sha256.New, secret)
	mac.Write(challenge.Nonce)
	_, err = a.VerifyResponse("service-a", mac.Sum(nil))
	if err == nil {
		t.Fatal("expected an expired challenge to be rejected")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("expected an 'expired' error, got: %v", err)
	}

	// The expired challenge was consumed (one-shot): a second attempt
	// finds no pending challenge at all.
	_, err = a.VerifyResponse("service-a", mac.Sum(nil))
	if err == nil || !strings.Contains(err.Error(), "no pending challenge") {
		t.Errorf("expected 'no pending challenge' after expiry, got: %v", err)
	}
}

func TestVerifyResponseChallengeWithinTTL(t *testing.T) {
	secret := []byte("shared-secret")
	a := NewAuthenticator(secret, map[string][]string{
		"service-a": {"cap:read"},
	})
	a.SetChallengeTTL(1 * time.Minute)

	peer := challengeAuth(t, a, secret, "service-a")
	if peer.ServiceID != "service-a" {
		t.Errorf("expected service-a, got %q", peer.ServiceID)
	}
}

func TestValidateSession(t *testing.T) {
	secret := []byte("shared-secret")
	knownServices := map[string][]string{
		"service-a": {"cap:read", "cap:write"},
	}

	a := NewAuthenticator(secret, knownServices)

	identity := challengeAuth(t, a, secret, "service-a")

	validated, err := a.ValidateSession(identity.SessionToken)
	if err != nil {
		t.Fatalf("ValidateSession failed: %v", err)
	}
	if validated == nil {
		t.Fatal("expected validated identity, got nil")
	}
	if validated.ServiceID != "service-a" {
		t.Errorf("expected service-a, got %q", validated.ServiceID)
	}
}

func TestValidateSessionInvalidToken(t *testing.T) {
	knownServices := map[string][]string{
		"service-a": {"cap:read"},
	}

	a := NewAuthenticator([]byte("shared-secret"), knownServices)

	_, err := a.ValidateSession("invalid-token-12345")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestRevokeSession(t *testing.T) {
	secret := []byte("shared-secret")
	knownServices := map[string][]string{
		"service-a": {"cap:read"},
	}

	a := NewAuthenticator(secret, knownServices)

	identity := challengeAuth(t, a, secret, "service-a")

	a.RevokeSession(identity.SessionToken)

	_, err := a.ValidateSession(identity.SessionToken)
	if err == nil {
		t.Fatal("expected error after revoking session")
	}
}

func TestMultipleSessions(t *testing.T) {
	secret := []byte("shared-secret")
	knownServices := map[string][]string{
		"service-a": {"cap:read"},
		"service-b": {"cap:write"},
	}

	a := NewAuthenticator(secret, knownServices)

	idA := challengeAuth(t, a, secret, "service-a")
	idB := challengeAuth(t, a, secret, "service-b")

	if idA.SessionToken == idB.SessionToken {
		t.Error("expected different tokens for different services")
	}

	a.RevokeSession(idA.SessionToken)

	_, err := a.ValidateSession(idA.SessionToken)
	if err == nil {
		t.Error("expected error for revoked token A")
	}

	_, err = a.ValidateSession(idB.SessionToken)
	if err != nil {
		t.Errorf("token B should still be valid: %v", err)
	}
}
