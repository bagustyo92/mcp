/**
 * Bitbucket Cloud REST API v2.0 client.
 *
 * Docs: https://developer.atlassian.com/cloud/bitbucket/rest/
 */

import type {
  BitbucketAuth,
  PrInfo,
  PrMetadata,
  ReviewComment,
  PostCommentResult,
} from "../config/types.js";

const API_BASE = "https://api.bitbucket.org/2.0";

/**
 * Build Authorization headers for GET requests.
 *
 * Auth strategies for Bitbucket Cloud API v2:
 *
 * 1. Atlassian API Token (ATATT3x...) — recommended:
 *    Basic base64("email:api_token")
 *    Create token at: https://id.atlassian.com/manage-profile/security/api-tokens
 *
 * 2. Legacy App Password:
 *    Basic base64("username:app_password")
 *    Create at: https://bitbucket.org/account/settings/app-passwords/
 */
function getHeaders(auth: BitbucketAuth): Record<string, string> {
  let credentials: string;

  if (auth.email && auth.api_token) {
    // Atlassian API Token auth
    credentials = `${auth.email}:${auth.api_token}`;
  } else if (auth.username && auth.app_password) {
    // Legacy App Password auth
    credentials = `${auth.username}:${auth.app_password}`;
  } else {
    throw new Error(
      "Invalid auth config: provide either (email + api_token) or (username + app_password)."
    );
  }

  return {
    Authorization: `Basic ${Buffer.from(credentials).toString("base64")}`,
    Accept: "application/json",
  };
}

/**
 * Build Authorization headers for POST/PUT/PATCH requests (with Content-Type).
 */
function postHeaders(auth: BitbucketAuth): Record<string, string> {
  return {
    ...getHeaders(auth),
    "Content-Type": "application/json",
  };
}

/**
 * Fetch PR metadata (title, branches, author, state).
 */
export async function fetchPrMetadata(
  auth: BitbucketAuth,
  pr: PrInfo
): Promise<PrMetadata> {
  const url = `${API_BASE}/repositories/${pr.workspace}/${pr.repoSlug}/pullrequests/${pr.prId}`;

  const res = await fetch(url, { headers: getHeaders(auth) });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(
      `Bitbucket API error fetching PR metadata (${res.status}): ${text}`
    );
  }

  const data = (await res.json()) as Record<string, any>;

  return {
    title: data.title ?? "",
    description: data.description ?? "",
    sourceBranch: data.source?.branch?.name ?? "",
    targetBranch: data.destination?.branch?.name ?? "",
    author: data.author?.display_name ?? data.author?.nickname ?? "",
    state: data.state ?? "",
  };
}

/**
 * Fetch the unified diff of a PR by comparing source vs destination branches.
 *
 * Uses the branch-comparison diff endpoint (`/diff/{src}..{dst}`) instead of
 * the PR-specific `/pullrequests/{id}/diff` because the latter does not support
 * Bitbucket HTTP Access Tokens (Bearer auth).
 */
export async function fetchPrDiff(
  auth: BitbucketAuth,
  pr: PrInfo,
  sourceBranch: string,
  targetBranch: string
): Promise<string> {
  // Encode branch names to handle slashes, e.g. "feature/my-branch"
  const spec = `${encodeURIComponent(sourceBranch)}..${encodeURIComponent(targetBranch)}`;
  const url = `${API_BASE}/repositories/${pr.workspace}/${pr.repoSlug}/diff/${spec}`;

  const headers = getHeaders(auth);
  headers["Accept"] = "text/plain";

  const res = await fetch(url, { headers });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(
      `Bitbucket API error fetching diff (${res.status}): ${text}`
    );
  }

  return res.text();
}

/**
 * Extract the list of changed file paths directly from the unified diff text.
 * Format matched: `diff --git a/<path> b/<path>`
 * This avoids a separate diffstat API call (which requires additional token scopes).
 */
export function extractChangedFilesFromDiff(diff: string): string[] {
  const files: string[] = [];
  const regex = /^diff --git a\/.+ b\/(.+)$/gm;
  let match: RegExpExecArray | null;

  while ((match = regex.exec(diff)) !== null) {
    const filePath = match[1].trim();
    if (filePath && !files.includes(filePath)) {
      files.push(filePath);
    }
  }

  return files;
}

/**
 * Post an inline comment on a specific file + line in a PR.
 */
export async function postInlineComment(
  auth: BitbucketAuth,
  pr: PrInfo,
  comment: ReviewComment
): Promise<PostCommentResult> {
  const url = `${API_BASE}/repositories/${pr.workspace}/${pr.repoSlug}/pullrequests/${pr.prId}/comments`;

  const body: Record<string, any> = {
    content: {
      raw: comment.content,
    },
  };

  // Attach inline anchor if path and line are provided
  if (comment.path && comment.line > 0) {
    body.inline = {
      path: comment.path,
      to: comment.line,
    };
  }

  const res = await fetch(url, {
    method: "POST",
    headers: postHeaders(auth),
    body: JSON.stringify(body),
  });

  if (!res.ok) {
    const text = await res.text();
    return {
      path: comment.path,
      line: comment.line,
      success: false,
      error: `Bitbucket API error (${res.status}): ${text}`,
    };
  }

  const data = (await res.json()) as Record<string, any>;

  return {
    path: comment.path,
    line: comment.line,
    success: true,
    comment_url: data.links?.html?.href ?? undefined,
  };
}

/**
 * Post a general (non-inline) comment on a PR.
 */
export async function postGeneralComment(
  auth: BitbucketAuth,
  pr: PrInfo,
  content: string
): Promise<PostCommentResult> {
  const url = `${API_BASE}/repositories/${pr.workspace}/${pr.repoSlug}/pullrequests/${pr.prId}/comments`;

  const body = {
    content: {
      raw: content,
    },
  };

  const res = await fetch(url, {
    method: "POST",
    headers: postHeaders(auth),
    body: JSON.stringify(body),
  });

  if (!res.ok) {
    const text = await res.text();
    return {
      path: "",
      line: 0,
      success: false,
      error: `Bitbucket API error (${res.status}): ${text}`,
    };
  }

  const data = (await res.json()) as Record<string, any>;

  return {
    path: "",
    line: 0,
    success: true,
    comment_url: data.links?.html?.href ?? undefined,
  };
}
