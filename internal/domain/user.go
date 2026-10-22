package domain

import (
	"net/mail"
	"strings"
	"time"
)

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (c *Credentials) Normalize() {
	c.Email = strings.ToLower(strings.TrimSpace(c.Email))
}

func (c Credentials) Validate() error {
	if _, err := mail.ParseAddress(c.Email); err != nil {
		return &ValidationError{Field: "email", Message: "must be a valid email address"}
	}
	if len(c.Password) < 8 {
		return &ValidationError{Field: "password", Message: "must be at least 8 characters"}
	}
	if len(c.Password) > 72 {
		return &ValidationError{Field: "password", Message: "must be at most 72 characters"}
	}
	return nil
}
