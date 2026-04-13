package jira

import (
	"regexp"
	"strings"
)

var jiraKeyRegex = regexp.MustCompile(`[A-Z][A-Z0-9]+-\d+`)

// ExtractTicket extracts the first Jira ticket key from the given sources.
// It tries each source in order and returns the first match.
// Returns "unknown ticket" if no match is found.
func ExtractTicket(sources ...string) string {
	for _, s := range sources {
		upper := strings.ToUpper(s)
		match := jiraKeyRegex.FindString(upper)
		if match != "" {
			return match
		}
	}
	return "unknown ticket"
}

// BuildJiraURL constructs a full Jira URL from a base URL and ticket key.
// Returns empty string if baseURL or ticket is empty, or if ticket is "unknown ticket".
func BuildJiraURL(baseURL, ticket string) string {
	if baseURL == "" || ticket == "" || ticket == "unknown ticket" {
		return ""
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return baseURL + "/" + ticket
}
