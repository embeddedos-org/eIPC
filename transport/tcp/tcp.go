// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package tcp

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/embeddedos-org/eipc/transport"
)

// Transport implements the EIPC transport interface over TCP.
// Works on Linux, Windows, and macOS. Supports optional TLS/mTLS.
type Transport struct {
	mu        sync.Mutex
	listener  net.Listener
	tlsConfig *tls.Config
}

// New creates a new TCP transport.
func New() *Transport {
	return &Transport{}
}

// WithTLS configures TLS from cert/key/CA files.
func (t *Transport) WithTLS(certFile, keyFile, caFile string) error {
	cfg, err := LoadTLSConfig(certFile, keyFile, caFile)
	if err != nil {
		return err
	}
	t.tlsConfig = cfg
	return nil
}

// WithTLSConfig sets a pre-built TLS config.
func (t *Transport) WithTLSConfig(cfg *tls.Config) {
	t.tlsConfig = cfg
}

// TLSEnabled reports whether this transport will actually negotiate TLS:
// true only when a TLS config has been set (WithTLS, WithTLSConfig,
// SetupTLSFromEnv, SetupClientTLSFromEnv). This is the ground truth for
// log labels — it must be used instead of sniffing the TLS environment
// variables, which can claim TLS while the socket stays plaintext.
func (t *Transport) TLSEnabled() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.tlsConfig != nil
}

// If TLS is configured, wraps listener with tls.NewListener.
func (t *Transport) Listen(address string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	ln, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("tcp listen: %w", err)
	}

	if t.tlsConfig != nil {
		ln = tls.NewListener(ln, t.tlsConfig)
	}

	t.listener = ln
	return nil
}

// Dial connects to a remote TCP address and returns a Connection.
// If TLS is configured, uses tls.Dial; also enables TCP keepalive.
func (t *Transport) Dial(address string) (transport.Connection, error) {
	if t.tlsConfig != nil {
		clientTLS := t.tlsConfig.Clone()
		conn, err := tls.Dial("tcp", address, clientTLS)
		if err != nil {
			return nil, fmt.Errorf("tcp tls dial: %w", err)
		}
		return transport.NewConnWrapper(conn), nil
	}

	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("tcp dial: %w", err)
	}
	if tc, ok := conn.(*net.TCPConn); ok {
		_ = tc.SetKeepAlive(true)
		_ = tc.SetKeepAlivePeriod(30 * time.Second)
	}
	return transport.NewConnWrapper(conn), nil
}

// Accept waits for and returns the next inbound TCP connection.
// Enables keepalive on accepted TCP connections.
func (t *Transport) Accept() (transport.Connection, error) {
	t.mu.Lock()
	ln := t.listener
	t.mu.Unlock()

	if ln == nil {
		return nil, fmt.Errorf("tcp: not listening")
	}

	conn, err := ln.Accept()
	if err != nil {
		return nil, fmt.Errorf("tcp accept: %w", err)
	}

	if tc, ok := conn.(*net.TCPConn); ok {
		_ = tc.SetKeepAlive(true)
		_ = tc.SetKeepAlivePeriod(30 * time.Second)
	}

	return transport.NewConnWrapper(conn), nil
}

// Close shuts down the TCP listener.
func (t *Transport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.listener != nil {
		return t.listener.Close()
	}
	return nil
}

// Addr returns the listener's address. Returns "" if not listening.
func (t *Transport) Addr() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.listener != nil {
		return t.listener.Addr().String()
	}
	return ""
}

// SetupTLSFromEnv configures server-side TLS from environment variables.
// EIPC_TLS_CERT, EIPC_TLS_KEY for cert/key files.
// EIPC_TLS_CA for mTLS CA file.
// EIPC_TLS_AUTO_CERT=true for auto-generated self-signed cert.
//
// A cert path without its key path (or vice versa) is an error, not a
// silent fall back to plaintext: a half-configured server must fail
// loudly at startup rather than listen unencrypted while the operator
// believes TLS is on.
func (t *Transport) SetupTLSFromEnv() error {
	certFile := os.Getenv("EIPC_TLS_CERT")
	keyFile := os.Getenv("EIPC_TLS_KEY")

	if (certFile == "") != (keyFile == "") {
		return fmt.Errorf("EIPC_TLS_CERT and EIPC_TLS_KEY must be set together")
	}

	if certFile != "" && keyFile != "" {
		return t.WithTLS(certFile, keyFile, os.Getenv("EIPC_TLS_CA"))
	}

	if os.Getenv("EIPC_TLS_AUTO_CERT") == "true" {
		cfg, err := AutoTLSConfig()
		if err != nil {
			return err
		}
		t.tlsConfig = cfg
		return nil
	}

	return nil
}

// SetupClientTLSFromEnv configures client-side TLS from environment
// variables, for dialing a server whose certificate must be verified.
// EIPC_TLS_CA is the CA bundle the server is verified against (RootCAs);
// without it the client verifies against the system roots.
// EIPC_TLS_CERT/EIPC_TLS_KEY optionally present a client certificate
// for mTLS; they must be set together.
//
// This is the client counterpart of SetupTLSFromEnv and the only env
// setup a dialing client should use: the server setup builds a config
// with Certificates/ClientCAs and never RootCAs, so a client built from
// it cannot verify any server certificate and the only working dial was
// InsecureSkipVerify. EIPC_TLS_AUTO_CERT is ignored here — minting a
// self-signed certificate creates a server identity, not client trust.
func (t *Transport) SetupClientTLSFromEnv() error {
	caFile := os.Getenv("EIPC_TLS_CA")
	certFile := os.Getenv("EIPC_TLS_CERT")
	keyFile := os.Getenv("EIPC_TLS_KEY")

	if caFile == "" && certFile == "" && keyFile == "" {
		return nil
	}

	cfg, err := LoadClientTLSConfig(caFile, certFile, keyFile)
	if err != nil {
		return err
	}
	t.tlsConfig = cfg
	return nil
}
