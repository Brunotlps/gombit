package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const maxPasswordBytes = 72

// Hasher hashes and compares passwords.
type Hasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

type bcryptHasher struct {
	cost int
}

func newBcryptHasher(cost int) Hasher {
	if cost <= 0 {
		cost = bcrypt.DefaultCost
	}
	return bcryptHasher{cost: cost}
}

func (h bcryptHasher) Hash(password string) (string, error) {
	if err := checkPasswordLength(password); err != nil {
		return "", err
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("auth: hash password: %w", err)
	}
	return string(hashed), nil
}

func (h bcryptHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func checkPasswordLength(password string) error {
	if password == "" {
		return errPasswordRequired
	}
	if len(password) > maxPasswordBytes {
		return errPasswordTooLong
	}
	return nil
}
