// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package health

import (
	"testing"
	"time"
)

func TestRecordHeartbeat(t *testing.T) {
	svc := NewService(5*time.Second, 15*time.Second)
	svc.RecordHeartbeat("eni.min", "ready")

	if !svc.IsAlive("eni.min") {
		t.Error("peer should be alive after heartbeat")
	}
}

func TestIsAlive_Unknown(t *testing.T) {
	svc := NewService(5*time.Second, 15*time.Second)
	if svc.IsAlive("unknown") {
		t.Error("unknown peer should not be alive")
	}
}

func TestIsAlive_Timeout(t *testing.T) {
	svc := NewService(5*time.Second, 50*time.Millisecond)
	svc.RecordHeartbeat("eni.min", "ready")

	time.Sleep(100 * time.Millisecond)

	if svc.IsAlive("eni.min") {
		t.Error("peer should be dead after timeout")
	}
}

func TestAllPeers(t *testing.T) {
	svc := NewService(5*time.Second, 15*time.Second)
	svc.RecordHeartbeat("eni.min", "ready")
	svc.RecordHeartbeat("eai.agent", "running")

	peers := svc.AllPeers()
	if len(peers) != 2 {
		t.Fatalf("expected 2 peers, got %d", len(peers))
	}
}

func TestLivePeers(t *testing.T) {
	svc := NewService(5*time.Second, 50*time.Millisecond)
	svc.RecordHeartbeat("eni.min", "ready")
	svc.RecordHeartbeat("eai.agent", "running")

	time.Sleep(100 * time.Millisecond)

	svc.RecordHeartbeat("eai.agent", "running")

	live := svc.LivePeers()
	if len(live) != 1 {
		t.Fatalf("expected 1 live peer, got %d", len(live))
	}
	if live[0].ServiceID != "eai.agent" {
		t.Errorf("expected eai.agent, got %s", live[0].ServiceID)
	}
}

func TestInterval(t *testing.T) {
	svc := NewService(10*time.Second, 30*time.Second)
	if svc.Interval() != 10*time.Second {
		t.Errorf("expected 10s interval, got %v", svc.Interval())
	}
}

func TestDefaults(t *testing.T) {
	svc := NewService(0, 0)
	if svc.Interval() != 5*time.Second {
		t.Errorf("expected default 5s interval, got %v", svc.Interval())
	}
}

func TestHeartbeatUpdate(t *testing.T) {
	svc := NewService(5*time.Second, 15*time.Second)
	svc.RecordHeartbeat("eni.min", "starting")
	svc.RecordHeartbeat("eni.min", "ready")

	peers := svc.AllPeers()
	if len(peers) != 1 {
		t.Fatalf("expected 1 peer after update, got %d", len(peers))
	}
	if peers[0].Status != "ready" {
		t.Errorf("expected status 'ready', got %q", peers[0].Status)
	}
}
