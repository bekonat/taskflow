package auth

import (
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestJWTRoundTrip(t *testing.T) {
	m := NewJWTManager("secret", time.Hour)
	token, err := m.Issue(42)
	if err != nil {
		t.Fatal(err)
	}
	id, err := m.Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	if id != 42 {
		t.Fatalf("id = %d, want 42", id)
	}
}

func TestJWTExpired(t *testing.T) {
	m := NewJWTManager("secret", time.Minute)
	issued := time.Now()
	m.now = func() time.Time { return issued }
	token, _ := m.Issue(1)

	m.now = func() time.Time { return issued.Add(2 * time.Minute) }
	if _, err := m.Parse(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestJWTWrongSecret(t *testing.T) {
	token, _ := NewJWTManager("one", time.Hour).Issue(1)
	if _, err := NewJWTManager("two", time.Hour).Parse(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestJWTGarbage(t *testing.T) {
	if _, err := NewJWTManager("s", time.Hour).Parse("not.a.token"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}

func TestBcryptHasher(t *testing.T) {
	h := &BcryptHasher{Cost: bcrypt.MinCost}
	hash, err := h.Hash("password123")
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := h.Compare(hash, "password123"); err != nil || !ok {
		t.Fatalf("Compare(correct) = %v, %v", ok, err)
	}
	if ok, err := h.Compare(hash, "wrong"); err != nil || ok {
		t.Fatalf("Compare(wrong) = %v, %v", ok, err)
	}
	if _, err := h.Compare("garbage", "x"); err == nil {
		t.Fatal("Compare(garbage hash) returned nil error")
	}
}
