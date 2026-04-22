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
	"os/signal"
	"time"

	"github.com/embeddedos-org/eipc/config"
	"github.com/embeddedos-org/eipc/core"
	"github.com/embeddedos-org/eipc/protocol"
	"github.com/embeddedos-org/eipc/security/auth"
	"github.com/embeddedos-org/eipc/security/capability"
	"github.com/embeddedos-org/eipc/services/audit"
	"github.com/embeddedos-org/eipc/services/health"
	"github.com/embeddedos-org/eipc/services/registry"
	"github.com/embeddedos-org/eipc/transport"
	"github.com/embeddedos-org/eipc/transport/tcp"
)

func main() {
	addr := config.LoadListenAddr()
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}

	sharedSecret, err := config.LoadHMACKey()
	if err != nil {
		log.Fatalf("[CONFIG] %v", err)
	}

	sessionTTL := config.LoadSessionTTL()
	maxConns := config.LoadMaxConnections()

	authenticator := auth.NewAuthenticator(sharedSecret, map[string][]string{
		"nia.min":       {"ui:control", "device:read"},
		"nia.framework": {"ui:control", "device:read", "device:write"},
		"ail.min.agent": {"ui:control"},
		"ail.framework": {"ui:control", "device:read", "device:write", "system:restricted"},
		"ebot.client":   {"ai:chat"},
	})
	authenticator.SetSessionTTL(sessionTTL)

	capChecker := capability.NewChecker(map[string][]string{
		"ui:control":        {"ui.cursor.move", "ui.click", "ui.scroll"},
		"device:read":       {"device.sensor.read", "device.status"},
		"device:write":      {"device.actuator.write"},
		"system:restricted": {"system.reboot", "system.update"},
		"ai:chat":           {"ai.chat.send", "ai.complete.send"},
	})

	auditLogger, err := audit.NewFileLogger("")
	if err != nil {
		log.Fatalf("audit logger: %v", err)
	}
	defer auditLogger.Close()

	healthSvc := health.NewService(5*time.Second, 15*time.Second)

	reg := registry.NewRegistry()
	if err := reg.Register(registry.ServiceInfo{
		ServiceID:    "eipc-server",
		Capabilities: []string{"ui:control", "device:read", "device:write", "ai:chat"},
		Versions:     []uint16{1},
		MessageTypes: []core.MessageType{
			core.TypeIntent, core.TypeAck, core.TypeHeartbeat, core.TypeAudit,
			core.TypeChat, core.TypeComplete, core.TypeAuth, core.TypeChallenge, core.TypeAuthResponse,
		},
		Priority: core.PriorityP0,
	}); err != nil {
		log.Printf("[REGISTRY] failed to register eipc-server: %v", err)
	}
	if err := reg.Register(registry.ServiceInfo{
		ServiceID:    "ebot.client",
		Capabilities: []string{"ai:chat"},
		Versions:     []uint16{1},
		MessageTypes: []core.MessageType{core.TypeChat, core.TypeComplete, core.TypeAck, core.TypeAuth, core.TypeAuthResponse},
		Priority:     core.PriorityP1,
	}); err != nil {
		log.Printf("[REGISTRY] failed to register ebot.client: %v", err)
	}

	router := core.NewRouter()

	router.Handle(core.TypeIntent, func(msg core.Message) (*core.Message, error) {
		var intent core.IntentEvent
		codec := protocol.DefaultCodec()
		if err := codec.Unmarshal(msg.Payload, &intent); err != nil {
			return nil, fmt.Errorf("unmarshal intent: %w", err)
		}

		log.Printf("[INTENT] from=%s intent=%s confidence=%.2f session=%s",
			msg.Source, intent.Intent, intent.Confidence, intent.SessionID)

		if err := capChecker.Check([]string{msg.Capability}, "ui.cursor.move"); err != nil {
			log.Printf("[POLICY] DENIED: %v", err)
			if err := auditLogger.Log(audit.Entry{
				RequestID: msg.RequestID,
				Source:    msg.Source,
				Target:    "eipc-server",
				Action:    intent.Intent,
				Decision:  "denied",
				Result:    err.Error(),
			}); err != nil {
				log.Printf("[AUDIT] failed: %v", err)
			}
			return nil, err
		}

		log.Printf("[POLICY] ALLOWED: capability=%s action=%s", msg.Capability, intent.Intent)

		if err := auditLogger.Log(audit.Entry{
			RequestID: msg.RequestID,
			Source:    msg.Source,
			Target:    "eipc-server",
			Action:    intent.Intent,
			Decision:  "allowed",
			Result:    "success",
		}); err != nil {
			log.Printf("[AUDIT] failed: %v", err)
		}

		ackPayload, err := codec.Marshal(core.AckEvent{
			RequestID: msg.RequestID,
			Status:    "ok",
		})
		if err != nil {
			return nil, fmt.Errorf("marshal ack: %w", err)
		}

		ack := core.Message{
			Version:   core.ProtocolVersion,
			Type:      core.TypeAck,
			Source:    "eipc-server",
			Timestamp: time.Now().UTC(),
			SessionID: msg.SessionID,
			RequestID: msg.RequestID,
			Priority:  core.PriorityP0,
			Payload:   ackPayload,
		}
		return &ack, nil
	})

	router.Handle(core.TypeHeartbeat, func(msg core.Message) (*core.Message, error) {
		var hb core.HeartbeatEvent
		codec := protocol.DefaultCodec()
		if err := codec.Unmarshal(msg.Payload, &hb); err != nil {
			return nil, err
		}
		healthSvc.RecordHeartbeat(hb.Service, hb.Status)
		log.Printf("[HEARTBEAT] service=%s status=%s", hb.Service, hb.Status)
		return nil, nil
	})

	router.Handle(core.TypeChat, func(msg core.Message) (*core.Message, error) {
		var chatReq core.ChatRequestEvent
		codec := protocol.DefaultCodec()
		if err := codec.Unmarshal(msg.Payload, &chatReq); err != nil {
			return nil, fmt.Errorf("unmarshal chat request: %w", err)
		}

		if err := capChecker.Check([]string{msg.Capability}, "ai.chat.send"); err != nil {
			log.Printf("[POLICY] DENIED chat: %v", err)
			if err := auditLogger.Log(audit.Entry{
				RequestID: msg.RequestID,
				Source:    msg.Source,
				Target:    "eipc-server",
				Action:    "ai.chat.send",
				Decision:  "denied",
				Result:    err.Error(),
			}); err != nil {
				log.Printf("[AUDIT] failed: %v", err)
			}
			return nil, err
		}

		log.Printf("[CHAT] from=%s session=%s prompt=%q",
			msg.Source, chatReq.SessionID, chatReq.UserPrompt)

		if err := auditLogger.Log(audit.Entry{
			RequestID: msg.RequestID,
			Source:    msg.Source,
			Target:    "eai",
			Action:    "ai.chat.send",
			Decision:  "allowed",
			Result:    "forwarded",
		}); err != nil {
			log.Printf("[AUDIT] failed: %v", err)
		}

		// TODO: Forward to EAI agent loop. For now, echo acknowledgment.
		chatResp := core.ChatResponseEvent{
			SessionID:  chatReq.SessionID,
			Response:   fmt.Sprintf("[EIPC] Chat received: %s", chatReq.UserPrompt),
			Model:      chatReq.Model,
			TokensUsed: 0,
		}
		respPayload, err := codec.Marshal(chatResp)
		if err != nil {
			return nil, fmt.Errorf("marshal chat response: %w", err)
		}

		return &core.Message{
			Version:   core.ProtocolVersion,
			Type:      core.TypeChat,
			Source:    "eipc-server",
			Timestamp: time.Now().UTC(),
			SessionID: msg.SessionID,
			RequestID: msg.RequestID,
			Priority:  core.PriorityP1,
			Payload:   respPayload,
		}, nil
	})

	router.Handle(core.TypeComplete, func(msg core.Message) (*core.Message, error) {
		var completeReq core.CompleteRequestEvent
		codec := protocol.DefaultCodec()
		if err := codec.Unmarshal(msg.Payload, &completeReq); err != nil {
			return nil, fmt.Errorf("unmarshal complete request: %w", err)
		}

		if err := capChecker.Check([]string{msg.Capability}, "ai.complete.send"); err != nil {
			log.Printf("[POLICY] DENIED complete: %v", err)
			if err := auditLogger.Log(audit.Entry{
				RequestID: msg.RequestID,
				Source:    msg.Source,
				Target:    "eipc-server",
				Action:    "ai.complete.send",
				Decision:  "denied",
				Result:    err.Error(),
			}); err != nil {
				log.Printf("[AUDIT] failed: %v", err)
			}
			return nil, err
		}

		log.Printf("[COMPLETE] from=%s session=%s prompt=%q",
			msg.Source, completeReq.SessionID, completeReq.Prompt)

		if err := auditLogger.Log(audit.Entry{
			RequestID: msg.RequestID,
			Source:    msg.Source,
			Target:    "eai",
			Action:    "ai.complete.send",
			Decision:  "allowed",
			Result:    "forwarded",
		}); err != nil {
			log.Printf("[AUDIT] failed: %v", err)
		}

		completeResp := core.CompleteResponseEvent{
			SessionID:  completeReq.SessionID,
			Completion: fmt.Sprintf("[EIPC] Completion received: %s", completeReq.Prompt),
			Model:      completeReq.Model,
			TokensUsed: 0,
		}
		respPayload, err := codec.Marshal(completeResp)
		if err != nil {
			return nil, fmt.Errorf("marshal complete response: %w", err)
		}

		return &core.Message{
			Version:   core.ProtocolVersion,
			Type:      core.TypeComplete,
			Source:    "eipc-server",
			Timestamp: time.Now().UTC(),
			SessionID: msg.SessionID,
			RequestID: msg.RequestID,
			Priority:  core.PriorityP1,
			Payload:   respPayload,
		}, nil
	})

	tcpTransport := tcp.New()
	if err := tcpTransport.SetupTLSFromEnv(); err != nil {
		log.Fatalf("TLS setup: %v", err)
	}
	if err := tcpTransport.Listen(addr); err != nil {
		log.Fatalf("listen: %v", err)
	}
	defer tcpTransport.Close()

	tlsMode := "plaintext"
	if config.TLSEnabled() {
		tlsMode = "TLS"
	}
	log.Printf("EIPC server listening on %s [%s] (max_conns=%d, session_ttl=%s)",
		tcpTransport.Addr(), tlsMode, maxConns, sessionTTL)

	// Connection limit semaphore
	connSem := make(chan struct{}, maxConns)

	// Background session cleanup goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			removed := authenticator.CleanupExpired()
			if removed > 0 {
				log.Printf("[SESSION] cleaned up %d expired sessions", removed)
			}
		}
	}()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)
		<-sigCh
		log.Println("Shutting down...")
		tcpTransport.Close()
		os.Exit(0)
	}()

	codec := protocol.DefaultCodec()

	for {
		conn, err := tcpTransport.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			return
		}

		select {
		case connSem <- struct{}{}:
			go func() {
				defer func() { <-connSem }()
				handleConnection(conn, authenticator, codec, sharedSecret, router, auditLogger, capChecker)
			}()
		default:
			log.Printf("[CONN] rejected connection from %s: max connections (%d) reached", conn.RemoteAddr(), maxConns)
			if err := auditLogger.Log(audit.Entry{
				Source:   conn.RemoteAddr(),
				Target:   "eipc-server",
				Action:   "connect",
				Decision: "denied",
				Result:   "connection limit exceeded",
			}); err != nil {
				log.Printf("[AUDIT] failed: %v", err)
			}
			conn.Close()
		}
	}
}

func handleConnection(
	conn transport.Connection,
	authenticator *auth.Authenticator,
	codec protocol.Codec,
	hmacKey []byte,
	router *core.Router,
	auditLogger audit.Logger,
	capChecker *capability.Checker,
) {
	defer conn.Close()
	log.Printf("[CONN] new connection from %s", conn.RemoteAddr())

	endpoint := core.NewServerEndpoint(conn, codec, hmacKey)

	// Auth timeout: 10s
	authDone := make(chan struct{})
	go func() {
		select {
		case <-authDone:
		case <-time.After(10 * time.Second):
			log.Printf("[AUTH] timeout waiting for auth from %s", conn.RemoteAddr())
			conn.Close()
		}
	}()

	// Step 1: Receive auth request
	authMsg, err := endpoint.Receive()
	if err != nil {
		log.Printf("[AUTH] failed to receive auth message: %v", err)
		close(authDone)
		return
	}
	if authMsg.Type != core.TypeAuth {
		log.Printf("[AUTH] expected TypeAuth, got %s", authMsg.Type)
		close(authDone)
		return
	}

	type authRequest struct {
		ServiceID string `json:"service_id"`
	}
	var authReq authRequest
	if err := json.Unmarshal(authMsg.Payload, &authReq); err != nil {
		log.Printf("[AUTH] bad auth payload: %v", err)
		close(authDone)
		return
	}

	// Step 2: Create challenge (send nonce)
	challenge, err := authenticator.CreateChallenge(authReq.ServiceID)
	if err != nil {
		log.Printf("[AUTH] REJECTED: %v", err)
		if err := auditLogger.Log(audit.Entry{
			RequestID: authMsg.RequestID,
			Source:    authReq.ServiceID,
			Target:    "eipc-server",
			Action:    "authenticate",
			Decision:  "denied",
			Result:    err.Error(),
		}); err != nil {
			log.Printf("[AUDIT] failed: %v", err)
		}
		type authResponse struct {
			Status string `json:"status"`
			Error  string `json:"error,omitempty"`
		}
		respPayload, err := codec.Marshal(authResponse{Status: "denied", Error: err.Error()})
		if err != nil {
			log.Printf("[AUTH] failed to marshal auth response: %v", err)
		} else {
			if err := endpoint.Send(core.Message{
				Version:   core.ProtocolVersion,
				Type:      core.TypeAuthResponse,
				Source:    "eipc-server",
				Timestamp: time.Now().UTC(),
				RequestID: authMsg.RequestID,
				Payload:   respPayload,
			}); err != nil {
				log.Printf("[AUTH] failed to send auth response: %v", err)
			}
		}
		close(authDone)
		return
	}

	type challengeMessage struct {
		Status string `json:"status"`
		Nonce  string `json:"nonce"`
	}
	challengePayload, err := codec.Marshal(challengeMessage{
		Status: "challenge",
		Nonce:  hex.EncodeToString(challenge.Nonce),
	})
	if err != nil {
		log.Printf("[AUTH] failed to marshal challenge: %v", err)
		close(authDone)
		return
	}
	if err := endpoint.Send(core.Message{
		Version:   core.ProtocolVersion,
		Type:      core.TypeChallenge,
		Source:    "eipc-server",
		Timestamp: time.Now().UTC(),
		RequestID: authMsg.RequestID,
		Payload:   challengePayload,
	}); err != nil {
		log.Printf("[AUTH] failed to send challenge: %v", err)
		close(authDone)
		return
	}

	// Step 3: Receive HMAC response
	responseMsg, err := endpoint.Receive()
	if err != nil {
		log.Printf("[AUTH] failed to receive challenge response: %v", err)
		close(authDone)
		return
	}
	if responseMsg.Type != core.TypeAuthResponse {
		log.Printf("[AUTH] expected TypeAuthResponse, got %s", responseMsg.Type)
		close(authDone)
		return
	}

	type challengeResponse struct {
		ServiceID string `json:"service_id"`
		Response  string `json:"response"`
	}
	var chalResp challengeResponse
	if err := json.Unmarshal(responseMsg.Payload, &chalResp); err != nil {
		log.Printf("[AUTH] bad challenge response: %v", err)
		close(authDone)
		return
	}

	responseBytes, err := hex.DecodeString(chalResp.Response)
	if err != nil {
		log.Printf("[AUTH] bad response encoding: %v", err)
		close(authDone)
		return
	}

	// Step 4: Verify response
	peer, err := authenticator.VerifyResponse(authReq.ServiceID, responseBytes)
	if err != nil {
		log.Printf("[AUTH] REJECTED (challenge-response): %v", err)
		if err := auditLogger.Log(audit.Entry{
			RequestID: authMsg.RequestID,
			Source:    authReq.ServiceID,
			Target:    "eipc-server",
			Action:    "authenticate",
			Decision:  "denied",
			Result:    "challenge-response failed",
		}); err != nil {
			log.Printf("[AUDIT] failed: %v", err)
		}
		type authResponse struct {
			Status string `json:"status"`
			Error  string `json:"error,omitempty"`
		}
		respPayload, err := codec.Marshal(authResponse{Status: "denied", Error: err.Error()})
		if err != nil {
			log.Printf("[AUTH] failed to marshal auth response: %v", err)
		} else {
			if err := endpoint.Send(core.Message{
				Version:   core.ProtocolVersion,
				Type:      core.TypeAuthResponse,
				Source:    "eipc-server",
				Timestamp: time.Now().UTC(),
				RequestID: authMsg.RequestID,
				Payload:   respPayload,
			}); err != nil {
				log.Printf("[AUTH] failed to send auth response: %v", err)
			}
		}
		close(authDone)
		return
	}

	close(authDone) // Auth completed successfully

	log.Printf("[AUTH] ACCEPTED: service=%s token=%s...%s caps=%v",
		peer.ServiceID, peer.SessionToken[:8], peer.SessionToken[len(peer.SessionToken)-8:], peer.Capabilities)

	if err := auditLogger.Log(audit.Entry{
		RequestID: authMsg.RequestID,
		Source:    peer.ServiceID,
		Target:    "eipc-server",
		Action:    "authenticate",
		Decision:  "allowed",
		Result:    "session created",
	}); err != nil {
		log.Printf("[AUDIT] failed: %v", err)
	}

	// Set peer capabilities on the endpoint for validation
	endpoint.SetPeerCapabilities(peer.Capabilities)

	type authResult struct {
		Status       string   `json:"status"`
		SessionToken string   `json:"session_token"`
		Capabilities []string `json:"capabilities"`
	}
	respPayload, err := codec.Marshal(authResult{
		Status:       "ok",
		SessionToken: peer.SessionToken,
		Capabilities: peer.Capabilities,
	})
	if err != nil {
		log.Printf("[AUTH] failed to marshal auth result: %v", err)
		return
	}
	if err := endpoint.Send(core.Message{
		Version:   core.ProtocolVersion,
		Type:      core.TypeAuthResponse,
		Source:    "eipc-server",
		Timestamp: time.Now().UTC(),
		RequestID: authMsg.RequestID,
		Payload:   respPayload,
	}); err != nil {
		log.Printf("[AUTH] failed to send auth response: %v", err)
		return
	}

	// Message loop with idle timeout and capability enforcement
	for {
		msg, err := endpoint.Receive()
		if err != nil {
			log.Printf("[CONN] connection closed: %v", err)
			return
		}

		// Check session TTL
		if peer.IsExpired() {
			log.Printf("[SESSION] expired for %s", peer.ServiceID)
			if err := auditLogger.Log(audit.Entry{
				Source:   peer.ServiceID,
				Target:   "eipc-server",
				Action:   "session_check",
				Decision: "denied",
				Result:   "session expired",
			}); err != nil {
				log.Printf("[AUDIT] failed: %v", err)
			}
			return
		}

		// Enforce capability binding
		if err := endpoint.ValidateCapability(msg.Capability); err != nil {
			log.Printf("[CAPABILITY] DENIED: %s tried %s", peer.ServiceID, msg.Capability)
			if err := auditLogger.Log(audit.Entry{
				RequestID: msg.RequestID,
				Source:    peer.ServiceID,
				Target:    "eipc-server",
				Action:    msg.Capability,
				Decision:  "denied",
				Result:    "capability violation",
			}); err != nil {
				log.Printf("[AUDIT] failed: %v", err)
			}
			errPayload, err := codec.Marshal(core.AckEvent{
				RequestID: msg.RequestID,
				Status:    "error",
				Error:     err.Error(),
			})
			if err != nil {
				log.Printf("[CAPABILITY] failed to marshal error: %v", err)
			} else {
				if err := endpoint.Send(core.Message{
					Version:   core.ProtocolVersion,
					Type:      core.TypeAck,
					Source:    "eipc-server",
					Timestamp: time.Now().UTC(),
					SessionID: msg.SessionID,
					RequestID: msg.RequestID,
					Priority:  core.PriorityP0,
					Payload:   errPayload,
				}); err != nil {
					log.Printf("[CAPABILITY] failed to send error: %v", err)
				}
			}
			continue
		}

		resp, err := router.Dispatch(msg)
		if err != nil {
			log.Printf("[DISPATCH] error: %v", err)
			errPayload, err := codec.Marshal(core.AckEvent{
				RequestID: msg.RequestID,
				Status:    "error",
				Error:     err.Error(),
			})
			if err != nil {
				log.Printf("[DISPATCH] failed to marshal error: %v", err)
			} else {
				if err := endpoint.Send(core.Message{
					Version:   core.ProtocolVersion,
					Type:      core.TypeAck,
					Source:    "eipc-server",
					Timestamp: time.Now().UTC(),
					SessionID: msg.SessionID,
					RequestID: msg.RequestID,
					Priority:  core.PriorityP0,
					Payload:   errPayload,
				}); err != nil {
					log.Printf("[DISPATCH] failed to send error: %v", err)
				}
			}
			continue
		}

		if resp != nil {
			if err := endpoint.Send(*resp); err != nil {
				log.Printf("[SEND] error: %v", err)
				return
			}
		}
	}
}

// computeChallengeResponse computes HMAC-SHA256(secret, nonce) for client-side auth.
