package config

import (
	"strings"
	"testing"
	"time"
)

func validSecret() string { return strings.Repeat("x", minJWTSecretLen) }

func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"TASKLAND_ADDR", "TASKLAND_DB_PATH", "TASKLAND_JWT_SECRET", "TASKLAND_JWT_TTL"} {
		t.Setenv(k, "")
	}
}

func TestLoad_Defaults(t *testing.T) {
	clearEnv(t)
	t.Setenv("TASKLAND_JWT_SECRET", validSecret())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q, want :8080", cfg.Addr)
	}
	if cfg.DBPath != "taskland.db" {
		t.Errorf("DBPath = %q, want taskland.db", cfg.DBPath)
	}
	if cfg.JWTTTL != 24*time.Hour {
		t.Errorf("JWTTTL = %s, want 24h", cfg.JWTTTL)
	}
}

func TestLoad_Overrides(t *testing.T) {
	clearEnv(t)
	t.Setenv("TASKLAND_ADDR", ":9999")
	t.Setenv("TASKLAND_DB_PATH", "/tmp/x.db")
	t.Setenv("TASKLAND_JWT_SECRET", strings.Repeat("s", 40))
	t.Setenv("TASKLAND_JWT_TTL", "1h30m")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Addr != ":9999" || cfg.DBPath != "/tmp/x.db" {
		t.Errorf("unexpected cfg: %+v", cfg)
	}
	if cfg.JWTSecret != strings.Repeat("s", 40) {
		t.Errorf("JWTSecret not read from env")
	}
	if cfg.JWTTTL != 90*time.Minute {
		t.Errorf("JWTTTL = %s, want 1h30m", cfg.JWTTTL)
	}
}

func TestLoad_MissingSecret(t *testing.T) {
	clearEnv(t)
	if _, err := Load(); err == nil {
		t.Fatal("expected an error when TASKLAND_JWT_SECRET is unset")
	}
}

func TestLoad_ShortSecret(t *testing.T) {
	clearEnv(t)
	t.Setenv("TASKLAND_JWT_SECRET", "tooshort")
	if _, err := Load(); err == nil {
		t.Fatal("expected an error for a short TASKLAND_JWT_SECRET")
	}
}

func TestLoad_BadTTL(t *testing.T) {
	clearEnv(t)
	t.Setenv("TASKLAND_JWT_SECRET", validSecret())
	t.Setenv("TASKLAND_JWT_TTL", "not-a-duration")
	if _, err := Load(); err == nil {
		t.Fatal("expected an error for an unparseable TASKLAND_JWT_TTL")
	}
}

func TestLoad_NonPositiveTTL(t *testing.T) {
	clearEnv(t)
	t.Setenv("TASKLAND_JWT_SECRET", validSecret())
	t.Setenv("TASKLAND_JWT_TTL", "-5m")
	if _, err := Load(); err == nil {
		t.Fatal("expected an error for a non-positive TASKLAND_JWT_TTL")
	}
}
