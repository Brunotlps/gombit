package auth

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestBcryptHasherRoundTrip(t *testing.T) {
	h := newBcryptHasher(bcrypt.MinCost)
	hash, err := h.Hash("correct-horse")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if err := h.Compare(hash, "correct-horse"); err != nil {
		t.Fatalf("Compare() error = %v, want nil", err)
	}
	if err := h.Compare(hash, "wrong-password"); err == nil {
		t.Fatal("Compare(wrong) error = nil, want mismatch")
	}
}

func TestBcryptHasherRejectsTooLongPassword(t *testing.T) {
	h := newBcryptHasher(bcrypt.MinCost)
	_, err := h.Hash(strings.Repeat("a", maxPasswordBytes+1))
	if err == nil {
		t.Fatal("Hash() error = nil, want too-long error")
	}
	if !strings.Contains(err.Error(), "72") {
		t.Fatalf("Hash() error = %v, want 72-byte limit", err)
	}
}
