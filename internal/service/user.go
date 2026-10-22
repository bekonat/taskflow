package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/bekonat/taskflow/internal/domain"
)

type UserService struct {
	users  UserRepository
	hasher PasswordHasher
	tokens TokenIssuer
}

func NewUserService(users UserRepository, hasher PasswordHasher, tokens TokenIssuer) *UserService {
	return &UserService{users: users, hasher: hasher, tokens: tokens}
}

func (s *UserService) Register(ctx context.Context, creds domain.Credentials) (*domain.User, error) {
	creds.Normalize()
	if err := creds.Validate(); err != nil {
		return nil, err
	}
	hash, err := s.hasher.Hash(creds.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	user := &domain.User{Email: creds.Email, PasswordHash: hash}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) Login(ctx context.Context, creds domain.Credentials) (string, error) {
	creds.Normalize()
	user, err := s.users.GetByEmail(ctx, creds.Email)
	if errors.Is(err, domain.ErrNotFound) {
		return "", domain.ErrInvalidCredentials
	}
	if err != nil {
		return "", err
	}
	ok, err := s.hasher.Compare(user.PasswordHash, creds.Password)
	if err != nil {
		return "", fmt.Errorf("compare password: %w", err)
	}
	if !ok {
		return "", domain.ErrInvalidCredentials
	}
	return s.tokens.Issue(user.ID)
}

func (s *UserService) Get(ctx context.Context, id int64) (*domain.User, error) {
	return s.users.GetByID(ctx, id)
}
