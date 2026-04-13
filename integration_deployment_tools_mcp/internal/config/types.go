package config

// AppConfig is the top-level configuration loaded from config.yaml.
type AppConfig struct {
	Auth         AuthConfig       `yaml:"auth"`
	GChat        GChatConfig      `yaml:"gchat"`
	Confluence   ConfluenceConfig `yaml:"confluence"`
	Repositories []RepoConfig     `yaml:"repositories"`
}

// AuthConfig holds authentication credentials for Bitbucket and Confluence.
type AuthConfig struct {
	Bitbucket BitbucketAuth `yaml:"bitbucket"`
}

// BitbucketAuth supports two authentication strategies:
//   - Atlassian API Token: email + api_token (recommended, works for both Bitbucket and Confluence)
//   - Legacy App Password: username + app_password
type BitbucketAuth struct {
	Email       string `yaml:"email"`
	APIToken    string `yaml:"api_token"`
	Username    string `yaml:"username"`
	AppPassword string `yaml:"app_password"`
}

// GChatConfig holds Google Chat notification settings.
type GChatConfig struct {
	WebhookURL  string `yaml:"webhook_url"`
	JiraBaseURL string `yaml:"jira_base_url"`
}

// ConfluenceConfig holds Confluence deployment document settings.
type ConfluenceConfig struct {
	BaseURL        string `yaml:"base_url"`
	SpaceKey       string `yaml:"space_key"`
	ParentPageID   string `yaml:"parent_page_id"`
	TemplatePageID string `yaml:"template_page_id"`
}

// RepoConfig maps a repository to its deployment configuration.
type RepoConfig struct {
	RepoSlug      string                    `yaml:"repo_slug"`
	DefaultBranch string                    `yaml:"default_branch"`
	TagPattern    string                    `yaml:"tag_pattern"`
	Pipelines     map[string]PipelineConfig `yaml:"pipelines"`
}

// PipelineConfig holds per-environment pipeline settings.
type PipelineConfig struct {
	Name    string `yaml:"name"`
	RefType string `yaml:"ref_type"`
}
