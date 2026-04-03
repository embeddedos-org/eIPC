// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package encryption

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	key := make([]byte, KeySize)
	rand.Read(key)

	plaintext := []byte("hello EIPC encryption")

	ciphertext, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if bytes.Equal(ciphertext[NonceSize:], plaintext) {
		t.Error("ciphertext should not equal plaintext")
	}

	decrypted, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted %q != plaintext %q", decrypted, plaintext)
	}
}

func TestEncrypt_WrongKeySize(t *testing.T) {
	_, err := Encrypt([]byte("short"), []byte("data"))
	if err == nil {
		t.Fatal("expected error for wrong key size")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key := make([]byte, KeySize)
	rand.Read(key)

	ciphertext, _ := Encrypt(key, []byte("secret data"))

	wrongKey := make([]byte, KeySize)
	rand.Read(wrongKey)

	_, err := Decrypt(wrongKey, ciphertext)
	if err == nil {
		t.Fatal("expected error for wrong key")
	}
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	key := make([]byte, KeySize)
	rand.Read(key)

	ciphertext, _ := Encrypt(key, []byte("secret data"))

	// Tamper with the ciphertext (not the nonce)
	if len(ciphertext) > NonceSize+1 {
		ciphertext[NonceSize+1] ^= 0xff
	}

	_, err := Decrypt(key, ciphertext)
	if err == nil {
		t.Fatal("expected error for tampered ciphertext")
	}
}

func TestDecrypt_TooShort(t *testing.T) {
	key := make([]byte, KeySize)
	rand.Read(key)

	_, err := Decrypt(key, []byte{1, 2, 3})
	if err == nil {
		t.Fatal("expected error for ciphertext too short")
	}
}

func TestEncryptDecrypt_EmptyPlaintext(t *testing.T) {
	key := make([]byte, KeySize)
	rand.Read(key)

	ciphertext, err := Encrypt(key, []byte{})
	if err != nil {
		t.Fatalf("Encrypt empty: %v", err)
	}

	decrypted, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt empty: %v", err)
	}

	if len(decrypted) != 0 {
		t.Errorf("expected empty plaintext, got %d bytes", len(decrypted))
	}
}

func TestEncryptDecrypt_LargePayload(t *testing.T) {
	key := make([]byte, KeySize)
	rand.Read(key)

	plaintext := make([]byte, 1<<16) // 64KB
	rand.Read(plaintext)

	ciphertext, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt large: %v", err)
	}

	decrypted, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt large: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Error("large payload round-trip failed")
	}
}

func TestEncrypt_UniqueNonces(t *testing.T) {
	key := make([]byte, KeySize)
	rand.Read(key)

	ct1, _ := Encrypt(key, []byte("same data"))
	ct2, _ := Encrypt(key, []byte("same data"))

	if bytes.Equal(ct1, ct2) {
		t.Error("two encryptions of same data should produce different ciphertexts (different nonces)")
	}
}
