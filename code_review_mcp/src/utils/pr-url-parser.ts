/**
 * Parses PR URLs from various Git hosting platforms into a normalized PrInfo.
 * Currently supports Bitbucket Cloud.
 */

import type { PrInfo } from "../config/types.js";

/**
 * Bitbucket Cloud URL format:
 *   https://bitbucket.org/{workspace}/{repo}/pull-requests/{id}
 */
const BITBUCKET_PR_REGEX =
  /^https?:\/\/bitbucket\.org\/([^/]+)\/([^/]+)\/pull-requests\/(\d+)/i;

/**
 * Parse a PR URL into structured info.
 * Throws if the URL format is unrecognized.
 */
export function parsePrUrl(url: string): PrInfo {
  const trimmed = url.trim();

  const bbMatch = trimmed.match(BITBUCKET_PR_REGEX);
  if (bbMatch) {
    return {
      platform: "bitbucket",
      workspace: bbMatch[1],
      repoSlug: bbMatch[2],
      prId: parseInt(bbMatch[3], 10),
    };
  }

  throw new Error(
    `Unsupported PR URL format: ${trimmed}\n` +
      `Supported formats:\n` +
      `  - Bitbucket: https://bitbucket.org/{workspace}/{repo}/pull-requests/{id}`
  );
}
