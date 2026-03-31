package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"time"

	"bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp/internal/config"
)

// FetchPRMetadata retrieves PR details (title, branches, author, state).
func (c *Client) FetchPRMetadata(ctx context.Context, auth config.BitbucketAuth, pr PRInfo) (*PRMetadata, error) {
	apiURL := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d", apiBase, pr.Workspace, pr.RepoSlug, pr.PRId)

	headers, err := getHeaders(auth)
	if err != nil {
		return nil, err
	}

	data, status, err := c.doRequest(ctx, "GET", apiURL, headers, "")
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("bitbucket API error fetching PR metadata (%d): %s", status, string(data))
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse PR metadata: %w", err)
	}

	return &PRMetadata{
		Title:        jsonString(raw, "title"),
		Description:  jsonString(raw, "description"),
		SourceBranch: jsonNestedString(raw, "source", "branch", "name"),
		TargetBranch: jsonNestedString(raw, "destination", "branch", "name"),
		Author:       jsonNestedStringOr(raw, "author", "display_name", "nickname"),
		AuthorUUID:   jsonNestedString(raw, "author", "uuid"),
		State:        jsonString(raw, "state"),
	}, nil
}

// FetchBranchDiff fetches the unified diff between two branches.
func (c *Client) FetchBranchDiff(ctx context.Context, auth config.BitbucketAuth, workspace, repoSlug, sourceBranch, targetBranch string) (string, error) {
	spec := url.PathEscape(sourceBranch) + ".." + url.PathEscape(targetBranch)
	apiURL := fmt.Sprintf("%s/repositories/%s/%s/diff/%s", apiBase, workspace, repoSlug, spec)

	headers, err := getHeaders(auth)
	if err != nil {
		return "", err
	}
	headers["Accept"] = "text/plain"

	data, status, err := c.doRequest(ctx, "GET", apiURL, headers, "")
	if err != nil {
		return "", err
	}
	if status < 200 || status >= 300 {
		return "", fmt.Errorf("bitbucket API error fetching diff (%d): %s", status, string(data))
	}

	return string(data), nil
}

// ExtractChangedFiles parses file paths from a unified diff.
// Matches lines like: diff --git a/path/to/file b/path/to/file
var diffFileRegex = regexp.MustCompile(`(?m)^diff --git a/.+ b/(.+)$`)

func ExtractChangedFiles(diff string) []string {
	matches := diffFileRegex.FindAllStringSubmatch(diff, -1)
	seen := make(map[string]bool)
	var files []string

	for _, m := range matches {
		path := m[1]
		if !seen[path] {
			seen[path] = true
			files = append(files, path)
		}
	}
	return files
}

// CreatePR creates a new pull request on Bitbucket.
func (c *Client) CreatePR(ctx context.Context, auth config.BitbucketAuth, workspace, repoSlug string, req CreatePRRequest) (*CreatePRResponse, error) {
	apiURL := fmt.Sprintf("%s/repositories/%s/%s/pullrequests", apiBase, workspace, repoSlug)

	headers, err := postHeaders(auth)
	if err != nil {
		return nil, err
	}

	// Build the request body per Bitbucket API spec
	body := map[string]any{
		"title":               req.Title,
		"description":         req.Description,
		"source":              map[string]any{"branch": map[string]string{"name": req.SourceBranch}},
		"destination":         map[string]any{"branch": map[string]string{"name": req.TargetBranch}},
		"close_source_branch": req.CloseSourceBranch,
	}

	if len(req.Reviewers) > 0 {
		reviewers := make([]map[string]string, len(req.Reviewers))
		for i, uuid := range req.Reviewers {
			reviewers[i] = map[string]string{"uuid": uuid}
		}
		body["reviewers"] = reviewers
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal PR body: %w", err)
	}

	data, status, err := c.doRequest(ctx, "POST", apiURL, headers, string(bodyJSON))
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("bitbucket API error creating PR (%d): %s", status, string(data))
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse create PR response: %w", err)
	}

	prID := int(jsonFloat(raw, "id"))
	prURL := jsonNestedString(raw, "links", "html", "href")

	return &CreatePRResponse{
		PRURL: prURL,
		PRId:  prID,
	}, nil
}

// PostInlineComment posts a comment on a specific file+line in a PR.
func (c *Client) PostInlineComment(ctx context.Context, auth config.BitbucketAuth, pr PRInfo, comment ReviewComment) PostCommentResult {
	apiURL := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d/comments", apiBase, pr.Workspace, pr.RepoSlug, pr.PRId)

	headers, err := postHeaders(auth)
	if err != nil {
		return PostCommentResult{Path: comment.Path, Line: comment.Line, Success: false, Error: err.Error()}
	}

	body := map[string]any{
		"content": map[string]string{"raw": comment.Content},
	}

	if comment.Path != "" && comment.Line > 0 {
		body["inline"] = map[string]any{
			"path": comment.Path,
			"to":   comment.Line,
		}
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return PostCommentResult{Path: comment.Path, Line: comment.Line, Success: false, Error: err.Error()}
	}

	data, status, err := c.doRequest(ctx, "POST", apiURL, headers, string(bodyJSON))
	if err != nil {
		return PostCommentResult{Path: comment.Path, Line: comment.Line, Success: false, Error: err.Error()}
	}
	if status < 200 || status >= 300 {
		return PostCommentResult{Path: comment.Path, Line: comment.Line, Success: false, Error: fmt.Sprintf("bitbucket API error (%d): %s", status, string(data))}
	}

	var raw map[string]any
	_ = json.Unmarshal(data, &raw)

	return PostCommentResult{
		Path:       comment.Path,
		Line:       comment.Line,
		Success:    true,
		CommentURL: jsonNestedString(raw, "links", "html", "href"),
	}
}

// PostGeneralComment posts a non-inline comment on a PR.
func (c *Client) PostGeneralComment(ctx context.Context, auth config.BitbucketAuth, pr PRInfo, content string) PostCommentResult {
	apiURL := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d/comments", apiBase, pr.Workspace, pr.RepoSlug, pr.PRId)

	headers, err := postHeaders(auth)
	if err != nil {
		return PostCommentResult{Success: false, Error: err.Error()}
	}

	body := map[string]any{
		"content": map[string]string{"raw": content},
	}

	bodyJSON, _ := json.Marshal(body)

	data, status, err := c.doRequest(ctx, "POST", apiURL, headers, string(bodyJSON))
	if err != nil {
		return PostCommentResult{Success: false, Error: err.Error()}
	}
	if status < 200 || status >= 300 {
		return PostCommentResult{Success: false, Error: fmt.Sprintf("bitbucket API error (%d): %s", status, string(data))}
	}

	var raw map[string]any
	_ = json.Unmarshal(data, &raw)

	return PostCommentResult{
		Success:    true,
		CommentURL: jsonNestedString(raw, "links", "html", "href"),
	}
}

// FetchDefaultReviewers retrieves the effective default reviewers for a repository.
func (c *Client) FetchDefaultReviewers(ctx context.Context, auth config.BitbucketAuth, workspace, repoSlug string) ([]Reviewer, error) {
	apiURL := fmt.Sprintf("%s/repositories/%s/%s/effective-default-reviewers", apiBase, workspace, repoSlug)

	headers, err := getHeaders(auth)
	if err != nil {
		return nil, err
	}

	data, status, err := c.doRequest(ctx, "GET", apiURL, headers, "")
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("bitbucket API error fetching default reviewers (%d): %s", status, string(data))
	}

	var result struct {
		Values []struct {
			UUID        string `json:"uuid"`
			DisplayName string `json:"display_name"`
		} `json:"values"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse default reviewers: %w", err)
	}

	reviewers := make([]Reviewer, len(result.Values))
	for i, v := range result.Values {
		reviewers[i] = Reviewer{UUID: v.UUID, DisplayName: v.DisplayName}
	}
	return reviewers, nil
}

// PostCommentWithDelay posts a comment and sleeps for rate limiting.
func (c *Client) PostCommentWithDelay(ctx context.Context, auth config.BitbucketAuth, pr PRInfo, comment ReviewComment) PostCommentResult {
	var result PostCommentResult

	if comment.Path == "" || comment.Line <= 0 {
		result = c.PostGeneralComment(ctx, auth, pr, comment.Content)
	} else {
		result = c.PostInlineComment(ctx, auth, pr, comment)
	}

	// Rate limiting delay
	time.Sleep(200 * time.Millisecond)

	return result
}

// JSON helper functions for safe nested access.

func jsonString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func jsonFloat(m map[string]any, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}

func jsonNestedString(m map[string]any, keys ...string) string {
	current := m
	for i, key := range keys {
		v, ok := current[key]
		if !ok {
			return ""
		}
		if i == len(keys)-1 {
			if s, ok := v.(string); ok {
				return s
			}
			return ""
		}
		nested, ok := v.(map[string]any)
		if !ok {
			return ""
		}
		current = nested
	}
	return ""
}

func jsonNestedStringOr(m map[string]any, parent, key1, key2 string) string {
	p, ok := m[parent]
	if !ok {
		return ""
	}
	pm, ok := p.(map[string]any)
	if !ok {
		return ""
	}
	if s := jsonString(pm, key1); s != "" {
		return s
	}
	return jsonString(pm, key2)
}
