package auth

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

type JWTManager struct {
	secret []byte
	ttl    time.Duration
	issuer string
	now    func() time.Time
}

func NewJWTManager(secret string, ttl time.Duration) *JWTManager {
	return &JWTManager{
		secret: []byte(secret),
		ttl:    ttl,
		issuer: "taskflow",
		now:    time.Now,
	}
}

func (m *JWTManager) Issue(userID int64) (string, error) {
	now := m.now()
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		Issuer:    m.issuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *JWTManager) Parse(tokenString string) (int64, error) {
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		return m.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(m.issuer),
		jwt.WithTimeFunc(m.now),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	id, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: bad subject", ErrInvalidToken)
	}
	return id, nil
}
