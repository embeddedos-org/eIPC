// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package tcp

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

// writeCertKeyFiles generates a self-signed pair via the package's own
// generator and writes it as PEM files for LoadTLSConfig to consume.
func writeCertKeyFiles(t *testing.T, dir string) (certFile, keyFile string) {
	t.Helper()
	cert, err := GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert: %v", err)
	}
	if len(cert.Certificate) != 1 {
		t.Fatalf("generated cert chain length: got %d, want 1", len(cert.Certificate))
	}
	key, ok := cert.PrivateKey.(*ecdsa.PrivateKey)
	if !ok {
		t.Fatalf("generated private key type: got %T, want *ecdsa.PrivateKey", cert.PrivateKey)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	certFile = filepath.Join(dir, "cert.pem")
	keyFile = filepath.Join(dir, "key.pem")
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(certFile, certPEM, 0o600); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	return certFile, keyFile
}

func TestGenerateSelfSignedCert(t *testing.T) {
	cert, err := GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert: %v", err)
	}
	if len(cert.Certificate) != 1 {
		t.Fatalf("chain length: got %d, want 1", len(cert.Certificate))
	}
	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("parse generated cert: %v", err)
	}
	if len(parsed.DNSNames) != 1 || parsed.DNSNames[0] != "localhost" {
		t.Errorf("DNSNames: got %v, want [localhost]", parsed.DNSNames)
	}
	if parsed.Subject.CommonName != "eipc-server" {
		t.Errorf("CommonName: got %q, want %q", parsed.Subject.CommonName, "eipc-server")
	}
	wantUsage := x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
	if parsed.KeyUsage != wantUsage {
		t.Errorf("KeyUsage: got %v, want %v", parsed.KeyUsage, wantUsage)
	}
	if len(parsed.ExtKeyUsage) != 2 ||
		parsed.ExtKeyUsage[0] != x509.ExtKeyUsageServerAuth ||
		parsed.ExtKeyUsage[1] != x509.ExtKeyUsageClientAuth {
		t.Errorf("ExtKeyUsage: got %v, want [ServerAuth ClientAuth]", parsed.ExtKeyUsage)
	}
	pub, ok := parsed.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		t.Fatalf("public key type: got %T, want *ecdsa.PublicKey", parsed.PublicKey)
	}
	if pub.Curve.Params().Name != "P-256" {
		t.Errorf("public key curve: got %s, want P-256", pub.Curve.Params().Name)
	}
	if parsed.NotAfter.Before(parsed.NotBefore) {
		t.Errorf("validity window inverted: NotBefore %v, NotAfter %v", parsed.NotBefore, parsed.NotAfter)
	}
}

func TestLoadTLSConfigMissingFiles(t *testing.T) {
	dir := t.TempDir()
	if _, err := LoadTLSConfig(filepath.Join(dir, "nope.crt"), filepath.Join(dir, "nope.key"), ""); err == nil {
		t.Fatal("LoadTLSConfig with missing cert/key files: got nil error, want error")
	}
}

func TestLoadTLSConfigNoCA(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := writeCertKeyFiles(t, dir)
	cfg, err := LoadTLSConfig(certFile, keyFile, "")
	if err != nil {
		t.Fatalf("LoadTLSConfig: %v", err)
	}
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion: got %x, want TLS 1.2 (%x)", cfg.MinVersion, tls.VersionTLS12)
	}
	if len(cfg.Certificates) != 1 {
		t.Errorf("Certificates: got %d, want 1", len(cfg.Certificates))
	}
	if cfg.ClientAuth != tls.NoClientCert {
		t.Errorf("ClientAuth without CA: got %v, want NoClientCert", cfg.ClientAuth)
	}
	if cfg.ClientCAs != nil {
		t.Errorf("ClientCAs without CA: got non-nil pool, want nil")
	}
}

func TestLoadTLSConfigWithCAEnablesMTLS(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := writeCertKeyFiles(t, dir)
	// The generated self-signed cert doubles as the CA bundle for this
	// test: AppendCertsFromPEM only needs one parseable certificate.
	cfg, err := LoadTLSConfig(certFile, keyFile, certFile)
	if err != nil {
		t.Fatalf("LoadTLSConfig with CA: %v", err)
	}
	if cfg.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Errorf("ClientAuth with CA: got %v, want RequireAndVerifyClientCert", cfg.ClientAuth)
	}
	if cfg.ClientCAs == nil {
		t.Error("ClientCAs with CA: got nil, want non-nil pool")
	}
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion: got %x, want TLS 1.2 (%x)", cfg.MinVersion, tls.VersionTLS12)
	}
}

func TestLoadTLSConfigBadCA(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := writeCertKeyFiles(t, dir)

	badCA := filepath.Join(dir, "bad-ca.pem")
	if err := os.WriteFile(badCA, []byte("this is not a PEM certificate\n"), 0o600); err != nil {
		t.Fatalf("write bad CA: %v", err)
	}
	if _, err := LoadTLSConfig(certFile, keyFile, badCA); err == nil {
		t.Fatal("LoadTLSConfig with unparseable CA PEM: got nil error, want error")
	}

	if _, err := LoadTLSConfig(certFile, keyFile, filepath.Join(dir, "missing-ca.pem")); err == nil {
		t.Fatal("LoadTLSConfig with missing CA file: got nil error, want error")
	}
}

func TestAutoTLSConfig(t *testing.T) {
	// Pinned behaviour at HEAD: AutoTLSConfig's doc comment says it is
	// "Controlled by EIPC_TLS_AUTO_CERT env var", but the implementation
	// does not read the variable — it always generates a certificate.
	// These tests pin what the code does today; the mismatch is recorded
	// in the commit that lands this file.
	for _, val := range []string{"", "0", "1"} {
		t.Setenv("EIPC_TLS_AUTO_CERT", val)
		cfg, err := AutoTLSConfig()
		if err != nil {
			t.Fatalf("AutoTLSConfig (env %q): %v", val, err)
		}
		if len(cfg.Certificates) != 1 {
			t.Errorf("AutoTLSConfig (env %q): got %d certificates, want 1", val, len(cfg.Certificates))
		}
		if cfg.MinVersion != tls.VersionTLS12 {
			t.Errorf("AutoTLSConfig (env %q): MinVersion got %x, want TLS 1.2", val, cfg.MinVersion)
		}
	}
}
