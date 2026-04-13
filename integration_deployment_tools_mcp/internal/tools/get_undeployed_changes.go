package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"bitbucket.org/mid-kelola-indonesia/integration-deployment-tools-mcp/internal/bitbucket"
	"bitbucket.org/mid-kelola-indonesia/integration-deployment-tools-mcp/internal/config"
	"bitbucket.org/mid-kelola-indonesia/integration-deployment-tools-mcp/internal/gchat"
	"bitbucket.org/mid-kelola-indonesia/integration-deployment-tools-mcp/internal/jira"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GetUndeployedChangesInput is the input schema.
type GetUndeployedChangesInput struct {
	RepoSlug         string `json:"repo_slug,omitempty" jsonschema:"Full repo slug (workspace/repo). Empty means all configured repos."`
	SendNotification bool   `json:"send_notification,omitempty" jsonschema:"Send results to Google Chat (default false)"`
}

// UndeployedChange represents a single undeployed change.
type UndeployedChange struct {
	JiraTicket string `json:"jira_ticket"`
	JiraURL    string `json:"jira_url,omitempty"`
	PRURL      string `json:"pr_url,omitempty"`
	PRTitle    string `json:"pr_title,omitempty"`
	Author     string `json:"author"`
	CommitHash string `json:"commit_hash"`
	Summary    string `json:"summary"`
}

// RepoUndeployedChanges holds undeployed changes for a single repository.
type RepoUndeployedChanges struct {
	RepoSlug         string             `json:"repo_slug"`
	LatestTag        string             `json:"latest_tag"`
	TotalUndeployed  int                `json:"total_undeployed"`
	Changes          []UndeployedChange `json:"changes"`
	NotificationSent bool               `json:"notification_sent"`
	Error            string             `json:"error,omitempty"`
}

// GetUndeployedChangesOutput is the structured output.
type GetUndeployedChangesOutput struct {
	Repositories []RepoUndeployedChanges `json:"repositories"`
}

// RegisterGetUndeployedChanges registers the get_undeployed_changes tool.
func RegisterGetUndeployedChanges(server *mcp.Server, cfg *config.AppConfig, client *bitbucket.Client, notifier *gchat.Notifier) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name: "get_undeployed_changes",
			Description: "Identify code changes that have been merged to the default branch but not yet " +
				"deployed (no production tag). Returns a per-repo list of undeployed PRs with Jira tickets. " +
				"Optionally sends a summary to Google Chat.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input GetUndeployedChangesInput) (*mcp.CallToolResult, GetUndeployedChangesOutput, error) {
			// Resolve target repos
			var targetRepos []config.RepoConfig
			if input.RepoSlug != "" {
				repoCfg := config.FindRepoConfig(cfg, input.RepoSlug)
				if repoCfg == nil {
					return errorResult(fmt.Sprintf("Repository %q not found in config", input.RepoSlug)), GetUndeployedChangesOutput{}, nil
				}
				targetRepos = []config.RepoConfig{*repoCfg}
			} else {
				for _, r := range cfg.Repositories {
					if r.RepoSlug != "*" {
						targetRepos = append(targetRepos, r)
					}
				}
			}

			if len(targetRepos) == 0 {
				return errorResult("No repositories configured"), GetUndeployedChangesOutput{}, nil
			}

			output := GetUndeployedChangesOutput{
				Repositories: make([]RepoUndeployedChanges, 0, len(targetRepos)),
			}

			for _, repoCfg := range targetRepos {
				result := processRepoUndeployed(ctx, cfg, client, repoCfg)
				output.Repositories = append(output.Repositories, result)
			}

			// Send a single combined notification covering every repo (including up-to-date ones).
			if input.SendNotification && cfg.GChat.WebhookURL != "" {
				summaries := make([]gchat.RepoNotifSummary, 0, len(output.Repositories))
				for _, r := range output.Repositories {
					entries := make([]gchat.ChangeEntry, 0, len(r.Changes))
					for _, c := range r.Changes {
						entries = append(entries, gchat.ChangeEntry{
							JiraTicket: c.JiraTicket,
							JiraURL:    c.JiraURL,
							PRTitle:    c.PRTitle,
							PRURL:      c.PRURL,
							Author:     c.Author,
							PRSummary:  c.Summary,
						})
					}
					summaries = append(summaries, gchat.RepoNotifSummary{
						RepoSlug:  r.RepoSlug,
						LatestTag: r.LatestTag,
						Changes:   entries,
						Error:     r.Error,
					})
				}
				// Mark all repos as notification-sent on success
				if err := notifier.NotifyAllReposUndeployed(ctx, cfg.GChat.WebhookURL, summaries); err == nil {
					for i := range output.Repositories {
						output.Repositories[i].NotificationSent = true
					}
				}
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

func processRepoUndeployed(ctx context.Context, cfg *config.AppConfig, client *bitbucket.Client, repoCfg config.RepoConfig) RepoUndeployedChanges {
	result := RepoUndeployedChanges{RepoSlug: repoCfg.RepoSlug}

	workspace, repo, err := parseRepoSlug(repoCfg.RepoSlug)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	branch := repoCfg.DefaultBranch
	if branch == "" {
		branch = "master"
	}

	// Get latest tag
	latestTag, err := client.GetLatestTag(ctx, cfg.Auth.Bitbucket, workspace, repo)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get latest tag: %s", err)
		return result
	}

	if latestTag == nil {
		result.LatestTag = "(none)"
		// No tag means all commits are undeployed; fetch last 30
		commits, err := client.ListCommits(ctx, cfg.Auth.Bitbucket, workspace, repo, branch, 30)
		if err != nil {
			result.Error = fmt.Sprintf("failed to list commits: %s", err)
			return result
		}
		result.Changes = commitsToChanges(ctx, cfg, client, workspace, repo, commits)
	} else {
		result.LatestTag = latestTag.Name

		// Get commits after the latest tag
		commits, err := client.GetCommitsAfterHash(ctx, cfg.Auth.Bitbucket, workspace, repo, branch, latestTag.Hash)
		if err != nil {
			result.Error = fmt.Sprintf("failed to get commits after tag: %s", err)
			return result
		}
		result.Changes = commitsToChanges(ctx, cfg, client, workspace, repo, commits)
	}

	result.TotalUndeployed = len(result.Changes)

	return result
}

func commitsToChanges(ctx context.Context, cfg *config.AppConfig, client *bitbucket.Client, workspace, repo string, commits []bitbucket.Commit) []UndeployedChange {
	seen := make(map[string]bool)
	var changes []UndeployedChange

	for _, commit := range commits {
		// Try to find associated PR
		prs, _ := client.GetPRsForCommit(ctx, cfg.Auth.Bitbucket, workspace, repo, commit.Hash)

		if len(prs) > 0 {
			pr := prs[0]
			ticket := jira.ExtractTicket(pr.SourceBranch, pr.Title, pr.Description, commit.Message)
			if seen[ticket] {
				continue
			}
			seen[ticket] = true

			changes = append(changes, UndeployedChange{
				JiraTicket: ticket,
				JiraURL:    jira.BuildJiraURL(cfg.GChat.JiraBaseURL, ticket),
				PRURL:      pr.URL,
				PRTitle:    pr.Title,
				Author:     pr.Author,
				CommitHash: commit.Hash,
				Summary:    truncateString(firstLineOf(pr.Description), 120),
			})
		} else {
			// No PR found — use commit info
			ticket := jira.ExtractTicket(commit.Message)
			if seen[ticket] {
				continue
			}
			seen[ticket] = true

			firstLine := firstLineOf(commit.Message)
			changes = append(changes, UndeployedChange{
				JiraTicket: ticket,
				JiraURL:    jira.BuildJiraURL(cfg.GChat.JiraBaseURL, ticket),
				Author:     commit.Author,
				CommitHash: commit.Hash,
				Summary:    truncateString(firstLine, 120),
			})
		}
	}

	return changes
}

func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}

func firstLineOf(s string) string {
	for i, c := range s {
		if c == '\n' {
			return s[:i]
		}
	}
	return s
}
