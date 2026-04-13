# Plan: Integration Deployment Tools MCP

## TL;DR

Build a Go-based MCP server that automates deployment workflows: identifying undeployed code changes (merged to master but no production tag), creating release tags, generating Confluence deployment documents, and triggering Bitbucket pipelines. Follows the same architecture as the existing `pull_request_maker_mcp` — uses `github.com/modelcontextprotocol/go-sdk`, YAML config, manual DI, and stdio transport.

---

## Steps

### Phase 1: Project Scaffolding & Configuration

1. **Initialize Go module and project structure** — Create `go.mod` with module `bitbucket.org/mid-kelola-indonesia/integration-deployment-tools-mcp`, set up directory layout matching pull_request_maker_mcp pattern.

2. **Define configuration types** (`internal/config/types.go`) — Create structs for:
   - `AppConfig` (top-level: Auth, GChat, Confluence, Repositories)
   - `AuthConfig` (Bitbucket + Confluence/Atlassian creds — same API token works for both)
   - `GChatConfig` (webhook_url, jira_base_url)
   - `ConfluenceConfig` (base_url, space_key, parent_page_id, template_page_id)
   - `RepositoryConfig` (repo_slug, default_branch, tag_pattern, pipelines map, gchat_webhook_url override)
   - `PipelineConfig` (per environment: name, ref_type — "tag" or "commit")

3. **Implement config loader** (`internal/config/loader.go`) — YAML loading with validation (auth required, at least 1 repo). Include `FindRepoConfig(slug)` lookup with `"*"` wildcard fallback. *Reuse pattern from* `pull_request_maker_mcp/internal/config/loader.go`.

4. **Create config.example.yaml** — Full example covering all repos (TDA, disbursement, form, notification, etc.) with pipeline configs per environment.

5. **Create Makefile** — `build`, `run`, `clean`, `copy-config`, `test` targets matching existing pattern.

6. **Create .gitignore** — Ignore `config.yaml`, `bin/`, vendor.

### Phase 2: Bitbucket Client (Core API Layer)

7. **Implement auth module** (`internal/bitbucket/auth.go`) — Reuse same auth pattern: email+api_token or username+app_password → Basic Auth header. *Copy directly from* `pull_request_maker_mcp/internal/bitbucket/auth.go`.

8. **Implement HTTP client** (`internal/bitbucket/client.go`) — HTTP client with 30s timeout and `doRequest(ctx, method, url, headers, body)` helper. *Copy directly from* `pull_request_maker_mcp/internal/bitbucket/client.go`.

9. **Implement tag operations** (`internal/bitbucket/tags.go`):
   - `ListTags(ctx, auth, workspace, repo, sort, pagelen)` — `GET /2.0/repositories/{ws}/{repo}/refs/tags?sort=-target.date&pagelen={n}` — returns list of tags with name + target commit hash + date
   - `GetLatestTag(ctx, auth, workspace, repo)` — wrapper that calls ListTags with pagelen=1
   - `CreateTag(ctx, auth, workspace, repo, tagName, commitHash)` — `POST /2.0/repositories/{ws}/{repo}/refs/tags` with body `{"name":"...","target":{"hash":"..."}}`

10. **Implement commit operations** (`internal/bitbucket/commits.go`):
    - `ListCommits(ctx, auth, workspace, repo, branch, pagelen)` — `GET /2.0/repositories/{ws}/{repo}/commits/{branch}` — paginated
    - `GetCommitsAfterHash(ctx, auth, workspace, repo, branch, afterHash)` — iterates commit pages until reaching `afterHash`, returns all commits before it
    - `GetLatestCommit(ctx, auth, workspace, repo, branch)` — ListCommits with pagelen=1

11. **Implement PR lookup** (`internal/bitbucket/pullrequests.go`):
    - `GetPRsForCommit(ctx, auth, workspace, repo, commitHash)` — `GET /2.0/repositories/{ws}/{repo}/commit/{hash}/pullrequests` — returns associated merged PRs with title, description, source branch, author

12. **Implement pipeline operations** (`internal/bitbucket/pipelines.go`):
    - `TriggerPipeline(ctx, auth, workspace, repo, refType, refName, pipelineName)` — `POST /2.0/repositories/{ws}/{repo}/pipelines/` with body specifying `target.type=pipeline_ref_target`, `ref_type` (branch/tag), `ref_name`, and `selector.pattern`
    - Returns pipeline UUID and build number

13. **Define shared types** (`internal/bitbucket/types.go`) — Structs for Tag, Commit, PullRequest, PipelineRun, and API response envelopes with pagination.

### Phase 3: Supporting Modules

14. **Implement Jira ticket extractor** (`internal/jira/extractor.go`):
    - `ExtractTicket(sources ...string)` — tries each source string in order, returns first match of `[A-Z][A-Z0-9]+-\d+` regex, or `"unknown ticket"` if none found
    - `BuildJiraURL(baseURL, ticketKey)` — generates full Jira link
    - *Reuse regex pattern from* `pull_request_maker_mcp/internal/gchat/notifier.go` `ExtractJiraTicket()`

15. **Implement tag version parser** (`internal/tagversion/parser.go`):
    - `ParseSemver(tagName)` — parse `v1.2.3` or `1.2.3` into major/minor/patch
    - `IncrementPatch(tagName)` — `v1.2.3` → `v1.2.4`
    - `IncrementMinor(tagName)` → `v1.2.3` → `v1.3.0`
    - Support detecting prefix (e.g., `v`, `release-`) and preserving it
    - If tag doesn't match semver, return error and let user specify manually

16. **Implement Google Chat notifier** (`internal/gchat/notifier.go`):
    - `NotifyUndeployedChanges(ctx, webhookURL, message UndeployedChangesMessage)` — formats and sends list of undeployed changes
    - `NotifyTagCreated(ctx, webhookURL, message TagCreatedMessage)` — formats tag creation notification
    - `NotifyPipelineTriggered(ctx, webhookURL, message PipelineMessage)` — formats pipeline trigger notification
    - *Extend pattern from* `pull_request_maker_mcp/internal/gchat/notifier.go`

17. **Implement Confluence client** (`internal/confluence/client.go`):
    - HTTP client with Atlassian API auth (same email+api_token as Bitbucket)
    - `GetPage(ctx, auth, baseURL, pageID)` — `GET /wiki/api/v2/pages/{id}?body-format=storage` — returns title + storage-format body
    - `CreatePage(ctx, auth, baseURL, spaceID, parentID, title, body)` — `POST /wiki/api/v2/pages` — creates new page under parent with storage-format body
    - `GetSpaceID(ctx, auth, baseURL, spaceKey)` — `GET /wiki/api/v2/spaces?keys={key}` — resolves space key to space ID (needed by v2 API)

### Phase 4: MCP Tools Implementation

18. **Tool: `get_undeployed_changes`** (`internal/tools/get_undeployed_changes.go`):
    - **Input**: `repo_slug` (string, optional — empty means all configured repos), `send_notification` (bool, optional — default false)
    - **Logic**:
      a. Resolve target repos (single or all from config)
      b. For each repo: call `GetLatestTag()` → `GetCommitsAfterHash()` to find commits after tag
      c. For each commit: call `GetPRsForCommit()` to find associated PR
      d. Extract Jira ticket from: PR source branch → PR title → PR description → commit message (in that priority order)
      e. Deduplicate results by Jira ticket
      f. If `send_notification=true`, format and send to Google Chat via `NotifyUndeployedChanges()`
    - **Output**: Per-repo list of `{jira_ticket, jira_url, pr_url, pr_title, author, commit_hash, summary}` with `latest_tag`, `total_undeployed`, and `notification_sent`

19. **Tool: `create_release_tag`** (`internal/tools/create_release_tag.go`):
    - **Input**: `repo_slug` (string, required), `tag_name` (string, optional — auto-increment if empty), `commit_hash` (string, optional — latest master if empty), `send_notification` (bool, optional)
    - **Logic**:
      a. Fetch latest tag via `GetLatestTag()`
      b. If `tag_name` empty: parse latest tag with `tagversion.IncrementPatch()` to generate next version
      c. If `commit_hash` empty: fetch latest commit on default branch via `GetLatestCommit()`
      d. Create tag via `CreateTag()`
      e. If `send_notification=true`, notify Google Chat via `NotifyTagCreated()`
    - **Output**: `{tag_name, commit_hash, previous_tag, repo_slug, tag_url, notification_sent}`

20. **Tool: `create_deployment_document`** (`internal/tools/create_deployment_doc.go`):
    - **Input**: `title` (string, required), `reference_page_id` (string, optional — overrides config `template_page_id`), `changes_summary` (string, optional — injected into doc), `tag_name` (string, optional), `repo_slugs` (string[], optional — for multi-repo deployments)
    - **Logic**:
      a. Resolve reference page ID (input override → config `template_page_id`)
      b. Fetch reference page content via `confluence.GetPage()`
      c. Clone content: replace title, date, version/tag info, and optionally inject changes summary
      d. Create new page via `confluence.CreatePage()` under configured `parent_page_id`
    - **Output**: `{page_id, page_url, title, based_on_page_id}`

21. **Tool: `trigger_pipeline`** (`internal/tools/trigger_pipeline.go`):
    - **Input**: `repo_slug` (string, required), `environment` (string, required — matches pipeline config key e.g. "production", "staging"), `ref_name` (string, optional — auto-detected)
    - **Logic**:
      a. Look up pipeline config for repo + environment
      b. Determine ref: for production → use latest tag (via `GetLatestTag()`) unless `ref_name` specified; for non-prod → use `ref_name` or latest commit hash on default branch
      c. Determine ref_type: production uses "tag", non-prod uses "branch"
      d. Call `TriggerPipeline()` with resolved parameters
      e. Return pipeline run info
    - **Output**: `{pipeline_uuid, pipeline_url, build_number, ref_type, ref_name, pipeline_name, repo_slug}`

### Phase 5: Entry Point & Wiring

22. **Create main.go** — Following pull_request_maker_mcp pattern:
    - Parse `-config` flag
    - Load config via `config.Load()`
    - Create dependencies: `bitbucket.NewClient()`, `gchat.NewNotifier()`, `confluence.NewClient()`
    - Create MCP server with `mcp.NewServer(&mcp.Implementation{Name: "integration-deployment-tools-mcp", Version: "1.0.0"}, nil)`
    - Register 4 tools
    - Graceful shutdown with signal handling
    - Run with `mcp.StdioTransport{}`

23. **Create README.md** — Tool descriptions, setup instructions, VS Code MCP config example, config reference.

---

## Relevant Files

### Reuse from pull_request_maker_mcp (reference/copy patterns)
- `pull_request_maker_mcp/main.go` — MCP server setup, DI, signal handling, stdio transport
- `pull_request_maker_mcp/internal/config/types.go` — Config struct pattern
- `pull_request_maker_mcp/internal/config/loader.go` — YAML loader, validation, FindProjectConfig, LoadFileContent
- `pull_request_maker_mcp/internal/bitbucket/auth.go` — `buildBasicAuth()`, `getHeaders()`, `postHeaders()`
- `pull_request_maker_mcp/internal/bitbucket/client.go` — `NewClient()`, `doRequest()` pattern
- `pull_request_maker_mcp/internal/gchat/notifier.go` — `Notifier` struct, `ExtractJiraTicket()`, Google Chat webhook pattern
- `pull_request_maker_mcp/internal/tools/get_pr_diff.go` — `mcp.AddTool()` registration pattern with typed input/output
- `pull_request_maker_mcp/Makefile` — Build targets

### New files to create
- `integration_deployment_tools_mcp/main.go`
- `integration_deployment_tools_mcp/go.mod`
- `integration_deployment_tools_mcp/Makefile`
- `integration_deployment_tools_mcp/config.example.yaml`
- `integration_deployment_tools_mcp/.gitignore`
- `integration_deployment_tools_mcp/README.md`
- `integration_deployment_tools_mcp/internal/config/types.go`
- `integration_deployment_tools_mcp/internal/config/loader.go`
- `integration_deployment_tools_mcp/internal/bitbucket/auth.go`
- `integration_deployment_tools_mcp/internal/bitbucket/client.go`
- `integration_deployment_tools_mcp/internal/bitbucket/types.go`
- `integration_deployment_tools_mcp/internal/bitbucket/tags.go`
- `integration_deployment_tools_mcp/internal/bitbucket/commits.go`
- `integration_deployment_tools_mcp/internal/bitbucket/pullrequests.go`
- `integration_deployment_tools_mcp/internal/bitbucket/pipelines.go`
- `integration_deployment_tools_mcp/internal/confluence/client.go`
- `integration_deployment_tools_mcp/internal/confluence/types.go`
- `integration_deployment_tools_mcp/internal/gchat/notifier.go`
- `integration_deployment_tools_mcp/internal/jira/extractor.go`
- `integration_deployment_tools_mcp/internal/tagversion/parser.go`
- `integration_deployment_tools_mcp/internal/tools/get_undeployed_changes.go`
- `integration_deployment_tools_mcp/internal/tools/create_release_tag.go`
- `integration_deployment_tools_mcp/internal/tools/create_deployment_doc.go`
- `integration_deployment_tools_mcp/internal/tools/trigger_pipeline.go`
- `integration_deployment_tools_mcp/internal/tools/helpers.go`

---

## Verification

1. **Build**: `make build` compiles without errors
2. **Config validation**: Verify `config.Load()` fails with clear errors for missing auth, empty repo list, invalid pipeline configs
3. **Unit tests** for:
   - `jira.ExtractTicket()` — various branch/PR name formats, "unknown ticket" fallback
   - `tagversion.ParseSemver()` / `IncrementPatch()` — `v1.2.3`, `1.2.3`, `v0.0.1`, edge cases
   - `config.FindRepoConfig()` — exact match, wildcard fallback, no match
   - `gchat` message builders — correct formatting
4. **Manual integration test**: Register in VS Code, verify each tool responds correctly via Copilot chat:
   - "Show undeployed changes for mid-kelola-indonesia/talenta-data-api"
   - "Create a release tag for talenta-data-api"
   - "Create a deployment document for v1.2.4 release"
   - "Trigger staging pipeline for talenta-data-api"
5. **Bitbucket API correctness**: Verify tag listing returns correct latest tag, commit traversal stops at tag commit, pipeline trigger uses correct ref_type/ref_name

---

## Decisions

- **Go SDK**: Use `github.com/modelcontextprotocol/go-sdk` v1.4.1+ matching pull_request_maker_mcp
- **Auth sharing**: Bitbucket and Confluence use the same Atlassian API token (same `email` + `api_token`), so single auth block with `confluence.base_url` specified separately
- **Tag version auto-increment**: Only semver patterns (`v1.2.3`) are auto-incremented; non-semver tags require manual `tag_name` input
- **Pipeline ref resolution**: Production environments always use tag ref_type; non-production always use branch ref_type with commit hash selector. This is enforced by config's `ref_type` field per environment
- **Confluence API**: Use v2 REST API (`/wiki/api/v2/pages`) which requires space ID (resolved from space key). Storage format for page body (XHTML)
- **Scope boundary**: This MCP is for deployment workflow automation only — no code review, no PR creation (those live in pull_request_maker_mcp)
- **Google Chat notifications** are opt-in (via `send_notification` parameter) — the tool always returns data regardless

## Config Structure (config.example.yaml)

```yaml
auth:
  bitbucket:
    email: "your-email@example.com"
    api_token: "ATATT3x..."

gchat:
  webhook_url: "https://chat.googleapis.com/v1/spaces/XXXXX/messages?key=...&token=..."
  jira_base_url: "https://your-org.atlassian.net/browse"

confluence:
  base_url: "https://your-org.atlassian.net"
  space_key: "DEPLOY"
  parent_page_id: "123456789"
  template_page_id: "987654321"

repositories:
  - repo_slug: "mid-kelola-indonesia/talenta-data-api"
    default_branch: "master"
    tag_pattern: "semver"   # "semver" for auto-increment, or "manual" to require explicit tag name
    pipelines:
      production:
        name: "deploy-to-production"
        ref_type: "tag"
      staging:
        name: "deploy-to-staging"
        ref_type: "branch"
    gchat_webhook_url: ""   # optional override

  - repo_slug: "mid-kelola-indonesia/disbursement-service"
    default_branch: "master"
    tag_pattern: "semver"
    pipelines:
      production:
        name: "deploy-to-production"
        ref_type: "tag"
      staging:
        name: "deploy-to-staging"
        ref_type: "branch"

  - repo_slug: "*"  # wildcard fallback
    default_branch: "master"
    tag_pattern: "manual"
    pipelines: {}
```

## Further Considerations

1. **Confluence template variables**: The deployment doc cloning needs placeholder replacement (e.g., `{{TAG_NAME}}`, `{{DATE}}`, `{{CHANGES}}`). Recommendation: use simple string replacement on the storage-format body, document supported variables in README.
2. **Pagination for large commit histories**: If many commits exist between tags, the `GetCommitsAfterHash` function may need to paginate through many pages. Recommendation: set a configurable max page limit (default 10 pages × 30 commits = 300 commits) to avoid runaway API calls.
3. **Pipeline status polling**: The `trigger_pipeline` tool currently fires-and-forgets. A future enhancement could poll pipeline status. Recommendation: keep it fire-and-forget for v1, return the pipeline URL for manual checking.
