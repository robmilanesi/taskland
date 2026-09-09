// Package config loads the application configuration from the environment.
package config

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"time"
)

// minJWTSecretLen is the minimum accepted length for the HS256 signing secret.
const minJWTSecretLen = 32

// Config holds every runtime setting for the server.
type Config struct {
	Addr      string
	DBPath    string
	JWTSecret string
	JWTTTL    time.Duration
}

// Load reads the configuration from TASKLAND_* environment variables, applying
// defaults where possible. It fails when TASKLAND_JWT_SECRET is missing or too
// short, or when TASKLAND_JWT_TTL is not a positive Go duration.
func Load() (Config, error) {
	cfg := Config{
		Addr:   cmp.Or(os.Getenv("TASKLAND_ADDR"), ":8080"),
		DBPath: cmp.Or(os.Getenv("TASKLAND_DB_PATH"), "taskland.db"),
	}

	cfg.JWTSecret = os.Getenv("TASKLAND_JWT_SECRET")
	if cfg.JWTSecret == "" {
		return Config{}, errors.New("TASKLAND_JWT_SECRET is required")
	}
	if len(cfg.JWTSecret) < minJWTSecretLen {
		return Config{}, fmt.Errorf("TASKLAND_JWT_SECRET must be at least %d characters", minJWTSecretLen)
	}

	ttl := cmp.Or(os.Getenv("TASKLAND_JWT_TTL"), "24h")
	parsed, err := time.ParseDuration(ttl)
	if err != nil {
		return Config{}, fmt.Errorf("invalid TASKLAND_JWT_TTL %q: %w", ttl, err)
	}
	if parsed <= 0 {
		return Config{}, fmt.Errorf("TASKLAND_JWT_TTL must be positive, got %s", parsed)
	}
	cfg.JWTTTL = parsed

	return cfg, nil
}
