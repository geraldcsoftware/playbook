package playbook

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Playbook struct {
	Name  string
	Hosts []string
	File  string
}

// Parse reads a playbook for Host Resolution, rejecting Playbook Hosts that
// are Ansible patterns rather than names an SSH Alias could match.
func Parse(path string) (Playbook, error) {
	return parse(path, validateHostPattern)
}

// ParseAnyPattern reads a playbook whose hosts come from an Explicit
// Inventory, so any Ansible host pattern is accepted.
func ParseAnyPattern(path string) (Playbook, error) {
	return parse(path, func(string) error { return nil })
}

func parse(path string, validate func(string) error) (Playbook, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Playbook{}, fmt.Errorf("reading playbook: %w", err)
	}

	var raw []map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Playbook{}, fmt.Errorf("parsing playbook YAML: %w", err)
	}

	if len(raw) == 0 {
		return Playbook{}, fmt.Errorf("playbook contains no plays")
	}

	first := raw[0]
	name, _ := first["name"].(string)

	hosts, err := extractHosts(first["hosts"], validate)
	if err != nil {
		return Playbook{}, err
	}

	return Playbook{
		Name:  name,
		Hosts: hosts,
		File:  path,
	}, nil
}

func extractHosts(v interface{}, validate func(string) error) ([]string, error) {
	if v == nil {
		return nil, fmt.Errorf("playbook has no 'hosts' field")
	}

	switch val := v.(type) {
	case string:
		if err := validate(val); err != nil {
			return nil, err
		}
		return []string{val}, nil
	case []interface{}:
		var hosts []string
		for _, item := range val {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("hosts list contains non-string value: %v", item)
			}
			if err := validate(s); err != nil {
				return nil, err
			}
			hosts = append(hosts, s)
		}
		return hosts, nil
	default:
		return nil, fmt.Errorf("unsupported hosts type: %T", v)
	}
}

func validateHostPattern(host string) error {
	if host == "all" {
		return fmt.Errorf("host pattern 'all' is not supported — this tool resolves individual hosts from ~/.ssh/config")
	}
	for _, ch := range []string{":", "&", "!", "*"} {
		if strings.Contains(host, ch) {
			return fmt.Errorf("host pattern '%s' contains '%s' — Ansible patterns are not supported, use explicit hostnames", host, ch)
		}
	}
	return nil
}
