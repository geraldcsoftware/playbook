package doctor

import (
	"os"
	"testing"
)

func TestCheckBinary_Exists(t *testing.T) {
	result := CheckBinary("ls")
	if !result.OK {
		t.Errorf("expected ls to be found, got error: %s", result.Detail)
	}
	if result.Name != "ls" {
		t.Errorf("expected name 'ls', got '%s'", result.Name)
	}
}

func TestCheckBinary_NotExists(t *testing.T) {
	result := CheckBinary("nonexistent-binary-xyz-123")
	if result.OK {
		t.Error("expected binary to not be found")
	}
}

func TestCheckEnvVar_Set(t *testing.T) {
	t.Setenv("TEST_DOCTOR_VAR", "hello")
	result := CheckEnvVar("TEST_DOCTOR_VAR")
	if !result.OK {
		t.Error("expected env var to be set")
	}
}

func TestCheckEnvVar_NotSet(t *testing.T) {
	result := CheckEnvVar("DEFINITELY_NOT_SET_XYZ_123")
	if result.OK {
		t.Error("expected env var to not be set")
	}
}

func TestCheckFile_Exists(t *testing.T) {
	dir := t.TempDir()
	f := dir + "/test.txt"
	os.WriteFile(f, []byte("content"), 0644)

	result := CheckFile(f, "test file")
	if !result.OK {
		t.Errorf("expected file to exist: %s", result.Detail)
	}
}

func TestCheckFile_NotExists(t *testing.T) {
	result := CheckFile("/nonexistent/file.txt", "test file")
	if result.OK {
		t.Error("expected file to not exist")
	}
}

func TestRunAll(t *testing.T) {
	checks := []Check{
		CheckBinary("ls"),
		CheckEnvVar("PATH"),
	}
	allOK := true
	for _, c := range checks {
		if !c.OK {
			allOK = false
		}
	}
	if !allOK {
		t.Error("expected all basic checks to pass")
	}
}
