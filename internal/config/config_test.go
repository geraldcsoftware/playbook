package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_FromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgPath, []byte("default_user: testuser\ncredential_provider: aac\n"), 0644)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultUser != "testuser" {
		t.Errorf("expected DefaultUser=testuser, got %s", cfg.DefaultUser)
	}
	if cfg.CredentialProvider != "aac" {
		t.Errorf("expected CredentialProvider=aac, got %s", cfg.CredentialProvider)
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultUser != "gchifanzwa" {
		t.Errorf("expected default user gchifanzwa, got %s", cfg.DefaultUser)
	}
	if cfg.CredentialProvider != "aac" {
		t.Errorf("expected default provider aac, got %s", cfg.CredentialProvider)
	}
}

func TestLoadConfig_MissingFileUsesDefaults(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultUser != "gchifanzwa" {
		t.Errorf("expected default user, got %s", cfg.DefaultUser)
	}
}
