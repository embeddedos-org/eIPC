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

// Listen starts a TCP listener on the given address (e.g. "127.0.0.1:9090").
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

// SetupTLSFromEnv configures TLS from environment variables.
// EIPC_TLS_CERT, EIPC_TLS_KEY for cert/key files.
// EIPC_TLS_CA for mTLS CA file.
// EIPC_TLS_AUTO_CERT=true for auto-generated self-signed cert.
func (t *Transport) SetupTLSFromEnv() error {
	certFile := os.Getenv("EIPC_TLS_CERT")
	keyFile := os.Getenv("EIPC_TLS_KEY")

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
