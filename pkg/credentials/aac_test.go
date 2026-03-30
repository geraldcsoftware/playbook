package credentials

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAACProvider_Fetch_EmptyItemID(t *testing.T) {
	p := NewAACProvider("")
	_, err := p.Fetch()
	if err == nil {
		t.Error("expected error for empty item ID")
	}
}

func TestAACProvider_Fetch_ParsesJSON(t *testing.T) {
	dir := t.TempDir()
	fakeAAC := filepath.Join(dir, "aac")
	script := "#!/bin/sh\necho '{\"credential\":{\"password\":\"secret123\",\"username\":\"user\"},\"success\":true}'\n"
	os.WriteFile(fakeAAC, []byte(script), 0755)

	p := &AACProvider{
		ItemID:     "test-item-id",
		BinaryPath: fakeAAC,
	}

	password, err := p.Fetch()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if password != "secret123" {
		t.Errorf("expected password 'secret123', got '%s'", password)
	}
}

func TestAACProvider_Fetch_FailsOnBadJSON(t *testing.T) {
	dir := t.TempDir()
	fakeAAC := filepath.Join(dir, "aac")
	script := "#!/bin/sh\necho 'not json'\n"
	os.WriteFile(fakeAAC, []byte(script), 0755)

	p := &AACProvider{
		ItemID:     "test-item-id",
		BinaryPath: fakeAAC,
	}

	_, err := p.Fetch()
	if err == nil {
		t.Error("expected error for bad JSON")
	}
}

func TestAACProvider_Fetch_FailsOnEmptyPassword(t *testing.T) {
	dir := t.TempDir()
	fakeAAC := filepath.Join(dir, "aac")
	script := "#!/bin/sh\necho '{\"credential\":{\"password\":\"\",\"username\":\"user\"},\"success\":true}'\n"
	os.WriteFile(fakeAAC, []byte(script), 0755)

	p := &AACProvider{
		ItemID:     "test-item-id",
		BinaryPath: fakeAAC,
	}

	_, err := p.Fetch()
	if err == nil {
		t.Error("expected error for empty password")
	}
}
