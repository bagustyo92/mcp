package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/bitbucket"
	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/config"
	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/prdesc"
	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/urlparser"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CreatePRInput is the input schema for the create_pr tool.
type CreatePRInput struct {
	RepoSlug          string   `json:"repo_slug" jsonschema:"Full repo slug (workspace/repo) e.g. mid-kelola-indonesia/talenta-data-api"`
	SourceBranch      string   `json:"source_branch" jsonschema:"Source branch name (feature branch)"`
	TargetBranch      string   `json:"target_branch,omitempty" jsonschema:"Target branch for the PR. Falls back to project config default if omitted"`
	Title             string   `json:"title" jsonschema:"REQUIRED FORMAT: '{ticket_id}: {ticket title} - [FULL_COPILOT]'. Extract ticket_id from the source branch name (e.g. feature/TD-1234-foo -> TD-1234). Use NO-TICKET if none found. The suffix ' - [FULL_COPILOT]' is mandatory."`
	Description       string   `json:"description" jsonschema:"Pull request description in markdown. Must follow the PR description template provided by get_branch_diff."`
	Reviewers         []string `json:"reviewers,omitempty" jsonschema:"List of Bitbucket user UUIDs to assign as reviewers. Falls back to config defaults if omitted"`
	CloseSourceBranch *bool    `json:"close_source_branch,omitempty" jsonschema:"Whether to close the source branch after merge. Falls back to config default if omitted"`
}

// CreatePROutput is the structured output returned by create_pr.
type CreatePROutput struct {
	PRURL  string `json:"pr_url"`
	PRId   int    `json:"pr_id"`
	Status string `json:"status"`
}

// RegisterCreatePR registers the create_pr tool on the MCP server.
func RegisterCreatePR(server *mcp.Server, cfg *config.AppConfig, client *bitbucket.Client, templateLoader *prdesc.TemplateLoader) {
	// Determine default description mode from config (fall back to comprehensive).
	descriptionMode := "comprehensive"
	for _, p := range cfg.Projects {
		if p.DescriptionMode != "" {
			descriptionMode = p.DescriptionMode
			break
		}
	}
	prTemplate := templateLoader.Load(descriptionMode)

	mcp.AddTool(server,
		&mcp.Tool{
			Name: "create_pr",
			Description: "Create a new pull request on Bitbucket.\n\n" +
				"TITLE FORMAT (mandatory):\n" +
				"  {ticket_id}: {ticket title} - [FULL_COPILOT]\n" +
				"  - Extract ticket_id from source branch (e.g. feature/TD-1234-foo → TD-1234).\n" +
				"  - Use NO-TICKET if no ticket ID found in branch name.\n" +
				"  - The ' - [FULL_COPILOT]' suffix is mandatory and must never be omitted.\n\n" +
				"DESCRIPTION FORMAT (mandatory — use the template below exactly):\n\n" +
				prTemplate,
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input CreatePRInput) (*mcp.CallToolResult, CreatePROutput, error) {
			workspace, repo, err := urlparser.ParseRepoSlug(input.RepoSlug)
			if err != nil {
				return errorResult(err.Error()), CreatePROutput{}, nil
			}

			// Enforce the required [FULL_COPILOT] title suffix regardless of what the LLM provided.
			const requiredSuffix = " - [FULL_COPILOT]"
			if !strings.HasSuffix(input.Title, requiredSuffix) {
				// Strip any existing partial suffix to avoid duplication before appending.
				input.Title = strings.TrimSuffix(input.Title, " - [FULL_COPILOT]")
				input.Title = strings.TrimSuffix(input.Title, " [FULL_COPILOT]")
				input.Title = strings.TrimRight(input.Title, " -") + requiredSuffix
			}

			projectCfg := config.FindProjectConfig(cfg, workspace, repo)

			// Resolve target branch
			targetBranch := input.TargetBranch
			if targetBranch == "" && projectCfg != nil {
				targetBranch = projectCfg.DefaultTargetBranch
			}
			if targetBranch == "" {
				targetBranch = "master"
			}

			// Resolve reviewers
			reviewers := input.Reviewers
			if len(reviewers) == 0 && projectCfg != nil {
				reviewers = projectCfg.DefaultReviewers
			}

			// Resolve close_source_branch
			closeSource := false
			if input.CloseSourceBranch != nil {
				closeSource = *input.CloseSourceBranch
			} else if projectCfg != nil {
				closeSource = projectCfg.CloseSourceBranch
			}

			prReq := bitbucket.CreatePRRequest{
				Title:             input.Title,
				Description:       input.Description,
				SourceBranch:      input.SourceBranch,
				TargetBranch:      targetBranch,
				Reviewers:         reviewers,
				CloseSourceBranch: closeSource,
			}

			result, err := client.CreatePR(ctx, cfg.Auth.Bitbucket, workspace, repo, prReq)
			if err != nil {
				return errorResult(fmt.Sprintf("Error creating PR: %s", err)), CreatePROutput{}, nil
			}

			output := CreatePROutput{
				PRURL:  result.PRURL,
				PRId:   result.PRId,
				Status: "created",
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
