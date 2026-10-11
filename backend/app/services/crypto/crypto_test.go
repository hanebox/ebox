package crypto

import (
	"strings"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	if err := Init("base64:test-key-for-unit-tests"); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	cases := []string{
		"hello world",
		"",
		"unicode: 🔐 café",
		strings.Repeat("long-secret-", 100),
	}

	for _, plaintext := range cases {
		encrypted, err := Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encrypt(%q) error = %v", plaintext, err)
		}

		if !strings.HasPrefix(encrypted, payloadPrefix) {
			t.Fatalf("Encrypt(%q) = %q, missing version prefix", plaintext, encrypted)
		}

		decrypted, err := Decrypt(encrypted)
		if err != nil {
			t.Fatalf("Decrypt() error = %v", err)
		}

		if decrypted != plaintext {
			t.Fatalf("Decrypt() = %q, want %q", decrypted, plaintext)
		}
	}
}

func TestEncryptProducesUniqueCiphertexts(t *testing.T) {
	if err := Init("base64:test-key-for-unit-tests"); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	first, err := Encrypt("same input")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	second, err := Encrypt("same input")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	if first == second {
		t.Fatal("Encrypt() produced identical ciphertexts for the same input; nonce reuse")
	}
}

func TestDecryptRejectsTamperedPayload(t *testing.T) {
	if err := Init("base64:test-key-for-unit-tests"); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	encrypted, err := Encrypt("sensitive")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	tampered := encrypted[:len(encrypted)-2] + "aa"

	if _, err := Decrypt(tampered); err == nil {
		t.Fatal("Decrypt() accepted a tampered payload")
	}

	if _, err := Decrypt("not-a-payload"); err == nil {
		t.Fatal("Decrypt() accepted an unknown payload format")
	}
}

func TestInitRejectsEmptyKey(t *testing.T) {
	if err := Init("   "); err == nil {
		t.Fatal("Init() accepted an empty APP_KEY")
	}
}

func TestHashTokenIsStable(t *testing.T) {
	first := HashToken("token")
	second := HashToken("token")

	if first != second {
		t.Fatalf("HashToken() not stable: %q != %q", first, second)
	}

	if first == HashToken("other") {
		t.Fatal("HashToken() collided for different inputs")
	}

	if len(first) != 64 {
		t.Fatalf("HashToken() length = %d, want 64", len(first))
	}
}

func TestRandomToken(t *testing.T) {
	first, err := RandomToken(32)
	if err != nil {
		t.Fatalf("RandomToken() error = %v", err)
	}

	second, err := RandomToken(32)
	if err != nil {
		t.Fatalf("RandomToken() error = %v", err)
	}

	if first == second {
		t.Fatal("RandomToken() returned duplicate values")
	}

	if len(first) != 64 {
		t.Fatalf("RandomToken(32) length = %d, want 64 hex chars", len(first))
	}
}
