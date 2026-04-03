// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package integrity

import (
	"bytes"
	"testing"
)

func TestSign(t *testing.T) {
	key := []byte("test-key-32-bytes-long-here!!!")
	data := []byte("hello world")

	mac := Sign(key, data)
	if len(mac) != 32 {
		t.Fatalf("expected 32-byte MAC, got %d", len(mac))
	}

	mac2 := Sign(key, data)
	if !bytes.Equal(mac, mac2) {
		t.Error("signing same data with same key should produce identical MAC")
	}
}

func TestVerify(t *testing.T) {
	key := []byte("test-key-32-bytes-long-here!!!")
	data := []byte("hello world")

	mac := Sign(key, data)
	if !Verify(key, data, mac) {
		t.Error("valid MAC should verify successfully")
	}
}

func TestVerify_TamperedData(t *testing.T) {
	key := []byte("test-key-32-bytes-long-here!!!")
	data := []byte("hello world")

	mac := Sign(key, data)

	tampered := []byte("hello World")
	if Verify(key, tampered, mac) {
		t.Error("tampered data should fail verification")
	}
}

func TestVerify_WrongKey(t *testing.T) {
	key := []byte("test-key-32-bytes-long-here!!!")
	data := []byte("hello world")

	mac := Sign(key, data)

	wrongKey := []byte("wrong-key-32-bytes-long-here!!")
	if Verify(wrongKey, data, mac) {
		t.Error("wrong key should fail verification")
	}
}

func TestVerify_EmptyKey(t *testing.T) {
	key := []byte{}
	data := []byte("hello world")

	mac := Sign(key, data)
	if !Verify(key, data, mac) {
		t.Error("empty key should still produce consistent HMAC")
	}
}

func TestVerify_EmptyData(t *testing.T) {
	key := []byte("test-key-32-bytes-long-here!!!")
	data := []byte{}

	mac := Sign(key, data)
	if len(mac) != 32 {
		t.Fatalf("expected 32-byte MAC for empty data, got %d", len(mac))
	}
	if !Verify(key, data, mac) {
		t.Error("empty data should verify successfully")
	}
}

func TestVerify_TruncatedMAC(t *testing.T) {
	key := []byte("test-key-32-bytes-long-here!!!")
	data := []byte("hello world")

	mac := Sign(key, data)
	truncated := mac[:16]
	if Verify(key, data, truncated) {
		t.Error("truncated MAC should fail verification")
	}
}
