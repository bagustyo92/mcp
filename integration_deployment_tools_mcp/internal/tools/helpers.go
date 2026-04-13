package tools

import (
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// errorResult creates a CallToolResult with IsError set to true.
func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
		IsError: true,
	}
}

// parseRepoSlug splits "workspace/repo" into its two parts.
func parseRepoSlug(slug string) (workspace, repo string, err error) {
	parts := strings.SplitN(slug, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid repo slug %q: expected workspace/repo", slug)
	}
	return parts[0], parts[1], nil
}

// resolveWebhookURL returns the per-repo webhook URL or falls back to the global one.
func resolveWebhookURL(repoURL, globalURL string) string {
	if repoURL != "" {
		return repoURL
	}
	return globalURL
}
