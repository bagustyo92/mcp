#!/usr/bin/env node

/**
 * Code Review MCP Server
 *
 * An MCP server that automates code review on Bitbucket pull requests.
 * It provides two tools:
 *   - get_pr_diff: Fetches the PR diff, metadata, and review instructions
 *   - post_pr_comments: Posts user-approved review comments on the PR
 *
 * Usage:
 *   node dist/index.js
 *
 * Configuration:
 *   Place a config.yaml in the project root (see config.example.yaml).
 */

import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { loadConfig } from "./config/loader.js";
import { registerGetPrDiff } from "./tools/get-pr-diff.js";
import { registerPostPrComments } from "./tools/post-comments.js";

async function main(): Promise<void> {
  // 1. Load configuration
  const config = loadConfig();

  // 2. Create the MCP server
  const server = new McpServer({
    name: "code-review-mcp",
    version: "1.0.0",
  });

  // 3. Register tools
  registerGetPrDiff(server, config);
  registerPostPrComments(server, config);

  // 4. Start the server with stdio transport (for VS Code integration)
  const transport = new StdioServerTransport();
  await server.connect(transport);

  console.error("Code Review MCP server started (stdio transport)");
}

main().catch((error) => {
  console.error("Fatal error starting MCP server:", error);
  process.exit(1);
});
