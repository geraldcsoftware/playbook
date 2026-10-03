package ssh

import (
	"testing"
	"time"
)

func TestCheckReachability_Localhost(t *testing.T) {
	err := CheckReachability("127.0.0.1", 1, 2*time.Second)
	if err == nil {
		t.Error("expected unreachable error for port 1")
	}
}

func TestCheckReachability_InvalidHost(t *testing.T) {
	err := CheckReachability("host.invalid.test", 22, 2*time.Second)
	if err == nil {
		t.Error("expected error for invalid host")
	}
}

func TestPreflightResult_AllPassed(t *testing.T) {
	results := []HostPreflightResult{
		{Host: "a", Reachable: true, HostKeyVerified: true},
		{Host: "b", Reachable: true, HostKeyVerified: true},
	}
	if !AllPassed(results) {
		t.Error("expected all passed")
	}
}

func TestPreflightResult_OneFailed(t *testing.T) {
	results := []HostPreflightResult{
		{Host: "a", Reachable: true, HostKeyVerified: true},
		{Host: "b", Reachable: false, Error: "timeout"},
	}
	if AllPassed(results) {
		t.Error("expected not all passed")
	}
}

func TestRunPreflight_ChecksEachAliasOnce(t *testing.T) {
	hosts := []ResolvedHost{
		{Alias: "db01", Hostname: "127.0.0.1", Port: 1},
		{Alias: "web01", Hostname: "127.0.0.1", Port: 1},
		{Alias: "db01", Hostname: "127.0.0.1", Port: 1},
	}

	results := RunPreflight(hosts, 2*time.Second)
	if len(results) != 2 || results[0].Alias != "db01" || results[1].Alias != "web01" {
		t.Fatalf("results = %+v, want one each for db01 then web01", results)
	}
	if got := results[0].Address(); got != "127.0.0.1:1" {
		t.Errorf("Address() = %q, want 127.0.0.1:1", got)
	}
}
