package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/bitbucket"
	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/config"
	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/urlparser"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GetPRDiffInput is the input schema for the get_pr_diff tool.
type GetPRDiffInput struct {
	PRURL string `json:"pr_url" jsonschema:"The full URL of the Bitbucket pull request"`
}

// GetPRDiffOutput is the structured output returned by get_pr_diff.
type GetPRDiffOutput struct {
	PRURL              string   `json:"pr_url"`
	Platform           string   `json:"platform"`
	Workspace          string   `json:"workspace"`
	RepoSlug           string   `json:"repo_slug"`
	PRId               int      `json:"pr_id"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	SourceBranch       string   `json:"source_branch"`
	TargetBranch       string   `json:"target_branch"`
	Author             string   `json:"author"`
	State              string   `json:"state"`
	ChangedFiles       []string `json:"changed_files"`
	Diff               string   `json:"diff"`
	ReviewInstructions string   `json:"review_instructions"`
	ProjectGuidelines  string   `json:"project_guidelines"`
	InstructionsToLLM  string   `json:"instructions_to_llm"`
}

// RegisterGetPRDiff registers the get_pr_diff tool on the MCP server.
func RegisterGetPRDiff(server *mcp.Server, cfg *config.AppConfig, client *bitbucket.Client) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name: "get_pr_diff",
			Description: "Fetch the diff, metadata, review instructions, and project guidelines for a Bitbucket pull request. " +
				"Use this as the first step when reviewing a PR. The returned data includes everything needed to perform a thorough code review.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input GetPRDiffInput) (*mcp.CallToolResult, GetPRDiffOutput, error) {
			prInfo, err := urlparser.ParsePRURL(input.PRURL)
			if err != nil {
				return errorResult(err.Error()), GetPRDiffOutput{}, nil
			}

			projectCfg := config.FindProjectConfig(cfg, prInfo.Workspace, prInfo.RepoSlug)

			reviewInstructions := ""
			projectGuidelines := ""
			if projectCfg != nil {
				reviewInstructions = config.LoadFileContent(projectCfg.ReviewInstructions)
				projectGuidelines = config.LoadFileContent(projectCfg.ProjectGuidelines)
			}

			metadata, err := client.FetchPRMetadata(ctx, cfg.Auth.Bitbucket, *prInfo)
			if err != nil {
				return errorResult(fmt.Sprintf("Error fetching PR metadata: %s", err)), GetPRDiffOutput{}, nil
			}

			diff, err := client.FetchBranchDiff(ctx, cfg.Auth.Bitbucket, prInfo.Workspace, prInfo.RepoSlug, metadata.SourceBranch, metadata.TargetBranch)
			if err != nil {
				return errorResult(fmt.Sprintf("Error fetching diff: %s", err)), GetPRDiffOutput{}, nil
			}

			changedFiles := bitbucket.ExtractChangedFiles(diff)

			if reviewInstructions == "" {
				reviewInstructions = "No review instructions configured for this project."
			}
			if projectGuidelines == "" {
				projectGuidelines = "No project guidelines configured for this project."
			}

			output := GetPRDiffOutput{
				PRURL:              input.PRURL,
				Platform:           prInfo.Platform,
				Workspace:          prInfo.Workspace,
				RepoSlug:           prInfo.RepoSlug,
				PRId:               prInfo.PRId,
				Title:              metadata.Title,
				Description:        metadata.Description,
				SourceBranch:       metadata.SourceBranch,
				TargetBranch:       metadata.TargetBranch,
				Author:             metadata.Author,
				State:              metadata.State,
				ChangedFiles:       changedFiles,
				Diff:               diff,
				ReviewInstructions: reviewInstructions,
				ProjectGuidelines:  projectGuidelines,
				InstructionsToLLM:  llmReviewInstructions,
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

const llmReviewInstructions = `You are a senior software engineer performing a code review. ` +
	`Analyze the diff above against the review instructions and project guidelines. ` +
	`For each issue found, provide:
  - A numbered comment (1, 2, 3, ...)
  - The file path and line number
  - Category: Security | Quality | Performance | Style | Bug | Readability
  - Priority: Critical | High | Medium | Low
  - Clear description of the issue
  - Suggested fix with code snippet if applicable

Present ALL comments as a numbered list. The user will then choose which ones to post. ` +
	`After the user selects comments, use the post_pr_comments tool to post them.`

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
		IsError: true,
	}
}
