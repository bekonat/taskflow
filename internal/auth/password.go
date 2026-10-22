package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type BcryptHasher struct {
	Cost int
}

func NewBcryptHasher() *BcryptHasher {
	return &BcryptHasher{Cost: bcrypt.DefaultCost}
}

func (h *BcryptHasher) Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), h.Cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (h *BcryptHasher) Compare(hash, password string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
