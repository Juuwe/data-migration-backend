package config

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsDotEnv(t *testing.T) {
	const key = "WEB_PORT"
	original, existed := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(key, original)
		} else {
			_ = os.Unsetenv(key)
		}
	})

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("WEB_PORT=9090\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.WebServer.Port != 9090 {
		t.Fatalf("WebServer.Port = %d, want 9090", cfg.WebServer.Port)
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	t.Setenv("WEB_PORT", "not-a-port")

	_, err := Load(filepath.Join(t.TempDir(), "missing.env"))

	if err == nil {
		t.Fatal("Load() error = nil, want invalid port error")
	}
}

func TestDatabaseDSNEscapesCredentials(t *testing.T) {
	database := Database{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "secret@value",
		Name:     "data_migration",
		SSLMode:  "disable",
		Timezone: "Europe/Moscow",
	}

	parsed, err := url.Parse(database.DSN())
	if err != nil {
		t.Fatalf("parse DSN: %v", err)
	}
	password, _ := parsed.User.Password()
	if password != database.Password {
		t.Fatalf("password = %q, want %q", password, database.Password)
	}
	if parsed.Path != "/data_migration" {
		t.Fatalf("path = %q", parsed.Path)
	}
}
