// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package registry

import (
	"testing"

	"github.com/embeddedos-org/eipc/core"
)

func TestRegister(t *testing.T) {
	reg := NewRegistry()
	err := reg.Register(ServiceInfo{
		ServiceID:    "eni.min",
		Capabilities: []string{"ui:control"},
		Versions:     []uint16{1},
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
}

func TestRegister_EmptyID(t *testing.T) {
	reg := NewRegistry()
	err := reg.Register(ServiceInfo{})
	if err == nil {
		t.Fatal("expected error for empty service_id")
	}
}

func TestLookup(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(ServiceInfo{
		ServiceID:    "eni.min",
		Capabilities: []string{"ui:control"},
	})

	info, err := reg.Lookup("eni.min")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if info.ServiceID != "eni.min" {
		t.Errorf("expected 'eni.min', got %q", info.ServiceID)
	}
}

func TestLookup_NotFound(t *testing.T) {
	reg := NewRegistry()
	_, err := reg.Lookup("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent service")
	}
}

func TestDeregister(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(ServiceInfo{ServiceID: "eni.min"})
	reg.Deregister("eni.min")

	_, err := reg.Lookup("eni.min")
	if err == nil {
		t.Fatal("expected error after deregistration")
	}
}

func TestList(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(ServiceInfo{ServiceID: "eni.min"})
	reg.Register(ServiceInfo{ServiceID: "eai.agent"})

	list := reg.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 services, got %d", len(list))
	}
}

func TestFindByCapability(t *testing.T) {
	reg := NewRegistry()
	reg.Register(ServiceInfo{ServiceID: "eni.min", Capabilities: []string{"ui:control"}})
	reg.Register(ServiceInfo{ServiceID: "eai.agent", Capabilities: []string{"ai:chat"}})
	reg.Register(ServiceInfo{ServiceID: "tool.svc", Capabilities: []string{"ui:control", "device:read"}})

	results := reg.FindByCapability("ui:control")
	if len(results) != 2 {
		t.Fatalf("expected 2 services with ui:control, got %d", len(results))
	}
}

func TestFindByCapability_None(t *testing.T) {
	reg := NewRegistry()
	reg.Register(ServiceInfo{ServiceID: "eni.min", Capabilities: []string{"ui:control"}})

	results := reg.FindByCapability("nonexistent")
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestRegister_WithMessageTypes(t *testing.T) {
	reg := NewRegistry()
	reg.Register(ServiceInfo{
		ServiceID:    "eni.min",
		Capabilities: []string{"ui:control"},
		MessageTypes: []core.MessageType{core.TypeIntent, core.TypeHeartbeat},
		Priority:     core.PriorityP0,
	})

	info, _ := reg.Lookup("eni.min")
	if len(info.MessageTypes) != 2 {
		t.Errorf("expected 2 message types, got %d", len(info.MessageTypes))
	}
	if info.Priority != core.PriorityP0 {
		t.Errorf("expected P0 priority, got %d", info.Priority)
	}
}

func TestRegister_Update(t *testing.T) {
	reg := NewRegistry()
	reg.Register(ServiceInfo{ServiceID: "eni.min", Capabilities: []string{"ui:control"}})
	reg.Register(ServiceInfo{ServiceID: "eni.min", Capabilities: []string{"ui:control", "device:read"}})

	info, _ := reg.Lookup("eni.min")
	if len(info.Capabilities) != 2 {
		t.Errorf("expected updated capabilities with 2 entries, got %d", len(info.Capabilities))
	}
}
