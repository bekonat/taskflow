package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("JWT_SECRET", "0123456789abcdef")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("TOKEN_TTL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != ":8080" || cfg.TokenTTL != 24*time.Hour {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		ttl    string
	}{
		{"short secret", "short", "1h"},
		{"bad ttl", "0123456789abcdef", "forever"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", tt.secret)
			t.Setenv("TOKEN_TTL", tt.ttl)
			if _, err := Load(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
