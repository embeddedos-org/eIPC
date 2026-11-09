// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package broker

import (
	"errors"
	"testing"

	"github.com/embeddedos-org/eipc/core"
)

func testPeer() *core.AuthenticatedPeer {
	return &core.AuthenticatedPeer{
		ServiceID:    "eni.min",
		SessionToken: "tok-abc123",
		Capabilities: []string{"ui:control"},
	}
}

func TestBrokerRouteFor_OK(t *testing.T) {
	b := NewBroker(nil, nil)
	ep := &mockEndpoint{}
	_ = b.Subscribe(&Subscriber{ServiceID: "eai.agent", Endpoint: ep, Priority: core.PriorityP1})
	b.AddRoute(core.TypeIntent, "eai.agent")

	msg := core.NewMessage(core.TypeIntent, "eni.min", []byte(`{}`))
	msg.SessionID = "tok-abc123"
	msg.Capability = "ui:control"
	results, err := b.RouteFor(msg, testPeer())
	if err != nil {
		t.Fatalf("RouteFor rejected a valid message: %v", err)
	}
	if len(results) != 1 || len(ep.sent) != 1 {
		t.Fatalf("expected 1 delivery, got %d results / %d sent", len(results), len(ep.sent))
	}
}

func TestBrokerRouteFor_ForgedSourceDeliversNothing(t *testing.T) {
	b := NewBroker(nil, nil)
	ep := &mockEndpoint{}
	_ = b.Subscribe(&Subscriber{ServiceID: "eai.agent", Endpoint: ep, Priority: core.PriorityP1})
	b.AddRoute(core.TypeIntent, "eai.agent")

	msg := core.NewMessage(core.TypeIntent, "eipc-server", []byte(`{}`))
	msg.SessionID = "tok-abc123"
	_, err := b.RouteFor(msg, testPeer())
	if !errors.Is(err, core.ErrAuth) {
		t.Fatalf("expected ErrAuth for forged source, got %v", err)
	}
	if len(ep.sent) != 0 {
		t.Fatalf("forged message must not be delivered, got %d sent", len(ep.sent))
	}
}

func TestBrokerRouteFor_WrongSessionDeliversNothing(t *testing.T) {
	b := NewBroker(nil, nil)
	ep := &mockEndpoint{}
	_ = b.Subscribe(&Subscriber{ServiceID: "eai.agent", Endpoint: ep, Priority: core.PriorityP1})
	b.AddRoute(core.TypeIntent, "eai.agent")

	msg := core.NewMessage(core.TypeIntent, "eni.min", []byte(`{}`))
	msg.SessionID = "tok-other-session"
	_, err := b.RouteFor(msg, testPeer())
	if !errors.Is(err, core.ErrAuth) {
		t.Fatalf("expected ErrAuth for wrong session, got %v", err)
	}
	if len(ep.sent) != 0 {
		t.Fatalf("wrong-session message must not be delivered, got %d sent", len(ep.sent))
	}
}

func TestBrokerRouteFor_NilPeerBehavesLikeRoute(t *testing.T) {
	b := NewBroker(nil, nil)
	ep := &mockEndpoint{}
	_ = b.Subscribe(&Subscriber{ServiceID: "eai.agent", Endpoint: ep, Priority: core.PriorityP1})
	b.AddRoute(core.TypeIntent, "eai.agent")

	msg := core.NewMessage(core.TypeIntent, "unbound", []byte(`{}`))
	results, err := b.RouteFor(msg, nil)
	if err != nil {
		t.Fatalf("nil peer should route like Route, got %v", err)
	}
	if len(results) != 1 || len(ep.sent) != 1 {
		t.Fatalf("expected 1 delivery, got %d results / %d sent", len(results), len(ep.sent))
	}
}

func TestBrokerFanoutFor_ForgedDeliversNothing(t *testing.T) {
	b := NewBroker(nil, nil)
	ep1 := &mockEndpoint{}
	ep2 := &mockEndpoint{}
	_ = b.Subscribe(&Subscriber{ServiceID: "svc1", Endpoint: ep1})
	_ = b.Subscribe(&Subscriber{ServiceID: "svc2", Endpoint: ep2})

	msg := core.NewMessage(core.TypeHeartbeat, "eipc-server", []byte(`{}`))
	msg.SessionID = "tok-abc123"
	_, err := b.FanoutFor(msg, testPeer())
	if !errors.Is(err, core.ErrAuth) {
		t.Fatalf("expected ErrAuth for forged fanout, got %v", err)
	}
	if len(ep1.sent) != 0 || len(ep2.sent) != 0 {
		t.Fatal("forged fanout must reach no subscriber")
	}
}

func TestBrokerFanoutFor_OK(t *testing.T) {
	b := NewBroker(nil, nil)
	ep1 := &mockEndpoint{}
	ep2 := &mockEndpoint{}
	_ = b.Subscribe(&Subscriber{ServiceID: "svc1", Endpoint: ep1})
	_ = b.Subscribe(&Subscriber{ServiceID: "svc2", Endpoint: ep2})

	msg := core.NewMessage(core.TypeHeartbeat, "eni.min", []byte(`{}`))
	msg.SessionID = "tok-abc123"
	results, err := b.FanoutFor(msg, testPeer())
	if err != nil {
		t.Fatalf("FanoutFor rejected a valid message: %v", err)
	}
	if len(results) != 2 || len(ep1.sent) != 1 || len(ep2.sent) != 1 {
		t.Fatalf("expected fanout to 2 subscribers, got %d results", len(results))
	}
}
