// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package core

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// ReconnectPolicy configures automatic reconnection behavior.
type ReconnectPolicy struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	BackoffFactor  float64
}

// DefaultReconnectPolicy returns a sensible default reconnect policy.
func DefaultReconnectPolicy() ReconnectPolicy {
	return ReconnectPolicy{
		MaxRetries:     10,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     30 * time.Second,
		BackoffFactor:  2.0,
	}
}

// Backoff calculates the backoff duration for the given attempt number.
func (p ReconnectPolicy) Backoff(attempt int) time.Duration {
	if attempt <= 0 {
		return p.InitialBackoff
	}
	backoff := float64(p.InitialBackoff) * math.Pow(p.BackoffFactor, float64(attempt))
	if backoff > float64(p.MaxBackoff) {
		backoff = float64(p.MaxBackoff)
	}
	return time.Duration(backoff)
}

// HeartbeatConfig configures periodic heartbeat sending.
type HeartbeatConfig struct {
	Interval  time.Duration
	ServiceID string
}

// HeartbeatSender sends periodic heartbeat messages over an endpoint.
type HeartbeatSender struct {
	endpoint Endpoint
	config   HeartbeatConfig
	stopCh   chan struct{}
	stopped  bool
	mu       sync.Mutex
}

// NewHeartbeatSender creates a heartbeat sender for the given endpoint.
func NewHeartbeatSender(endpoint Endpoint, config HeartbeatConfig) *HeartbeatSender {
	if config.Interval == 0 {
		config.Interval = 5 * time.Second
	}
	return &HeartbeatSender{
		endpoint: endpoint,
		config:   config,
		stopCh:   make(chan struct{}),
	}
}

// Start begins sending heartbeats in a background goroutine.
func (h *HeartbeatSender) Start() {
	go func() {
		ticker := time.NewTicker(h.config.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-h.stopCh:
				return
			case <-ticker.C:
				msg := Message{
					Version:   ProtocolVersion,
					Type:      TypeHeartbeat,
					Source:    h.config.ServiceID,
					Timestamp: time.Now().UTC(),
					Priority:  PriorityP3,
					Payload:   []byte(fmt.Sprintf(`{"service":"%s","status":"alive"}`, h.config.ServiceID)),
				}
				if err := h.endpoint.Send(msg); err != nil {
					fmt.Printf("heartbeat send failed: %v\n", err)
				}
			}
		}
	}()
}

// Stop halts heartbeat sending.
func (h *HeartbeatSender) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.stopped {
		close(h.stopCh)
		h.stopped = true
	}
}

// GracefulShutdown closes an endpoint after draining in-flight messages.
// It waits up to timeout before forcing closure.
func GracefulShutdown(endpoint Endpoint, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() {
		done <- endpoint.Close()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timed out after %v", timeout)
	}
}
