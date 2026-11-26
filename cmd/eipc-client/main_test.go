// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/embeddedos-org/eipc/core"
	"github.com/embeddedos-org/eipc/protocol"
	"github.com/embeddedos-org/eipc/transport/tcp"
)

// runMainArg is the first argument the test passes to its own binary to ask
// for the client's main() instead of the test runner. Dispatch is on argv,
// not the environment: an inherited variable must never be able to turn a
// normal `go test` run into a client run and silently skip every test.
const runMainArg = "-eipc-client-test-run-main"

// TestMain lets a test run the client's main() in a child process (it
// ends with log.Fatalf on errors, which would end the test binary).
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == runMainArg {
		// Drop the marker so main() sees the address as os.Args[1].
		os.Args = append(os.Args[:1], os.Args[2:]...)
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
	// Generated per run: no key material is committed in the fixture.
	keyBytes := make([]byte, 16)
	if _, err := rand.Read(keyBytes); err != nil {
		t.Fatalf("generate test key: %v", err)
	}
	key := []byte(hex.EncodeToString(keyBytes))

	srv := tcp.New()
	if err := srv.Listen("127.0.0.1:0"); err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer srv.Close()

	// Reports a message the client sent after it was told "denied".
	afterDenial := make(chan core.MessageType, 1)

	go func() {
		defer close(afterDenial)
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
			"error":  "auth: challenge expired for \"nia.min\"\nFAKE log line",
		})

		// A client that honours the denial sends nothing more and closes,
		// so this Receive must end in an error (EOF), not a message.
		if msg, err := ep.Receive(); err == nil {
			afterDenial <- msg.Type
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], runMainArg, srv.Addr())
	cmd.Env = append(os.Environ(),
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
	if !strings.Contains(string(out), `[AUTH] rejected: "auth: challenge expired`) {
		t.Errorf("client did not report the server's rejection")
	}
	// The rejection text is quoted for the log, so the newline the server
	// embedded must not appear as a real line break in the output.
	if strings.Contains(string(out), "\nFAKE log line") {
		t.Errorf("server-supplied error forged a log line; it must be escaped")
	}
	if typ, ok := <-afterDenial; ok {
		t.Errorf("client sent %s after a denied auth response; it must stop", typ)
	}
}
