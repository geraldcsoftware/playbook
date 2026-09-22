package credentials

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

type BWSProvider struct {
	AccessToken string
	SecretName  string
	BinaryPath  string // override for testing; defaults to "bws" via PATH lookup if empty
}

func NewBWSProvider(accessToken string, secretName string) *BWSProvider {
	return &BWSProvider{
		AccessToken: accessToken,
		SecretName:  secretName,
	}
}

// BWSArgs builds the argument list for a bws invocation, appending the global
// flags that every call site needs.
//
// Colour is disabled explicitly rather than left at the bws default of "auto":
// under auto, bws wraps its JSON in ANSI escape sequences whenever it decides
// it is writing to a terminal, or when a variable such as FORCE_COLOR or
// CLICOLOR_FORCE is set in the environment, and those escapes break parsing.
func BWSArgs(accessToken string, args ...string) []string {
	full := make([]string, 0, len(args)+6)
	full = append(full, args...)
	full = append(full, "--access-token", accessToken)
	full = append(full, "--output", "json")
	full = append(full, "--color", "no")
	return full
}

func (p *BWSProvider) bwsPath() (string, error) {
	if p.BinaryPath != "" {
		return p.BinaryPath, nil
	}
	return exec.LookPath("bws")
}

type bwsSecret struct {
	ID    string `json:"id"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (p *BWSProvider) Fetch() (string, error) {
	if p.AccessToken == "" {
		return "", fmt.Errorf("BWS access token is empty — set the access token env var or pass --access-token")
	}
	if p.SecretName == "" {
		return "", fmt.Errorf("BWS secret name is empty — set secret_name in config or pass --secret-name")
	}

	bwsBin, err := p.bwsPath()
	if err != nil {
		return "", fmt.Errorf("bws not found on PATH: %w", err)
	}

	cmd := exec.Command(bwsBin, BWSArgs(p.AccessToken, "secret", "list")...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("bws secret list failed: %w", err)
	}

	var secrets []bwsSecret
	if err := json.Unmarshal(out, &secrets); err != nil {
		return "", fmt.Errorf("parsing bws response: %w", err)
	}

	for _, s := range secrets {
		if s.Key == p.SecretName {
			if s.Value == "" {
				return "", fmt.Errorf("BWS secret '%s' exists but has an empty value", p.SecretName)
			}
			return s.Value, nil
		}
	}

	return "", fmt.Errorf("BWS secret '%s' not found — check the secret name and project access. Run 'playbook doctor' to diagnose", p.SecretName)
}
