// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package main

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/embeddedos-org/eipc/core"
	"github.com/embeddedos-org/eipc/protocol"
	"github.com/embeddedos-org/eipc/transport/tcp"
)

// TestMain lets a test run the client's main() in a child process (it
// ends with log.Fatalf on errors, which would end the test binary).
func TestMain(m *testing.M) {
	if os.Getenv("EIPC_CLIENT_TEST_RUN_MAIN") == "1" {
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// The server answers a failed challenge-response with a TypeAuthResponse
// whose status is "denied" (cmd/eipc-server, VerifyResponse error), then
// closes the connection. The client must report the rejection, not a
// successful login.
func TestClientReportsDeniedAuthResponse(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")

	srv := tcp.New()
	if err := srv.Listen("127.0.0.1:0"); err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer srv.Close()

	go func() {
		conn, err := srv.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		codec := protocol.DefaultCodec()
		ep := core.NewServerEndpoint(conn, codec, key)
		send := func(typ core.MessageType, payload map[string]string) {
			b, err := codec.Marshal(payload)
			if err != nil {
				return
			}
			_ = ep.Send(core.Message{
				Version:   core.ProtocolVersion,
				Type:      typ,
				Source:    "eipc-server",
				Timestamp: time.Now().UTC(),
				Payload:   b,
			})
		}
		if _, err := ep.Receive(); err != nil { // auth request
			return
		}
		send(core.TypeChallenge, map[string]string{
			"status": "challenge",
			"nonce":  "00112233445566778899aabbccddeeff",
		})
		if _, err := ep.Receive(); err != nil { // challenge response
			return
		}
		send(core.TypeAuthResponse, map[string]string{
			"status": "denied",
			"error":  `auth: challenge expired for "nia.min"`,
		})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], srv.Addr())
	cmd.Env = append(os.Environ(),
		"EIPC_CLIENT_TEST_RUN_MAIN=1",
		"EIPC_HMAC_KEY="+string(key),
		"EIPC_KEY_FILE=",
		"EIPC_TLS_CA=",
		"EIPC_TLS_CERT=",
		"EIPC_TLS_KEY=",
	)
	out, err := cmd.CombinedOutput()
	t.Logf("client output:\n%s", out)

	if err == nil {
		t.Errorf("client exited 0 after a denied auth response")
	}
	if strings.Contains(string(out), "Authenticated!") {
		t.Errorf("client logged a successful login after a denied auth response")
	}
	if !strings.Contains(string(out), "[AUTH] rejected: auth: challenge expired") {
		t.Errorf("client did not report the server's rejection")
	}
}
