// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/embeddedos-org/eipc/core"
)

// PeerIdentity represents an authenticated EIPC peer.
type PeerIdentity struct {
	ServiceID    string        `json:"service_id"`
	Capabilities []string      `json:"capabilities"`
	SessionToken string        `json:"session_token"`
	CreatedAt    time.Time     `json:"created_at"`
	SessionTTL   time.Duration `json:"-"`
}

// IsExpired returns true if the session has exceeded its TTL.
func (p *PeerIdentity) IsExpired() bool {
	if p.SessionTTL <= 0 {
		return false
	}
	return time.Since(p.CreatedAt) > p.SessionTTL
}

// Challenge represents a pending challenge-response authentication.
type Challenge struct {
	ServiceID string
	Nonce     []byte
	CreatedAt time.Time
}

// Authenticator validates peer credentials and issues session tokens.
type Authenticator struct {
	mu                sync.RWMutex
	sharedSecret      []byte
	knownServices     map[string][]string // service_id → allowed capabilities
	activeSessions    map[string]*PeerIdentity
	pendingChallenges map[string]*Challenge // serviceID → pending challenge
	sessionTTL        time.Duration
}

// NewAuthenticator creates an authenticator with the given shared secret.
// knownServices maps service IDs to their allowed capability sets.
func NewAuthenticator(sharedSecret []byte, knownServices map[string][]string) *Authenticator {
	known := make(map[string][]string, len(knownServices))
	for k, v := range knownServices {
		caps := make([]string, len(v))
		copy(caps, v)
		known[k] = caps
	}
	return &Authenticator{
		sharedSecret:      sharedSecret,
		knownServices:     known,
		activeSessions:    make(map[string]*PeerIdentity),
		pendingChallenges: make(map[string]*Challenge),
		sessionTTL:        1 * time.Hour,
	}
}

// SetSessionTTL configures the default session TTL for new sessions.
func (a *Authenticator) SetSessionTTL(ttl time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sessionTTL = ttl
}

// CreateChallenge generates a 32-byte nonce challenge for the given service ID.
// Returns error if the service is unknown.
func (a *Authenticator) CreateChallenge(serviceID string) (*Challenge, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, ok := a.knownServices[serviceID]; !ok {
		return nil, fmt.Errorf("%w: unknown service %q", core.ErrAuth, serviceID)
	}

	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	challenge := &Challenge{
		ServiceID: serviceID,
		Nonce:     nonce,
		CreatedAt: time.Now().UTC(),
	}
	a.pendingChallenges[serviceID] = challenge
	return challenge, nil
}

// VerifyResponse verifies the HMAC-SHA256 response to a challenge.
// On success, creates and returns a PeerIdentity with session token.
func (a *Authenticator) VerifyResponse(serviceID string, response []byte) (*PeerIdentity, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	challenge, ok := a.pendingChallenges[serviceID]
	if !ok {
		return nil, fmt.Errorf("%w: no pending challenge for %q", core.ErrAuth, serviceID)
	}
	delete(a.pendingChallenges, serviceID)

	// Compute expected response: HMAC-SHA256(sharedSecret, nonce)
	mac := hmac.New(sha256.New, a.sharedSecret)
	mac.Write(challenge.Nonce)
	expected := mac.Sum(nil)

	if !hmac.Equal(expected, response) {
		return nil, fmt.Errorf("%w: challenge-response verification failed for %q", core.ErrAuth, serviceID)
	}

	capsSrc := a.knownServices[serviceID]
	caps := make([]string, len(capsSrc))
	copy(caps, capsSrc)

	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	peer := &PeerIdentity{
		ServiceID:    serviceID,
		Capabilities: caps,
		SessionToken: token,
		CreatedAt:    time.Now().UTC(),
		SessionTTL:   a.sessionTTL,
	}
	a.activeSessions[token] = peer
	return peer, nil
}

// Authenticate validates a peer's service ID and returns a PeerIdentity
// with a fresh session token. (Legacy simple auth, kept for backward compat.)
func (a *Authenticator) Authenticate(serviceID string) (*PeerIdentity, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	capsSrc, ok := a.knownServices[serviceID]
	if !ok {
		return nil, fmt.Errorf("%w: unknown service %q", core.ErrAuth, serviceID)
	}

	caps := make([]string, len(capsSrc))
	copy(caps, capsSrc)

	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	peer := &PeerIdentity{
		ServiceID:    serviceID,
		Capabilities: caps,
		SessionToken: token,
		CreatedAt:    time.Now().UTC(),
		SessionTTL:   a.sessionTTL,
	}
	a.activeSessions[token] = peer
	return peer, nil
}

// ValidateSession checks whether a session token is valid and not expired.
func (a *Authenticator) ValidateSession(token string) (*PeerIdentity, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	peer, ok := a.activeSessions[token]
	if !ok {
		return nil, fmt.Errorf("%w: invalid session token", core.ErrAuth)
	}
	if peer.IsExpired() {
		return nil, fmt.Errorf("%w: session expired", core.ErrAuth)
	}
	return peer, nil
}

// SharedSecret returns the shared HMAC key for message signing.
func (a *Authenticator) SharedSecret() []byte {
	return a.sharedSecret
}

// RevokeSession removes a session.
func (a *Authenticator) RevokeSession(token string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.activeSessions, token)
}

// CleanupExpired removes all expired sessions. Returns count removed.
func (a *Authenticator) CleanupExpired() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	removed := 0
	for token, peer := range a.activeSessions {
		if peer.IsExpired() {
			delete(a.activeSessions, token)
			removed++
		}
	}
	return removed
}

// ActiveSessionCount returns the number of active sessions.
func (a *Authenticator) ActiveSessionCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.activeSessions)
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
