package auth

import (
	"encoding/base64"
	"fmt"
	"os"
	"runtime"
)

// encrypt applies a simple XOR obfuscation + base64 encoding.
// This is NOT cryptographically strong — it's obfuscation to prevent
// accidental exposure (e.g., in log files). The key is machine-specific.
func encrypt(data []byte) []byte {
	key := encryptionKey()
	xored := xorBytes(data, key)
	encoded := base64.StdEncoding.EncodeToString(xored)
	return []byte(encoded)
}

// decrypt reverses encrypt.
func decrypt(data []byte) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}
	key := encryptionKey()
	return xorBytes(decoded, key), nil
}

// encryptionKey builds a machine-specific key from hostname + username.
func encryptionKey() []byte {
	hostname, _ := os.Hostname()
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME") // Windows
	}
	if username == "" {
		username = "mgc-default"
	}

	seed := fmt.Sprintf("%s:%s:%s:mgc-v1", hostname, username, runtime.GOOS)
	// Stretch the seed to at least 32 bytes using a simple pattern
	key := make([]byte, 32)
	seedBytes := []byte(seed)
	for i := range key {
		key[i] = seedBytes[i%len(seedBytes)]
	}
	return key
}

// xorBytes XORs data with the key (cycling).
func xorBytes(data, key []byte) []byte {
	result := make([]byte, len(data))
	for i, b := range data {
		result[i] = b ^ key[i%len(key)]
	}
	return result
}
