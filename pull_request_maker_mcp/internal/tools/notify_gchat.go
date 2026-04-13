package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/bitbucket"
	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/config"
	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/gchat"
	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/urlparser"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NotifyGChatInput is the input schema for the notify_gchat tool.
type NotifyGChatInput struct {
	PRURL       string `json:"pr_url" jsonschema:"The full URL of the Bitbucket pull request"`
	Title       string `json:"title" jsonschema:"Pull request title"`
	Author      string `json:"author" jsonschema:"Author display name"`
	AuthorUUID  string `json:"author_uuid,omitempty" jsonschema:"Author UUID from Bitbucket (displayed in preference to display name)"`
	Description string `json:"description,omitempty" jsonschema:"Pull request description (will be truncated if too long)"`
	// SourceBranch is used to extract the Jira ticket key automatically.
	SourceBranch string `json:"source_branch,omitempty" jsonschema:"Source branch name — used to extract Jira ticket key automatically"`
}

// NotifyGChatOutput is the structured output returned by notify_gchat.
type NotifyGChatOutput struct {
	Sent       bool   `json:"sent"`
	TicketKey  string `json:"ticket_key,omitempty"`
	TicketURL  string `json:"ticket_url,omitempty"`
	WebhookURL string `json:"webhook_url_used"`
}

// RegisterNotifyGChat registers the notify_gchat tool on the MCP server.
func RegisterNotifyGChat(server *mcp.Server, cfg *config.AppConfig, bbClient *bitbucket.Client, notifier *gchat.Notifier) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name: "notify_gchat",
			Description: "Send a pull request notification to a Google Chat space via webhook. " +
				"Call this after create_pr to notify the team. " +
				"Automatically extracts the Jira ticket key from the source branch name or PR title. " +
				"Also automatically fetches the author UUID from Bitbucket if not provided.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input NotifyGChatInput) (*mcp.CallToolResult, NotifyGChatOutput, error) {
			// Resolve which webhook URL to use (project-specific override or global)
			webhookURL := cfg.GChat.WebhookURL
			if webhookURL == "" {
				return errorResult("gchat.webhook_url is not configured in config.yaml"), NotifyGChatOutput{}, nil
			}

			// Parse PR URL to get workspace, repo, and PR ID
			prInfo, err := urlparser.ParsePRURL(input.PRURL)
			if err != nil {
				return errorResult(fmt.Sprintf("Invalid PR URL: %s", err)), NotifyGChatOutput{}, nil
			}

			// Check for per-project webhook override
			projectCfg := config.FindProjectConfig(cfg, prInfo.Workspace, prInfo.RepoSlug)
			if projectCfg != nil && projectCfg.GChatWebhookURL != "" {
				webhookURL = projectCfg.GChatWebhookURL
			}

			// If AuthorUUID not provided, fetch it from Bitbucket
			authorUUID := input.AuthorUUID
			if authorUUID == "" {
				prMetadata, err := bbClient.FetchPRMetadata(ctx, cfg.Auth.Bitbucket, *prInfo)
				if err == nil {
					authorUUID = prMetadata.AuthorUUID
				}
			}

			// Extract Jira ticket
			ticketKey := gchat.ExtractJiraTicket(input.SourceBranch)
			if ticketKey == "" {
				ticketKey = gchat.ExtractJiraTicket(input.Title)
			}
			ticketURL := gchat.JiraURL(cfg.GChat.JiraBaseURL, ticketKey)

			// Resolve author's email for mention
			authorEmail := config.FindAuthorEmail(cfg, input.Author, authorUUID)

			msg := gchat.PRMessage{
				Title:        input.Title,
				Author:       input.Author,
				AuthorUUID:   authorUUID,
				AuthorEmail:  authorEmail,
				Description:  input.Description,
				PRURL:        input.PRURL,
				SourceBranch: input.SourceBranch,
				JiraBaseURL:  cfg.GChat.JiraBaseURL,
			}

			if err := notifier.NotifyPRCreated(ctx, webhookURL, msg); err != nil {
				return errorResult(fmt.Sprintf("Failed to send GChat notification: %s", err)), NotifyGChatOutput{}, nil
			}

			output := NotifyGChatOutput{
				Sent:       true,
				TicketKey:  ticketKey,
				TicketURL:  ticketURL,
				WebhookURL: webhookURL,
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
