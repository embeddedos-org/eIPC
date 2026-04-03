// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

// Minimal EIPC server example.
// Run: go run ./examples/hello-eipc/server/
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/embeddedos-org/eipc/core"
	"github.com/embeddedos-org/eipc/protocol"
	"github.com/embeddedos-org/eipc/transport/tcp"
)

func main() {
	secret := []byte("hello-eipc-demo-secret-key-32b!")
	codec := protocol.DefaultCodec()

	t := tcp.New()
	if err := t.Listen("127.0.0.1:9090"); err != nil {
		log.Fatal(err)
	}
	defer t.Close()
	fmt.Println("EIPC server listening on 127.0.0.1:9090")

	for {
		conn, err := t.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go func() {
			defer conn.Close()
			ep := core.NewServerEndpoint(conn, codec, secret)

			msg, err := ep.Receive()
			if err != nil {
				log.Printf("receive: %v", err)
				return
			}
			fmt.Printf("received: type=%s source=%s payload=%s\n", msg.Type, msg.Source, msg.Payload)

			ack := core.Message{
				Version:   core.ProtocolVersion,
				Type:      core.TypeAck,
				Source:    "hello-server",
				Timestamp: time.Now().UTC(),
				RequestID: msg.RequestID,
				Priority:  core.PriorityP0,
				Payload:   []byte(`{"status":"ok","message":"Hello from EIPC!"}`),
			}
			ep.Send(ack)
		}()
	}
}
