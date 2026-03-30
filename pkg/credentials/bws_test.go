package credentials

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBWSProvider_Fetch_FindsSecretByName(t *testing.T) {
	dir := t.TempDir()
	fakeBWS := filepath.Join(dir, "bws")
	script := "#!/bin/sh\necho '[{\"key\":\"DBPASS\",\"value\":\"dbsecret123\",\"id\":\"id1\"},{\"key\":\"MYANSIBLEPWD\",\"value\":\"ansible456\",\"id\":\"id2\"}]'\n"
	os.WriteFile(fakeBWS, []byte(script), 0755)

	p := &BWSProvider{
		AccessToken: "fake-token",
		SecretName:  "MYANSIBLEPWD",
		BinaryPath:  fakeBWS,
	}

	password, err := p.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if password != "ansible456" {
		t.Errorf("expected 'ansible456', got '%s'", password)
	}
}

func TestBWSProvider_Fetch_SecretNotFound(t *testing.T) {
	dir := t.TempDir()
	fakeBWS := filepath.Join(dir, "bws")
	script := "#!/bin/sh\necho '[{\"key\":\"OTHER\",\"value\":\"val\",\"id\":\"id1\"}]'\n"
	os.WriteFile(fakeBWS, []byte(script), 0755)

	p := &BWSProvider{
		AccessToken: "fake-token",
		SecretName:  "NONEXISTENT",
		BinaryPath:  fakeBWS,
	}

	_, err := p.Fetch()
	if err == nil {
		t.Error("expected error for missing secret")
	}
}

func TestBWSProvider_Fetch_EmptyAccessToken(t *testing.T) {
	p := NewBWSProvider("", "MYSECRET")
	_, err := p.Fetch()
	if err == nil {
		t.Error("expected error for empty access token")
	}
}

func TestBWSProvider_Fetch_EmptySecretName(t *testing.T) {
	p := NewBWSProvider("some-token", "")
	_, err := p.Fetch()
	if err == nil {
		t.Error("expected error for empty secret name")
	}
}

func TestBWSProvider_Fetch_EmptySecretValue(t *testing.T) {
	dir := t.TempDir()
	fakeBWS := filepath.Join(dir, "bws")
	script := "#!/bin/sh\necho '[{\"key\":\"MYANSIBLEPWD\",\"value\":\"\",\"id\":\"id1\"}]'\n"
	os.WriteFile(fakeBWS, []byte(script), 0755)

	p := &BWSProvider{
		AccessToken: "fake-token",
		SecretName:  "MYANSIBLEPWD",
		BinaryPath:  fakeBWS,
	}

	_, err := p.Fetch()
	if err == nil {
		t.Error("expected error for empty secret value")
	}
}
