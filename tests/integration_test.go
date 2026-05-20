// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package tests

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/embeddedos-org/eipc/core"
	"github.com/embeddedos-org/eipc/protocol"
	"github.com/embeddedos-org/eipc/security/auth"
	"github.com/embeddedos-org/eipc/security/capability"
	"github.com/embeddedos-org/eipc/transport/tcp"
)

const testSecret = "test-secret-key-32-bytes-long!!"

func init() {
	os.Setenv("EIPC_HMAC_KEY", testSecret)
}

func TestChallengeResponseAuth(t *testing.T) {
	authenticator := auth.NewAuthenticator([]byte(testSecret), map[string][]string{
		"test.client": {"ui:control"},
	})

	challenge, err := authenticator.CreateChallenge("test.client")
	if err != nil {
		t.Fatalf("CreateChallenge: %v", err)
	}
	if len(challenge.Nonce) != 32 {
		t.Fatalf("expected 32-byte nonce, got %d", len(challenge.Nonce))
	}

	mac := hmac.New(sha256.New, []byte(testSecret))
	mac.Write(challenge.Nonce)
	response := mac.Sum(nil)

	peer, err := authenticator.VerifyResponse("test.client", response)
	if err != nil {
		t.Fatalf("VerifyResponse: %v", err)
	}
	if peer.ServiceID != "test.client" {
		t.Errorf("expected service_id 'test.client', got %q", peer.ServiceID)
	}
	if len(peer.Capabilities) != 1 || peer.Capabilities[0] != "ui:control" {
		t.Errorf("unexpected capabilities: %v", peer.Capabilities)
	}
	if peer.SessionToken == "" {
		t.Error("expected non-empty session token")
	}
}

func TestChallengeResponseAuth_WrongSecret(t *testing.T) {
	authenticator := auth.NewAuthenticator([]byte(testSecret), map[string][]string{
		"test.client": {"ui:control"},
	})

	challenge, err := authenticator.CreateChallenge("test.client")
	if err != nil {
		t.Fatalf("CreateChallenge: %v", err)
	}

	mac := hmac.New(sha256.New, []byte("wrong-secret-key-definitely-bad!"))
	mac.Write(challenge.Nonce)
	wrongResponse := mac.Sum(nil)

	_, err = authenticator.VerifyResponse("test.client", wrongResponse)
	if err == nil {
		t.Fatal("expected error for wrong secret, got nil")
	}
}

func TestChallengeResponseAuth_UnknownService(t *testing.T) {
	authenticator := auth.NewAuthenticator([]byte(testSecret), map[string][]string{
		"test.client": {"ui:control"},
	})

	_, err := authenticator.CreateChallenge("unknown.service")
	if err == nil {
		t.Fatal("expected error for unknown service")
	}
}

func TestCapabilityEnforcement(t *testing.T) {
	checker := capability.NewChecker(map[string][]string{
		"ui:control": {"ui.cursor.move", "ui.click"},
		"ai:chat":    {"ai.chat.send", "ai.complete.send"},
	})

	if err := checker.Check([]string{"ui:control"}, "ui.cursor.move"); err != nil {
		t.Errorf("expected ui:control to permit ui.cursor.move: %v", err)
	}

	if err := checker.Check([]string{"ai:chat"}, "ai.chat.send"); err != nil {
		t.Errorf("expected ai:chat to permit ai.chat.send: %v", err)
	}

	if err := checker.Check([]string{"ui:control"}, "ai.chat.send"); err == nil {
		t.Error("expected ui:control NOT to permit ai.chat.send")
	}

	if err := checker.Check([]string{"ai:chat"}, "ui.cursor.move"); err == nil {
		t.Error("expected ai:chat NOT to permit ui.cursor.move")
	}
}

func TestSessionTTL(t *testing.T) {
	authenticator := auth.NewAuthenticator([]byte(testSecret), map[string][]string{
		"test.client": {"ui:control"},
	})
	authenticator.SetSessionTTL(100 * time.Millisecond)

	peer, err := authenticator.Authenticate("test.client")
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}

	if peer.IsExpired() {
		t.Error("peer should not be expired immediately")
	}

	time.Sleep(150 * time.Millisecond)

	if !peer.IsExpired() {
		t.Error("peer should be expired after TTL")
	}

	removed := authenticator.CleanupExpired()
	if removed != 1 {
		t.Errorf("expected 1 expired session cleaned, got %d", removed)
	}
}

func TestChatCompleteMessageTypes(t *testing.T) {
	if core.TypeChat != "chat" {
		t.Errorf("expected TypeChat='chat', got %q", core.TypeChat)
	}
	if core.TypeComplete != "complete" {
		t.Errorf("expected TypeComplete='complete', got %q", core.TypeComplete)
	}

	if core.MsgTypeToByte(core.TypeChat) != 'c' {
		t.Errorf("expected chat wire byte 'c', got %c", core.MsgTypeToByte(core.TypeChat))
	}
	if core.MsgTypeToByte(core.TypeComplete) != 'C' {
		t.Errorf("expected complete wire byte 'C', got %c", core.MsgTypeToByte(core.TypeComplete))
	}
}

func TestChatRequestEventSerialization(t *testing.T) {
	codec := protocol.DefaultCodec()
	req := core.ChatRequestEvent{
		SessionID:  "sess-123",
		UserPrompt: "Hello EIPC",
		Model:      "llama3",
		MaxTokens:  512,
	}

	data, err := codec.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed core.ChatRequestEvent
	if err := codec.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if parsed.SessionID != "sess-123" || parsed.UserPrompt != "Hello EIPC" {
		t.Errorf("roundtrip mismatch: %+v", parsed)
	}
}

func TestEbot_EIPC_EAI_ChatFlow(t *testing.T) {
	secret := []byte(testSecret)
	codec := protocol.DefaultCodec()

	authenticator := auth.NewAuthenticator(secret, map[string][]string{
		"ebot.client": {"ai:chat"},
	})
	authenticator.SetSessionTTL(1 * time.Hour)

	tcpTransport := tcp.New()
	if err := tcpTransport.Listen("127.0.0.1:0"); err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer tcpTransport.Close()
	addr := tcpTransport.Addr()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := tcpTransport.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		ep := core.NewServerEndpoint(conn, codec, secret)

		authMsg, _ := ep.Receive()
		var authReq struct {
			ServiceID string `json:"service_id"`
		}
		_ = json.Unmarshal(authMsg.Payload, &authReq)

		challenge, _ := authenticator.CreateChallenge(authReq.ServiceID)
		chalPayload, _ := codec.Marshal(map[string]string{
			"status": "challenge",
			"nonce":  hex.EncodeToString(challenge.Nonce),
		})
		_ = ep.Send(core.Message{Version: 1, Type: core.TypeAck, Source: "server",
			Timestamp: time.Now().UTC(), Payload: chalPayload})

		respMsg, _ := ep.Receive()
		var chalResp struct {
			Response string `json:"response"`
		}
		_ = json.Unmarshal(respMsg.Payload, &chalResp)
		respBytes, _ := hex.DecodeString(chalResp.Response)

		peer, _ := authenticator.VerifyResponse(authReq.ServiceID, respBytes)
		ep.SetPeerCapabilities(peer.Capabilities)

		resultPayload, _ := codec.Marshal(map[string]interface{}{
			"status":        "ok",
			"session_token": peer.SessionToken,
			"capabilities":  peer.Capabilities,
		})
		_ = ep.Send(core.Message{Version: 1, Type: core.TypeAck, Source: "server",
			Timestamp: time.Now().UTC(), Payload: resultPayload})

		chatMsg, _ := ep.Receive()
		var chatReq core.ChatRequestEvent
		_ = codec.Unmarshal(chatMsg.Payload, &chatReq)

		chatResp := core.ChatResponseEvent{
			SessionID:  chatReq.SessionID,
			Response:   "Echo: " + chatReq.UserPrompt,
			TokensUsed: 5,
		}
		respPayload, _ := codec.Marshal(chatResp)
		_ = ep.Send(core.Message{Version: 1, Type: core.TypeChat, Source: "server",
			Timestamp: time.Now().UTC(), Payload: respPayload})
	}()

	clientTransport := tcp.New()
	conn, err := clientTransport.Dial(addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	ep := core.NewClientEndpoint(conn, codec, secret, "")

	authPayload, _ := codec.Marshal(map[string]string{"service_id": "ebot.client"})
	ep.Send(core.Message{Version: 1, Type: core.TypeAck, Source: "ebot.client",
		Timestamp: time.Now().UTC(), RequestID: "auth-1", Payload: authPayload})

	chalMsg, _ := ep.Receive()
	var chalData struct {
		Nonce string `json:"nonce"`
	}
	_ = json.Unmarshal(chalMsg.Payload, &chalData)
	nonceBytes, _ := hex.DecodeString(chalData.Nonce)

	mac := hmac.New(sha256.New, secret)
	mac.Write(nonceBytes)
	response := mac.Sum(nil)

	chalRespPayload, _ := codec.Marshal(map[string]string{
		"service_id": "ebot.client",
		"response":   hex.EncodeToString(response),
	})
	ep.Send(core.Message{Version: 1, Type: core.TypeAck, Source: "ebot.client",
		Timestamp: time.Now().UTC(), RequestID: "auth-2", Payload: chalRespPayload})

	authResult, _ := ep.Receive()
	var result struct {
		Status string `json:"status"`
	}
	json.Unmarshal(authResult.Payload, &result)
	if result.Status != "ok" {
		t.Fatalf("auth failed: %s", result.Status)
	}

	chatReq := core.ChatRequestEvent{
		SessionID:  "ebot-test",
		UserPrompt: "Hello from ebot",
		Model:      "test",
	}
	chatPayload, _ := codec.Marshal(chatReq)
	ep.Send(core.Message{Version: 1, Type: core.TypeChat, Source: "ebot.client",
		Timestamp: time.Now().UTC(), RequestID: "chat-1", Capability: "ai:chat",
		Payload: chatPayload})

	chatResp, err := ep.Receive()
	if err != nil {
		t.Fatalf("receive chat response: %v", err)
	}

	var resp core.ChatResponseEvent
	if err := codec.Unmarshal(chatResp.Payload, &resp); err != nil {
		t.Fatalf("unmarshal chat response: %v", err)
	}

	if resp.Response != "Echo: Hello from ebot" {
		t.Errorf("unexpected response: %q", resp.Response)
	}

	<-serverDone
}
