/**
 * Configuration types for the Code Review MCP server.
 */

/**
 * Bitbucket authentication config.
 *
 * Option 1 (Recommended): Atlassian API Token
 *   - Create at: https://id.atlassian.com/manage-profile/security/api-tokens
 *   - Set: email (your Atlassian account email) + api_token
 *   - Auth: Basic base64("email:api_token")
 *
 * Option 2 (Legacy): Bitbucket App Password
 *   - Create at: https://bitbucket.org/account/settings/app-passwords/
 *   - Set: username + app_password
 *   - Auth: Basic base64("username:app_password")
 */
export interface BitbucketAuth {
  /** Atlassian account email — used together with api_token. */
  email?: string;
  /** Atlassian API Token (ATATT3x...) — paired with email. */
  api_token?: string;
  /** Legacy: Bitbucket username (used with app_password). */
  username?: string;
  /** Legacy: Bitbucket App Password (used with username). */
  app_password?: string;
}

export interface AuthConfig {
  bitbucket: BitbucketAuth;
}

export interface ProjectConfig {
  /** Bitbucket repo slug, e.g. "mid-kelola-indonesia/talenta-data-api". Use "*" for default fallback. */
  repo_slug: string;
  /** Absolute path to a markdown file with code review instructions. */
  review_instructions: string | null;
  /** Absolute path to a markdown file with project guidelines. */
  project_guidelines: string | null;
}

export interface AppConfig {
  auth: AuthConfig;
  projects: ProjectConfig[];
}

export interface PrInfo {
  platform: "bitbucket";
  workspace: string;
  repoSlug: string;
  prId: number;
}

export interface PrMetadata {
  title: string;
  description: string;
  sourceBranch: string;
  targetBranch: string;
  author: string;
  state: string;
}

export interface ReviewComment {
  /** File path relative to repo root */
  path: string;
  /** Line number in the new version of the file (for inline comments) */
  line: number;
  /** The review comment content (markdown) */
  content: string;
}

export interface PostCommentResult {
  path: string;
  line: number;
  success: boolean;
  error?: string;
  comment_url?: string;
}
