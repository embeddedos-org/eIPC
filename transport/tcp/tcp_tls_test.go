// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package tcp

import (
	"crypto/tls"
	"path/filepath"
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
	// The file path engages only when cert AND key are both set; a
	// lone cert path is ignored and the transport stays unconfigured.
	dir := t.TempDir()
	certFile, _ := writeCertKeyFiles(t, dir)
	clearTLSEnv(t, certFile, "", "", "")

	tr := New()
	if err := tr.SetupTLSFromEnv(); err != nil {
		t.Fatalf("SetupTLSFromEnv: %v", err)
	}
	if tr.tlsConfig != nil {
		t.Errorf("tlsConfig: got non-nil, want nil for a cert path without a key path")
	}
}

// TestTLSRoundTrip drives a real TLS listener/dial pair and exchanges
// protocol frames in both directions, mirroring TestTCPBidirectional
// over TLS. The generated server certificate is not a CA and its only
// SAN is "localhost", so the client's config skips verification —
// certificate verification itself is pinned at the config level by
// the tls_config_test.go tests; this test pins the transport wiring
// (tls.NewListener on Listen, tls.Dial on Dial).
func TestTLSRoundTrip(t *testing.T) {
	serverCfg, err := AutoTLSConfig()
	if err != nil {
		t.Fatalf("AutoTLSConfig: %v", err)
	}

	srv := New()
	srv.WithTLSConfig(serverCfg)
	if err := srv.Listen("127.0.0.1:0"); err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer srv.Close()

	client := New()
	client.WithTLSConfig(&tls.Config{InsecureSkipVerify: true}) //nolint:gosec // test-only client config; see comment above

	addr := srv.Addr()

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
