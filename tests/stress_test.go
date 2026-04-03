// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package tests

import (
	"crypto/rand"
	"sync"
	"testing"
	"time"

	"github.com/embeddedos-org/eipc/core"
	"github.com/embeddedos-org/eipc/protocol"
	"github.com/embeddedos-org/eipc/transport/tcp"
)

func TestStress_LargePayload(t *testing.T) {
	secret := []byte(testSecret)
	codec := protocol.DefaultCodec()

	transport := tcp.New()
	if err := transport.Listen("127.0.0.1:0"); err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer transport.Close()
	addr := transport.Addr()

	payload := make([]byte, 512*1024) // 512KB
	rand.Read(payload)

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := transport.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		ep := core.NewServerEndpoint(conn, codec, secret)
		msg, err := ep.Receive()
		if err != nil {
			t.Errorf("server receive: %v", err)
			return
		}
		if len(msg.Payload) != len(payload) {
			t.Errorf("expected payload %d bytes, got %d", len(payload), len(msg.Payload))
		}
	}()

	clientTransport := tcp.New()
	conn, err := clientTransport.Dial(addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	ep := core.NewClientEndpoint(conn, codec, secret, "")
	err = ep.Send(core.Message{
		Version:   core.ProtocolVersion,
		Type:      core.TypeIntent,
		Source:    "stress.client",
		Timestamp: time.Now().UTC(),
		RequestID: "large-1",
		Priority:  core.PriorityP0,
		Payload:   payload,
	})
	if err != nil {
		t.Fatalf("send large payload: %v", err)
	}

	<-done
}

func TestStress_ConcurrentClients(t *testing.T) {
	secret := []byte(testSecret)
	codec := protocol.DefaultCodec()
	numClients := 10

	serverTransport := tcp.New()
	if err := serverTransport.Listen("127.0.0.1:0"); err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer serverTransport.Close()
	addr := serverTransport.Addr()

	var serverWg sync.WaitGroup
	serverWg.Add(numClients)

	go func() {
		for i := 0; i < numClients; i++ {
			conn, err := serverTransport.Accept()
			if err != nil {
				return
			}
			go func() {
				defer serverWg.Done()
				defer conn.Close()
				ep := core.NewServerEndpoint(conn, codec, secret)
				msg, err := ep.Receive()
				if err != nil {
					t.Errorf("server receive: %v", err)
					return
				}

				resp := core.Message{
					Version:   core.ProtocolVersion,
					Type:      core.TypeAck,
					Source:    "server",
					Timestamp: time.Now().UTC(),
					RequestID: msg.RequestID,
					Priority:  core.PriorityP0,
					Payload:   []byte(`{"status":"ok"}`),
				}
				ep.Send(resp)
			}()
		}
	}()

	var clientWg sync.WaitGroup
	errors := make(chan error, numClients)

	for i := 0; i < numClients; i++ {
		clientWg.Add(1)
		go func(id int) {
			defer clientWg.Done()

			ct := tcp.New()
			conn, err := ct.Dial(addr)
			if err != nil {
				errors <- err
				return
			}
			defer conn.Close()

			ep := core.NewClientEndpoint(conn, codec, secret, "")
			err = ep.Send(core.Message{
				Version:   core.ProtocolVersion,
				Type:      core.TypeIntent,
				Source:    "client",
				Timestamp: time.Now().UTC(),
				RequestID: "concurrent",
				Priority:  core.PriorityP1,
				Payload:   []byte(`{"intent":"test"}`),
			})
			if err != nil {
				errors <- err
				return
			}

			_, err = ep.Receive()
			if err != nil {
				errors <- err
			}
		}(i)
	}

	clientWg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("client error: %v", err)
	}

	serverWg.Wait()
}

func TestStress_MessageOrdering(t *testing.T) {
	secret := []byte(testSecret)
	codec := protocol.DefaultCodec()
	numMessages := 50

	serverTransport := tcp.New()
	if err := serverTransport.Listen("127.0.0.1:0"); err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer serverTransport.Close()
	addr := serverTransport.Addr()

	received := make(chan string, numMessages)
	done := make(chan struct{})

	go func() {
		defer close(done)
		conn, err := serverTransport.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		ep := core.NewServerEndpoint(conn, codec, secret)
		for i := 0; i < numMessages; i++ {
			msg, err := ep.Receive()
			if err != nil {
				t.Errorf("receive %d: %v", i, err)
				return
			}
			received <- msg.RequestID
		}
	}()

	ct := tcp.New()
	conn, err := ct.Dial(addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	ep := core.NewClientEndpoint(conn, codec, secret, "")
	for i := 0; i < numMessages; i++ {
		err := ep.Send(core.Message{
			Version:   core.ProtocolVersion,
			Type:      core.TypeIntent,
			Source:    "ordering.client",
			Timestamp: time.Now().UTC(),
			RequestID: string(rune('A' + i%26)),
			Priority:  core.PriorityP0,
			Payload:   []byte(`{}`),
		})
		if err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
	}

	<-done
	close(received)

	count := 0
	for range received {
		count++
	}
	if count != numMessages {
		t.Errorf("expected %d messages, received %d", numMessages, count)
	}
}
