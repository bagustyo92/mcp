package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"

	"bitbucket.org/mid-kelola-indonesia/integration-deployment-tools-mcp/internal/config"
)

const maxCommitPages = 10

// ListCommits returns recent commits on a branch.
func (c *Client) ListCommits(ctx context.Context, auth config.BitbucketAuth, workspace, repo, branch string, pagelen int) ([]Commit, error) {
	if pagelen <= 0 {
		pagelen = 30
	}
	apiURL := fmt.Sprintf("%s/repositories/%s/%s/commits/%s?pagelen=%d", apiBase, workspace, repo, branch, pagelen)

	headers := getHeaders(auth)

	data, status, err := c.doRequest(ctx, "GET", apiURL, headers, "")
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("bitbucket API error listing commits (%d): %s", status, string(data))
	}

	return parseCommitsResponse(data)
}

// GetCommitsAfterHash returns all commits on a branch that come after the given hash.
// It paginates through the commit history, stopping when it finds the target hash.
func (c *Client) GetCommitsAfterHash(ctx context.Context, auth config.BitbucketAuth, workspace, repo, branch, afterHash string) ([]Commit, error) {
	headers := getHeaders(auth)

	var allCommits []Commit
	nextURL := fmt.Sprintf("%s/repositories/%s/%s/commits/%s?pagelen=30", apiBase, workspace, repo, branch)

	for page := 0; page < maxCommitPages && nextURL != ""; page++ {
		data, status, err := c.doRequest(ctx, "GET", nextURL, headers, "")
		if err != nil {
			return nil, err
		}
		if status < 200 || status >= 300 {
			return nil, fmt.Errorf("bitbucket API error listing commits (%d): %s", status, string(data))
		}

		var resp struct {
			Values []map[string]any `json:"values"`
			Next   string           `json:"next"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("parse commits response: %w", err)
		}

		for _, v := range resp.Values {
			hash := jsonString(v, "hash")
			// Stop when we reach the tagged commit
			if len(hash) >= 12 && len(afterHash) >= 12 && hash[:12] == afterHash[:12] {
				return allCommits, nil
			}

			commit := parseCommit(v)
			allCommits = append(allCommits, commit)
		}

		nextURL = resp.Next
	}

	return allCommits, nil
}

// GetCommitsBetweenHashes returns commits reachable from toRef but not from fromRef.
// Uses the Bitbucket API include/exclude syntax for efficiency.
// toRef and fromRef can be a commit hash or a tag/branch name.
func (c *Client) GetCommitsBetweenHashes(ctx context.Context, auth config.BitbucketAuth, workspace, repo, fromRef, toRef string) ([]Commit, error) {
	headers := getHeaders(auth)

	var allCommits []Commit
	nextURL := fmt.Sprintf("%s/repositories/%s/%s/commits?include=%s&exclude=%s&pagelen=30",
		apiBase, workspace, repo, toRef, fromRef)

	for page := 0; page < maxCommitPages && nextURL != ""; page++ {
		data, status, err := c.doRequest(ctx, "GET", nextURL, headers, "")
		if err != nil {
			return nil, err
		}
		if status < 200 || status >= 300 {
			return nil, fmt.Errorf("bitbucket API error listing commits between refs (%d): %s", status, string(data))
		}

		var resp struct {
			Values []map[string]any `json:"values"`
			Next   string           `json:"next"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return nil, fmt.Errorf("parse commits response: %w", err)
		}

		for _, v := range resp.Values {
			allCommits = append(allCommits, parseCommit(v))
		}

		nextURL = resp.Next
	}

	return allCommits, nil
}

// GetLatestCommit returns the most recent commit on a branch.
func (c *Client) GetLatestCommit(ctx context.Context, auth config.BitbucketAuth, workspace, repo, branch string) (*Commit, error) {
	commits, err := c.ListCommits(ctx, auth, workspace, repo, branch, 1)
	if err != nil {
		return nil, err
	}
	if len(commits) == 0 {
		return nil, fmt.Errorf("no commits found on branch %s", branch)
	}
	return &commits[0], nil
}

func parseCommitsResponse(data []byte) ([]Commit, error) {
	var resp paginatedResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse commits response: %w", err)
	}

	commits := make([]Commit, 0, len(resp.Values))
	for _, v := range resp.Values {
		commits = append(commits, parseCommit(v))
	}
	return commits, nil
}

func parseCommit(v map[string]any) Commit {
	commit := Commit{
		Hash:    jsonString(v, "hash"),
		Message: jsonString(v, "message"),
		Date:    jsonString(v, "date"),
	}
	if author, ok := v["author"].(map[string]any); ok {
		if user, ok := author["user"].(map[string]any); ok {
			commit.Author = jsonString(user, "display_name")
		} else {
			commit.Author = jsonString(author, "raw")
		}
	}
	return commit
}
