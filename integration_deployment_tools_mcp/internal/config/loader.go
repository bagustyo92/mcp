package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads and validates the YAML configuration file.
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

	if len(cfg.Repositories) == 0 {
		return fmt.Errorf("config validation error: at least one entry in 'repositories' is required")
	}

	for i, r := range cfg.Repositories {
		if strings.TrimSpace(r.RepoSlug) == "" {
			return fmt.Errorf("config validation error: repositories[%d].repo_slug is required", i)
		}
	}

	return nil
}

// FindRepoConfig looks up a repository config by its full slug (workspace/repo).
// Falls back to a wildcard ("*") entry if no exact match is found.
func FindRepoConfig(cfg *AppConfig, slug string) *RepoConfig {
	lower := strings.ToLower(slug)
	for i := range cfg.Repositories {
		if strings.ToLower(cfg.Repositories[i].RepoSlug) == lower {
			return &cfg.Repositories[i]
		}
	}
	for i := range cfg.Repositories {
		if cfg.Repositories[i].RepoSlug == "*" {
			return &cfg.Repositories[i]
		}
	}
	return nil
}

// FindRepoConfigByParts looks up a repository config by workspace and repo separately.
func FindRepoConfigByParts(cfg *AppConfig, workspace, repo string) *RepoConfig {
	return FindRepoConfig(cfg, workspace+"/"+repo)
}
