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

func TestDoctorCmd_Runs(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"doctor"})
	// Just verify it doesn't panic — some checks may fail depending on installed tools
	_ = root.Execute()
}
