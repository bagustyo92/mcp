package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"bitbucket.org/mid-kelola-indonesia/integration-deployment-tools-mcp/internal/bitbucket"
	"bitbucket.org/mid-kelola-indonesia/integration-deployment-tools-mcp/internal/config"
	"bitbucket.org/mid-kelola-indonesia/integration-deployment-tools-mcp/internal/gchat"
	"bitbucket.org/mid-kelola-indonesia/integration-deployment-tools-mcp/internal/tagversion"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CreateReleaseTagInput is the input schema.
type CreateReleaseTagInput struct {
	RepoSlug         string `json:"repo_slug" jsonschema:"required,Full repo slug (workspace/repo)"`
	TagName          string `json:"tag_name,omitempty" jsonschema:"Tag name to create. If empty, auto-increments patch from latest tag."`
	CommitHash       string `json:"commit_hash,omitempty" jsonschema:"Commit hash to tag. If empty, uses HEAD of default branch."`
	SendNotification bool   `json:"send_notification,omitempty" jsonschema:"Send notification to Google Chat (default false)"`
}

// CreateReleaseTagOutput is the structured output.
type CreateReleaseTagOutput struct {
	RepoSlug    string `json:"repo_slug"`
	TagName     string `json:"tag_name"`
	CommitHash  string `json:"commit_hash"`
	PreviousTag string `json:"previous_tag"`
	TagURL      string `json:"tag_url"`
}

// RegisterCreateReleaseTag registers the create_release_tag tool.
func RegisterCreateReleaseTag(server *mcp.Server, cfg *config.AppConfig, client *bitbucket.Client, notifier *gchat.Notifier) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name: "create_release_tag",
			Description: "Create a Git tag on a repository's default branch to mark a release. " +
				"When tag_name is empty the latest tag's patch version is auto-incremented. " +
				"When commit_hash is empty the HEAD of the default branch is used.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input CreateReleaseTagInput) (*mcp.CallToolResult, CreateReleaseTagOutput, error) {
			if input.RepoSlug == "" {
				return errorResult("repo_slug is required"), CreateReleaseTagOutput{}, nil
			}

			repoCfg := config.FindRepoConfig(cfg, input.RepoSlug)
			if repoCfg == nil {
				return errorResult(fmt.Sprintf("Repository %q not found in config", input.RepoSlug)), CreateReleaseTagOutput{}, nil
			}

			workspace, repo, err := parseRepoSlug(input.RepoSlug)
			if err != nil {
				return errorResult(err.Error()), CreateReleaseTagOutput{}, nil
			}

			branch := repoCfg.DefaultBranch
			if branch == "" {
				branch = "master"
			}

			// Determine previous tag
			latestTag, _ := client.GetLatestTag(ctx, cfg.Auth.Bitbucket, workspace, repo)
			previousTag := ""
			if latestTag != nil {
				previousTag = latestTag.Name
			}

			// Resolve tag name
			tagName := input.TagName
			if tagName == "" {
				if previousTag == "" {
					tagName = "v1.0.0"
				} else {
					sv, err := tagversion.ParseSemver(previousTag)
					if err != nil {
						return errorResult(fmt.Sprintf("Cannot auto-increment tag %q: %s", previousTag, err)), CreateReleaseTagOutput{}, nil
					}
					tagName = tagversion.IncrementPatch(sv)
				}
			}

			// Resolve commit hash
			commitHash := input.CommitHash
			if commitHash == "" {
				latest, err := client.GetLatestCommit(ctx, cfg.Auth.Bitbucket, workspace, repo, branch)
				if err != nil {
					return errorResult(fmt.Sprintf("Failed to get latest commit on %s: %s", branch, err)), CreateReleaseTagOutput{}, nil
				}
				commitHash = latest.Hash
			}

			// Create the tag
			createdTag, err := client.CreateTag(ctx, cfg.Auth.Bitbucket, workspace, repo, tagName, commitHash)
			if err != nil {
				return errorResult(fmt.Sprintf("Failed to create tag: %s", err)), CreateReleaseTagOutput{}, nil
			}

			tagURL := fmt.Sprintf("https://bitbucket.org/%s/%s/src/%s", workspace, repo, createdTag.Name)

			output := CreateReleaseTagOutput{
				RepoSlug:    input.RepoSlug,
				TagName:     createdTag.Name,
				CommitHash:  commitHash,
				PreviousTag: previousTag,
				TagURL:      tagURL,
			}

			// Notify
			if input.SendNotification {
				if cfg.GChat.WebhookURL != "" {
					msg := gchat.TagCreatedMessage{
						RepoSlug:    input.RepoSlug,
						TagName:     createdTag.Name,
						CommitHash:  commitHash,
						PreviousTag: previousTag,
						TagURL:      tagURL,
					}
					_ = notifier.NotifyTagCreated(ctx, cfg.GChat.WebhookURL, msg)
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
