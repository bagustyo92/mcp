package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"bitbucket.org/mid-kelola-indonesia/integration-deployment-tools-mcp/internal/bitbucket"
	"bitbucket.org/mid-kelola-indonesia/integration-deployment-tools-mcp/internal/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GetChangesByTagInput is the input schema.
type GetChangesByTagInput struct {
	RepoSlug  string `json:"repo_slug" jsonschema:"required,Full repo slug (workspace/repo)"`
	TagName   string `json:"tag_name,omitempty" jsonschema:"Tag to inspect. If empty, uses the latest tag."`
	FromTag   string `json:"from_tag,omitempty" jsonschema:"Baseline tag to compare from. If empty, automatically uses the previous tag before tag_name."`
	ListTags  bool   `json:"list_tags,omitempty" jsonschema:"If true, returns the list of recent tags so you can pick a version to inspect. No changes are returned."`
	TagsLimit int    `json:"tags_limit,omitempty" jsonschema:"Number of recent tags to return when list_tags=true (default 10, max 50)."`
}

// GetChangesByTagOutput is the structured output.
type GetChangesByTagOutput struct {
	RepoSlug     string             `json:"repo_slug"`
	TagName      string             `json:"tag_name,omitempty"`
	FromTag      string             `json:"from_tag,omitempty"`
	TotalChanges int                `json:"total_changes,omitempty"`
	Changes      []UndeployedChange `json:"changes,omitempty"`
	Tags         []bitbucket.Tag    `json:"tags,omitempty"`
	Error        string             `json:"error,omitempty"`
}

// RegisterGetChangesByTag registers the get_changes_by_tag tool.
func RegisterGetChangesByTag(server *mcp.Server, cfg *config.AppConfig, client *bitbucket.Client) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name: "get_changes_by_tag",
			Description: "List Jira tickets and PRs included in a specific release tag for a repository. " +
				"Supports comparing between any two tags for full flexibility. " +
				"Use list_tags=true first to explore available tags, then specify tag_name (and optionally from_tag) to get the changes between them. " +
				"When from_tag is omitted, the previous tag is used automatically.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input GetChangesByTagInput) (*mcp.CallToolResult, GetChangesByTagOutput, error) {
			if input.RepoSlug == "" {
				return errorResult("repo_slug is required"), GetChangesByTagOutput{}, nil
			}

			repoCfg := config.FindRepoConfig(cfg, input.RepoSlug)
			if repoCfg == nil {
				return errorResult(fmt.Sprintf("Repository %q not found in config", input.RepoSlug)), GetChangesByTagOutput{}, nil
			}

			workspace, repo, err := parseRepoSlug(input.RepoSlug)
			if err != nil {
				return errorResult(err.Error()), GetChangesByTagOutput{}, nil
			}

			// list_tags mode: return tag list so the model can pick a version
			if input.ListTags {
				limit := input.TagsLimit
				if limit <= 0 {
					limit = 10
				}
				if limit > 50 {
					limit = 50
				}
				tags, err := client.ListTags(ctx, cfg.Auth.Bitbucket, workspace, repo, limit)
				if err != nil {
					return errorResult(fmt.Sprintf("Failed to list tags: %s", err)), GetChangesByTagOutput{}, nil
				}
				output := GetChangesByTagOutput{
					RepoSlug: input.RepoSlug,
					Tags:     tags,
				}
				resultJSON, _ := json.Marshal(output)
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{Text: string(resultJSON)},
					},
				}, output, nil
			}

			// Resolve the target tag
			var targetTag *bitbucket.Tag
			if input.TagName == "" {
				targetTag, err = client.GetLatestTag(ctx, cfg.Auth.Bitbucket, workspace, repo)
				if err != nil {
					return errorResult(fmt.Sprintf("Failed to get latest tag: %s", err)), GetChangesByTagOutput{}, nil
				}
				if targetTag == nil {
					return errorResult("No tags found in this repository"), GetChangesByTagOutput{}, nil
				}
			} else {
				targetTag, err = client.GetTagByName(ctx, cfg.Auth.Bitbucket, workspace, repo, input.TagName)
				if err != nil {
					return errorResult(fmt.Sprintf("Failed to get tag %q: %s", input.TagName, err)), GetChangesByTagOutput{}, nil
				}
				if targetTag == nil {
					return errorResult(fmt.Sprintf("Tag %q not found", input.TagName)), GetChangesByTagOutput{}, nil
				}
			}

			// Resolve the baseline tag
			var fromTagName string
			var fromRef string
			if input.FromTag != "" {
				fromTag, err := client.GetTagByName(ctx, cfg.Auth.Bitbucket, workspace, repo, input.FromTag)
				if err != nil {
					return errorResult(fmt.Sprintf("Failed to get from_tag %q: %s", input.FromTag, err)), GetChangesByTagOutput{}, nil
				}
				if fromTag == nil {
					return errorResult(fmt.Sprintf("from_tag %q not found", input.FromTag)), GetChangesByTagOutput{}, nil
				}
				fromTagName = fromTag.Name
				fromRef = fromTag.Hash
			} else {
				prevTag, err := client.GetPreviousTag(ctx, cfg.Auth.Bitbucket, workspace, repo, targetTag.Name)
				if err != nil {
					return errorResult(fmt.Sprintf("Failed to find previous tag: %s", err)), GetChangesByTagOutput{}, nil
				}
				if prevTag != nil {
					fromTagName = prevTag.Name
					fromRef = prevTag.Hash
				}
			}

			// Fetch commits between the two tags
			var commits []bitbucket.Commit
			if fromRef == "" {
				// No baseline — list commits up to the tag (last 50)
				branch := repoCfg.DefaultBranch
				if branch == "" {
					branch = "master"
				}
				commits, err = client.GetCommitsAfterHash(ctx, cfg.Auth.Bitbucket, workspace, repo, branch, "")
				if err != nil {
					return errorResult(fmt.Sprintf("Failed to list commits: %s", err)), GetChangesByTagOutput{}, nil
				}
				// Trim to commits up to and including the target tag hash
				commits = commitsUpToHash(commits, targetTag.Hash)
			} else {
				commits, err = client.GetCommitsBetweenHashes(ctx, cfg.Auth.Bitbucket, workspace, repo, fromRef, targetTag.Hash)
				if err != nil {
					return errorResult(fmt.Sprintf("Failed to get commits between tags: %s", err)), GetChangesByTagOutput{}, nil
				}
			}

			changes := commitsToChanges(ctx, cfg, client, workspace, repo, commits)

			output := GetChangesByTagOutput{
				RepoSlug:     input.RepoSlug,
				TagName:      targetTag.Name,
				FromTag:      fromTagName,
				TotalChanges: len(changes),
				Changes:      changes,
			}

			resultJSON, _ := json.Marshal(output)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: string(resultJSON)},
				},
			}, output, nil
		},
	)
}

// commitsUpToHash returns commits from the slice up to (but not including) the given hash.
// The slice is assumed to be in newest-first order.
func commitsUpToHash(commits []bitbucket.Commit, hash string) []bitbucket.Commit {
	if hash == "" {
		return commits
	}
	for i, c := range commits {
		if len(c.Hash) >= 12 && len(hash) >= 12 && c.Hash[:12] == hash[:12] {
			return commits[:i+1] // include the tagged commit itself
		}
	}
	return commits
}
