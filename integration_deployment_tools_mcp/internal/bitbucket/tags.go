package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"

	"bitbucket.org/mid-kelola-indonesia/integration-deployment-tools-mcp/internal/config"
)

// ListTags returns recent tags for a repository, sorted by newest first.
func (c *Client) ListTags(ctx context.Context, auth config.BitbucketAuth, workspace, repo string, pagelen int) ([]Tag, error) {
	if pagelen <= 0 {
		pagelen = 10
	}
	apiURL := fmt.Sprintf("%s/repositories/%s/%s/refs/tags?sort=-target.date&pagelen=%d", apiBase, workspace, repo, pagelen)

	headers := getHeaders(auth)

	data, status, err := c.doRequest(ctx, "GET", apiURL, headers, "")
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("bitbucket API error listing tags (%d): %s", status, string(data))
	}

	var resp paginatedResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse tags response: %w", err)
	}

	tags := make([]Tag, 0, len(resp.Values))
	for _, v := range resp.Values {
		tag := Tag{
			Name: jsonString(v, "name"),
		}
		if target, ok := v["target"].(map[string]any); ok {
			tag.Hash = jsonString(target, "hash")
			tag.Date = jsonString(target, "date")
			if author, ok := target["author"].(map[string]any); ok {
				if user, ok := author["user"].(map[string]any); ok {
					tag.Author = jsonString(user, "display_name")
				}
			}
		}
		tags = append(tags, tag)
	}

	return tags, nil
}

// GetLatestTag returns the most recent tag for a repository.
func (c *Client) GetLatestTag(ctx context.Context, auth config.BitbucketAuth, workspace, repo string) (*Tag, error) {
	tags, err := c.ListTags(ctx, auth, workspace, repo, 1)
	if err != nil {
		return nil, err
	}
	if len(tags) == 0 {
		return nil, nil
	}
	return &tags[0], nil
}

// GetTagByName returns a specific tag by name, or nil if not found.
func (c *Client) GetTagByName(ctx context.Context, auth config.BitbucketAuth, workspace, repo, tagName string) (*Tag, error) {
	apiURL := fmt.Sprintf("%s/repositories/%s/%s/refs/tags/%s", apiBase, workspace, repo, tagName)

	headers := getHeaders(auth)

	data, status, err := c.doRequest(ctx, "GET", apiURL, headers, "")
	if err != nil {
		return nil, err
	}
	if status == 404 {
		return nil, nil
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("bitbucket API error getting tag %q (%d): %s", tagName, status, string(data))
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse tag response: %w", err)
	}

	tag := &Tag{
		Name: jsonString(raw, "name"),
	}
	if target, ok := raw["target"].(map[string]any); ok {
		tag.Hash = jsonString(target, "hash")
		tag.Date = jsonString(target, "date")
		if author, ok := target["author"].(map[string]any); ok {
			if user, ok := author["user"].(map[string]any); ok {
				tag.Author = jsonString(user, "display_name")
			}
		}
	}
	return tag, nil
}

// GetPreviousTag returns the tag that comes just before the given tag in the sorted list.
// Returns nil if the given tag is the oldest or not found.
func (c *Client) GetPreviousTag(ctx context.Context, auth config.BitbucketAuth, workspace, repo, tagName string) (*Tag, error) {
	// Fetch enough tags to find the previous one (sorted newest first by date)
	tags, err := c.ListTags(ctx, auth, workspace, repo, 50)
	if err != nil {
		return nil, err
	}

	for i, t := range tags {
		if t.Name == tagName {
			if i+1 < len(tags) {
				return &tags[i+1], nil
			}
			return nil, nil
		}
	}
	return nil, nil
}

// CreateTag creates a new tag pointing to the given commit hash.
func (c *Client) CreateTag(ctx context.Context, auth config.BitbucketAuth, workspace, repo, tagName, commitHash string) (*Tag, error) {
	apiURL := fmt.Sprintf("%s/repositories/%s/%s/refs/tags", apiBase, workspace, repo)

	headers := postHeaders(auth)

	body := map[string]any{
		"name":   tagName,
		"target": map[string]string{"hash": commitHash},
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal tag body: %w", err)
	}

	data, status, err := c.doRequest(ctx, "POST", apiURL, headers, string(bodyJSON))
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("bitbucket API error creating tag (%d): %s", status, string(data))
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse create tag response: %w", err)
	}

	tag := &Tag{
		Name: jsonString(raw, "name"),
	}
	if target, ok := raw["target"].(map[string]any); ok {
		tag.Hash = jsonString(target, "hash")
		tag.Date = jsonString(target, "date")
	}

	return tag, nil
}
