/**
 * MCP Tool: get_pr_diff
 *
 * Given a PR URL, fetches the diff, PR metadata, and loads the matching
 * review instructions + project guidelines from the config.
 *
 * Returns everything the LLM needs to perform a code review.
 */

import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import type { AppConfig } from "../config/types.js";
import { findProjectConfig, loadFileContent } from "../config/loader.js";
import { parsePrUrl } from "../utils/pr-url-parser.js";
import {
  fetchPrMetadata,
  fetchPrDiff,
  extractChangedFilesFromDiff,
} from "../providers/bitbucket.js";

export function registerGetPrDiff(server: McpServer, config: AppConfig): void {
  server.tool(
    "get_pr_diff",
    "Fetch the diff, metadata, review instructions, and project guidelines for a Bitbucket pull request. " +
      "Use this as the first step when reviewing a PR. The returned data includes everything needed to perform a thorough code review.",
    {
      pr_url: z
        .string()
        .url()
        .describe(
          "The full URL of the Bitbucket pull request, e.g. https://bitbucket.org/workspace/repo/pull-requests/123"
        ),
    },
    async ({ pr_url }) => {
      try {
        // 1. Parse the PR URL
        const prInfo = parsePrUrl(pr_url);

        // 2. Find project config for this repo
        const projectConfig = findProjectConfig(
          config,
          prInfo.workspace,
          prInfo.repoSlug
        );

        // 3. Load instruction files from disk
        const reviewInstructions = loadFileContent(
          projectConfig?.review_instructions ?? null
        );
        const projectGuidelines = loadFileContent(
          projectConfig?.project_guidelines ?? null
        );

        // 4. Fetch PR metadata first (need branches for diff endpoint)
        const metadata = await fetchPrMetadata(config.auth.bitbucket, prInfo);

        // 5. Fetch diff using branch-comparison endpoint (works with Bearer tokens)
        const diff = await fetchPrDiff(
          config.auth.bitbucket,
          prInfo,
          metadata.sourceBranch,
          metadata.targetBranch
        );

        // Extract changed files from the diff (avoids a separate diffstat API call)
        const changedFiles = extractChangedFilesFromDiff(diff);

        // 5. Build the response
        const result = {
          pr_url,
          platform: prInfo.platform,
          workspace: prInfo.workspace,
          repo_slug: prInfo.repoSlug,
          pr_id: prInfo.prId,
          metadata: {
            title: metadata.title,
            description: metadata.description,
            source_branch: metadata.sourceBranch,
            target_branch: metadata.targetBranch,
            author: metadata.author,
            state: metadata.state,
          },
          changed_files: changedFiles,
          diff,
          review_instructions: reviewInstructions ?? "No review instructions configured for this project.",
          project_guidelines: projectGuidelines ?? "No project guidelines configured for this project.",
          instructions_to_llm:
            "You are a senior software engineer performing a code review. " +
            "Analyze the diff above against the review instructions and project guidelines. " +
            "For each issue found, provide:\n" +
            "  - A numbered comment (1, 2, 3, ...)\n" +
            "  - The file path and line number\n" +
            "  - Category: Security | Quality | Performance | Style | Bug | Readability\n" +
            "  - Priority: Critical | High | Medium | Low\n" +
            "  - Clear description of the issue\n" +
            "  - Suggested fix with code snippet if applicable\n\n" +
            "Present ALL comments as a numbered list. The user will then choose which ones to post. " +
            "After the user selects comments, use the post_pr_comments tool to post them.",
        };

        return {
          content: [
            {
              type: "text" as const,
              text: JSON.stringify(result, null, 2),
            },
          ],
        };
      } catch (error) {
        const message =
          error instanceof Error ? error.message : String(error);
        return {
          content: [
            {
              type: "text" as const,
              text: `Error fetching PR data: ${message}`,
            },
          ],
          isError: true,
        };
      }
    }
  );
}
