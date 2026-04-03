// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

// eipc-cli is a debugging command-line tool for sending and receiving EIPC messages.
//
// Usage:
//
//	eipc-cli send --addr HOST:PORT --type chat --payload '{"text":"hello"}'
//	eipc-cli listen --addr HOST:PORT
//	eipc-cli inspect --addr HOST:PORT
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/embeddedos-org/eipc/config"
	"github.com/embeddedos-org/eipc/core"
	"github.com/embeddedos-org/eipc/protocol"
	"github.com/embeddedos-org/eipc/transport/tcp"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "send":
		cmdSend(os.Args[2:])
	case "listen":
		cmdListen(os.Args[2:])
	case "ping":
		cmdPing(os.Args[2:])
	case "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`eipc-cli — EIPC debugging tool

Usage:
  eipc-cli <command> [options]

Commands:
  send     Send a single message to an EIPC server
  listen   Connect and print incoming messages
  ping     Send a heartbeat and wait for response
  help     Show this help message

Environment:
  EIPC_HMAC_KEY        Shared HMAC key (required)
  EIPC_LISTEN_ADDR     Default server address (optional)`)
}

func cmdSend(args []string) {
	fs := flag.NewFlagSet("send", flag.ExitOnError)
	addr := fs.String("addr", config.LoadListenAddr(), "server address")
	msgType := fs.String("type", "chat", "message type (intent|chat|heartbeat|ack)")
	payload := fs.String("payload", `{}`, "JSON payload")
	source := fs.String("source", "eipc-cli", "source service ID")
	capability := fs.String("cap", "", "capability header")
	fs.Parse(args)

	hmacKey, err := config.LoadHMACKey()
	if err != nil {
		log.Fatalf("HMAC key: %v", err)
	}

	t := tcp.New()
	conn, err := t.Dial(*addr)
	if err != nil {
		log.Fatalf("dial %s: %v", *addr, err)
	}
	defer conn.Close()

	ep := core.NewClientEndpoint(conn, protocol.DefaultCodec(), hmacKey, "")

	mt := core.MessageType(*msgType)
	msg := core.Message{
		Version:    core.ProtocolVersion,
		Type:       mt,
		Source:     *source,
		Timestamp:  time.Now().UTC(),
		RequestID:  fmt.Sprintf("cli-%d", time.Now().UnixNano()),
		Priority:   core.PriorityP1,
		Capability: *capability,
		Payload:    []byte(*payload),
	}

	if err := ep.Send(msg); err != nil {
		log.Fatalf("send: %v", err)
	}

	fmt.Printf("sent type=%s to=%s size=%d bytes\n", *msgType, *addr, len(*payload))

	resp, err := ep.Receive()
	if err != nil {
		fmt.Printf("no response (connection closed or timeout)\n")
		return
	}

	printMessage("response", resp)
}

func cmdListen(args []string) {
	fs := flag.NewFlagSet("listen", flag.ExitOnError)
	addr := fs.String("addr", config.LoadListenAddr(), "server address")
	count := fs.Int("count", 0, "max messages to receive (0=unlimited)")
	fs.Parse(args)

	hmacKey, err := config.LoadHMACKey()
	if err != nil {
		log.Fatalf("HMAC key: %v", err)
	}

	t := tcp.New()
	conn, err := t.Dial(*addr)
	if err != nil {
		log.Fatalf("dial %s: %v", *addr, err)
	}
	defer conn.Close()

	ep := core.NewClientEndpoint(conn, protocol.DefaultCodec(), hmacKey, "")

	fmt.Printf("listening on %s ...\n", *addr)
	received := 0
	for {
		msg, err := ep.Receive()
		if err != nil {
			log.Fatalf("receive: %v", err)
		}
		received++
		printMessage(fmt.Sprintf("msg#%d", received), msg)

		if *count > 0 && received >= *count {
			break
		}
	}
}

func cmdPing(args []string) {
	fs := flag.NewFlagSet("ping", flag.ExitOnError)
	addr := fs.String("addr", config.LoadListenAddr(), "server address")
	fs.Parse(args)

	hmacKey, err := config.LoadHMACKey()
	if err != nil {
		log.Fatalf("HMAC key: %v", err)
	}

	t := tcp.New()
	conn, err := t.Dial(*addr)
	if err != nil {
		log.Fatalf("dial %s: %v", *addr, err)
	}
	defer conn.Close()

	ep := core.NewClientEndpoint(conn, protocol.DefaultCodec(), hmacKey, "")

	start := time.Now()
	msg := core.Message{
		Version:   core.ProtocolVersion,
		Type:      core.TypeHeartbeat,
		Source:    "eipc-cli",
		Timestamp: time.Now().UTC(),
		RequestID: "ping",
		Priority:  core.PriorityP0,
		Payload:   []byte(`{"service":"eipc-cli","status":"ping"}`),
	}

	if err := ep.Send(msg); err != nil {
		log.Fatalf("send ping: %v", err)
	}

	_, err = ep.Receive()
	elapsed := time.Since(start)

	if err != nil {
		fmt.Printf("ping %s: no response (%v)\n", *addr, err)
		return
	}
	fmt.Printf("ping %s: rtt=%v\n", *addr, elapsed)
}

func printMessage(label string, msg core.Message) {
	var payload interface{}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		payload = string(msg.Payload)
	}

	payloadJSON, _ := json.MarshalIndent(payload, "  ", "  ")
	fmt.Printf("[%s] type=%s source=%s req=%s priority=P%d cap=%s\n  payload: %s\n",
		label, msg.Type, msg.Source, msg.RequestID, msg.Priority, msg.Capability, payloadJSON)
}
