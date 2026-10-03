package ssh

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

// HostPreflightResult is the outcome of the pre-flight check for one
// Target: whether its SSH Alias's effective address and port accept a TCP
// connection.
type HostPreflightResult struct {
	Alias           string
	Host            string
	Port            int
	Reachable       bool
	HostKeyVerified bool
	Error           string
}

// Address is the host:port the check dialled.
func (r HostPreflightResult) Address() string {
	return net.JoinHostPort(r.Host, strconv.Itoa(r.Port))
}

func CheckReachability(host string, port int, timeout time.Duration) error {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
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

// RunPreflight checks, concurrently, that each Target accepts a TCP
// connection at the address and port OpenSSH effectively applies to it. A
// Target listed more than once is checked once; results follow the order
// in which each SSH Alias first appears.
func RunPreflight(hosts []ResolvedHost, timeout time.Duration) []HostPreflightResult {
	var unique []ResolvedHost
	seen := map[string]bool{}
	for _, h := range hosts {
		if !seen[h.Alias] {
			seen[h.Alias] = true
			unique = append(unique, h)
		}
	}

	results := make([]HostPreflightResult, len(unique))
	done := make(chan struct{}, len(unique))

	for i, h := range unique {
		go func() {
			r := HostPreflightResult{Alias: h.Alias, Host: h.Hostname, Port: h.Port}
			if err := CheckReachability(h.Hostname, h.Port, timeout); err != nil {
				r.Error = err.Error()
			} else {
				r.Reachable = true
				r.HostKeyVerified = true
			}
			results[i] = r
			done <- struct{}{}
		}()
	}

	for range unique {
		<-done
	}

	return results
}
