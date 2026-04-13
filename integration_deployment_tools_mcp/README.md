# Integration Deployment Tools MCP

A Go-based MCP (Model Context Protocol) server that provides deployment workflow automation tools for Bitbucket repositories. It enables AI assistants (e.g. VS Code Copilot, Claude) to detect undeployed changes, create release tags, generate Confluence deployment docs, and trigger Bitbucket pipelines.

## Tools

| Tool | Description |
|------|-------------|
| `get_undeployed_changes` | Identify code changes merged to the default branch but not yet released (no tag). Returns per-repo lists of PRs with Jira tickets. Optionally sends a summary to Google Chat. |
| `create_release_tag` | Create a Git tag on a repository to mark a release. Auto-increments the patch version when no tag name is provided. |
| `create_deployment_doc` | Create a Confluence page from a template, populated with deployment details (changes summary, tag, repositories). |
| `trigger_pipeline` | Trigger a Bitbucket Pipeline for a given repository and environment (e.g. staging, production). |

## Prerequisites

- Go 1.25+
- Bitbucket Cloud account with API access (App Password or API Token)
- (Optional) Google Chat webhook URL for notifications
- (Optional) Confluence Cloud account for deployment documentation

## Setup

### 1. Clone & Build

```bash
cd mcp/integration_deployment_tools_mcp
make copy-config   # creates config.yaml from config.example.yaml
make build         # compiles to bin/deployment-tools-mcp
```

### 2. Configure

Edit `config.yaml` with your credentials and repository settings:

```yaml
auth:
  bitbucket:
    email: "you@example.com"
    api_token: "your-bitbucket-api-token"

gchat:
  webhook_url: "https://chat.googleapis.com/v1/spaces/..."
  jira_base_url: "https://yourcompany.atlassian.net/browse"

confluence:
  base_url: "https://yourcompany.atlassian.net"
  space_key: "DEPLOY"
  parent_page_id: "123456"
  template_page_id: "789012"

repositories:
  - repo_slug: "workspace/repo-name"
    default_branch: "master"
    pipelines:
      production:
        name: "deploy-production"
        ref_type: "tag"
      ppe:
        name: "deploy-ppe"
        ref_type: "commit"
```

See [`config.example.yaml`](config.example.yaml) for a full example with multiple repositories.

### 3. VS Code MCP Configuration

Add to your VS Code `settings.json` (or `.vscode/mcp.json`):

```json
{
  "mcp": {
    "servers": {
      "integration-deployment-tools": {
        "type": "stdio",
        "command": "/absolute/path/to/bin/deployment-tools-mcp",
        "args": ["-config", "/absolute/path/to/config.yaml"]
      }
    }
  }
}
```

Or in `.vscode/mcp.json`:

```json
{
  "servers": {
    "integration-deployment-tools": {
      "type": "stdio",
      "command": "${workspaceFolder}/mcp/integration_deployment_tools_mcp/bin/deployment-tools-mcp",
      "args": ["-config", "${workspaceFolder}/mcp/integration_deployment_tools_mcp/config.yaml"]
    }
  }
}
```

## Tool Details

### get_undeployed_changes

Detects PRs merged after the latest production tag.

**Input:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `repo_slug` | string | No | Full slug (`workspace/repo`). Empty = all configured repos. |
| `send_notification` | bool | No | Send results to Google Chat. Default `false`. |

**Output:** JSON with per-repo list of undeployed changes including Jira ticket, PR URL, author, and commit hash.

**Example prompt:**
> "What changes haven't been deployed yet for talenta-data-api?"

---

### create_release_tag

Creates a semver Git tag on the repository.

**Input:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `repo_slug` | string | Yes | Full slug (`workspace/repo`). |
| `tag_name` | string | No | Explicit tag name. If empty, auto-increments patch from latest tag. |
| `commit_hash` | string | No | Commit to tag. If empty, uses HEAD of default branch. |
| `send_notification` | bool | No | Notify Google Chat. Default `false`. |

**Output:** JSON with tag name, commit hash, previous tag, and tag URL.

**Example prompt:**
> "Create a release tag for disbursement-service and notify the team."

---

### create_deployment_doc

Creates a Confluence page from a template.

**Input:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `title` | string | Yes | Page title. |
| `reference_page_id` | string | No | Template page ID. Falls back to config's `template_page_id`. |
| `changes_summary` | string | No | Summary text to embed. |
| `tag_name` | string | No | Release tag name. |
| `repo_slugs` | []string | No | Repositories in this deployment. |

**Output:** JSON with page ID, URL, title, and space key.

**Template placeholders:** `{{TITLE}}`, `{{DATE}}`, `{{TAG}}`, `{{CHANGES}}`, `{{REPOS}}`

**Example prompt:**
> "Create a deployment doc titled 'Sprint 24.3 Release' with the changes from the last tag."

---

### trigger_pipeline

Triggers a Bitbucket Pipeline.

**Input:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `repo_slug` | string | Yes | Full slug (`workspace/repo`). |
| `environment` | string | Yes | Target environment (e.g. `staging`, `production`). |
| `ref_name` | string | No | Branch or tag. If empty, auto-detects from config. |
| `send_notification` | bool | No | Notify Google Chat. Default `false`. |

**Output:** JSON with pipeline UUID, build number, URL, and ref details.

**Example prompt:**
> "Deploy talenta-data-api to staging."

## Configuration Reference

### auth.bitbucket

| Field | Description |
|-------|-------------|
| `email` | Bitbucket account email (used with API token auth) |
| `api_token` | Bitbucket API token |
| `username` | Alternative: Bitbucket username (used with App Password) |
| `app_password` | Alternative: Bitbucket App Password |

### gchat

| Field | Description |
|-------|-------------|
| `webhook_url` | Default Google Chat webhook URL |
| `jira_base_url` | Jira base URL for building ticket links (e.g. `https://company.atlassian.net/browse`) |

### confluence

| Field | Description |
|-------|-------------|
| `base_url` | Confluence Cloud base URL |
| `space_key` | Space key for deployment docs |
| `parent_page_id` | Parent page under which docs are created |
| `template_page_id` | Default template page ID to clone |

### repositories[]

| Field | Description |
|-------|-------------|
| `repo_slug` | Full slug `workspace/repo-name` (use `*` for wildcard fallback) |
| `default_branch` | Default branch name (default: `master`) |
| `gchat_webhook_url` | Per-repo webhook override |
| `pipelines.<env>.name` | Custom pipeline name for the environment |
| `pipelines.<env>.ref_type` | `branch` or `tag` |

## Development

```bash
make build    # Build binary
make run      # Build and run
make test     # Run tests
make clean    # Remove build artifacts
```

## Project Structure

```
integration_deployment_tools_mcp/
├── main.go                          # Entry point, DI wiring
├── config.example.yaml              # Example configuration
├── Makefile                         # Build commands
├── internal/
│   ├── config/
│   │   ├── types.go                 # Config structs
│   │   └── loader.go               # YAML loader + validation
│   ├── bitbucket/
│   │   ├── auth.go                  # Auth helpers
│   │   ├── client.go               # HTTP client
│   │   ├── types.go                # API response types
│   │   ├── tags.go                 # Tag operations
│   │   ├── commits.go             # Commit operations
│   │   ├── pullrequests.go        # PR operations
│   │   ├── pipelines.go           # Pipeline operations
│   │   └── helpers.go             # JSON utilities
│   ├── confluence/
│   │   └── client.go              # Confluence API client
│   ├── gchat/
│   │   └── notifier.go            # Google Chat webhook
│   ├── jira/
│   │   ├── extractor.go           # Jira ticket extraction
│   │   └── extractor_test.go      # Tests
│   ├── tagversion/
│   │   ├── parser.go              # Semver parsing
│   │   └── parser_test.go         # Tests
│   └── tools/
│       ├── helpers.go              # Shared tool utilities
│       ├── get_undeployed_changes.go
│       ├── create_release_tag.go
│       ├── create_deployment_doc.go
│       └── trigger_pipeline.go
└── bin/                             # Build output (gitignored)
```
