// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadHMACKey_EnvVar(t *testing.T) {
	os.Setenv("EIPC_HMAC_KEY", "test-secret-key")
	defer os.Unsetenv("EIPC_HMAC_KEY")
	os.Unsetenv("EIPC_KEY_FILE")

	key, err := LoadHMACKey()
	if err != nil {
		t.Fatalf("LoadHMACKey: %v", err)
	}
	if string(key) != "test-secret-key" {
		t.Errorf("expected 'test-secret-key', got %q", string(key))
	}
}

func TestLoadHMACKey_File(t *testing.T) {
	os.Unsetenv("EIPC_HMAC_KEY")

	tmpFile := filepath.Join(t.TempDir(), "hmac.key")
	os.WriteFile(tmpFile, []byte("file-based-key"), 0600)

	os.Setenv("EIPC_KEY_FILE", tmpFile)
	defer os.Unsetenv("EIPC_KEY_FILE")

	key, err := LoadHMACKey()
	if err != nil {
		t.Fatalf("LoadHMACKey from file: %v", err)
	}
	if string(key) != "file-based-key" {
		t.Errorf("expected 'file-based-key', got %q", string(key))
	}
}

func TestLoadHMACKey_Missing(t *testing.T) {
	os.Unsetenv("EIPC_HMAC_KEY")
	os.Unsetenv("EIPC_KEY_FILE")

	_, err := LoadHMACKey()
	if err == nil {
		t.Fatal("expected error when no key source configured")
	}
}

func TestLoadHMACKey_BadFile(t *testing.T) {
	os.Unsetenv("EIPC_HMAC_KEY")
	os.Setenv("EIPC_KEY_FILE", "/nonexistent/path/key.dat")
	defer os.Unsetenv("EIPC_KEY_FILE")

	_, err := LoadHMACKey()
	if err == nil {
		t.Fatal("expected error for nonexistent key file")
	}
}

func TestLoadSessionTTL_Default(t *testing.T) {
	os.Unsetenv("EIPC_SESSION_TTL")
	ttl := LoadSessionTTL()
	if ttl != 1*time.Hour {
		t.Errorf("expected default 1h, got %v", ttl)
	}
}

func TestLoadSessionTTL_Custom(t *testing.T) {
	os.Setenv("EIPC_SESSION_TTL", "30m")
	defer os.Unsetenv("EIPC_SESSION_TTL")

	ttl := LoadSessionTTL()
	if ttl != 30*time.Minute {
		t.Errorf("expected 30m, got %v", ttl)
	}
}

func TestLoadSessionTTL_Invalid(t *testing.T) {
	os.Setenv("EIPC_SESSION_TTL", "not-a-duration")
	defer os.Unsetenv("EIPC_SESSION_TTL")

	ttl := LoadSessionTTL()
	if ttl != 1*time.Hour {
		t.Errorf("expected default 1h for invalid input, got %v", ttl)
	}
}

func TestLoadMaxConnections_Default(t *testing.T) {
	os.Unsetenv("EIPC_MAX_CONNECTIONS")
	max := LoadMaxConnections()
	if max != 64 {
		t.Errorf("expected default 64, got %d", max)
	}
}

func TestLoadMaxConnections_Custom(t *testing.T) {
	os.Setenv("EIPC_MAX_CONNECTIONS", "128")
	defer os.Unsetenv("EIPC_MAX_CONNECTIONS")

	max := LoadMaxConnections()
	if max != 128 {
		t.Errorf("expected 128, got %d", max)
	}
}

func TestLoadMaxConnections_Invalid(t *testing.T) {
	os.Setenv("EIPC_MAX_CONNECTIONS", "abc")
	defer os.Unsetenv("EIPC_MAX_CONNECTIONS")

	max := LoadMaxConnections()
	if max != 64 {
		t.Errorf("expected default 64 for invalid input, got %d", max)
	}
}

func TestLoadListenAddr_Default(t *testing.T) {
	os.Unsetenv("EIPC_LISTEN_ADDR")
	addr := LoadListenAddr()
	if addr != "127.0.0.1:9090" {
		t.Errorf("expected default '127.0.0.1:9090', got %q", addr)
	}
}

func TestLoadListenAddr_Custom(t *testing.T) {
	os.Setenv("EIPC_LISTEN_ADDR", "0.0.0.0:8080")
	defer os.Unsetenv("EIPC_LISTEN_ADDR")

	addr := LoadListenAddr()
	if addr != "0.0.0.0:8080" {
		t.Errorf("expected '0.0.0.0:8080', got %q", addr)
	}
}

func TestTLSEnabled(t *testing.T) {
	os.Unsetenv("EIPC_TLS_CERT")
	os.Unsetenv("EIPC_TLS_AUTO_CERT")
	if TLSEnabled() {
		t.Error("TLS should be disabled by default")
	}

	os.Setenv("EIPC_TLS_CERT", "/path/to/cert.pem")
	defer os.Unsetenv("EIPC_TLS_CERT")
	if !TLSEnabled() {
		t.Error("TLS should be enabled with EIPC_TLS_CERT set")
	}
}

func TestTLSEnabled_AutoCert(t *testing.T) {
	os.Unsetenv("EIPC_TLS_CERT")
	os.Setenv("EIPC_TLS_AUTO_CERT", "true")
	defer os.Unsetenv("EIPC_TLS_AUTO_CERT")

	if !TLSEnabled() {
		t.Error("TLS should be enabled with EIPC_TLS_AUTO_CERT=true")
	}
}
