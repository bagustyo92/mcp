/**
 * MCP Tool: post_pr_comments
 *
 * Posts user-approved review comments as inline comments on a Bitbucket PR.
 * Accepts the PR URL and an array of comments with file path, line number, and content.
 */

import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import type { AppConfig, ReviewComment } from "../config/types.js";
import { parsePrUrl } from "../utils/pr-url-parser.js";
import {
  postInlineComment,
  postGeneralComment,
} from "../providers/bitbucket.js";

export function registerPostPrComments(
  server: McpServer,
  config: AppConfig
): void {
  server.tool(
    "post_pr_comments",
    "Post review comments on a Bitbucket pull request. " +
      "Each comment can be an inline comment (attached to a specific file and line) or a general comment. " +
      "Use this after the user has selected which review comments to post from the get_pr_diff analysis.",
    {
      pr_url: z
        .string()
        .url()
        .describe(
          "The full URL of the Bitbucket pull request, e.g. https://bitbucket.org/workspace/repo/pull-requests/123"
        ),
      comments: z
        .array(
          z.object({
            path: z
              .string()
              .describe(
                "File path relative to repo root. Leave empty string for general (non-inline) comments."
              ),
            line: z
              .number()
              .int()
              .min(0)
              .describe(
                "Line number in the new file version for inline comment. Use 0 for general comments."
              ),
            content: z
              .string()
              .describe(
                "The review comment content in markdown format."
              ),
          })
        )
        .min(1)
        .describe(
          "Array of review comments to post. Each comment has a file path, line number, and content."
        ),
    },
    async ({ pr_url, comments }) => {
      try {
        // 1. Parse the PR URL
        const prInfo = parsePrUrl(pr_url);
        const auth = config.auth.bitbucket;

        // 2. Post each comment
        const results = [];

        for (const comment of comments) {
          const reviewComment: ReviewComment = {
            path: comment.path,
            line: comment.line,
            content: comment.content,
          };

          let result;
          if (!comment.path || comment.line <= 0) {
            // General comment (not attached to a file/line)
            result = await postGeneralComment(
              auth,
              prInfo,
              comment.content
            );
          } else {
            // Inline comment
            result = await postInlineComment(auth, prInfo, reviewComment);
          }

          results.push(result);

          // Small delay between posts to avoid rate limiting
          await sleep(200);
        }

        // 3. Build summary
        const posted = results.filter((r) => r.success).length;
        const failed = results.filter((r) => !r.success).length;

        const summary = {
          total: comments.length,
          posted,
          failed,
          results,
        };

        return {
          content: [
            {
              type: "text" as const,
              text: JSON.stringify(summary, null, 2),
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
              text: `Error posting comments: ${message}`,
            },
          ],
          isError: true,
        };
      }
    }
  );
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
