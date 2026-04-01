// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// LoadHMACKey loads the shared HMAC key from environment or file.
// Priority: EIPC_HMAC_KEY env var → EIPC_KEY_FILE env var → error.
func LoadHMACKey() ([]byte, error) {
	if key := os.Getenv("EIPC_HMAC_KEY"); key != "" {
		return []byte(key), nil
	}

	if path := os.Getenv("EIPC_KEY_FILE"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read key file %q: %w", path, err)
		}
		return data, nil
	}

	return nil, fmt.Errorf("EIPC_HMAC_KEY or EIPC_KEY_FILE must be set")
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

// TLSEnabled returns true if TLS cert files are configured or auto-cert is enabled.
func TLSEnabled() bool {
	return os.Getenv("EIPC_TLS_CERT") != "" || os.Getenv("EIPC_TLS_AUTO_CERT") == "true"
}
