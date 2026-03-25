package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type AnsibleConfig struct {
	DefaultArgs []string `yaml:"default_args"`
}

type Config struct {
	DefaultUser        string        `yaml:"default_user"`
	CredentialProvider string        `yaml:"credential_provider"`
	Ansible            AnsibleConfig `yaml:"ansible"`
}

func defaults() Config {
	return Config{
		DefaultUser:        "gchifanzwa",
		CredentialProvider: "aac",
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

	if cfg.DefaultUser == "" {
		cfg.DefaultUser = defaults().DefaultUser
	}
	if cfg.CredentialProvider == "" {
		cfg.CredentialProvider = defaults().CredentialProvider
	}

	return cfg, nil
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return home + "/.config/playbook/config.yaml"
}
