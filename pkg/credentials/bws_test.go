package credentials

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
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

func TestBWSArgs_DisablesColourAndForcesJSON(t *testing.T) {
	got := BWSArgs("fake-token", "secret", "list")
	want := []string{"secret", "list", "--access-token", "fake-token", "--output", "json", "--color", "no"}

	if !slices.Equal(got, want) {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestBWSProvider_Fetch_PassesColourNoToBWS(t *testing.T) {
	dir := t.TempDir()
	fakeBWS := filepath.Join(dir, "bws")
	argsFile := filepath.Join(dir, "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + argsFile +
		"\necho '[{\"key\":\"MYANSIBLEPWD\",\"value\":\"ansible456\",\"id\":\"id1\"}]'\n"
	os.WriteFile(fakeBWS, []byte(script), 0755)

	p := &BWSProvider{
		AccessToken: "fake-token",
		SecretName:  "MYANSIBLEPWD",
		BinaryPath:  fakeBWS,
	}

	if _, err := p.Fetch(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recorded, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("reading recorded args: %v", err)
	}
	args := strings.Fields(string(recorded))

	if !slices.Contains(args, "--color") {
		t.Fatalf("expected --color in bws args, got %v", args)
	}
	i := slices.Index(args, "--color")
	if i+1 >= len(args) || args[i+1] != "no" {
		t.Errorf("expected '--color no', got %v", args[i:])
	}
}

// The stub honours --color no the way bws does, so this test fails if the flag
// is ever dropped: without it the stub emits ANSI-wrapped JSON and Fetch fails
// to parse the response.
func TestBWSProvider_Fetch_ColourFlagKeepsResponseParseable(t *testing.T) {
	dir := t.TempDir()
	fakeBWS := filepath.Join(dir, "bws")
	json := `[{"key":"MYANSIBLEPWD","value":"ansible456","id":"id1"}]`
	script := "#!/bin/sh\n" +
		"colour=1\n" +
		"prev=\"\"\n" +
		"for a in \"$@\"; do\n" +
		"  if [ \"$prev\" = \"--color\" ] && [ \"$a\" = \"no\" ]; then colour=0; fi\n" +
		"  prev=\"$a\"\n" +
		"done\n" +
		"if [ \"$colour\" = \"1\" ]; then\n" +
		"  printf '\\033[32m" + json + "\\033[0m\\n'\n" +
		"else\n" +
		"  printf '" + json + "\\n'\n" +
		"fi\n"
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
