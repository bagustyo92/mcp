package config

// AppConfig is the top-level configuration loaded from config.yaml.
type AppConfig struct {
	Auth     AuthConfig      `yaml:"auth"`
	GChat    GChatConfig     `yaml:"gchat"`
	Projects []ProjectConfig `yaml:"projects"`
}

// GChatConfig holds Google Chat notification settings.
type GChatConfig struct {
	// WebhookURL is the incoming webhook URL for the Google Chat space.
	// Create at: Google Chat space → Apps & integrations → Webhooks → Add webhook
	WebhookURL string `yaml:"webhook_url"`

	// JiraBaseURL is the base URL for Jira ticket links, e.g. "https://mekari.atlassian.net/browse".
	// Used to auto-generate ticket links from the branch name or PR title.
	JiraBaseURL string `yaml:"jira_base_url"`

	// UserMappings maps Bitbucket authors to their Google Chat user IDs for @mentions.
	// Lookup is done by BitbucketName first, then BitbucketUUID as fallback.
	// To find a user's Google Chat ID, use the Admin SDK Directory API or ask the user
	// to share their ID from their Google Chat profile.
	UserMappings []UserMapping `yaml:"user_mappings"`
}

// UserMapping maps a Bitbucket user to their email for display in Google Chat mentions.
type UserMapping struct {
	// BitbucketName is the Bitbucket display name (e.g. "Bagus Prasetyo"). Used as primary lookup key.
	BitbucketName string `yaml:"bitbucket_name"`

	// BitbucketUUID is the Bitbucket user UUID (e.g. "{abc-123}"). Optional fallback lookup key.
	BitbucketUUID string `yaml:"bitbucket_uuid,omitempty"`

	// Email is the user's email address (e.g. "bagus@example.com").
	// Will be displayed in the Google Chat message as "Author: bagus@example.com".
	Email string `yaml:"email"`
}

// AuthConfig holds authentication credentials for supported providers.
type AuthConfig struct {
	Bitbucket BitbucketAuth `yaml:"bitbucket"`
}

// BitbucketAuth supports two authentication strategies:
//   - Atlassian API Token: email + api_token (recommended)
//   - Legacy App Password: username + app_password
type BitbucketAuth struct {
	Email       string `yaml:"email"`
	APIToken    string `yaml:"api_token"`
	Username    string `yaml:"username"`
	AppPassword string `yaml:"app_password"`
}

// ProjectConfig maps a repository to its review instructions, guidelines,
// PR description settings, and reviewer defaults.
type ProjectConfig struct {
	// RepoSlug is "workspace/repo" or "*" for wildcard fallback.
	RepoSlug string `yaml:"repo_slug"`

	// ReviewInstructions is an absolute path to a markdown file with code review criteria.
	ReviewInstructions string `yaml:"review_instructions"`

	// ProjectGuidelines is an absolute path to a markdown file with project-specific guidelines.
	ProjectGuidelines string `yaml:"project_guidelines"`

	// DefaultTargetBranch is the default branch to compare against (e.g. "develop", "master").
	DefaultTargetBranch string `yaml:"default_target_branch"`

	// DescriptionMode controls the PR description detail level: "comprehensive" or "concise".
	DescriptionMode string `yaml:"description_mode"`

	// PRDescriptionTemplate is an absolute path to a custom markdown template for PR descriptions.
	// If empty, the built-in template matching DescriptionMode is used.
	PRDescriptionTemplate string `yaml:"pr_description_template"`

	// DefaultReviewers is a list of Bitbucket user UUIDs to assign as reviewers.
	DefaultReviewers []string `yaml:"default_reviewers"`

	// CloseSourceBranch controls whether the source branch is deleted after merge.
	CloseSourceBranch bool `yaml:"close_source_branch"`

	// GChatWebhookURL overrides the top-level gchat.webhook_url for this specific project.
	// Leave empty to use the global webhook.
	GChatWebhookURL string `yaml:"gchat_webhook_url"`
}
