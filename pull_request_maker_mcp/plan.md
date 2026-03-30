# Plan: Pull Request Maker MCP (Go)

## TL;DR
Build a Go MCP server that combines code review + PR creation into a single workflow. The user tells Copilot to "create a PR for my branch", and the MCP fetches the diff, lets Copilot review it using configurable instructions, generates a standardized PR description (comprehensive or concise), and creates the PR on Bitbucket — all from within VS Code.

This replaces the existing TypeScript `code_review_mcp` with a Go rewrite AND adds PR creation + description generation capabilities.

---

## Phase 1: Project Scaffolding & Config

**Steps:**

1. Initialize Go module at `mcp/pull_request_maker_mcp/`
   - `go mod init bitbucket.org/mid-kelola-indonesia/pull-request-maker-mcp`
   - Add dependency: `github.com/modelcontextprotocol/go-sdk@latest`
   - Add dependency: `gopkg.in/yaml.v3` for YAML config parsing

2. Create config system (`internal/config/`)
   - `types.go` — Config structs: `AppConfig`, `BitbucketAuth`, `ProjectConfig`, `PRDescriptionConfig`
   - `loader.go` — YAML loader with validation, project matcher (exact + wildcard `*` fallback), file content loader
   - `config.example.yaml` — Template with all options documented
   - Config extends existing `code_review_mcp` config with new fields:
     - `pr_description_template` — path to markdown template for PR descriptions
     - `description_mode` — `"comprehensive"` or `"concise"` (default: comprehensive)
     - `default_target_branch` — per-project default target branch (e.g., `develop`, `master`)
     - `default_reviewers` — list of reviewer UUIDs per project
     - `close_source_branch` — boolean, default false

3. Create `config.yaml` (gitignored) and `.gitignore`

**Files:**
- `mcp/pull_request_maker_mcp/go.mod`
- `mcp/pull_request_maker_mcp/internal/config/types.go`
- `mcp/pull_request_maker_mcp/internal/config/loader.go`
- `mcp/pull_request_maker_mcp/config.example.yaml`
- `mcp/pull_request_maker_mcp/.gitignore`

---

## Phase 2: Bitbucket API Client (Go Port)

**Steps:**

4. Create Bitbucket provider (`internal/bitbucket/`)
   - `auth.go` — Auth header builder (email+API token OR username+app_password → Basic auth)
   - `client.go` — HTTP client wrapper with `net/http`, base URL, auth headers, error handling
   - `pr.go` — PR-specific operations:
     - `FetchPRMetadata(ctx, prInfo)` → title, description, branches, author, state
     - `FetchPRDiff(ctx, prInfo, src, dst)` → unified diff text (using branch-comparison endpoint)
     - `ExtractChangedFiles(diff)` → []string from diff header parsing
     - `CreatePR(ctx, workspace, repoSlug, req)` → PR URL + ID (POST `/repositories/{ws}/{repo}/pullrequests`)
     - `PostInlineComment(ctx, prInfo, comment)` → success/failure
     - `PostGeneralComment(ctx, prInfo, content)` → success/failure
     - `FetchDefaultReviewers(ctx, workspace, repoSlug)` → []Reviewer (GET `/effective-default-reviewers`)
   - `types.go` — Request/response structs for Bitbucket API

5. Create URL parser utility (`internal/urlparser/`)
   - `parser.go` — Parse Bitbucket PR URLs and repo URLs into `PRInfo` struct
   - Support formats: `https://bitbucket.org/{workspace}/{repo}/pull-requests/{id}`

**Files:**
- `mcp/pull_request_maker_mcp/internal/bitbucket/auth.go`
- `mcp/pull_request_maker_mcp/internal/bitbucket/client.go`
- `mcp/pull_request_maker_mcp/internal/bitbucket/pr.go`
- `mcp/pull_request_maker_mcp/internal/bitbucket/types.go`
- `mcp/pull_request_maker_mcp/internal/urlparser/parser.go`

---

## Phase 3: MCP Tools Implementation

**Steps:**

6. Create MCP tool: `get_branch_diff` (`internal/tools/get_branch_diff.go`)
   - Input: `repo_slug` (string), `source_branch` (string), `target_branch` (string, optional — falls back to config default)
   - Flow:
     1. Match project config by repo_slug
     2. Fetch diff between source and target branches
     3. Extract changed files from diff
     4. Load review_instructions and project_guidelines from disk
     5. Return diff + metadata + instructions to LLM
   - Output: JSON with diff, changed_files, review_instructions, project_guidelines, instructions_to_llm

7. Create MCP tool: `review_and_create_pr` (`internal/tools/create_pr.go`)
   - Input: `repo_slug` (string), `source_branch` (string), `target_branch` (string, optional), `title` (string), `description` (string — LLM-generated), `reviewers` ([]string, optional — UUIDs), `close_source_branch` (bool, optional), `draft` (bool, optional)
   - Flow:
     1. Match project config for defaults
     2. Call Bitbucket API to create PR
     3. Return PR URL + ID
   - Output: JSON with pr_url, pr_id, status

8. Create MCP tool: `post_review_comments` (`internal/tools/post_comments.go`)
   - Port of existing `post_pr_comments` from TypeScript
   - Input: `pr_url` (string), `comments` (array of {path, line, content})
   - Flow: Post each comment with 200ms delay, return success/failure summary

9. Create MCP tool: `get_pr_diff` (`internal/tools/get_pr_diff.go`)
   - Port of existing `get_pr_diff` from TypeScript — backward compatible
   - Input: `pr_url` (string)
   - Flow: Parse URL → fetch metadata + diff → load instructions → return package

10. *(Optional, future)* Create MCP tool: `merge_pr`
    - Input: `pr_url` (string), `merge_strategy` (string)
    - Flow: Call Bitbucket merge endpoint

**Files:**
- `mcp/pull_request_maker_mcp/internal/tools/get_branch_diff.go`
- `mcp/pull_request_maker_mcp/internal/tools/create_pr.go`
- `mcp/pull_request_maker_mcp/internal/tools/post_comments.go`
- `mcp/pull_request_maker_mcp/internal/tools/get_pr_diff.go`

---

## Phase 4: PR Description Template System

**Steps:**

11. Create PR description template system (`internal/prdesc/`)
    - `template.go` — Template loader + renderer
    - Templates stored as markdown files in `pr_templates/` directory
    - Two built-in templates:
      - `comprehensive.md` — Full details: Summary, Motivation, Changes, Testing, Screenshots, Checklist
      - `concise.md` — Brief: Summary, Key Changes, Testing Done
    - Config selects which template via `description_mode`
    - The template is returned to the LLM as part of `get_branch_diff` output, so Copilot generates the description following the template
    - Instructions tell the LLM: "Generate a PR description following this template format based on the diff"

**Files:**
- `mcp/pull_request_maker_mcp/internal/prdesc/template.go`
- `mcp/pull_request_maker_mcp/pr_templates/comprehensive.md`
- `mcp/pull_request_maker_mcp/pr_templates/concise.md`

---

## Phase 5: Server Entry Point & Build

**Steps:**

12. Create MCP server entry point (`main.go`)
    - Load config
    - Create MCP server with `mcp.NewServer`
    - Register all tools via `mcp.AddTool`
    - Run with `mcp.StdioTransport{}` (VS Code integration)
    - Graceful shutdown via signal handling

13. Create Makefile
    - `make build` — `go build -o bin/pr-maker-mcp`
    - `make run` — build + run
    - `make test` — run tests
    - `make copy-config` — copy config.example.yaml to config.yaml

14. Create VS Code MCP configuration example
    - Document in README the settings.json snippet to register the MCP server

**Files:**
- `mcp/pull_request_maker_mcp/main.go`
- `mcp/pull_request_maker_mcp/Makefile`

---

## Phase 6: Documentation

**Steps:**

15. Create README.md
    - Overview: What the MCP does (combined review + PR creation)
    - Prerequisites: Go 1.23+, Bitbucket API token
    - Setup: build, configure, register in VS Code
    - Usage examples (Copilot chat commands):
      - "Review changes on branch X against develop and create a PR"
      - "Create a PR for feature/my-branch to develop with concise description"
      - "Review this PR: https://bitbucket.org/..."
    - Tool reference table
    - Configuration reference
    - PR Description modes (comprehensive vs concise)
    - Security notes

**Files:**
- `mcp/pull_request_maker_mcp/README.md`

---

## Relevant Files

### Existing (reference/port from):
- `mcp/code_review_mcp/src/config/types.ts` — Config type definitions to port
- `mcp/code_review_mcp/src/config/loader.ts` — Config loading logic to port
- `mcp/code_review_mcp/src/providers/bitbucket.ts` — Bitbucket API client to port (auth, fetch metadata, fetch diff, post comments)
- `mcp/code_review_mcp/src/tools/get-pr-diff.ts` — Tool logic to port
- `mcp/code_review_mcp/src/tools/post-comments.ts` — Tool logic to port
- `mcp/code_review_mcp/src/utils/pr-url-parser.ts` — URL parser to port
- `mcp/code_review_mcp/config.yaml` — Reference for project config structure
- `mcp/code_review_mcp/review_instructions/` — Reuse existing review instruction markdown files

### Go MCP SDK reference:
- `disbursement-service/.github/go-mcp-server.instructions.md` — Official Go MCP SDK patterns (`mcp.NewServer`, `mcp.AddTool`, struct-based tools with `jsonschema` tags, `StdioTransport`)

### New files to create:
- `mcp/pull_request_maker_mcp/go.mod`
- `mcp/pull_request_maker_mcp/main.go`
- `mcp/pull_request_maker_mcp/Makefile`
- `mcp/pull_request_maker_mcp/.gitignore`
- `mcp/pull_request_maker_mcp/config.example.yaml`
- `mcp/pull_request_maker_mcp/internal/config/types.go`
- `mcp/pull_request_maker_mcp/internal/config/loader.go`
- `mcp/pull_request_maker_mcp/internal/bitbucket/auth.go`
- `mcp/pull_request_maker_mcp/internal/bitbucket/client.go`
- `mcp/pull_request_maker_mcp/internal/bitbucket/pr.go`
- `mcp/pull_request_maker_mcp/internal/bitbucket/types.go`
- `mcp/pull_request_maker_mcp/internal/urlparser/parser.go`
- `mcp/pull_request_maker_mcp/internal/tools/get_branch_diff.go`
- `mcp/pull_request_maker_mcp/internal/tools/create_pr.go`
- `mcp/pull_request_maker_mcp/internal/tools/post_comments.go`
- `mcp/pull_request_maker_mcp/internal/tools/get_pr_diff.go`
- `mcp/pull_request_maker_mcp/internal/prdesc/template.go`
- `mcp/pull_request_maker_mcp/pr_templates/comprehensive.md`
- `mcp/pull_request_maker_mcp/pr_templates/concise.md`
- `mcp/pull_request_maker_mcp/README.md`

---

## Verification

1. **Build**: `cd mcp/pull_request_maker_mcp && go build -o bin/pr-maker-mcp` — must compile without errors
2. **Unit tests**: `go test ./...` — test config loader, URL parser, diff file extraction, template rendering
3. **Config validation**: Load config.example.yaml, verify parsing and validation logic
4. **Integration test (manual)**: 
   - Register MCP in VS Code settings.json
   - Reload VS Code window
   - In Copilot chat: "Get the diff for branch feature/test against develop in mid-kelola-indonesia/talenta-data-api"
   - Verify diff + review instructions returned correctly
5. **PR creation test (manual)**:
   - In Copilot chat: "Create a PR for branch feature/test to develop in mid-kelola-indonesia/talenta-data-api with title 'Test PR'"
   - Verify PR created on Bitbucket with correct title, description, branch, reviewers
6. **Code review test (manual)**:
   - In Copilot chat: "Review this PR: https://bitbucket.org/mid-kelola-indonesia/talenta-data-api/pull-requests/123"
   - Verify backward compatibility with existing `get_pr_diff` + `post_pr_comments` workflow
7. **Description modes**: Test both `comprehensive` and `concise` description templates produce different LLM instruction formats

---

## Decisions

- **Language**: Go (per user request — easier to read and debug than TypeScript)
- **Go MCP SDK**: `github.com/modelcontextprotocol/go-sdk` (official SDK, documented in workspace)
- **Config format**: YAML (consistent with existing `code_review_mcp`)
- **Auth**: Same dual-strategy as existing (email+API token OR username+app_password → HTTP Basic)
- **Transport**: Stdio only (VS Code Copilot integration)
- **Scope includes**: Code review tools (port from TS), PR creation tool, PR description generation, configurable templates
- **Scope excludes**: Auto-merge (deferred to future iteration), reviewer auto-assignment by expertise (deferred — only manual/config-based reviewers for now)
- **Review instructions**: Reuse existing markdown files from `code_review_mcp/review_instructions/` (shared path in config)
- **Backward compatible**: The `get_pr_diff` and `post_pr_comments` tools preserve the same interface as the TypeScript version

## Further Considerations

1. **Shared review instructions**: The review instruction markdown files live in `code_review_mcp/review_instructions/`. Should we move them to a shared location (e.g., `mcp/review_instructions/`) so both MCPs can reference them? **Recommendation**: Yes, move to shared location, update config paths.

2. **Deprecating TypeScript MCP**: Once the Go MCP is stable, should we deprecate `code_review_mcp`? **Recommendation**: Yes — the Go version is a superset. Keep the TS version around briefly for fallback, then remove.

3. **Config compatibility**: The new Go MCP config is a superset of the existing TS config. Should we make it fully backward-compatible (accept the old config format without new fields)? **Recommendation**: Yes — use defaults for all new fields so existing configs work.
