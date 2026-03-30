package tools

import (
	"context"
	"encoding/json"

	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/bitbucket"
	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/config"
	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/urlparser"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PostPRCommentsInput is the input schema for the post_pr_comments tool.
type PostPRCommentsInput struct {
	PRURL    string            `json:"pr_url" jsonschema:"The full URL of the Bitbucket pull request"`
	Comments []CommentToPost   `json:"comments" jsonschema:"Array of review comments to post"`
}

// CommentToPost represents a single comment to be posted.
type CommentToPost struct {
	Path    string `json:"path" jsonschema:"File path relative to repo root. Empty string for general comments"`
	Line    int    `json:"line" jsonschema:"Line number in the new file version. 0 for general comments"`
	Content string `json:"content" jsonschema:"The review comment content in markdown"`
}

// PostPRCommentsOutput is the structured output returned by post_pr_comments.
type PostPRCommentsOutput struct {
	Total   int                          `json:"total"`
	Posted  int                          `json:"posted"`
	Failed  int                          `json:"failed"`
	Results []bitbucket.PostCommentResult `json:"results"`
}

// RegisterPostPRComments registers the post_pr_comments tool on the MCP server.
func RegisterPostPRComments(server *mcp.Server, cfg *config.AppConfig, client *bitbucket.Client) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name: "post_pr_comments",
			Description: "Post review comments on a Bitbucket pull request. " +
				"Each comment can be an inline comment (attached to a specific file and line) or a general comment. " +
				"Use this after the user has selected which review comments to post from the get_pr_diff or get_branch_diff analysis.",
		},
		func(ctx context.Context, req *mcp.CallToolRequest, input PostPRCommentsInput) (*mcp.CallToolResult, PostPRCommentsOutput, error) {
			prInfo, err := urlparser.ParsePRURL(input.PRURL)
			if err != nil {
				return errorResult(err.Error()), PostPRCommentsOutput{}, nil
			}

			results := make([]bitbucket.PostCommentResult, 0, len(input.Comments))

			for _, c := range input.Comments {
				comment := bitbucket.ReviewComment{
					Path:    c.Path,
					Line:    c.Line,
					Content: c.Content,
				}
				result := client.PostCommentWithDelay(ctx, cfg.Auth.Bitbucket, *prInfo, comment)
				results = append(results, result)
			}

			posted := 0
			failed := 0
			for _, r := range results {
				if r.Success {
					posted++
				} else {
					failed++
				}
			}

			output := PostPRCommentsOutput{
				Total:   len(input.Comments),
				Posted:  posted,
				Failed:  failed,
				Results: results,
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
