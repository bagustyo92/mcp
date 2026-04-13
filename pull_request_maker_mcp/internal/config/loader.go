package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads and validates the config file from the given path.
// If configPath is empty, it defaults to config.yaml next to the executable.
func Load(configPath string) (*AppConfig, error) {
	if configPath == "" {
		exe, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("resolve executable path: %w", err)
		}
		configPath = filepath.Join(filepath.Dir(exe), "config.yaml")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf(
			"config file not found at %s\nCopy config.example.yaml to config.yaml and fill in your credentials",
			configPath,
		)
	}

	var cfg AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validate(cfg *AppConfig) error {
	bb := cfg.Auth.Bitbucket

	hasToken := strings.TrimSpace(bb.Email) != "" && strings.TrimSpace(bb.APIToken) != ""
	hasAppPassword := strings.TrimSpace(bb.Username) != "" && strings.TrimSpace(bb.AppPassword) != ""

	if !hasToken && !hasAppPassword {
		return fmt.Errorf(
			"config validation error: provide either:\n" +
				"  - auth.bitbucket.email + auth.bitbucket.api_token (Atlassian API Token, recommended)\n" +
				"  - auth.bitbucket.username + auth.bitbucket.app_password (legacy App Password)",
		)
	}

	if len(cfg.Projects) == 0 {
		return fmt.Errorf("config validation error: at least one entry in 'projects' is required")
	}

	for i, p := range cfg.Projects {
		if strings.TrimSpace(p.RepoSlug) == "" {
			return fmt.Errorf("config validation error: projects[%d].repo_slug is required", i)
		}
	}

	return nil
}

// FindProjectConfig matches a workspace/repo-slug to a project config entry.
// Falls back to a wildcard "*" entry if no exact match is found.
func FindProjectConfig(cfg *AppConfig, workspace, repoSlug string) *ProjectConfig {
	fullSlug := strings.ToLower(workspace + "/" + repoSlug)

	for i := range cfg.Projects {
		if strings.ToLower(cfg.Projects[i].RepoSlug) == fullSlug {
			return &cfg.Projects[i]
		}
	}

	// Wildcard fallback
	for i := range cfg.Projects {
		if cfg.Projects[i].RepoSlug == "*" {
			return &cfg.Projects[i]
		}
	}

	return nil
}

// FindProjectConfigBySlug matches a full "workspace/repo" slug directly.
func FindProjectConfigBySlug(cfg *AppConfig, slug string) *ProjectConfig {
	lower := strings.ToLower(slug)

	for i := range cfg.Projects {
		if strings.ToLower(cfg.Projects[i].RepoSlug) == lower {
			return &cfg.Projects[i]
		}
	}

	// Wildcard fallback
	for i := range cfg.Projects {
		if cfg.Projects[i].RepoSlug == "*" {
			return &cfg.Projects[i]
		}
	}

	return nil
}

// FindAuthorEmail looks up the email for a Bitbucket author.
// It searches by display name (case-insensitive) first, then by UUID as fallback.
// Returns an empty string if no mapping is found.
func FindAuthorEmail(cfg *AppConfig, authorName, authorUUID string) string {
	for _, m := range cfg.GChat.UserMappings {
		if m.BitbucketName != "" && strings.EqualFold(m.BitbucketName, authorName) {
			return m.Email
		}
	}

	if authorUUID != "" {
		cleanUUID := strings.Trim(authorUUID, "{}")
		for _, m := range cfg.GChat.UserMappings {
			cleanMappingUUID := strings.Trim(m.BitbucketUUID, "{}")
			if cleanMappingUUID != "" && strings.EqualFold(cleanMappingUUID, cleanUUID) {
				return m.Email
			}
		}
	}

	return ""
}

// LoadFileContent reads a file from disk. Returns empty string if path is empty or file not found.
func LoadFileContent(filePath string) string {
	if strings.TrimSpace(filePath) == "" {
		return ""
	}

	resolved, err := filepath.Abs(filePath)
	if err != nil {
		return ""
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		return ""
	}

	return string(data)
}
