package credentials

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

type AACProvider struct {
	ItemID     string
	BinaryPath string // override for testing; defaults to "aac" via PATH lookup if empty
}

func NewAACProvider(itemID string) *AACProvider {
	return &AACProvider{ItemID: itemID}
}

func (p *AACProvider) aacPath() (string, error) {
	if p.BinaryPath != "" {
		return p.BinaryPath, nil
	}
	return exec.LookPath("aac")
}

type aacResponse struct {
	Credential struct {
		Password string `json:"password"`
		Username string `json:"username"`
	} `json:"credential"`
	Success bool `json:"success"`
}

func (p *AACProvider) Fetch() (string, error) {
	if p.ItemID == "" {
		return "", fmt.Errorf("credential item ID is empty — set the item ID env var or check your config")
	}

	aacBin, err := p.aacPath()
	if err != nil {
		return "", fmt.Errorf("aac not found on PATH: %w", err)
	}

	cmd := exec.Command(aacBin, "connect", "--id", p.ItemID, "--output", "json")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("aac connect failed: %w", err)
	}

	var resp aacResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return "", fmt.Errorf("parsing aac response: %w", err)
	}

	if resp.Credential.Password == "" {
		return "", fmt.Errorf("aac returned empty password for item %s", p.ItemID)
	}

	return resp.Credential.Password, nil
}
