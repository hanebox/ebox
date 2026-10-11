package account

import (
	"strings"
	"testing"
)

func TestHashPasswordAndVerify(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("HashPassword() = %q, want argon2id encoded hash", hash)
	}

	if !VerifyPassword(hash, "correct horse battery staple") {
		t.Fatal("VerifyPassword() rejected the correct password")
	}

	if VerifyPassword(hash, "wrong password") {
		t.Fatal("VerifyPassword() accepted an incorrect password")
	}
}

func TestHashPasswordSalted(t *testing.T) {
	first, err := HashPassword("same password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	second, err := HashPassword("same password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if first == second {
		t.Fatal("HashPassword() produced identical hashes; salt is not random")
	}
}

func TestVerifyPasswordRejectsMalformedHashes(t *testing.T) {
	cases := []string{
		"",
		"plaintext",
		"$argon2id$v=19$m=19456,t=2,p=1$notbase64$notbase64",
		"$argon2id$v=99$m=19456,t=2,p=1$c2FsdA$aGFzaA",
		"$bcrypt$v=19$m=19456,t=2,p=1$c2FsdA$aGFzaA",
	}

	for _, encoded := range cases {
		if VerifyPassword(encoded, "anything") {
			t.Fatalf("VerifyPassword(%q) accepted a malformed hash", encoded)
		}
	}
}
