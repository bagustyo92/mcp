#!/usr/bin/env node
/**
 * Quick auth test script.
 * Run: node test-auth.mjs
 *
 * Verifies your config.yaml credentials work against the Bitbucket API
 * before reloading VS Code.
 */

import { readFileSync } from "node:fs";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { parse as parseYaml } from "yaml";

const __dirname = dirname(fileURLToPath(import.meta.url));
const configPath = resolve(__dirname, "config.yaml");

let config;
try {
  config = parseYaml(readFileSync(configPath, "utf-8"));
} catch (e) {
  console.error("❌ Could not read config.yaml:", e.message);
  process.exit(1);
}

const auth = config.auth?.bitbucket;
if (!auth) {
  console.error("❌ No auth.bitbucket found in config.yaml");
  process.exit(1);
}

let credentials;
if (auth.email && auth.api_token) {
  credentials = `${auth.email}:${auth.api_token}`;
  console.log(`🔑 Using Atlassian API Token (email: ${auth.email})`);
} else if (auth.username && auth.app_password) {
  credentials = `${auth.username}:${auth.app_password}`;
  console.log(`🔑 Using App Password (username: ${auth.username})`);
} else {
  console.error("❌ Invalid auth config — need email+api_token or username+app_password");
  process.exit(1);
}

const encoded = Buffer.from(credentials).toString("base64");
const url = "https://api.bitbucket.org/2.0/user";

console.log("\n📡 Testing against Bitbucket API...");

try {
  const res = await fetch(url, {
    headers: {
      Authorization: `Basic ${encoded}`,
      Accept: "application/json",
    },
  });

  const body = await res.json();

  if (res.ok) {
    console.log(`✅ Auth successful! Logged in as: ${body.display_name} (${body.account_id})`);
  } else {
    console.error(`❌ Auth failed (${res.status}): ${JSON.stringify(body)}`);
    console.error("\n💡 Fix: Update 'email' in config.yaml to your exact Atlassian account email.");
    console.error("   Your Atlassian email is the one you use to log in at bitbucket.org");
  }
} catch (e) {
  console.error("❌ Network error:", e.message);
}
