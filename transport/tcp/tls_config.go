// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package tcp

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// GenerateSelfSignedCert creates a P-256 ECDSA self-signed certificate
// and returns the TLS certificate suitable for tls.Config.
func GenerateSelfSignedCert() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate serial: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"EoS EIPC"},
			CommonName:   "eipc-server",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("create certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("marshal key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return tls.X509KeyPair(certPEM, keyPEM)
}

// loadCertPool reads a PEM file and returns a pool holding its certificates.
// It is shared by the server and client constructors so a CA file is parsed
// exactly once, the same way, on both sides.
func loadCertPool(caFile string) (*x509.CertPool, error) {
	caFile = filepath.Clean(caFile)
	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read CA file: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}
	return pool, nil
}

// LoadTLSConfig builds a *tls.Config from cert/key/CA file paths.
// If caFile is provided, mTLS (RequireAndVerifyClientCert) is enabled.
func LoadTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	certFile = filepath.Clean(certFile)
	keyFile = filepath.Clean(keyFile)

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load key pair: %w", err)
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	if caFile != "" {
		pool, err := loadCertPool(caFile)
		if err != nil {
			return nil, err
		}
		cfg.ClientCAs = pool
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return cfg, nil
}

// LoadClientTLSConfig builds a *tls.Config for dialing a server.
//
// Argument order is CA-first because the CA is the point of a client
// config: caFile is the CA bundle the server's certificate is verified
// against (RootCAs). Pass "" to verify against the system roots.
// certFile/keyFile optionally present a client certificate for mTLS;
// they must be given together or not at all.
//
// Verification is always on: the returned config never sets
// InsecureSkipVerify. Before this constructor existed, the only way to
// dial a server with a self-signed or private-CA certificate was
// InsecureSkipVerify, because the server-side LoadTLSConfig sets
// Certificates/ClientCAs and never RootCAs.
func LoadClientTLSConfig(caFile, certFile, keyFile string) (*tls.Config, error) {
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if caFile != "" {
		pool, err := loadCertPool(caFile)
		if err != nil {
			return nil, err
		}
		cfg.RootCAs = pool
	}

	if (certFile == "") != (keyFile == "") {
		return nil, fmt.Errorf("client certificate and key must be set together")
	}
	if certFile != "" {
		certFile = filepath.Clean(certFile)
		keyFile = filepath.Clean(keyFile)
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("load client key pair: %w", err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}

	return cfg, nil
}

// AutoTLSConfig generates a self-signed cert for development use.
// It is called by SetupTLSFromEnv when EIPC_TLS_AUTO_CERT is exactly
// "true"; this function itself reads no environment variables.
func AutoTLSConfig() (*tls.Config, error) {
	cert, err := GenerateSelfSignedCert()
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}
