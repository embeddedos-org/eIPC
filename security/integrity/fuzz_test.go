// SPDX-License-Identifier: MIT
// Copyright (c) 2026 EoS Project

package integrity

import (
	"testing"
)

func FuzzHMACVerify(f *testing.F) {
	f.Add([]byte("key"), []byte("data"), []byte("mac"))
	f.Add([]byte("secret-key-32bytes!!"), []byte("hello world"), Sign([]byte("secret-key-32bytes!!"), []byte("hello world")))
	f.Add([]byte{}, []byte{}, []byte{})

	f.Fuzz(func(t *testing.T, key, data, mac []byte) {
		result := Verify(key, data, mac)

		validMAC := Sign(key, data)
		validResult := Verify(key, data, validMAC)
		if !validResult {
			t.Error("signing then verifying with same key/data should always succeed")
		}

		_ = result
	})
}
