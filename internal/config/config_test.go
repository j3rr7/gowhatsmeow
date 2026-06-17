package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	cfgContent := `
server:
  port: "9090"
  host: "localhost"
database:
  type: "postgres"
  url: "postgres://root@localhost/test"
logger:
  level: "error"
  encoding: "json"
`
	err := os.WriteFile("config.yaml", []byte(cfgContent), 0644)
	if err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}
	defer os.Remove("config.yaml")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Server.Port != "9090" {
		t.Errorf("expected server port 9090, got %s", cfg.Server.Port)
	}
	if cfg.Server.Host != "localhost" {
		t.Errorf("expected server host localhost, got %s", cfg.Server.Host)
	}
	if cfg.Database.Type != "postgres" {
		t.Errorf("expected database type postgres, got %s", cfg.Database.Type)
	}
	if cfg.Database.URL != "postgres://root@localhost/test" {
		t.Errorf("expected database url, got %s", cfg.Database.URL)
	}
	if cfg.Logger.Level != "error" {
		t.Errorf("expected logger level error, got %s", cfg.Logger.Level)
	}
	if cfg.Logger.Encoding != "json" {
		t.Errorf("expected logger encoding json, got %s", cfg.Logger.Encoding)
	}
}
