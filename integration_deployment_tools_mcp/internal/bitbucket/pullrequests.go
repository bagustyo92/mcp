package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"

	"bitbucket.org/mid-kelola-indonesia/integration-deployment-tools-mcp/internal/config"
)

// GetPRsForCommit returns merged pull requests associated with a given commit hash.
func (c *Client) GetPRsForCommit(ctx context.Context, auth config.BitbucketAuth, workspace, repo, commitHash string) ([]PullRequest, error) {
	apiURL := fmt.Sprintf("%s/repositories/%s/%s/commit/%s/pullrequests?pagelen=10", apiBase, workspace, repo, commitHash)

	headers := getHeaders(auth)

	data, status, err := c.doRequest(ctx, "GET", apiURL, headers, "")
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("bitbucket API error fetching PRs for commit (%d): %s", status, string(data))
	}

	var resp paginatedResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse PRs response: %w", err)
	}

	prs := make([]PullRequest, 0, len(resp.Values))
	for _, v := range resp.Values {
		pr := PullRequest{
			ID:    int(jsonFloat(v, "id")),
			Title: jsonString(v, "title"),
		}

		if desc, ok := v["description"].(string); ok {
			pr.Description = desc
		}

		if source, ok := v["source"].(map[string]any); ok {
			if branch, ok := source["branch"].(map[string]any); ok {
				pr.SourceBranch = jsonString(branch, "name")
			}
		}

		if author, ok := v["author"].(map[string]any); ok {
			pr.Author = jsonString(author, "display_name")
			if pr.Author == "" {
				pr.Author = jsonString(author, "nickname")
			}
			pr.AuthorUUID = jsonString(author, "uuid")
		}

		if links, ok := v["links"].(map[string]any); ok {
			if html, ok := links["html"].(map[string]any); ok {
				pr.URL = jsonString(html, "href")
			}
		}

		prs = append(prs, pr)
	}

	return prs, nil
}
