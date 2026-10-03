package config

import (
	"errors"
	"os"
	"os/user"
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
	if cfg.DefaultUser != "" {
		t.Errorf("expected no built-in default user, got %s", cfg.DefaultUser)
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
	if cfg.DefaultUser != "" {
		t.Errorf("expected no built-in default user, got %s", cfg.DefaultUser)
	}
	if cfg.CredentialProvider != "aac" {
		t.Errorf("expected default provider aac, got %s", cfg.CredentialProvider)
	}
}

func TestEffectiveDefaultUser_ConfiguredValueWins(t *testing.T) {
	cfg := Config{DefaultUser: "deploy"}
	if got := cfg.EffectiveDefaultUser(); got != "deploy" {
		t.Errorf("expected deploy, got %s", got)
	}
}

func TestEffectiveDefaultUser_FallsBackToLocalAccount(t *testing.T) {
	stubCurrentUser(t, &user.User{Username: "local-operator"}, nil)
	t.Setenv("USER", "env-operator")

	if got := (Config{}).EffectiveDefaultUser(); got != "local-operator" {
		t.Errorf("expected local-operator, got %s", got)
	}
}

func TestEffectiveDefaultUser_FallsBackToUSERWhenLookupFails(t *testing.T) {
	stubCurrentUser(t, nil, errors.New("lookup failed"))
	t.Setenv("USER", "env-operator")

	if got := (Config{}).EffectiveDefaultUser(); got != "env-operator" {
		t.Errorf("expected env-operator, got %s", got)
	}
}

func stubCurrentUser(t *testing.T, u *user.User, err error) {
	t.Helper()
	orig := currentUser
	currentUser = func() (*user.User, error) { return u, err }
	t.Cleanup(func() { currentUser = orig })
}

func TestLoadConfig_BWSFields(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgPath, []byte("credential_provider: bws\nbws:\n  access_token_env: MY_BWS_TOKEN\n  secret_name: MYPASSWORD\n"), 0644)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.CredentialProvider != "bws" {
		t.Errorf("expected provider bws, got %s", cfg.CredentialProvider)
	}
	if cfg.BWS.AccessTokenEnv != "MY_BWS_TOKEN" {
		t.Errorf("expected access_token_env MY_BWS_TOKEN, got %s", cfg.BWS.AccessTokenEnv)
	}
	if cfg.BWS.SecretName != "MYPASSWORD" {
		t.Errorf("expected secret_name MYPASSWORD, got %s", cfg.BWS.SecretName)
	}
}

func TestLoadConfig_AACFields(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgPath, []byte("credential_provider: aac\naac:\n  item_id_env: MY_CUSTOM_VAR\n"), 0644)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AAC.ItemIDEnv != "MY_CUSTOM_VAR" {
		t.Errorf("expected item_id_env MY_CUSTOM_VAR, got %s", cfg.AAC.ItemIDEnv)
	}
}

func TestLoadConfig_DefaultAACItemIDEnv(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AAC.ItemIDEnv != "BW_EUS_ITEM_ID" {
		t.Errorf("expected default aac.item_id_env BW_EUS_ITEM_ID, got %s", cfg.AAC.ItemIDEnv)
	}
}

func TestLoadConfig_DefaultBWSAccessTokenEnv(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BWS.AccessTokenEnv != "BWS_ACCESS_TOKEN" {
		t.Errorf("expected default bws.access_token_env BWS_ACCESS_TOKEN, got %s", cfg.BWS.AccessTokenEnv)
	}
}
