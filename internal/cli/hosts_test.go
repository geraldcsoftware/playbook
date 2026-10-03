package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestHostsAddCmd_MarkedDeprecated(t *testing.T) {
	cmd := newHostsAddCmd()
	if !strings.Contains(cmd.Short, "deprecated") {
		t.Errorf("expected short description to mention deprecation, got %q", cmd.Short)
	}
}

func TestPrintHostsAddDeprecation(t *testing.T) {
	var buf bytes.Buffer
	printHostsAddDeprecation(&buf)
	out := buf.String()

	if !strings.Contains(out, "\033[33m") {
		t.Error("expected warning to be printed in yellow")
	}
	if !strings.Contains(out, "deprecated") {
		t.Errorf("expected deprecation notice, got %q", out)
	}
	if !strings.Contains(out, "~/.ssh/config") {
		t.Errorf("expected pointer to ~/.ssh/config, got %q", out)
	}
}
