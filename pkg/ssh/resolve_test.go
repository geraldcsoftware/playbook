package ssh

import (
	"testing"
)

var testHosts = []SSHHost{
	{Alias: "db-prod.eus.v.co.zw", HostName: "db-prod.eus.v.co.zw", User: "gchifanzwa", IdentityFile: "~/.ssh/id_rsa_db_prod", Port: 22},
	{Alias: "db-staging.eus.v.co.zw", HostName: "db-staging.eus.v.co.zw", User: "gchifanzwa", IdentityFile: "~/.ssh/id_rsa_db_staging", Port: 22},
	{Alias: "web-01.eus.v.co.zw", HostName: "web-01.eus.v.co.zw", User: "deploy", IdentityFile: "~/.ssh/id_ed25519_web01", Port: 22},
}

func TestResolve_ExactMatch(t *testing.T) {
	results, err := Resolve("db-prod.eus.v.co.zw", testHosts, "gchifanzwa")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Hostname != "db-prod.eus.v.co.zw" {
		t.Errorf("expected hostname db-prod.eus.v.co.zw, got %s", results[0].Hostname)
	}
}

func TestResolve_SubstringUnique(t *testing.T) {
	results, err := Resolve("web-01", testHosts, "gchifanzwa")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Hostname != "web-01.eus.v.co.zw" {
		t.Errorf("expected web-01.eus.v.co.zw, got %s", results[0].Hostname)
	}
}

func TestResolve_SubstringAmbiguous(t *testing.T) {
	_, err := Resolve("db-", testHosts, "gchifanzwa")
	if err == nil {
		t.Fatal("expected ambiguous match error")
	}
	ambErr, ok := err.(*AmbiguousMatchError)
	if !ok {
		t.Fatalf("expected *AmbiguousMatchError, got %T", err)
	}
	if len(ambErr.Candidates) != 2 {
		t.Errorf("expected 2 candidates, got %d", len(ambErr.Candidates))
	}
}

func TestResolve_NoMatch(t *testing.T) {
	_, err := Resolve("nonexistent", testHosts, "gchifanzwa")
	if err == nil {
		t.Fatal("expected no match error")
	}
}

func TestResolve_FallbackUser(t *testing.T) {
	results, err := Resolve("web-01", testHosts, "gchifanzwa")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].User != "deploy" {
		t.Errorf("expected user from ssh config 'deploy', got %s", results[0].User)
	}
}

func TestResolve_DefaultUser(t *testing.T) {
	hosts := []SSHHost{
		{Alias: "bare-host", HostName: "bare-host.example.com", Port: 22},
	}
	results, err := Resolve("bare-host", hosts, "fallback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].User != "fallback" {
		t.Errorf("expected fallback user, got %s", results[0].User)
	}
}
