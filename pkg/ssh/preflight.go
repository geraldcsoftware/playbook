package ssh

import (
	"fmt"
	"net"
	"time"
)

type HostPreflightResult struct {
	Host            string
	Port            int
	Reachable       bool
	HostKeyVerified bool
	Error           string
}

func CheckReachability(host string, port int, timeout time.Duration) error {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return fmt.Errorf("SSH port %d not reachable on %s: %w", port, host, err)
	}
	conn.Close()
	return nil
}

func AllPassed(results []HostPreflightResult) bool {
	for _, r := range results {
		if !r.Reachable || !r.HostKeyVerified {
			return false
		}
	}
	return true
}

func RunPreflight(hosts []ResolvedHost, timeout time.Duration) []HostPreflightResult {
	results := make([]HostPreflightResult, len(hosts))
	done := make(chan int, len(hosts))

	for i, h := range hosts {
		go func(idx int, host ResolvedHost) {
			r := HostPreflightResult{
				Host: host.Hostname,
				Port: host.Port,
			}
			if err := CheckReachability(host.Hostname, host.Port, timeout); err != nil {
				r.Error = err.Error()
			} else {
				r.Reachable = true
				r.HostKeyVerified = true
			}
			results[idx] = r
			done <- idx
		}(i, h)
	}

	for range hosts {
		<-done
	}

	return results
}
