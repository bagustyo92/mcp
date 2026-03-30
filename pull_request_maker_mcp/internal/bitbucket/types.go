package bitbucket

// PRInfo identifies a specific pull request on Bitbucket.
type PRInfo struct {
	Platform  string `json:"platform"`
	Workspace string `json:"workspace"`
	RepoSlug  string `json:"repo_slug"`
	PRId      int    `json:"pr_id"`
}

// PRMetadata contains pull request details returned by the Bitbucket API.
type PRMetadata struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch"`
	Author       string `json:"author"`
	State        string `json:"state"`
}

// ReviewComment represents a single review comment to post on a PR.
type ReviewComment struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

// PostCommentResult is the outcome of posting a single comment.
type PostCommentResult struct {
	Path       string `json:"path"`
	Line       int    `json:"line"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
	CommentURL string `json:"comment_url,omitempty"`
}

// CreatePRRequest holds parameters for creating a new pull request.
type CreatePRRequest struct {
	Title             string   `json:"title"`
	Description       string   `json:"description"`
	SourceBranch      string   `json:"source_branch"`
	TargetBranch      string   `json:"target_branch"`
	Reviewers         []string `json:"reviewers,omitempty"`
	CloseSourceBranch bool     `json:"close_source_branch"`
}

// CreatePRResponse is returned after successfully creating a PR.
type CreatePRResponse struct {
	PRURL string `json:"pr_url"`
	PRId  int    `json:"pr_id"`
}

// Reviewer represents a Bitbucket user assigned as a reviewer.
type Reviewer struct {
	UUID        string `json:"uuid"`
	DisplayName string `json:"display_name"`
}
