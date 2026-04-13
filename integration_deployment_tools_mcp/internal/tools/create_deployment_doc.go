package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"bitbucket.org/mid-kelola-indonesia/integration-deployment-tools-mcp/internal/config"
	"bitbucket.org/mid-kelola-indonesia/integration-deployment-tools-mcp/internal/confluence"
	"github.com/modelcontextprotocol/go-sdk/mcp"
) // CreateDeploymentDocInput is the input schema.
type CreateDeploymentDocInput struct {
	Title           string   `json:"title" jsonschema:"required,Document title (e.g. Sprint 24.1 Deployment)"`
	ReferencePageID string   `json:"reference_page_id,omitempty" jsonschema:"Confluence page ID to clone as template. If empty, uses config's template_page_id."`
	ChangesSummary  string   `json:"changes_summary,omitempty" jsonschema:"Human-readable or markdown summary of changes to embed in the page."`
	TagName         string   `json:"tag_name,omitempty" jsonschema:"Release tag name to include in the document."`
	RepoSlugs       []string `json:"repo_slugs,omitempty" jsonschema:"List of repo slugs involved in this deployment."`
}

// CreateDeploymentDocOutput is the structured output.
type CreateDeploymentDocOutput struct {
	PageID   string `json:"page_id"`
	PageURL  string `json:"page_url"`
	Title    string `json:"title"`
	SpaceKey string `json:"space_key"`
}

// RegisterCreateDeploymentDoc registers the create_deployment_doc tool.
func RegisterCreateDeploymentDoc(server *mcp.Server, cfg *config.AppConfig, confClient *confluence.Client) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name: "create_deployment_doc",
			Description: "Create a Confluence deployment document by cloning a template page and populating it " +
				"with deployment details (changes summary, tag name, repositories). Useful for maintaining " +
				"deployment records and audit trails.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input CreateDeploymentDocInput) (*mcp.CallToolResult, CreateDeploymentDocOutput, error) {
			if input.Title == "" {
				return errorResult("title is required"), CreateDeploymentDocOutput{}, nil
			}

			if cfg.Confluence.BaseURL == "" {
				return errorResult("Confluence is not configured (confluence.base_url is empty)"), CreateDeploymentDocOutput{}, nil
			}

			// Resolve template page
			templatePageID := strings.TrimSpace(input.ReferencePageID)
			if templatePageID == "" {
				templatePageID = strings.TrimSpace(cfg.Confluence.TemplatePageID)
			}

			// Build page body
			var bodyContent string
			baseURL := cfg.Confluence.BaseURL
			auth := cfg.Auth.Bitbucket

			if templatePageID != "" {
				page, err := confClient.GetPage(ctx, auth, baseURL, templatePageID)
				if err != nil {
					return errorResult(fmt.Sprintf("Failed to fetch template page %s: %s", templatePageID, err)), CreateDeploymentDocOutput{}, nil
				}
				bodyContent = page.Body
				// Replace known placeholders
				bodyContent = strings.ReplaceAll(bodyContent, "{{TITLE}}", input.Title)
				bodyContent = strings.ReplaceAll(bodyContent, "{{DATE}}", time.Now().Format("2006-01-02"))
				bodyContent = strings.ReplaceAll(bodyContent, "{{TAG}}", input.TagName)
				bodyContent = strings.ReplaceAll(bodyContent, "{{CHANGES}}", input.ChangesSummary)
				bodyContent = strings.ReplaceAll(bodyContent, "{{REPOS}}", strings.Join(input.RepoSlugs, ", "))
			} else {
				// Build a default page from scratch
				bodyContent = buildDefaultDeploymentPage(input)
			}

			// Resolve space ID
			spaceID, err := confClient.GetSpaceID(ctx, auth, baseURL, cfg.Confluence.SpaceKey)
			if err != nil {
				return errorResult(fmt.Sprintf("Failed to resolve space %q: %s", cfg.Confluence.SpaceKey, err)), CreateDeploymentDocOutput{}, nil
			}

			createdPage, err := confClient.CreatePage(ctx, auth, baseURL, spaceID, cfg.Confluence.ParentPageID, input.Title, bodyContent)
			if err != nil {
				return errorResult(fmt.Sprintf("Failed to create page: %s", err)), CreateDeploymentDocOutput{}, nil
			}

			output := CreateDeploymentDocOutput{
				PageID:   createdPage.ID,
				PageURL:  createdPage.URL,
				Title:    createdPage.Title,
				SpaceKey: cfg.Confluence.SpaceKey,
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

func buildDefaultDeploymentPage(input CreateDeploymentDocInput) string {
	var sb strings.Builder
	sb.WriteString("<h1>")
	sb.WriteString(input.Title)
	sb.WriteString("</h1>\n")
	sb.WriteString("<p><strong>Date:</strong> ")
	sb.WriteString(time.Now().Format("2006-01-02 15:04"))
	sb.WriteString("</p>\n")

	if input.TagName != "" {
		sb.WriteString("<p><strong>Release Tag:</strong> ")
		sb.WriteString(input.TagName)
		sb.WriteString("</p>\n")
	}

	if len(input.RepoSlugs) > 0 {
		sb.WriteString("<h2>Repositories</h2>\n<ul>\n")
		for _, r := range input.RepoSlugs {
			sb.WriteString("<li>")
			sb.WriteString(r)
			sb.WriteString("</li>\n")
		}
		sb.WriteString("</ul>\n")
	}

	if input.ChangesSummary != "" {
		sb.WriteString("<h2>Changes Summary</h2>\n")
		sb.WriteString("<p>")
		sb.WriteString(input.ChangesSummary)
		sb.WriteString("</p>\n")
	}

	return sb.String()
}
