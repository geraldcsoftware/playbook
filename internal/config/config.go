package config

import (
	"errors"
	"fmt"
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

// ErrNotFound reports that no config file exists at the path given to Load.
var ErrNotFound = errors.New("config file not found")

// ParseError reports that the config file at Path exists but is not valid
// YAML for a Config.
type ParseError struct {
	Path string
	Err  error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("cannot parse %s: %v", e.Path, e.Err)
}

func (e *ParseError) Unwrap() error { return e.Err }

// Load reads the config file at path, filling anything it leaves unset
// with the built-in defaults. It always returns a usable Config: when the
// file cannot be used, the Config holds the defaults alone and the error
// says why. A missing file yields an error matching ErrNotFound, a file
// that is not valid YAML yields a *ParseError, and any other failure to
// read the file is returned as is. An empty path means no config file and
// yields the defaults with no error.
func Load(path string) (Config, error) {
	cfg := defaults()

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, fmt.Errorf("%w: %s", ErrNotFound, path)
		}
		return cfg, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return defaults(), &ParseError{Path: path, Err: err}
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
