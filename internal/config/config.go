package config

import (
	"os"
	"os/user"

	"gopkg.in/yaml.v3"
)

type AnsibleConfig struct {
	DefaultArgs []string `yaml:"default_args"`
}

type BWSConfig struct {
	AccessTokenEnv string `yaml:"access_token_env"`
	SecretName     string `yaml:"secret_name"`
}

type AACConfig struct {
	ItemIDEnv string `yaml:"item_id_env"`
}

type Config struct {
	DefaultUser        string        `yaml:"default_user"`
	CredentialProvider string        `yaml:"credential_provider"`
	Ansible            AnsibleConfig `yaml:"ansible"`
	BWS                BWSConfig     `yaml:"bws"`
	AAC                AACConfig     `yaml:"aac"`
}

func defaults() Config {
	return Config{
		CredentialProvider: "aac",
		AAC: AACConfig{
			ItemIDEnv: "BW_EUS_ITEM_ID",
		},
		BWS: BWSConfig{
			AccessTokenEnv: "BWS_ACCESS_TOKEN",
		},
	}
}

func Load(path string) (Config, error) {
	cfg := defaults()

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return defaults(), err
	}

	d := defaults()
	if cfg.CredentialProvider == "" {
		cfg.CredentialProvider = d.CredentialProvider
	}
	if cfg.AAC.ItemIDEnv == "" {
		cfg.AAC.ItemIDEnv = d.AAC.ItemIDEnv
	}
	if cfg.BWS.AccessTokenEnv == "" {
		cfg.BWS.AccessTokenEnv = d.BWS.AccessTokenEnv
	}

	return cfg, nil
}

// currentUser looks up the operating system account running playbook. It is
// a variable so tests can simulate a failed lookup.
var currentUser = user.Current

// EffectiveDefaultUser returns the Default User: default_user from the
// config file when set, otherwise the operator's operating system account
// name, falling back to $USER only if that lookup fails.
func (c Config) EffectiveDefaultUser() string {
	if c.DefaultUser != "" {
		return c.DefaultUser
	}
	if u, err := currentUser(); err == nil && u.Username != "" {
		return u.Username
	}
	return os.Getenv("USER")
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return home + "/.config/playbook/config.yaml"
}
