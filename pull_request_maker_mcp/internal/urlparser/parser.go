package urlparser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/bitbucket"
)

var bitbucketPRRegex = regexp.MustCompile(
	`^https?://bitbucket\.org/([^/]+)/([^/]+)/pull-requests/(\d+)`,
)

// ParsePRURL extracts workspace, repo slug, and PR ID from a Bitbucket PR URL.
func ParsePRURL(rawURL string) (*bitbucket.PRInfo, error) {
	trimmed := strings.TrimSpace(rawURL)

	matches := bitbucketPRRegex.FindStringSubmatch(trimmed)
	if matches != nil {
		prID, _ := strconv.Atoi(matches[3])
		return &bitbucket.PRInfo{
			Platform:  "bitbucket",
			Workspace: matches[1],
			RepoSlug:  matches[2],
			PRId:      prID,
		}, nil
	}

	return nil, fmt.Errorf(
		"unsupported PR URL format: %s\nSupported formats:\n  - Bitbucket: https://bitbucket.org/{workspace}/{repo}/pull-requests/{id}",
		trimmed,
	)
}

// ParseRepoSlug splits a "workspace/repo" slug into workspace and repo parts.
func ParseRepoSlug(slug string) (workspace, repo string, err error) {
	parts := strings.SplitN(slug, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid repo slug format: %q (expected workspace/repo)", slug)
	}
	return parts[0], parts[1], nil
}
