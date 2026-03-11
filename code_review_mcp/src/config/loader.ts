/**
 * Loads and validates the YAML configuration file.
 */

import { readFileSync, existsSync } from "node:fs";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { parse as parseYaml } from "yaml";
import type { AppConfig, ProjectConfig } from "./types.js";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

/** Default config path: <project_root>/config.yaml */
const DEFAULT_CONFIG_PATH = resolve(__dirname, "..", "..", "config.yaml");

/**
 * Load the config from disk. Falls back to the default path if none is provided.
 */
export function loadConfig(configPath?: string): AppConfig {
  const cfgPath = configPath ?? DEFAULT_CONFIG_PATH;

  if (!existsSync(cfgPath)) {
    throw new Error(
      `Config file not found at: ${cfgPath}\n` +
        `Copy config.example.yaml to config.yaml and fill in your credentials.`
    );
  }

  const raw = readFileSync(cfgPath, "utf-8");
  const parsed = parseYaml(raw) as AppConfig;

  validateConfig(parsed);

  return parsed;
}

function validateConfig(cfg: AppConfig): void {
  const bb = cfg.auth?.bitbucket;
  if (!bb) {
    throw new Error("Config validation error: auth.bitbucket is required.");
  }

  const hasToken = Boolean(bb.email?.trim()) && Boolean(bb.api_token?.trim());
  const hasAppPassword = Boolean(bb.username?.trim()) && Boolean(bb.app_password?.trim());

  if (!hasToken && !hasAppPassword) {
    throw new Error(
      "Config validation error: provide either:\n" +
        "  - auth.bitbucket.email + auth.bitbucket.api_token (Atlassian API Token, recommended)\n" +
        "  - auth.bitbucket.username + auth.bitbucket.app_password (legacy App Password)"
    );
  }

  if (!Array.isArray(cfg.projects) || cfg.projects.length === 0) {
    throw new Error(
      "Config validation error: at least one entry in 'projects' is required."
    );
  }

  for (const p of cfg.projects) {
    if (!p.repo_slug) {
      throw new Error(
        "Config validation error: every project entry must have a 'repo_slug'."
      );
    }
  }
}

/**
 * Find the project config that matches a given workspace/repo-slug.
 * Falls back to a wildcard "*" entry if no exact match.
 */
export function findProjectConfig(
  config: AppConfig,
  workspace: string,
  repoSlug: string
): ProjectConfig | null {
  const fullSlug = `${workspace}/${repoSlug}`;

  // Exact match
  const exact = config.projects.find(
    (p) => p.repo_slug.toLowerCase() === fullSlug.toLowerCase()
  );
  if (exact) return exact;

  // Wildcard fallback
  const wildcard = config.projects.find((p) => p.repo_slug === "*");
  return wildcard ?? null;
}

/**
 * Read a file from disk (used for loading instruction / guideline markdown files).
 * Returns null if the path is null/undefined or file doesn't exist.
 */
export function loadFileContent(filePath: string | null | undefined): string | null {
  if (!filePath) return null;

  const resolved = resolve(filePath);
  if (!existsSync(resolved)) {
    console.error(`Warning: instruction file not found: ${resolved}`);
    return null;
  }

  return readFileSync(resolved, "utf-8");
}
