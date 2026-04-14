// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	addr := "127.0.0.1:9090"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}

	sharedSecret, err := config.LoadHMACKey()
	if err != nil {
		log.Fatalf("[CONFIG] %v", err)
	}

	serviceID := "nia.min"
	codec := protocol.DefaultCodec()

	log.Printf("EIPC client connecting to %s as %s", addr, serviceID)

	tcpTransport := tcp.New()
	if err := tcpTransport.SetupTLSFromEnv(); err != nil {
		log.Fatalf("TLS setup: %v", err)
	}

	conn, err := tcpTransport.Dial(addr)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	endpoint := core.NewClientEndpoint(conn, codec, sharedSecret, "")

	// Step 1: Send authentication request
	log.Println("[1] Sending authentication request...")
	type authRequest struct {
		ServiceID string `json:"service_id"`
	}
	authPayload, _ := codec.Marshal(authRequest{ServiceID: serviceID})

	if err := endpoint.Send(core.Message{
		Version:   core.ProtocolVersion,
		Type:      core.TypeAck,
		Source:    serviceID,
		Timestamp: time.Now().UTC(),
		RequestID: "auth-1",
		Payload:   authPayload,
	}); err != nil {
		log.Fatalf("send auth: %v", err)
	}

	// Step 2: Receive challenge (nonce)
	challengeMsg, err := endpoint.Receive()
	if err != nil {
		log.Fatalf("receive challenge: %v", err)
	}

	type challengeResponse struct {
		Status string `json:"status"`
		Nonce  string `json:"nonce"`
		Error  string `json:"error,omitempty"`
	}
	var challenge challengeResponse
	if err := json.Unmarshal(challengeMsg.Payload, &challenge); err != nil {
		log.Fatalf("unmarshal challenge: %v", err)
	}

	if challenge.Status == "denied" {
		log.Fatalf("[AUTH] rejected: %s", challenge.Error)
	}
	if len(challenge.Nonce) >= 16 {
		log.Printf("[2] Received challenge nonce: %s...%s",
			challenge.Nonce[:8], challenge.Nonce[len(challenge.Nonce)-8:])
	} else {
		log.Printf("[2] Received challenge nonce: %s", challenge.Nonce)
	}

	// Step 3: Compute HMAC-SHA256(secret, nonce) and send response
	nonceBytes, err := hex.DecodeString(challenge.Nonce)
	if err != nil {
		log.Fatalf("decode nonce: %v", err)
	}

	mac := hmac.New(sha256.New, sharedSecret)
	mac.Write(nonceBytes)
	response := mac.Sum(nil)

	type authChallengeResponse struct {
		ServiceID string `json:"service_id"`
		Response  string `json:"response"`
	}
	chalRespPayload, _ := codec.Marshal(authChallengeResponse{
		ServiceID: serviceID,
		Response:  hex.EncodeToString(response),
	})

	if err := endpoint.Send(core.Message{
		Version:   core.ProtocolVersion,
		Type:      core.TypeAck,
		Source:    serviceID,
		Timestamp: time.Now().UTC(),
		RequestID: "auth-2",
		Payload:   chalRespPayload,
	}); err != nil {
		log.Fatalf("send challenge response: %v", err)
	}

	// Step 4: Receive session token
	authResp, err := endpoint.Receive()
	if err != nil {
		log.Fatalf("receive auth response: %v", err)
	}

	type authResult struct {
		Status       string   `json:"status"`
		SessionToken string   `json:"session_token"`
		Capabilities []string `json:"capabilities"`
		Error        string   `json:"error,omitempty"`
	}
	var authRes authResult
	if err := json.Unmarshal(authResp.Payload, &authRes); err != nil {
		log.Fatalf("unmarshal auth response: %v", err)
	}

	sessionToken := authRes.SessionToken
	if len(sessionToken) >= 16 {
		log.Printf("[3] Authenticated! token=%s...%s caps=%v",
			sessionToken[:8], sessionToken[len(sessionToken)-8:], authRes.Capabilities)
	} else {
		log.Printf("[3] Authenticated! token=%s caps=%v", sessionToken, authRes.Capabilities)
	}

	// Step 5: Send HMAC-protected intent
	log.Println("[4] Sending intent: move_left (confidence=0.91)")
	intentPayload, _ := codec.Marshal(core.IntentEvent{
		Intent:     "move_left",
		Confidence: 0.91,
		SessionID:  sessionToken,
	})

	if err := endpoint.Send(core.Message{
		Version:    core.ProtocolVersion,
		Type:       core.TypeIntent,
		Source:     serviceID,
		Timestamp:  time.Now().UTC(),
		SessionID:  sessionToken,
		RequestID:  "req-1",
		Priority:   core.PriorityP0,
		Capability: "ui:control",
		Payload:    intentPayload,
	}); err != nil {
		log.Fatalf("send intent: %v", err)
	}

	// Step 6: Receive ack
	ackMsg, err := endpoint.Receive()
	if err != nil {
		log.Fatalf("receive ack: %v", err)
	}

	var ack core.AckEvent
	if err := json.Unmarshal(ackMsg.Payload, &ack); err != nil {
		log.Fatalf("unmarshal ack: %v", err)
	}

	log.Printf("[5] Received ACK: request_id=%s status=%s", ack.RequestID, ack.Status)

	// Step 7: Send heartbeat
	log.Println("[6] Sending heartbeat...")
	hbPayload, _ := codec.Marshal(core.HeartbeatEvent{
		Service: serviceID,
		Status:  "ready",
	})

	if err := endpoint.Send(core.Message{
		Version:   core.ProtocolVersion,
		Type:      core.TypeHeartbeat,
		Source:    serviceID,
		Timestamp: time.Now().UTC(),
		SessionID: sessionToken,
		RequestID: "hb-1",
		Priority:  core.PriorityP2,
		Payload:   hbPayload,
	}); err != nil {
		log.Fatalf("send heartbeat: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	endpoint.Close()
	fmt.Println()
	fmt.Println("=== EIPC Demo Complete (Hardened) ===")
	fmt.Println("End-to-end flow demonstrated:")
	fmt.Println("  1. Client connected to server (TLS if configured)")
	fmt.Println("  2. Server sent challenge nonce")
	fmt.Println("  3. Client proved secret via HMAC-SHA256 response")
	fmt.Println("  4. Server validated & issued session token")
	fmt.Println("  5. Client sent HMAC-protected intents")
	fmt.Println("  6. Server enforced capability (ui:control)")
	fmt.Println("  7. Audit events recorded")
	fmt.Println("  8. Acks returned to client")
}
