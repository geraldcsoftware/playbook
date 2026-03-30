package cli

import (
	"testing"
)

func TestRunCmd_RequiresPlaybookArg(t *testing.T) {
	cmd := newRunCmd()
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for missing playbook argument")
	}
}

func TestRunCmd_InvalidPlaybookFile(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"run", "/nonexistent/playbook.yml"})
	err := root.Execute()
	if err == nil {
		t.Error("expected error for nonexistent playbook")
	}
}

func TestRunCmd_HasProviderFlags(t *testing.T) {
	cmd := newRunCmd()

	flags := []struct {
		name      string
		shorthand string
	}{
		{"credential-provider", "p"},
		{"secret-name", "s"},
		{"access-token", "t"},
	}

	for _, f := range flags {
		flag := cmd.Flags().Lookup(f.name)
		if flag == nil {
			t.Errorf("expected flag --%s to exist", f.name)
			continue
		}
		if flag.Shorthand != f.shorthand {
			t.Errorf("expected --%s shorthand to be -%s, got -%s", f.name, f.shorthand, flag.Shorthand)
		}
	}
}

func TestRunCmd_UnknownProvider(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"run", "/nonexistent/playbook.yml", "--credential-provider", "invalid"})
	err := root.Execute()
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestDoctorCmd_Runs(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"doctor"})
	// Just verify it doesn't panic — some checks may fail depending on installed tools
	_ = root.Execute()
}
