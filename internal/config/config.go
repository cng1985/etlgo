package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	Addr    string
	DBPath  string
	DataDir string
	GinMode string
}

func New() (*Config, error) {
	dataDir := env("DATA_DIR", "data")
	abs, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, err
	}
	for _, sub := range []string{"samples", "uploads", "outputs"} {
		if err := os.MkdirAll(filepath.Join(abs, sub), 0o755); err != nil {
			return nil, err
		}
	}
	dbPath := env("DB_PATH", filepath.Join(abs, "etl.db"))
	return &Config{
		Addr:    env("ADDR", ":8080"),
		DBPath:  dbPath,
		DataDir: abs,
		GinMode: env("GIN_MODE", "release"),
	}, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
