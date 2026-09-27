package config

import "testing"

func TestLoadUsesDevelopmentDefaultSecretWhenUnset(t *testing.T) {
	t.Setenv("JWT_SECRET", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	if !cfg.JWT.UsingDefaultSecret {
		t.Error("UsingDefaultSecret should be true when JWT_SECRET is unset in development")
	}
	if cfg.JWT.Secret != developmentJWTSecret {
		t.Errorf("Secret = %q, want the development default", cfg.JWT.Secret)
	}
}

func TestLoadDoesNotFlagDefaultSecretWhenExplicitlySet(t *testing.T) {
	t.Setenv("JWT_SECRET", "an-explicitly-configured-secret-value")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	if cfg.JWT.UsingDefaultSecret {
		t.Error("UsingDefaultSecret should be false when JWT_SECRET is explicitly set")
	}
}

func TestLoadRejectsEmptySecretInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load should fail in production when JWT_SECRET is unset")
	}
}

func TestLoadBuildsPostgresDSN(t *testing.T) {
	t.Setenv("DB_HOST", "db.example.com")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_USER", "test user")
	t.Setenv("DB_PASSWORD", "p@ss")
	t.Setenv("DB_NAME", "test db")
	t.Setenv("DB_SSLMODE", "require")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	want := "postgres://test%20user:p%40ss@db.example.com:5433/test%20db?sslmode=require"
	if cfg.Database.DSN != want {
		t.Errorf("DSN = %q, want %q", cfg.Database.DSN, want)
	}
}

func TestLoadRejectsInvalidConnectionLimits(t *testing.T) {
	t.Setenv("DB_MAX_OPEN_CONNS", "0")

	if _, err := Load(); err == nil {
		t.Fatal("Load should reject DB_MAX_OPEN_CONNS < 1")
	}
}
