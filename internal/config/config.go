package config

import (
	"errors"
	"os"
	"time"
)

type Config struct {
	HTTPAddr        string
	DatabaseURL     string
	JWTSecret       string
	TokenTTL        time.Duration
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		ShutdownTimeout: 10 * time.Second,
	}

	ttl, err := time.ParseDuration(getEnv("TOKEN_TTL", "24h"))
	if err != nil {
		return Config{}, errors.New("TOKEN_TTL must be a valid duration")
	}
	cfg.TokenTTL = ttl

	if len(cfg.JWTSecret) < 16 {
		return Config{}, errors.New("JWT_SECRET must be at least 16 characters")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
