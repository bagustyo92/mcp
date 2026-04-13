//go:build integration

package confluence_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"bitbucket.org/mid-kelola-indonesia/integration-deployment-tools-mcp/internal/config"
	"bitbucket.org/mid-kelola-indonesia/integration-deployment-tools-mcp/internal/confluence"
)

// loadTestConfig loads config.yaml from the project root (two levels up from internal/confluence/).
func loadTestConfig(t *testing.T) *config.AppConfig {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}

	// internal/confluence/ -> internal/ -> project root
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	configPath := filepath.Join(projectRoot, "config.yaml")

	// Allow override via environment variable
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		configPath = envPath
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config from %s: %v", configPath, err)
	}
	return cfg
}

// TestGetSpaceID verifies that we can authenticate and resolve the configured space key.
func TestGetSpaceID(t *testing.T) {
	cfg := loadTestConfig(t)
	client := confluence.NewClient()
	ctx := context.Background()

	spaceKey := cfg.Confluence.SpaceKey
	if spaceKey == "" {
		t.Skip("confluence.space_key not configured")
	}

	t.Logf("Testing GetSpaceID for space key: %s", spaceKey)
	t.Logf("Confluence base URL: %s", cfg.Confluence.BaseURL)
	t.Logf("Auth: email=%s, api_token_set=%v", cfg.Auth.Bitbucket.Email, cfg.Auth.Bitbucket.APIToken != "")

	spaceID, err := client.GetSpaceID(ctx, cfg.Auth.Bitbucket, cfg.Confluence.BaseURL, spaceKey)
	if err != nil {
		t.Fatalf("GetSpaceID failed: %v\n\n"+
			"Troubleshooting:\n"+
			"  1. 401 Unauthorized -> API token invalid or account has no access to %s\n"+
			"     Generate a new token at: https://id.atlassian.net/manage-profile/security/api-tokens\n"+
			"     Ensure your Atlassian account (%s) has been invited to %s\n"+
			"  2. 403 Forbidden -> account exists but lacks page read permission\n"+
			"  3. 404 Not Found -> space key %q doesn't exist on this site\n",
			err,
			cfg.Confluence.BaseURL,
			cfg.Auth.Bitbucket.Email,
			cfg.Confluence.BaseURL,
			spaceKey,
		)
	}

	t.Logf("SUCCESS: Space %q resolved to ID: %s", spaceKey, spaceID)
}

// TestGetTemplatePage verifies that the configured template page can be fetched.
func TestGetTemplatePage(t *testing.T) {
	cfg := loadTestConfig(t)
	client := confluence.NewClient()
	ctx := context.Background()

	pageID := cfg.Confluence.TemplatePageID
	if pageID == "" {
		t.Skip("confluence.template_page_id not configured")
	}

	t.Logf("Testing GetPage for template page ID: %s", pageID)

	page, err := client.GetPage(ctx, cfg.Auth.Bitbucket, cfg.Confluence.BaseURL, pageID)
	if err != nil {
		t.Fatalf("GetPage (template) failed: %v", err)
	}

	t.Logf("SUCCESS: Template page fetched")
	t.Logf("  Title: %s", page.Title)
	t.Logf("  URL:   %s", page.URL)
	t.Logf("  Body length: %d chars", len(page.Body))
}

// TestGetParentPage verifies that the configured parent page can be fetched.
func TestGetParentPage(t *testing.T) {
	cfg := loadTestConfig(t)
	client := confluence.NewClient()
	ctx := context.Background()

	pageID := cfg.Confluence.ParentPageID
	if pageID == "" {
		t.Skip("confluence.parent_page_id not configured")
	}

	t.Logf("Testing GetPage for parent page ID: %s", pageID)

	page, err := client.GetPage(ctx, cfg.Auth.Bitbucket, cfg.Confluence.BaseURL, pageID)
	if err != nil {
		t.Fatalf("GetPage (parent) failed: %v", err)
	}

	t.Logf("SUCCESS: Parent page fetched")
	t.Logf("  Title: %s", page.Title)
	t.Logf("  URL:   %s", page.URL)
}

// TestRawAuthCheck sends a raw request and prints the HTTP status to help diagnose auth issues.
func TestRawAuthCheck(t *testing.T) {
	cfg := loadTestConfig(t)
	ctx := context.Background()

	// We test the /wiki/api/v2/spaces endpoint which is one of the simplest authenticated calls.
	url := fmt.Sprintf("%s/wiki/api/v2/spaces?limit=1", cfg.Confluence.BaseURL)

	t.Logf("Auth check URL: %s", url)
	t.Logf("Using email:    %s", cfg.Auth.Bitbucket.Email)

	// Re-use the client's exported behaviour via GetSpaceID with a dummy key to hit the endpoint.
	// Instead, we directly use the exported NewClient and verify error message.
	client := confluence.NewClient()
	_, err := client.GetSpaceID(ctx, cfg.Auth.Bitbucket, cfg.Confluence.BaseURL, "NONEXISTENT_SPACE_KEY_CHECK")
	if err != nil {
		// 401 -> credentials wrong or no access to site
		// 404 -> space not found (but auth passed!) - this is actually a success for auth
		t.Logf("Raw auth check result: %v", err)
		if contains(err.Error(), "401") {
			t.Error("FAILED: 401 Unauthorized\n" +
				"Your API token does not work for this Confluence site.\n" +
				"Steps to fix:\n" +
				"  1. Go to https://id.atlassian.net/manage-profile/security/api-tokens\n" +
				"  2. Create a NEW API token\n" +
				"  3. Ensure your account " + cfg.Auth.Bitbucket.Email + " has access to " + cfg.Confluence.BaseURL + "\n" +
				"     (Ask a Confluence admin to invite you if needed)\n" +
				"  4. Update config.yaml with the new token")
		} else if contains(err.Error(), "403") {
			t.Error("FAILED: 403 Forbidden — account exists but lacks permission to list spaces")
		} else {
			// Any other error (e.g., space not found with 200/404) means auth is likely fine
			t.Logf("Non-auth error (auth may be OK): %v", err)
		}
		return
	}

	t.Log("SUCCESS: Authentication and Confluence access confirmed")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
