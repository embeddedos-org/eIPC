// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

// Minimal EIPC client example.
// Run: go run ./examples/hello-eipc/client/
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
	conn, err := t.Dial("127.0.0.1:9090")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	ep := core.NewClientEndpoint(conn, codec, secret, "")

	msg := core.Message{
		Version:   core.ProtocolVersion,
		Type:      core.TypeChat,
		Source:    "hello-client",
		Timestamp: time.Now().UTC(),
		RequestID: "hello-1",
		Priority:  core.PriorityP1,
		Payload:   []byte(`{"text":"Hello, EIPC!"}`),
	}

	if err := ep.Send(msg); err != nil {
		log.Fatal(err)
	}
	fmt.Println("sent message to server")

	resp, err := ep.Receive()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("response: %s\n", resp.Payload)
}
