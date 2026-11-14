// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// unsetKeyEnv clears every key-related variable these tests may set, so
// one test's environment cannot leak into the next.
func unsetKeyEnv(t *testing.T) {
	t.Helper()
	for _, v := range []string{
		"EIPC_HMAC_KEY", "EIPC_KEY_FILE",
		"EIPC_HMAC_KEY_EIPC_SERVER", "EIPC_KEY_FILE_EIPC_SERVER",
		"EIPC_HMAC_KEY_EBOT_CLIENT", "EIPC_KEY_FILE_EBOT_CLIENT",
	} {
		os.Unsetenv(v)
	}
}

func TestLoadHMACKey_EnvVar(t *testing.T) {
	unsetKeyEnv(t)
	os.Setenv("EIPC_HMAC_KEY", "test-secret-key-that-is-32-bytes!")
	defer os.Unsetenv("EIPC_HMAC_KEY")

	key, err := LoadHMACKey()
	if err != nil {
		t.Fatalf("LoadHMACKey: %v", err)
	}
	if string(key) != "test-secret-key-that-is-32-bytes!" {
		t.Errorf("expected 'test-secret-key-that-is-32-bytes!', got %q", string(key))
	}
}

func TestLoadHMACKey_File(t *testing.T) {
	unsetKeyEnv(t)

	tmpFile := filepath.Join(t.TempDir(), "hmac.key")
	_ = os.WriteFile(tmpFile, []byte("file-based-key-that-is-32-bytes!"), 0600)

	os.Setenv("EIPC_KEY_FILE", tmpFile)
	defer os.Unsetenv("EIPC_KEY_FILE")

	key, err := LoadHMACKey()
	if err != nil {
		t.Fatalf("LoadHMACKey from file: %v", err)
	}
	if string(key) != "file-based-key-that-is-32-bytes!" {
		t.Errorf("expected 'file-based-key-that-is-32-bytes!', got %q", string(key))
	}
}

// A key file written with a trailing newline (e.g. `openssl rand -hex 32
// > hmac.key`) must load without the newline becoming part of the key.
func TestLoadHMACKey_FileTrimsTrailingNewline(t *testing.T) {
	unsetKeyEnv(t)

	tmpFile := filepath.Join(t.TempDir(), "hmac.key")
	_ = os.WriteFile(tmpFile, []byte("file-based-key-that-is-32-bytes!\n"), 0600)

	os.Setenv("EIPC_KEY_FILE", tmpFile)
	defer os.Unsetenv("EIPC_KEY_FILE")

	key, err := LoadHMACKey()
	if err != nil {
		t.Fatalf("LoadHMACKey from file: %v", err)
	}
	if string(key) != "file-based-key-that-is-32-bytes!" {
		t.Errorf("expected trailing newline trimmed, got %q", string(key))
	}
}

func TestLoadHMACKey_Missing(t *testing.T) {
	unsetKeyEnv(t)

	_, err := LoadHMACKey()
	if err == nil {
		t.Fatal("expected error when no key source configured")
	}
}

func TestLoadHMACKey_BadFile(t *testing.T) {
	unsetKeyEnv(t)
	os.Setenv("EIPC_KEY_FILE", "/nonexistent/path/key.dat")
	defer os.Unsetenv("EIPC_KEY_FILE")

	_, err := LoadHMACKey()
	if err == nil {
		t.Fatal("expected error for nonexistent key file")
	}
}

// Keys shorter than MinHMACKeyLength are rejected, from both sources:
// a short key silently weakens every frame's MAC, so load must fail
// closed instead of handing a weak key to the endpoints.
func TestLoadHMACKey_ShortEnvKeyRejected(t *testing.T) {
	unsetKeyEnv(t)
	os.Setenv("EIPC_HMAC_KEY", "too-short")
	defer os.Unsetenv("EIPC_HMAC_KEY")

	_, err := LoadHMACKey()
	if err == nil {
		t.Fatal("expected error for a key shorter than the minimum")
	}
	if !strings.Contains(err.Error(), "32-byte minimum") {
		t.Errorf("error should name the minimum length, got: %v", err)
	}
}

func TestLoadHMACKey_ShortFileKeyRejected(t *testing.T) {
	unsetKeyEnv(t)

	tmpFile := filepath.Join(t.TempDir(), "hmac.key")
	_ = os.WriteFile(tmpFile, []byte("short"), 0600)

	os.Setenv("EIPC_KEY_FILE", tmpFile)
	defer os.Unsetenv("EIPC_KEY_FILE")

	_, err := LoadHMACKey()
	if err == nil {
		t.Fatal("expected error for a short key file")
	}
}

func TestLoadHMACKey_ExactlyMinimumAccepted(t *testing.T) {
	unsetKeyEnv(t)
	os.Setenv("EIPC_HMAC_KEY", strings.Repeat("k", MinHMACKeyLength))
	defer os.Unsetenv("EIPC_HMAC_KEY")

	key, err := LoadHMACKey()
	if err != nil {
		t.Fatalf("LoadHMACKey: %v", err)
	}
	if len(key) != MinHMACKeyLength {
		t.Errorf("expected %d-byte key, got %d", MinHMACKeyLength, len(key))
	}
}

func TestLoadHMACKeyForService_EnvVar(t *testing.T) {
	unsetKeyEnv(t)
	os.Setenv("EIPC_HMAC_KEY", "global-key-that-is-32-bytes-long!")
	os.Setenv("EIPC_HMAC_KEY_EIPC_SERVER", "server-key-that-is-32-bytes-long!")
	defer os.Unsetenv("EIPC_HMAC_KEY")
	defer os.Unsetenv("EIPC_HMAC_KEY_EIPC_SERVER")

	key, err := LoadHMACKeyForService("eipc-server")
	if err != nil {
		t.Fatalf("LoadHMACKeyForService: %v", err)
	}
	if string(key) != "server-key-that-is-32-bytes-long!" {
		t.Errorf("expected the per-service key, got %q", string(key))
	}
}

func TestLoadHMACKeyForService_File(t *testing.T) {
	unsetKeyEnv(t)
	os.Setenv("EIPC_HMAC_KEY", "global-key-that-is-32-bytes-long!")
	defer os.Unsetenv("EIPC_HMAC_KEY")

	tmpFile := filepath.Join(t.TempDir(), "ebot.key")
	_ = os.WriteFile(tmpFile, []byte("ebot-client-key-that-is-32-bytes"), 0600)

	os.Setenv("EIPC_KEY_FILE_EBOT_CLIENT", tmpFile)
	defer os.Unsetenv("EIPC_KEY_FILE_EBOT_CLIENT")

	key, err := LoadHMACKeyForService("ebot.client")
	if err != nil {
		t.Fatalf("LoadHMACKeyForService: %v", err)
	}
	if string(key) != "ebot-client-key-that-is-32-bytes" {
		t.Errorf("expected the per-service file key, got %q", string(key))
	}
}

// A per-service env var beats a per-service file, and both beat the
// global key.
func TestLoadHMACKeyForService_Priority(t *testing.T) {
	unsetKeyEnv(t)
	os.Setenv("EIPC_HMAC_KEY", "global-key-that-is-32-bytes-long!")
	defer os.Unsetenv("EIPC_HMAC_KEY")

	tmpFile := filepath.Join(t.TempDir(), "server.key")
	_ = os.WriteFile(tmpFile, []byte("server-file-key-that-is-32-bytes"), 0600)
	os.Setenv("EIPC_KEY_FILE_EIPC_SERVER", tmpFile)
	defer os.Unsetenv("EIPC_KEY_FILE_EIPC_SERVER")

	key, err := LoadHMACKeyForService("eipc-server")
	if err != nil {
		t.Fatalf("LoadHMACKeyForService: %v", err)
	}
	if string(key) != "server-file-key-that-is-32-bytes" {
		t.Errorf("expected the per-service file key over the global key, got %q", string(key))
	}

	os.Setenv("EIPC_HMAC_KEY_EIPC_SERVER", "server-env-key-that-is-32-bytes!")
	defer os.Unsetenv("EIPC_HMAC_KEY_EIPC_SERVER")

	key, err = LoadHMACKeyForService("eipc-server")
	if err != nil {
		t.Fatalf("LoadHMACKeyForService: %v", err)
	}
	if string(key) != "server-env-key-that-is-32-bytes!" {
		t.Errorf("expected the per-service env key over the file key, got %q", string(key))
	}
}

// With no per-service key configured, the global key is the fallback.
func TestLoadHMACKeyForService_FallsBackToGlobal(t *testing.T) {
	unsetKeyEnv(t)
	os.Setenv("EIPC_HMAC_KEY", "global-key-that-is-32-bytes-long!")
	defer os.Unsetenv("EIPC_HMAC_KEY")

	key, err := LoadHMACKeyForService("eipc-server")
	if err != nil {
		t.Fatalf("LoadHMACKeyForService: %v", err)
	}
	if string(key) != "global-key-that-is-32-bytes-long!" {
		t.Errorf("expected the global key as fallback, got %q", string(key))
	}
}

func TestLoadHMACKeyForService_NoKeyAnywhere(t *testing.T) {
	unsetKeyEnv(t)

	_, err := LoadHMACKeyForService("eipc-server")
	if err == nil {
		t.Fatal("expected error when neither a per-service nor a global key is configured")
	}
}

func TestLoadHMACKeyForService_ShortPerServiceKeyRejected(t *testing.T) {
	unsetKeyEnv(t)
	os.Setenv("EIPC_HMAC_KEY", "global-key-that-is-32-bytes-long!")
	os.Setenv("EIPC_HMAC_KEY_EIPC_SERVER", "short")
	defer os.Unsetenv("EIPC_HMAC_KEY")
	defer os.Unsetenv("EIPC_HMAC_KEY_EIPC_SERVER")

	_, err := LoadHMACKeyForService("eipc-server")
	if err == nil {
		t.Fatal("expected error for a short per-service key")
	}
}

func TestLoadHMACKeyForService_InvalidServiceName(t *testing.T) {
	unsetKeyEnv(t)

	for _, name := range []string{"", "bad name", "bad/name", "bad$service"} {
		if _, err := LoadHMACKeyForService(name); err == nil {
			t.Errorf("expected error for service name %q", name)
		}
	}
}

func TestLoadHMACKeys(t *testing.T) {
	unsetKeyEnv(t)
	os.Setenv("EIPC_HMAC_KEY", "global-key-that-is-32-bytes-long!")
	os.Setenv("EIPC_HMAC_KEY_EIPC_SERVER", "server-key-that-is-32-bytes-long!")
	defer os.Unsetenv("EIPC_HMAC_KEY")
	defer os.Unsetenv("EIPC_HMAC_KEY_EIPC_SERVER")

	keys, err := LoadHMACKeys("eipc-server", "ebot.client")
	if err != nil {
		t.Fatalf("LoadHMACKeys: %v", err)
	}
	if string(keys["eipc-server"]) != "server-key-that-is-32-bytes-long!" {
		t.Errorf("eipc-server: expected its own key, got %q", string(keys["eipc-server"]))
	}
	if string(keys["ebot.client"]) != "global-key-that-is-32-bytes-long!" {
		t.Errorf("ebot.client: expected the global fallback, got %q", string(keys["ebot.client"]))
	}
}

func TestLoadHMACKeys_FailureNamesService(t *testing.T) {
	unsetKeyEnv(t)

	_, err := LoadHMACKeys("eipc-server")
	if err == nil {
		t.Fatal("expected error when no key is configured")
	}
	if !strings.Contains(err.Error(), "eipc-server") {
		t.Errorf("error should name the failing service, got: %v", err)
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
	os.Unsetenv("EIPC_TLS_KEY")
	os.Unsetenv("EIPC_TLS_AUTO_CERT")
	if TLSEnabled() {
		t.Error("TLS should be disabled by default")
	}

	// A lone cert without its key does not engage TLS: SetupTLSFromEnv
	// rejects the half-configured pair, so the predicate must not
	// report enabled either — that was the [TLS]-on-plaintext lie.
	os.Setenv("EIPC_TLS_CERT", "/path/to/cert.pem")
	defer os.Unsetenv("EIPC_TLS_CERT")
	if TLSEnabled() {
		t.Error("TLS should be disabled with EIPC_TLS_CERT set but no EIPC_TLS_KEY")
	}

	os.Setenv("EIPC_TLS_KEY", "/path/to/key.pem")
	defer os.Unsetenv("EIPC_TLS_KEY")
	if !TLSEnabled() {
		t.Error("TLS should be enabled with EIPC_TLS_CERT and EIPC_TLS_KEY set")
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
