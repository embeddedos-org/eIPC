// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// MinHMACKeyLength is the minimum accepted HMAC key length, in bytes.
// HMAC-SHA256 needs a key at least as long as its output (32 bytes) to
// use the full key space; anything shorter is rejected at load time
// rather than silently weakening every frame's MAC. This applies to the
// global key and to every per-service key.
const MinHMACKeyLength = 32

// validateHMACKey rejects keys shorter than MinHMACKeyLength. The key is
// never logged: a short key is reported by length only.
func validateHMACKey(key []byte, source string) error {
	if len(key) < MinHMACKeyLength {
		return fmt.Errorf("HMAC key from %s is %d bytes, below the %d-byte minimum",
			source, len(key), MinHMACKeyLength)
	}
	return nil
}

// readKeyFile reads a key file and strips trailing line endings and
// whitespace. Key files are commonly written with a trailing newline
// (e.g. `openssl rand -hex 32 > hmac.key`); without the trim that
// newline would silently become part of the key.
func readKeyFile(path string) ([]byte, error) {
	path = filepath.Clean(path)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read key file %q: %w", path, err)
	}
	return bytes.TrimRight(data, " \t\r\n"), nil
}

// LoadHMACKey loads the shared HMAC key from environment or file.
// Priority: EIPC_HMAC_KEY env var → EIPC_KEY_FILE env var → error.
//
// This is the legacy global key: one secret authenticates every
// service, so one leak compromises all of them. New deployments should
// use per-service keys via LoadHMACKeyForService; the global key
// remains as the fallback until the endpoints are wired to per-service
// keys (foundation PR-EIPC-5 follow-up). Keys shorter than
// MinHMACKeyLength are rejected.
func LoadHMACKey() ([]byte, error) {
	if key := os.Getenv("EIPC_HMAC_KEY"); key != "" {
		if err := validateHMACKey([]byte(key), "EIPC_HMAC_KEY"); err != nil {
			return nil, err
		}
		return []byte(key), nil
	}

	if path := os.Getenv("EIPC_KEY_FILE"); path != "" {
		data, err := readKeyFile(path)
		if err != nil {
			return nil, err
		}
		if err := validateHMACKey(data, "EIPC_KEY_FILE"); err != nil {
			return nil, err
		}
		return data, nil
	}

	return nil, fmt.Errorf("EIPC_HMAC_KEY or EIPC_KEY_FILE must be set")
}

// serviceEnvSuffix maps a service name to the suffix used in its
// per-service environment variables: upper-cased, with every character
// outside [A-Z0-9] replaced by "_". For example "eipc-server" and
// "ebot.client" become "EIPC_SERVER" and "EBOT_CLIENT".
func serviceEnvSuffix(service string) (string, error) {
	if service == "" {
		return "", fmt.Errorf("service name is required")
	}
	for _, r := range service {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' ||
			r >= '0' && r <= '9' || r == '.' || r == '-' || r == '_') {
			return "", fmt.Errorf("invalid service name %q: only letters, digits, '.', '-' and '_' are allowed", service)
		}
	}
	var b strings.Builder
	for _, r := range service {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r - 'a' + 'A')
		case r >= 'A' && r <= 'Z' || r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String(), nil
}

// LoadHMACKeyForService loads the HMAC key for one service, so each
// service can have its own purpose-separated secret instead of sharing
// the single global key. Priority:
//
//  1. EIPC_HMAC_KEY_<SERVICE> env var
//  2. EIPC_KEY_FILE_<SERVICE> env var (path to a key file)
//  3. The global key from LoadHMACKey (fallback until the endpoints
//     are wired to per-service keys; remove the global key once the
//     migration in docs/key-management.md is complete)
//
// <SERVICE> is the service name upper-cased with "."/"-"/"_" mapped to
// "_": service "eipc-server" reads EIPC_HMAC_KEY_EIPC_SERVER. Keys from
// any source must be at least MinHMACKeyLength bytes.
func LoadHMACKeyForService(service string) ([]byte, error) {
	suffix, err := serviceEnvSuffix(service)
	if err != nil {
		return nil, err
	}

	envVar := "EIPC_HMAC_KEY_" + suffix
	if key := os.Getenv(envVar); key != "" {
		if err := validateHMACKey([]byte(key), envVar); err != nil {
			return nil, err
		}
		return []byte(key), nil
	}

	fileVar := "EIPC_KEY_FILE_" + suffix
	if path := os.Getenv(fileVar); path != "" {
		data, err := readKeyFile(path)
		if err != nil {
			return nil, err
		}
		if err := validateHMACKey(data, fileVar); err != nil {
			return nil, err
		}
		return data, nil
	}

	return LoadHMACKey()
}

// LoadHMACKeys loads the HMAC key for each named service and returns
// them keyed by service name. It fails on the first service whose key
// cannot be loaded, naming that service in the error. A service with
// no per-service key falls back to the global key, exactly as
// LoadHMACKeyForService does.
func LoadHMACKeys(services ...string) (map[string][]byte, error) {
	keys := make(map[string][]byte, len(services))
	for _, service := range services {
		key, err := LoadHMACKeyForService(service)
		if err != nil {
			return nil, fmt.Errorf("load HMAC key for service %q: %w", service, err)
		}
		keys[service] = key
	}
	return keys, nil
}

// LoadSessionTTL reads the session TTL from EIPC_SESSION_TTL env var.
// Format is Go duration string (e.g. "1h", "30m"). Default: 1h.
func LoadSessionTTL() time.Duration {
	if s := os.Getenv("EIPC_SESSION_TTL"); s != "" {
		if d, err := time.ParseDuration(s); err == nil {
			return d
		}
	}
	return 1 * time.Hour
}

// LoadMaxConnections reads max connections from EIPC_MAX_CONNECTIONS env var.
// Default: 64.
func LoadMaxConnections() int {
	if s := os.Getenv("EIPC_MAX_CONNECTIONS"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			return n
		}
	}
	return 64
}

// LoadListenAddr reads the listen address from EIPC_LISTEN_ADDR env var.
// Default: 127.0.0.1:9090.
func LoadListenAddr() string {
	if addr := os.Getenv("EIPC_LISTEN_ADDR"); addr != "" {
		return addr
	}
	return "127.0.0.1:9090"
}

// TLSEnabled returns true if the TLS environment will actually engage the
// server transport: a cert/key pair, or auto-cert. A lone EIPC_TLS_CERT
// without EIPC_TLS_KEY does not engage TLS (SetupTLSFromEnv rejects a
// half-configured pair), so it must not report enabled here either —
// reporting it enabled is what let the server log [TLS] on a plaintext
// socket.
func TLSEnabled() bool {
	cert, key := os.Getenv("EIPC_TLS_CERT"), os.Getenv("EIPC_TLS_KEY")
	return (cert != "" && key != "") || os.Getenv("EIPC_TLS_AUTO_CERT") == "true"
}
