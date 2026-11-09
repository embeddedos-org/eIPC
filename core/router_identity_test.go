// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package core

import (
	"errors"
	"testing"
)

func TestRouterDispatchFor_OK(t *testing.T) {
	r := NewRouter()
	called := false
	r.Handle(TypeChat, func(msg Message) (*Message, error) {
		called = true
		return nil, nil
	})

	peer := testPeer()
	msg := testMessage()
	if _, err := r.DispatchFor(msg, peer); err != nil {
		t.Fatalf("DispatchFor rejected a valid message: %v", err)
	}
	if !called {
		t.Error("handler was not called for a valid message")
	}
}

func TestRouterDispatchFor_ForgedSourceNeverReachesHandler(t *testing.T) {
	r := NewRouter()
	called := false
	r.Handle(TypeChat, func(msg Message) (*Message, error) {
		called = true
		return nil, nil
	})

	msg := testMessage()
	msg.Source = "eipc-server"
	if _, err := r.DispatchFor(msg, testPeer()); !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth for forged source, got %v", err)
	}
	if called {
		t.Error("handler must not run for a forged message")
	}
}

func TestRouterDispatchFor_WrongSession(t *testing.T) {
	r := NewRouter()
	called := false
	r.Handle(TypeChat, func(msg Message) (*Message, error) {
		called = true
		return nil, nil
	})

	msg := testMessage()
	msg.SessionID = "tok-other-session"
	if _, err := r.DispatchFor(msg, testPeer()); !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth for wrong session, got %v", err)
	}
	if called {
		t.Error("handler must not run for a wrong-session message")
	}
}

func TestRouterDispatchFor_UngrantedCapability(t *testing.T) {
	r := NewRouter()
	called := false
	r.Handle(TypeChat, func(msg Message) (*Message, error) {
		called = true
		return nil, nil
	})

	msg := testMessage()
	msg.Capability = "device:write"
	if _, err := r.DispatchFor(msg, testPeer()); !errors.Is(err, ErrCapability) {
		t.Fatalf("expected ErrCapability for ungranted capability, got %v", err)
	}
	if called {
		t.Error("handler must not run for an ungranted capability")
	}
}

func TestRouterDispatchFor_NilPeerBehavesLikeDispatch(t *testing.T) {
	r := NewRouter()
	called := false
	r.Handle(TypeChat, func(msg Message) (*Message, error) {
		called = true
		return nil, nil
	})

	msg := testMessage()
	msg.Source = "unbound"
	msg.SessionID = ""
	if _, err := r.DispatchFor(msg, nil); err != nil {
		t.Fatalf("nil peer should dispatch like Dispatch, got %v", err)
	}
	if !called {
		t.Error("handler was not called for nil-peer dispatch")
	}
}

func TestRouterDispatchBatchFor_MixedBatch(t *testing.T) {
	r := NewRouter()
	var seen []string
	r.Handle(TypeChat, func(msg Message) (*Message, error) {
		seen = append(seen, msg.Source)
		return nil, nil
	})

	peer := testPeer()
	good := testMessage()
	forged := testMessage()
	forged.Source = "eipc-server"

	results := r.DispatchBatchFor([]Message{forged, good}, peer)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if !errors.Is(results[0].Err, ErrAuth) {
		t.Fatalf("expected ErrAuth for forged message, got %v", results[0].Err)
	}
	if results[1].Err != nil {
		t.Fatalf("expected valid message to dispatch, got %v", results[1].Err)
	}
	if len(seen) != 1 || seen[0] != "ebot.client" {
		t.Errorf("expected only the valid message dispatched, got %v", seen)
	}
}
