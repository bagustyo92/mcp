package jira

import "testing"

func TestExtractTicket(t *testing.T) {
	tests := []struct {
		name    string
		sources []string
		want    string
	}{
		{
			name:    "from branch name",
			sources: []string{"feature/TD-7900-add-payslips"},
			want:    "TD-7900",
		},
		{
			name:    "from PR title",
			sources: []string{"no-match", "(TD-1234) Add new endpoint"},
			want:    "TD-1234",
		},
		{
			name:    "from commit message",
			sources: []string{"no-match", "no-match-either", "fix: PLAT-456 resolve issue"},
			want:    "PLAT-456",
		},
		{
			name:    "no match returns unknown",
			sources: []string{"no-jira-here", "also nothing"},
			want:    "unknown ticket",
		},
		{
			name:    "empty sources",
			sources: []string{},
			want:    "unknown ticket",
		},
		{
			name:    "first source wins",
			sources: []string{"feature/ABC-111", "DEF-222 title"},
			want:    "ABC-111",
		},
		{
			name:    "case insensitive input",
			sources: []string{"feature/td-100-something"},
			want:    "TD-100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTicket(tt.sources...)
			if got != tt.want {
				t.Errorf("ExtractTicket(%v) = %q, want %q", tt.sources, got, tt.want)
			}
		})
	}
}

func TestBuildJiraURL(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		ticketKey string
		want      string
	}{
		{
			name:      "valid URL",
			baseURL:   "https://mekari.atlassian.net/browse",
			ticketKey: "TD-123",
			want:      "https://mekari.atlassian.net/browse/TD-123",
		},
		{
			name:      "trailing slash trimmed",
			baseURL:   "https://mekari.atlassian.net/browse/",
			ticketKey: "TD-123",
			want:      "https://mekari.atlassian.net/browse/TD-123",
		},
		{
			name:      "empty base URL",
			baseURL:   "",
			ticketKey: "TD-123",
			want:      "",
		},
		{
			name:      "unknown ticket",
			baseURL:   "https://mekari.atlassian.net/browse",
			ticketKey: "unknown ticket",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildJiraURL(tt.baseURL, tt.ticketKey)
			if got != tt.want {
				t.Errorf("BuildJiraURL(%q, %q) = %q, want %q", tt.baseURL, tt.ticketKey, got, tt.want)
			}
		})
	}
}
