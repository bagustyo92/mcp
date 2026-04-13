---
name: create-deployment-doc
description: >
  WORKFLOW SKILL — Orchestrates the full deployment document creation flow for Talenta services.
  Use this skill whenever the user wants to: create a deployment document, prepare a release doc,
  document pending deployments, list what needs to be deployed, create a Confluence deployment page,
  notify GChat about a deployment, or do anything related to "deployment prep" or "release notes"
  for any Talenta service (talenta-core, talenta-data-api, form-service, talenta-backyard, etc.).
  Also trigger this skill when the user says things like "prepare deployment", "what's pending deploy",
  "draft the deploy doc", "create the release page", or "announce to gchat".
  ALWAYS use this skill when deployment documentation or pre-deployment coordination is involved.
---

# Create Deployment Document

This skill orchestrates the full deployment document workflow:

1. **Fetch** undeployed changes (pending Jira tickets per service) via the MCP
2. **Draft** a Markdown deployment document for user review
3. **Post** to Confluence (only after explicit user confirmation)
4. **Announce** to Google Chat with the Confluence link (only after explicit user confirmation)

---

## Step 1 — Fetch Pending Changes (parallel)

Fetch undeployed changes for all repos **in parallel** — spawn one subagent per repo at the same time so the total wait is roughly the slowest single call rather than the sum of all of them.

The fixed list of repos to check is:

```
mid-kelola-indonesia/talenta-data-api
mid-kelola-indonesia/disbursement-service
mid-kelola-indonesia/form-service
mid-kelola-indonesia/notification-service
mid-kelola-indonesia/talenta-backyard
mid-kelola-indonesia/talenta-backyard-api
```

For each repo, call `mcp_integration-d_get_undeployed_changes` with that repo's slug:

```
get_undeployed_changes(repo_slug: "<workspace/repo>", send_notification: false)
```

Wait for all parallel calls to complete before moving on. Aggregate the results into a single list of per-repo summaries. Each entry gives you:
- `repo_slug`
- `latest_tag` — the most recent production tag
- `total_undeployed` — count of undeployed commits since that tag
- `changes[]` — list of `{ jira_ticket, jira_url, pr_url, pr_title, author, commit_hash, summary }`
- `error` (if the call failed — e.g. repo not found or no tags)

If the user asks for a different or smaller set of repos, respect that and only call for those repos.

---

## Step 2 — Draft the Deployment Document (Markdown)

Build a Markdown document following the structure below. Use `today's date` for the title and section headings.

Present the full Markdown to the user in the chat and say:
> "Here's the deployment document draft. Please review and let me know if you'd like any changes before I post it to Confluence."

**Do not post to Confluence or notify GChat at this point.** Wait for the user's explicit go-ahead.

### Document Structure

Use this exact template structure (adapt to the actual services found in the undeployed changes response):

```markdown
# Deployment — <DATE>

## SERVICES

| **SERVICES** | **JIRA RELEASE VERSION** | **DEPLOYER** | **START** | **FINISH** | **PIPELINE** |
|---|---|---|---|---|---|
| <SERVICE NAME> | *<jira_ticket(s) or "Please Input ... Here">* | | | | |
...

## RELEASE CRITERIA

| **WEB** | **MOBILE API** | ... |
|---|---|...|
| Manual Testing and No Bug<br>TC for Release<br>Exploratory Testing<br>UI & API Automation Test<br>Smoke Test<br>P0 Test<br>Unit Test Coverage Expected<br>UAT - Demo<br>Update Documentation*<br>User Manual / Guidebook<br>API Documentation | (same) | (same) |

## DEPLOYMENT CHECKLIST

### <SERVICE NAME>
<jira_url filtered by Jira label / fixVersion>

...one section per service with pending changes

## DEPLOYMENT STEPS

| **DEPLOYMENT STEPS** |
|---|
| **Service** | **Action** | **PIC** | **Status** | **Notes** |
| | | | | |

## POST-DEPLOYMENT STEPS

| **POST-DEPLOYMENT STEPS** |
|---|
| **Service** | **Action** | **PIC** | **Status** | **Notes** |
| | | | | |

## DEPLOYMENT LOGS

| **DEPLOYMENT LOGS** |
|---|
| **Service** | **Time** | **Event** | **Notes** |
| | | | |

## DEPLOYMENT FAILURE

| **DEPLOYMENT FAILURE** |
|---|
| **Service** | **Reason** | **Action Items** |
| | | |

## ROLLBACK REASON

| **ROLLBACK REASON** |
|---|
| **Service** | **Reason** |
| | |

## ROLLBACK STEPS

| **ROLLBACK STEPS** |
|---|
| **Service** | **Action** | **PIC** | **Status** | **Notes** |
| | | | | |
```

### Mapping repos to service names

Use this repo-to-service-name mapping when building the SERVICES table:

| Repo slug | Display name |
|---|---|
| `mid-kelola-indonesia/talenta-data-api` | TALENTA DATA API |
| `mid-kelola-indonesia/disbursement-service` | DISBURSEMENT SERVICE |
| `mid-kelola-indonesia/form-service` | FORM SERVICE |
| `mid-kelola-indonesia/notification-service` | NOTIFICATION SERVICE |
| `mid-kelola-indonesia/talenta-backyard` | TALENTA BACKYARD |
| `mid-kelola-indonesia/talenta-backyard-api` | TALENTA BACKYARD API |
| `talenta-core` | WEB CORE |
| `talenta-mobile-api` or `tma` | MOBILE API |
| `ledger` | LEDGER SERVICE |
| `fingerprint` | TALENTA FINGERPRINT API |

For repos not in this mapping, use the repo name (last path segment) formatted in UPPER CASE as the service name.

### Jira Release Version cell

- If the repo has undeployed changes, use the **first** `jira_ticket` value found in `changes[]` as the release version placeholder (or group tickets logically if multiple distinct projects appear).
- If there are no undeployed changes, use `*Please Input <Service> Jira Release Version Here*`.

### Deployment Checklist links

For each service that has pending changes, generate the Jira search URL:
```
https://jurnal.atlassian.net/issues/?jql=fixVersion%20%3D%20<TICKET>%20OR%20labels%20%3D%20<TICKET>%20ORDER%20BY%20parent%20ASC%2C%20updatedDate%20DESC
```
where `<TICKET>` is the Jira release label (e.g. `TD-TDA-2026-04-01`).

---

## Step 3 — Post to Confluence (requires user confirmation)

Only proceed after the user explicitly says something like "looks good", "post it", "create the page", "yes go ahead", or similar affirmation.

Call `mcp_integration-d_create_deployment_doc` with:
- `title`: the document title (e.g. `"Deployment — 2026-04-06"`)
- `changes_summary`: the full Markdown body of the document
- `tag_name`: the latest tag from any of the repos (use the most recent)
- `repo_slugs`: list of all repo slugs that had undeployed changes

The tool returns `page_url`. Store this URL — you'll need it for the GChat announcement.

Tell the user: "Confluence page created: <page_url>. Ready to announce to GChat?"

---

## Step 4 — Announce to GChat (requires user confirmation)

Only proceed after the user explicitly says something like "yes", "send it", "announce", "post to gchat", or similar.

Call `mcp_integration-d_get_undeployed_changes` with `send_notification: true` **only if** the user hasn't already approved a separate announcement — otherwise compose the message yourself and call the underlying notifier approach via any available GChat tool.

Actually: use `mcp_pr-maker-bitb_notify_gchat` if available, or note that the GChat notification is sent by setting `send_notification: true` on `get_undeployed_changes`.

The GChat message should include:
- Title: `*Deployment Ready — <DATE>*`
- Short list of services and their pending ticket counts
- Link to the Confluence page: `<page_url|📄 View Deployment Document>`

If no dedicated GChat notify tool is available, remind the user that they can re-call `get_undeployed_changes(send_notification: true)` to send the standard notification, or manually share the Confluence URL in the channel.

---

## Important guardrails

- **Never post to Confluence without user confirmation.** Always present the Markdown draft first.
- **Never announce to GChat without a separate user confirmation** after the Confluence page is created.
- If the MCP returns errors for some repos (e.g., no tags found), include those services in the document with a note like `⚠️ Could not fetch changes` and continue with the rest.
- Keep the document structure faithful to the template — don't collapse or omit sections even if they're empty. Empty rows with `| | | | |` are fine.
