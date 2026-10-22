package service

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/bekonat/taskflow/internal/domain"
	"github.com/bekonat/taskflow/internal/repository/memory"
)

type fakeHasher struct {
	err error
}

func (h fakeHasher) Hash(p string) (string, error) {
	if h.err != nil {
		return "", h.err
	}
	return "hashed:" + p, nil
}

func (h fakeHasher) Compare(hash, p string) (bool, error) {
	return hash == "hashed:"+p, nil
}

type fakeTokens struct{}

func (fakeTokens) Issue(id int64) (string, error) {
	return "token-" + strconv.FormatInt(id, 10), nil
}

func newUserService() *UserService {
	return NewUserService(memory.NewUserRepository(), fakeHasher{}, fakeTokens{})
}

func TestRegister(t *testing.T) {
	svc := newUserService()
	ctx := context.Background()

	user, err := svc.Register(ctx, domain.Credentials{Email: " Bob@Mail.com ", Password: "password1"})
	if err != nil {
		t.Fatal(err)
	}
	if user.ID == 0 || user.Email != "bob@mail.com" || user.PasswordHash != "hashed:password1" {
		t.Fatalf("unexpected user: %+v", user)
	}

	_, err = svc.Register(ctx, domain.Credentials{Email: "bob@mail.com", Password: "password2"})
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("err = %v, want ErrAlreadyExists", err)
	}
}

func TestRegisterValidation(t *testing.T) {
	_, err := newUserService().Register(context.Background(), domain.Credentials{Email: "bad", Password: "password1"})
	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("err = %v, want ValidationError", err)
	}
}

func TestRegisterHashError(t *testing.T) {
	svc := NewUserService(memory.NewUserRepository(), fakeHasher{err: errors.New("boom")}, fakeTokens{})
	if _, err := svc.Register(context.Background(), domain.Credentials{Email: "a@b.co", Password: "password1"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestLogin(t *testing.T) {
	svc := newUserService()
	ctx := context.Background()
	if _, err := svc.Register(ctx, domain.Credentials{Email: "a@b.co", Password: "password1"}); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		creds   domain.Credentials
		wantErr error
	}{
		{"ok", domain.Credentials{Email: "A@B.co", Password: "password1"}, nil},
		{"wrong password", domain.Credentials{Email: "a@b.co", Password: "nope"}, domain.ErrInvalidCredentials},
		{"unknown email", domain.Credentials{Email: "x@b.co", Password: "password1"}, domain.ErrInvalidCredentials},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := svc.Login(ctx, tt.creds)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && token == "" {
				t.Fatal("empty token")
			}
		})
	}
}
