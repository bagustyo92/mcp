package bitbucket

// Tag represents a Bitbucket repository tag.
type Tag struct {
	Name   string `json:"name"`
	Hash   string `json:"hash"`
	Date   string `json:"date"`
	Author string `json:"author"`
}

// Commit represents a Bitbucket commit.
type Commit struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
	Author  string `json:"author"`
	Date    string `json:"date"`
}

// PullRequest represents a merged pull request associated with a commit.
type PullRequest struct {
	ID           int    `json:"id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	SourceBranch string `json:"source_branch"`
	Author       string `json:"author"`
	AuthorUUID   string `json:"author_uuid"`
	URL          string `json:"url"`
}

// PipelineRun represents a triggered Bitbucket pipeline.
type PipelineRun struct {
	UUID        string `json:"uuid"`
	BuildNumber int    `json:"build_number"`
	URL         string `json:"url"`
}

// paginatedResponse is the generic envelope for paginated Bitbucket API responses.
type paginatedResponse struct {
	Values []map[string]any `json:"values"`
	Next   string           `json:"next"`
	Page   int              `json:"page"`
	Size   int              `json:"size"`
}
