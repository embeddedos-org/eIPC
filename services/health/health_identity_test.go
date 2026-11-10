// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package health

import (
	"errors"
	"testing"
	"time"

	"github.com/embeddedos-org/eipc/core"
)

func TestRecordHeartbeatFor_OK(t *testing.T) {
	svc := NewService(5*time.Second, 15*time.Second)
	peer := &core.AuthenticatedPeer{ServiceID: "eni.min", SessionToken: "tok-abc123"}

	if err := svc.RecordHeartbeatFor("eni.min", "ready", peer); err != nil {
		t.Fatalf("own heartbeat rejected: %v", err)
	}
	if !svc.IsAlive("eni.min") {
		t.Error("peer should be alive after its own heartbeat")
	}
}

func TestRecordHeartbeatFor_ForgedServiceRejected(t *testing.T) {
	svc := NewService(5*time.Second, 15*time.Second)
	peer := &core.AuthenticatedPeer{ServiceID: "eni.min", SessionToken: "tok-abc123"}

	// A peer authenticated as eni.min reports health for eai.agent.
	if err := svc.RecordHeartbeatFor("eai.agent", "ready", peer); !errors.Is(err, core.ErrAuth) {
		t.Fatalf("expected ErrAuth for forged heartbeat, got %v", err)
	}
	if svc.IsAlive("eai.agent") {
		t.Error("forged heartbeat must not mark the victim alive")
	}
	if len(svc.AllPeers()) != 0 {
		t.Fatalf("forged heartbeat must record nothing, got %d peers", len(svc.AllPeers()))
	}
	if svc.IsAlive("eni.min") {
		t.Error("forged heartbeat must not mark the sender alive either")
	}
}

func TestRecordHeartbeatFor_NilPeerBehavesLikeRecord(t *testing.T) {
	svc := NewService(5*time.Second, 15*time.Second)

	if err := svc.RecordHeartbeatFor("eni.min", "ready", nil); err != nil {
		t.Fatalf("nil peer should record like RecordHeartbeat, got %v", err)
	}
	if !svc.IsAlive("eni.min") {
		t.Error("peer should be alive after nil-peer heartbeat")
	}
}
