package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/bitbucket"
	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/config"
	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/prdesc"
	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/urlparser"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GetBranchDiffInput is the input schema for the get_branch_diff tool.
type GetBranchDiffInput struct {
	RepoSlug     string `json:"repo_slug" jsonschema:"Full repo slug (workspace/repo) e.g. mid-kelola-indonesia/talenta-data-api"`
	SourceBranch string `json:"source_branch" jsonschema:"Source branch name (feature branch)"`
	TargetBranch string `json:"target_branch,omitempty" jsonschema:"Target branch to compare against. Falls back to project config default if omitted"`
}

// GetBranchDiffOutput is the structured output returned by get_branch_diff.
type GetBranchDiffOutput struct {
	RepoSlug              string   `json:"repo_slug"`
	SourceBranch          string   `json:"source_branch"`
	TargetBranch          string   `json:"target_branch"`
	ChangedFiles          []string `json:"changed_files"`
	Diff                  string   `json:"diff"`
	ReviewInstructions    string   `json:"review_instructions"`
	ProjectGuidelines     string   `json:"project_guidelines"`
	PRDescriptionTemplate string   `json:"pr_description_template"`
	InstructionsToLLM     string   `json:"instructions_to_llm"`
}

// RegisterGetBranchDiff registers the get_branch_diff tool on the MCP server.
func RegisterGetBranchDiff(server *mcp.Server, cfg *config.AppConfig, client *bitbucket.Client, templateLoader *prdesc.TemplateLoader) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name: "get_branch_diff",
			Description: "Fetch the diff between two branches for code review and PR creation. " +
				"Returns the diff, changed files, review instructions, project guidelines, and a PR description template. " +
				"Use this before creating a PR to review changes and generate a PR description.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input GetBranchDiffInput) (*mcp.CallToolResult, GetBranchDiffOutput, error) {
			workspace, repo, err := urlparser.ParseRepoSlug(input.RepoSlug)
			if err != nil {
				return errorResult(err.Error()), GetBranchDiffOutput{}, nil
			}

			projectCfg := config.FindProjectConfig(cfg, workspace, repo)

			targetBranch := input.TargetBranch
			if targetBranch == "" && projectCfg != nil {
				targetBranch = projectCfg.DefaultTargetBranch
			}
			if targetBranch == "" {
				targetBranch = "master"
			}

			diff, err := client.FetchBranchDiff(ctx, cfg.Auth.Bitbucket, workspace, repo, input.SourceBranch, targetBranch)
			if err != nil {
				return errorResult(fmt.Sprintf("Error fetching branch diff: %s", err)), GetBranchDiffOutput{}, nil
			}

			changedFiles := bitbucket.ExtractChangedFiles(diff)

			reviewInstructions := ""
			projectGuidelines := ""
			if projectCfg != nil {
				reviewInstructions = config.LoadFileContent(projectCfg.ReviewInstructions)
				projectGuidelines = config.LoadFileContent(projectCfg.ProjectGuidelines)
			}

			if reviewInstructions == "" {
				reviewInstructions = "No review instructions configured for this project."
			}
			if projectGuidelines == "" {
				projectGuidelines = "No project guidelines configured for this project."
			}

			// Load the PR description template based on project config
			descriptionMode := "comprehensive"
			if projectCfg != nil && projectCfg.DescriptionMode != "" {
				descriptionMode = projectCfg.DescriptionMode
			}

			prTemplate := ""
			if projectCfg != nil && projectCfg.PRDescriptionTemplate != "" {
				prTemplate = config.LoadFileContent(projectCfg.PRDescriptionTemplate)
			}
			if prTemplate == "" {
				prTemplate = templateLoader.Load(descriptionMode)
			}

			output := GetBranchDiffOutput{
				RepoSlug:              input.RepoSlug,
				SourceBranch:          input.SourceBranch,
				TargetBranch:          targetBranch,
				ChangedFiles:          changedFiles,
				Diff:                  diff,
				ReviewInstructions:    reviewInstructions,
				ProjectGuidelines:     projectGuidelines,
				PRDescriptionTemplate: prTemplate,
				InstructionsToLLM:     llmBranchDiffInstructions,
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

const llmBranchDiffInstructions = `You are a senior software engineer. You have two tasks:

1. CODE REVIEW: Analyze the diff against the review instructions and project guidelines. For each issue found, provide:
   - A numbered comment (1, 2, 3, ...)
   - The file path and line number
   - Category: Security | Quality | Performance | Style | Bug | Readability
   - Priority: Critical | High | Medium | Low
   - Clear description of the issue
   - Suggested fix with code snippet if applicable

2. PR TITLE & DESCRIPTION: Generate a pull request title and description following the provided PR description template.

   CRITICAL — PR TITLE FORMAT (must be followed exactly):
   {ticket_id}: {ticket title} - [FULL_COPILOT]

   Rules for the title:
   - Extract {ticket_id} from the source branch name (e.g. "feature/TD-1234-add-attendance" → "TD-1234").
   - If no ticket ID is found in the branch name, use "NO-TICKET".
   - {ticket title} should be a short, human-readable summary of the change.
   - The suffix " - [FULL_COPILOT]" MUST always be appended — this is a required company identifier.

   Example titles:
   - "TD-1234: Add employee attendance endpoint - [FULL_COPILOT]"
   - "CORE-567: Fix payroll calculation bug - [FULL_COPILOT]"

   Fill in all description sections based on the actual code changes in the diff.

Present the review comments first, then the generated PR title and description. The user will choose which review comments to post and can use the generated title and description when creating the PR.`
