package ssh

import (
	"slices"
	"testing"
)

func TestSuggestAliases(t *testing.T) {
	hosts := []SSHHost{
		{Aliases: []string{"web-prod-01", "webmail"}},
		{Aliases: []string{"web-prod-02"}},
		{Aliases: []string{"web-staging"}},
		{Aliases: []string{"db01"}},
		{Aliases: []string{"mail"}},
	}

	tests := []struct {
		playbookHost string
		want         []string
	}{
		// Aliases containing it, nearest first then alphabetically, capped.
		{"web", []string{"webmail", "web-prod-01", "web-prod-02"}},
		{"web-prod-1", []string{"web-prod-01", "web-prod-02"}},
		{"db1", []string{"db01"}},
		{"mial", []string{"mail"}},
		// Two edits is too many for a three-character name.
		{"dbx", nil},
		{"nothing-like-it", nil},
	}
	for _, tt := range tests {
		if got := suggestAliases(tt.playbookHost, hosts); !slices.Equal(got, tt.want) {
			t.Errorf("suggestAliases(%q) = %q, want %q", tt.playbookHost, got, tt.want)
		}
	}
}

func TestEditDistance(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"db01", "db01", 0},
		{"db1", "db01", 1},
		{"mial", "mail", 1},
		{"kitten", "sitting", 3},
		{"", "abc", 3},
	}
	for _, tt := range tests {
		if got := editDistance(tt.a, tt.b); got != tt.want {
			t.Errorf("editDistance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
