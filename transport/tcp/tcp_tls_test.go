// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package tcp

import (
	"crypto/tls"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/embeddedos-org/eipc/core"
	"github.com/embeddedos-org/eipc/protocol"
)

// clearTLSEnv pins all four TLS env vars to the given values for the
// duration of a test, so the host environment cannot leak into the
// env-gate assertions.
func clearTLSEnv(t *testing.T, cert, key, ca, auto string) {
	t.Helper()
	t.Setenv("EIPC_TLS_CERT", cert)
	t.Setenv("EIPC_TLS_KEY", key)
	t.Setenv("EIPC_TLS_CA", ca)
	t.Setenv("EIPC_TLS_AUTO_CERT", auto)
}

func TestSetupTLSFromEnvNoEnv(t *testing.T) {
	clearTLSEnv(t, "", "", "", "")

	tr := New()
	if err := tr.SetupTLSFromEnv(); err != nil {
		t.Fatalf("SetupTLSFromEnv: %v", err)
	}
	if tr.tlsConfig != nil {
		t.Errorf("tlsConfig: got non-nil, want nil with no env set")
	}
}

func TestSetupTLSFromEnvAutoCert(t *testing.T) {
	clearTLSEnv(t, "", "", "", "true")

	tr := New()
	if err := tr.SetupTLSFromEnv(); err != nil {
		t.Fatalf("SetupTLSFromEnv: %v", err)
	}
	if tr.tlsConfig == nil {
		t.Fatal("tlsConfig: got nil, want a config with EIPC_TLS_AUTO_CERT=true")
	}
	if len(tr.tlsConfig.Certificates) != 1 {
		t.Errorf("certificates: got %d, want 1", len(tr.tlsConfig.Certificates))
	}
	if tr.tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion: got %#x, want TLS 1.2 (%#x)", tr.tlsConfig.MinVersion, tls.VersionTLS12)
	}
}

func TestSetupTLSFromEnvAutoCertRequiresExactTrue(t *testing.T) {
	// The gate is an exact "true" comparison: other truthy spellings
	// must not engage the auto-cert path.
	clearTLSEnv(t, "", "", "", "1")

	tr := New()
	if err := tr.SetupTLSFromEnv(); err != nil {
		t.Fatalf("SetupTLSFromEnv: %v", err)
	}
	if tr.tlsConfig != nil {
		t.Errorf("tlsConfig: got non-nil, want nil for EIPC_TLS_AUTO_CERT=1")
	}
}

func TestSetupTLSFromEnvCertKey(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := writeCertKeyFiles(t, dir)
	clearTLSEnv(t, certFile, keyFile, "", "")

	tr := New()
	if err := tr.SetupTLSFromEnv(); err != nil {
		t.Fatalf("SetupTLSFromEnv: %v", err)
	}
	if tr.tlsConfig == nil {
		t.Fatal("tlsConfig: got nil, want a config from cert/key env paths")
	}
	if len(tr.tlsConfig.Certificates) != 1 {
		t.Errorf("certificates: got %d, want 1", len(tr.tlsConfig.Certificates))
	}
	if tr.tlsConfig.ClientAuth != tls.NoClientCert {
		t.Errorf("ClientAuth: got %v, want NoClientCert without a CA file", tr.tlsConfig.ClientAuth)
	}
}

func TestSetupTLSFromEnvCertKeyWithCA(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := writeCertKeyFiles(t, dir)
	// The generated cert's own PEM parses as a CA file for the pool.
	clearTLSEnv(t, certFile, keyFile, certFile, "")

	tr := New()
	if err := tr.SetupTLSFromEnv(); err != nil {
		t.Fatalf("SetupTLSFromEnv: %v", err)
	}
	if tr.tlsConfig == nil {
		t.Fatal("tlsConfig: got nil, want a config from cert/key/CA env paths")
	}
	if tr.tlsConfig.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Errorf("ClientAuth: got %v, want RequireAndVerifyClientCert with a CA file", tr.tlsConfig.ClientAuth)
	}
	if tr.tlsConfig.ClientCAs == nil {
		t.Errorf("ClientCAs: got nil, want a pool with a CA file")
	}
}

func TestSetupTLSFromEnvMissingFiles(t *testing.T) {
	dir := t.TempDir()
	clearTLSEnv(t,
		filepath.Join(dir, "missing-cert.pem"),
		filepath.Join(dir, "missing-key.pem"),
		"", "")

	tr := New()
	if err := tr.SetupTLSFromEnv(); err == nil {
		t.Fatal("SetupTLSFromEnv: got nil error, want an error for missing cert/key files")
	}
	if tr.tlsConfig != nil {
		t.Errorf("tlsConfig: got non-nil after a failed load, want nil")
	}
}

func TestSetupTLSFromEnvCertWithoutKey(t *testing.T) {
	// A lone cert path is half-configured TLS: the server must fail
	// loudly at startup rather than silently listen on plaintext while
	// the operator believes TLS is on.
	dir := t.TempDir()
	certFile, _ := writeCertKeyFiles(t, dir)
	clearTLSEnv(t, certFile, "", "", "")

	tr := New()
	if err := tr.SetupTLSFromEnv(); err == nil {
		t.Fatal("SetupTLSFromEnv: got nil error, want an error for a cert path without a key path")
	}
	if tr.tlsConfig != nil {
		t.Errorf("tlsConfig: got non-nil after a failed load, want nil")
	}
}

func TestSetupTLSFromEnvKeyWithoutCert(t *testing.T) {
	// Symmetric with the cert-without-key case: a lone key path is
	// half-configured TLS and must fail, not fall back to plaintext.
	dir := t.TempDir()
	_, keyFile := writeCertKeyFiles(t, dir)
	clearTLSEnv(t, "", keyFile, "", "")

	tr := New()
	if err := tr.SetupTLSFromEnv(); err == nil {
		t.Fatal("SetupTLSFromEnv: got nil error, want an error for a key path without a cert path")
	}
	if tr.tlsConfig != nil {
		t.Errorf("tlsConfig: got non-nil after a failed load, want nil")
	}
}

func TestTransportTLSEnabled(t *testing.T) {
	clearTLSEnv(t, "", "", "", "")

	tr := New()
	if tr.TLSEnabled() {
		t.Error("TLSEnabled: got true on a fresh transport, want false")
	}

	cfg, err := AutoTLSConfig()
	if err != nil {
		t.Fatalf("AutoTLSConfig: %v", err)
	}
	tr.WithTLSConfig(cfg)
	if !tr.TLSEnabled() {
		t.Error("TLSEnabled: got false after WithTLSConfig, want true")
	}
}

func TestSetupClientTLSFromEnvNoEnv(t *testing.T) {
	clearTLSEnv(t, "", "", "", "")

	tr := New()
	if err := tr.SetupClientTLSFromEnv(); err != nil {
		t.Fatalf("SetupClientTLSFromEnv: %v", err)
	}
	if tr.tlsConfig != nil {
		t.Errorf("tlsConfig: got non-nil, want nil with no client TLS env set")
	}
	if tr.TLSEnabled() {
		t.Error("TLSEnabled: got true with no client TLS env set, want false")
	}
}

func TestSetupClientTLSFromEnvCA(t *testing.T) {
	dir := t.TempDir()
	certFile, _ := writeCertKeyFiles(t, dir)
	clearTLSEnv(t, "", "", certFile, "")

	tr := New()
	if err := tr.SetupClientTLSFromEnv(); err != nil {
		t.Fatalf("SetupClientTLSFromEnv: %v", err)
	}
	if tr.tlsConfig == nil {
		t.Fatal("tlsConfig: got nil, want a config with EIPC_TLS_CA set")
	}
	if tr.tlsConfig.RootCAs == nil {
		t.Error("RootCAs: got nil, want a pool from EIPC_TLS_CA")
	}
	if tr.tlsConfig.InsecureSkipVerify {
		t.Error("InsecureSkipVerify: got true, want false — verification is always on")
	}
	if !tr.TLSEnabled() {
		t.Error("TLSEnabled: got false with EIPC_TLS_CA set, want true")
	}
}

func TestSetupClientTLSFromEnvWithClientCert(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := writeCertKeyFiles(t, dir)
	clearTLSEnv(t, certFile, keyFile, certFile, "")

	tr := New()
	if err := tr.SetupClientTLSFromEnv(); err != nil {
		t.Fatalf("SetupClientTLSFromEnv: %v", err)
	}
	if tr.tlsConfig == nil {
		t.Fatal("tlsConfig: got nil, want a config with CA + client cert/key set")
	}
	if len(tr.tlsConfig.Certificates) != 1 {
		t.Errorf("Certificates: got %d, want 1 for mTLS client cert", len(tr.tlsConfig.Certificates))
	}
}

func TestSetupClientTLSFromEnvHalfClientCert(t *testing.T) {
	// A client cert without its key is half-configured, same as the
	// server side: fail instead of dialing with a silently dropped
	// client identity.
	dir := t.TempDir()
	certFile, _ := writeCertKeyFiles(t, dir)
	clearTLSEnv(t, certFile, "", "", "")

	tr := New()
	if err := tr.SetupClientTLSFromEnv(); err == nil {
		t.Fatal("SetupClientTLSFromEnv: got nil error, want an error for a client cert without a key")
	}
	if tr.tlsConfig != nil {
		t.Errorf("tlsConfig: got non-nil after a failed load, want nil")
	}
}

func TestSetupClientTLSFromEnvBadCA(t *testing.T) {
	dir := t.TempDir()
	badCA := filepath.Join(dir, "bad-ca.pem")
	if err := os.WriteFile(badCA, []byte("this is not a PEM certificate\n"), 0o600); err != nil {
		t.Fatalf("write bad CA: %v", err)
	}
	clearTLSEnv(t, "", "", badCA, "")

	tr := New()
	if err := tr.SetupClientTLSFromEnv(); err == nil {
		t.Fatal("SetupClientTLSFromEnv: got nil error, want an error for an unparseable CA file")
	}
}

// TestTLSRoundTrip drives a real TLS listener/dial pair and exchanges
// protocol frames in both directions, mirroring TestTCPBidirectional
// over TLS. The generated server certificate is self-signed with SAN
// "localhost", so the client verifies it by trusting that exact
// certificate as its RootCA — a real verified dial, with no
// InsecureSkipVerify anywhere in the test.
func TestTLSRoundTrip(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := writeCertKeyFiles(t, dir)

	serverCfg, err := LoadTLSConfig(certFile, keyFile, "")
	if err != nil {
		t.Fatalf("LoadTLSConfig: %v", err)
	}

	srv := New()
	srv.WithTLSConfig(serverCfg)
	if err := srv.Listen("127.0.0.1:0"); err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer srv.Close()

	clientCfg, err := LoadClientTLSConfig(certFile, "", "")
	if err != nil {
		t.Fatalf("LoadClientTLSConfig: %v", err)
	}
	if clientCfg.InsecureSkipVerify {
		t.Fatal("InsecureSkipVerify: got true, want false — the verified dial must not skip verification")
	}

	client := New()
	client.WithTLSConfig(clientCfg)

	// Dial by name so the "localhost" SAN on the generated certificate
	// matches the verified ServerName.
	_, port, err := net.SplitHostPort(srv.Addr())
	if err != nil {
		t.Fatalf("split host/port %q: %v", srv.Addr(), err)
	}
	addr := net.JoinHostPort("localhost", port)

	request := &protocol.Frame{
		Version: core.ProtocolVersion,
		MsgType: uint8('i'),
		Header:  []byte(`{"service_id":"client"}`),
		Payload: []byte(`{"intent":"ping"}`),
	}
	response := &protocol.Frame{
		Version: core.ProtocolVersion,
		MsgType: uint8('a'),
		Header:  []byte(`{"service_id":"server"}`),
		Payload: []byte(`{"status":"ok"}`),
	}

	errCh := make(chan error, 1)
	responseCh := make(chan *protocol.Frame, 1)

	go func() {
		srvConn, err := srv.Accept()
		if err != nil {
			errCh <- err
			return
		}
		defer srvConn.Close()

		if _, err := srvConn.Receive(); err != nil {
			errCh <- err
			return
		}
		if err := srvConn.Send(response); err != nil {
			errCh <- err
			return
		}
	}()

	clientConn, err := client.Dial(addr)
	if err != nil {
		t.Fatalf("tls dial: %v", err)
	}
	defer clientConn.Close()

	if err := clientConn.Send(request); err != nil {
		t.Fatalf("send: %v", err)
	}

	go func() {
		resp, err := clientConn.Receive()
		if err != nil {
			errCh <- err
			return
		}
		responseCh <- resp
	}()

	select {
	case err := <-errCh:
		t.Fatalf("error: %v", err)
	case resp := <-responseCh:
		if resp.MsgType != response.MsgType {
			t.Errorf("response type: got %d, want %d", resp.MsgType, response.MsgType)
		}
		if string(resp.Payload) != string(response.Payload) {
			t.Errorf("response payload: got %s, want %s", resp.Payload, response.Payload)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for TLS response")
	}
}

// TestTLSRoundTripWrongCARejected dials a TLS server while trusting a
// different certificate as the CA: the handshake must fail on
// verification, proving the verified dial in TestTLSRoundTrip is really
// verifying and not silently accepting any certificate.
func TestTLSRoundTripWrongCARejected(t *testing.T) {
	serverDir := t.TempDir()
	serverCert, serverKey := writeCertKeyFiles(t, serverDir)
	serverCfg, err := LoadTLSConfig(serverCert, serverKey, "")
	if err != nil {
		t.Fatalf("LoadTLSConfig: %v", err)
	}

	srv := New()
	srv.WithTLSConfig(serverCfg)
	if err := srv.Listen("127.0.0.1:0"); err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer srv.Close()

	// The server must Accept AND read so the handshake runs:
	// tls.Listener.Accept does not handshake — it completes on first
	// I/O. Without a Read the client never receives the certificate
	// and the dial fails with EOF instead of the verification error
	// this test pins. The Read fails once the client aborts the
	// handshake; that error is expected and discarded here.
	go func() {
		conn, err := srv.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = conn.Receive()
	}()

	// A second, unrelated self-signed certificate is the wrong trust
	// anchor: it shares nothing with the server's certificate.
	wrongDir := t.TempDir()
	wrongCert, _ := writeCertKeyFiles(t, wrongDir)

	clientCfg, err := LoadClientTLSConfig(wrongCert, "", "")
	if err != nil {
		t.Fatalf("LoadClientTLSConfig: %v", err)
	}

	client := New()
	client.WithTLSConfig(clientCfg)

	_, port, err := net.SplitHostPort(srv.Addr())
	if err != nil {
		t.Fatalf("split host/port %q: %v", srv.Addr(), err)
	}

	conn, err := client.Dial(net.JoinHostPort("localhost", port))
	if err == nil {
		conn.Close()
		t.Fatal("tls dial with the wrong CA: got nil error, want a certificate verification failure")
	}
	if !strings.Contains(err.Error(), "certificate") {
		t.Errorf("tls dial with the wrong CA: got %v, want an error mentioning the certificate", err)
	}
}
