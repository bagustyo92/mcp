# Code Review MCP

An MCP (Model Context Protocol) server that automates code review on Bitbucket pull requests via VS Code Copilot.

## How It Works

1. You provide a Bitbucket PR URL in Copilot chat
2. The MCP fetches the diff, PR metadata, and loads your project's review instructions/guidelines
3. Copilot analyzes the code and presents a numbered list of review comments
4. You pick which comments to post
5. The MCP posts the selected comments directly on the PR in Bitbucket

## Prerequisites

- **Node.js >= 18** (use `nvm use 22` if you have nvm)
- **Bitbucket App Password** with these scopes:
  - `Repositories: Read`
  - `Pull requests: Read`
  - `Pull requests: Write`

Create one at: https://bitbucket.org/account/settings/app-passwords/

## Setup

### 1. Install & Build

```bash
cd mcp/code_review_mcp
nvm use 22        # if using nvm
npm install
npm run build
```

### 2. Configure

```bash
cp config.example.yaml config.yaml
```

Edit `config.yaml` with your Bitbucket credentials and project paths:

```yaml
auth:
  bitbucket:
    username: "your-bitbucket-username"
    app_password: "your-app-password"

projects:
  - repo_slug: "mid-kelola-indonesia/talenta-data-api"
    review_instructions: null
    project_guidelines: "/absolute/path/to/.github/copilot-instructions.md"

  - repo_slug: "*"  # fallback for unmatched repos
    review_instructions: null
    project_guidelines: null
```

### 3. Add to VS Code

Add the MCP server to your VS Code settings (`.vscode/settings.json` or user settings):

```json
{
  "mcp": {
    "servers": {
      "code-review": {
        "command": "node",
        "args": ["/Users/mekari/Documents/mekari/talenta/mcp/code_review_mcp/dist/index.js"]
      }
    }
  }
}
```

> **Note**: If using nvm, you may need to use the full node path:
> ```bash
> which node  # get the path after running `nvm use 22`
> ```
> Then use that path as the `command` value.

### 4. Reload VS Code

After adding the MCP config, reload the VS Code window (`Cmd+Shift+P` → "Developer: Reload Window").

## Usage

In Copilot chat:

```
Review this PR: https://bitbucket.org/mid-kelola-indonesia/talenta-data-api/pull-requests/123
```

Copilot will:
1. Call `get_pr_diff` to fetch all PR data
2. Analyze the diff against your instructions
3. Present numbered review comments
4. Ask which ones you want to post

You reply:
```
Post comments 1, 3, 5
```

Copilot calls `post_pr_comments` and the comments appear on the PR.

## Tools

### `get_pr_diff`

| Parameter | Type   | Description |
|-----------|--------|-------------|
| `pr_url`  | string | Full Bitbucket PR URL |

Returns: PR metadata, full diff, changed files list, review instructions, project guidelines.

### `post_pr_comments`

| Parameter  | Type   | Description |
|------------|--------|-------------|
| `pr_url`   | string | Full Bitbucket PR URL |
| `comments` | array  | `[{ path, line, content }]` — review comments to post |

Returns: Summary with success/failure count per comment.

## Project Structure

```
code_review_mcp/
├── src/
│   ├── index.ts                 # MCP server entry point
│   ├── config/
│   │   ├── types.ts             # TypeScript interfaces
│   │   └── loader.ts            # Config file loader
│   ├── tools/
│   │   ├── get-pr-diff.ts       # get_pr_diff tool
│   │   └── post-comments.ts     # post_pr_comments tool
│   ├── providers/
│   │   └── bitbucket.ts         # Bitbucket API client
│   └── utils/
│       └── pr-url-parser.ts     # PR URL parser
├── config.example.yaml          # Example configuration
├── config.yaml                  # Your config (gitignored)
├── package.json
├── tsconfig.json
└── README.md
```

## Security

- `config.yaml` is gitignored — never commit it
- Credentials are only stored in the local config file
- All API calls use HTTPS with Bitbucket App Password (HTTP Basic)
