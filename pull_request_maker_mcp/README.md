# Pull Request Maker MCP

A Go-based [Model Context Protocol](https://modelcontextprotocol.io/) (MCP) server that combines **code review** and **pull request creation** into a single workflow for Bitbucket Cloud.

This is a Go rewrite and superset of the TypeScript `code_review_mcp`, adding PR creation and standardized description generation.

## Features

- **Code Review**: Fetches PR or branch diffs with configurable review instructions and project guidelines
- **PR Creation**: Creates pull requests on Bitbucket with configurable reviewers, target branch, and description
- **PR Description Templates**: Generates standardized PR descriptions in comprehensive or concise mode
- **Review Comments**: Posts inline and general review comments directly on Bitbucket PRs
- **Per-Project Config**: Different review instructions, guidelines, reviewers, and defaults per repository
- **Backward Compatible**: `get_pr_diff` and `post_pr_comments` tools match the TypeScript MCP interface

## Prerequisites

- Go 1.23+
- Bitbucket Cloud account with API access
- VS Code with GitHub Copilot (or any MCP-compatible client)

## Setup

### 1. Build

```bash
cd mcp/pull_request_maker_mcp
make build
```

### 2. Configure

```bash
make copy-config   # creates config.yaml from config.example.yaml
```

Edit `config.yaml` with your Bitbucket credentials and project mappings. See [Configuration](#configuration) for details.

### 3. Register in VS Code

Add to your VS Code `settings.json`:

```json
{
  "mcp": {
    "servers": {
      "pr-maker": {
        "command": "/full/path/to/mcp/pull_request_maker_mcp/bin/pr-maker-mcp",
        "args": ["-config", "/full/path/to/mcp/pull_request_maker_mcp/config.yaml"]
      }
    }
  }
}
```

Reload the VS Code window after adding the configuration.

## Tools

| Tool | Description |
|------|-------------|
| `get_pr_diff` | Fetch diff, metadata, and review instructions for an existing Bitbucket PR |
| `get_branch_diff` | Fetch diff between two branches for review + PR description generation |
| `create_pr` | Create a new pull request on Bitbucket |
| `post_pr_comments` | Post review comments (inline or general) on a Bitbucket PR |

### get_pr_diff

Fetches everything needed to review an existing PR.

**Input:**
- `pr_url` (string, required) - Full Bitbucket PR URL

**Example prompt:** "Review this PR: https://bitbucket.org/mid-kelola-indonesia/talenta-data-api/pull-requests/123"

### get_branch_diff

Fetches diff between branches and returns review instructions + PR description template.

**Input:**
- `repo_slug` (string, required) - e.g. `mid-kelola-indonesia/talenta-data-api`
- `source_branch` (string, required) - Feature branch name
- `target_branch` (string, optional) - Falls back to config default

**Example prompt:** "Review changes on branch feature/TD-1234 against develop in mid-kelola-indonesia/talenta-data-api and generate a PR description"

### create_pr

Creates a pull request on Bitbucket.

**Input:**
- `repo_slug` (string, required)
- `source_branch` (string, required)
- `target_branch` (string, optional) - Falls back to config default
- `title` (string, required)
- `description` (string, required) - Markdown PR description
- `reviewers` (string[], optional) - Bitbucket user UUIDs; falls back to config defaults
- `close_source_branch` (bool, optional) - Falls back to config default

**Example prompt:** "Create a PR for branch feature/TD-1234 to develop with title 'TD-1234: Add user validation'"

### post_pr_comments

Posts review comments on an existing PR.

**Input:**
- `pr_url` (string, required)
- `comments` (array, required) - Each has `path`, `line`, `content`

## Configuration

### Auth

Two authentication methods are supported:

**Option 1 (Recommended): Atlassian API Token**
```yaml
auth:
  bitbucket:
    email: "your-email@example.com"
    api_token: "ATATT3x..."
```
Create at: https://id.atlassian.com/manage-profile/security/api-tokens

**Option 2 (Legacy): App Password**
```yaml
auth:
  bitbucket:
    username: "your-username"
    app_password: "your-app-password"
```
Create at: https://bitbucket.org/account/settings/app-passwords/
Required permissions: Repositories (Read), Pull requests (Read + Write)

### Projects

Each project entry maps a repository to its review settings:

```yaml
projects:
  - repo_slug: "mid-kelola-indonesia/talenta-data-api"
    review_instructions: "/path/to/go-code-review-instruction.md"
    project_guidelines: "/path/to/copilot-instructions.md"
    default_target_branch: "master"
    description_mode: "comprehensive"    # or "concise"
    pr_description_template: ""          # optional custom template path
    default_reviewers:
      - "{uuid-reviewer-1}"
    close_source_branch: false

  - repo_slug: "*"                       # wildcard fallback
    default_target_branch: "master"
    description_mode: "comprehensive"
```

### PR Description Modes

- **comprehensive**: Full template with Summary, Motivation, Changes, Type of Change, Testing, Screenshots, and Checklist sections
- **concise**: Brief template with Summary, Key Changes, and Testing Done

You can also provide a custom template via `pr_description_template` path.

## Usage Examples

### Full workflow: Review + Create PR

```
User: "Review changes on branch feature/TD-1234 against develop in
       mid-kelola-indonesia/talenta-data-api and create a PR"
```

Copilot will:
1. Call `get_branch_diff` to fetch the diff and review instructions
2. Analyze the diff and present review findings
3. Generate a PR description using the configured template
4. Call `create_pr` to create the PR on Bitbucket

### Review an existing PR

```
User: "Review this PR: https://bitbucket.org/mid-kelola-indonesia/talenta-data-api/pull-requests/123"
```

### Post selected review comments

```
User: "Post comments 1, 3, and 5 to the PR"
```

## Project Structure

```
pull_request_maker_mcp/
├── main.go                          # Entry point
├── Makefile                         # Build commands
├── config.example.yaml              # Config template
├── internal/
│   ├── config/
│   │   ├── types.go                 # Config structs
│   │   └── loader.go                # YAML loader + validation
│   ├── bitbucket/
│   │   ├── types.go                 # API types
│   │   ├── auth.go                  # Auth header builder
│   │   ├── client.go                # HTTP client
│   │   └── pr.go                    # PR operations
│   ├── urlparser/
│   │   └── parser.go                # PR URL + repo slug parser
│   ├── tools/
│   │   ├── get_pr_diff.go           # get_pr_diff tool
│   │   ├── get_branch_diff.go       # get_branch_diff tool
│   │   ├── create_pr.go             # create_pr tool
│   │   └── post_comments.go         # post_pr_comments tool
│   └── prdesc/
│       ├── template.go              # Template loader (embed)
│       └── templates/
│           ├── comprehensive.md     # Full PR description template
│           └── concise.md           # Brief PR description template
└── pr_templates/                    # (empty, for user custom templates)
```

## Security Notes

- `config.yaml` contains API credentials and is gitignored
- Never commit `config.yaml` to version control
- Use Atlassian API Tokens (Option 1) over App Passwords when possible
- The MCP server runs locally via stdio transport only
