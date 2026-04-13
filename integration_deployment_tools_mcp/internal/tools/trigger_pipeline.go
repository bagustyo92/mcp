package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"bitbucket.org/mid-kelola-indonesia/integration-deployment-tools-mcp/internal/bitbucket"
	"bitbucket.org/mid-kelola-indonesia/integration-deployment-tools-mcp/internal/config"
	"bitbucket.org/mid-kelola-indonesia/integration-deployment-tools-mcp/internal/gchat"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TriggerPipelineInput is the input schema.
type TriggerPipelineInput struct {
	RepoSlug         string `json:"repo_slug" jsonschema:"required,Full repo slug (workspace/repo)"`
	Environment      string `json:"environment" jsonschema:"required,Target environment (e.g. staging, production)"`
	RefName          string `json:"ref_name,omitempty" jsonschema:"Branch/tag to deploy. If empty, auto-detects from pipeline config or uses default branch."`
	SendNotification bool   `json:"send_notification,omitempty" jsonschema:"Send notification to Google Chat (default false)"`
}

// TriggerPipelineOutput is the structured output.
type TriggerPipelineOutput struct {
	PipelineUUID string `json:"pipeline_uuid"`
	BuildNumber  int    `json:"build_number"`
	RepoSlug     string `json:"repo_slug"`
	Environment  string `json:"environment"`
	RefName      string `json:"ref_name"`
	PipelineURL  string `json:"pipeline_url"`
}

// RegisterTriggerPipeline registers the trigger_pipeline tool.
func RegisterTriggerPipeline(server *mcp.Server, cfg *config.AppConfig, client *bitbucket.Client, notifier *gchat.Notifier) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name: "trigger_pipeline",
			Description: "Trigger a Bitbucket Pipeline for a given repository and environment. " +
				"Pipeline configuration (custom pipeline name, ref type) is resolved from config. " +
				"If ref_name is empty, the default branch or configured ref is used.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input TriggerPipelineInput) (*mcp.CallToolResult, TriggerPipelineOutput, error) {
			if input.RepoSlug == "" {
				return errorResult("repo_slug is required"), TriggerPipelineOutput{}, nil
			}
			if input.Environment == "" {
				return errorResult("environment is required"), TriggerPipelineOutput{}, nil
			}

			repoCfg := config.FindRepoConfig(cfg, input.RepoSlug)
			if repoCfg == nil {
				return errorResult(fmt.Sprintf("Repository %q not found in config", input.RepoSlug)), TriggerPipelineOutput{}, nil
			}

			workspace, repo, err := parseRepoSlug(input.RepoSlug)
			if err != nil {
				return errorResult(err.Error()), TriggerPipelineOutput{}, nil
			}

			// Resolve pipeline config for the environment
			pipelineCfg, ok := repoCfg.Pipelines[strings.ToLower(input.Environment)]
			if !ok {
				// Try to build a reasonable default
				pipelineCfg = config.PipelineConfig{
					Name:    fmt.Sprintf("deploy-%s", strings.ToLower(input.Environment)),
					RefType: "branch",
				}
			}

			// Resolve ref name
			refName := input.RefName
			if refName == "" {
				if pipelineCfg.RefType == "tag" {
					// Use latest tag
					latestTag, err := client.GetLatestTag(ctx, cfg.Auth.Bitbucket, workspace, repo)
					if err != nil || latestTag == nil {
						return errorResult("Cannot auto-detect ref: no tags found and ref_name is empty"), TriggerPipelineOutput{}, nil
					}
					refName = latestTag.Name
				} else {
					branch := repoCfg.DefaultBranch
					if branch == "" {
						branch = "master"
					}
					refName = branch
				}
			}

			// Trigger the pipeline
			run, err := client.TriggerPipeline(ctx, cfg.Auth.Bitbucket, workspace, repo, pipelineCfg.Name, pipelineCfg.RefType, refName)
			if err != nil {
				return errorResult(fmt.Sprintf("Failed to trigger pipeline: %s", err)), TriggerPipelineOutput{}, nil
			}

			pipelineURL := fmt.Sprintf("https://bitbucket.org/%s/%s/addon/pipelines/home#!/results/%d", workspace, repo, run.BuildNumber)

			output := TriggerPipelineOutput{
				PipelineUUID: run.UUID,
				BuildNumber:  run.BuildNumber,
				RepoSlug:     input.RepoSlug,
				Environment:  input.Environment,
				RefName:      refName,
				PipelineURL:  pipelineURL,
			}

			// Notify
			if input.SendNotification {
				if cfg.GChat.WebhookURL != "" {
					msg := gchat.PipelineMessage{
						RepoSlug:    input.RepoSlug,
						Environment: input.Environment,
						RefName:     refName,
						PipelineURL: pipelineURL,
						BuildNumber: run.BuildNumber,
					}
					_ = notifier.NotifyPipelineTriggered(ctx, cfg.GChat.WebhookURL, msg)
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
