package ssh

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// EffectiveSettings are the connection settings OpenSSH applies to an SSH
// Alias once its whole client configuration has been evaluated, keyed by
// lower-case keyword.
type EffectiveSettings map[string]string

// User is the user OpenSSH connects to the SSH Alias as.
func (s EffectiveSettings) User() string { return s["user"] }

// EffectiveSettingsLookup reports the settings OpenSSH applies to an SSH
// Alias.
type EffectiveSettingsLookup interface {
	EffectiveSettings(alias string) (EffectiveSettings, error)
}

// OpenSSHLookup asks the ssh binary on PATH, with `ssh -G <alias>`.
type OpenSSHLookup struct{}

func (OpenSSHLookup) EffectiveSettings(alias string) (EffectiveSettings, error) {
	out, err := exec.Command("ssh", "-G", alias).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if stderr := strings.TrimSpace(string(exitErr.Stderr)); stderr != "" {
				return nil, fmt.Errorf("ssh -G %s: %s", alias, stderr)
			}
		}
		return nil, fmt.Errorf("ssh -G %s: %w", alias, err)
	}
	return parseEffectiveSettings(out), nil
}

// parseEffectiveSettings reads `ssh -G` output: one "keyword value" pair
// per line. A keyword listed more than once keeps its first value.
func parseEffectiveSettings(out []byte) EffectiveSettings {
	settings := EffectiveSettings{}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		keyword, value, _ := strings.Cut(strings.TrimSpace(scanner.Text()), " ")
		if keyword == "" {
			continue
		}
		keyword = strings.ToLower(keyword)
		if _, seen := settings[keyword]; !seen {
			settings[keyword] = strings.TrimSpace(value)
		}
	}
	return settings
}
