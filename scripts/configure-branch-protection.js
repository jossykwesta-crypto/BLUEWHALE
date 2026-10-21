#!/usr/bin/env node
/**
 * Enforces signed commits and branch protection rules via the GitHub REST API.
 *
 * Usage:
 *   GITHUB_TOKEN=<token> GITHUB_REPOSITORY=<owner/repo> node scripts/configure-branch-protection.js [--dry-run]
 */

const https = require("https");

const REPO = process.env.GITHUB_REPOSITORY || "progidyken-hub/BLUEWHALE";
const BRANCH = process.env.PROTECTED_BRANCH || "main";
const TOKEN = process.env.GITHUB_TOKEN || process.env.GH_TOKEN || "";
const DRY_RUN = process.argv.includes("--dry-run") || !TOKEN;

const BRANCH_PROTECTION_PAYLOAD = {
  required_status_checks: {
    strict: true,
    contexts: [
      "Lint PR Title (Conventional Commits)",
      "Test (Go 1.21)",
      "Test (Go 1.22)",
      "Test (Go 1.23)",
      "test",
    ],
  },
  enforce_admins: true,
  required_pull_request_reviews: {
    dismiss_stale_reviews: true,
    require_code_owner_reviews: false,
    required_approving_review_count: 1,
    require_last_push_approval: false,
  },
  restrictions: null,
  required_linear_history: true,
  allow_force_pushes: false,
  allow_deletions: false,
  block_creations: false,
  required_conversation_resolution: true,
  lock_branch: false,
  allow_fork_syncing: true,
};

function githubRequest(method, path, body) {
  return new Promise((resolve, reject) => {
    const payload = body ? JSON.stringify(body) : null;
    const req = https.request(
      {
        hostname: "api.github.com",
        path,
        method,
        headers: {
          Accept: "application/vnd.github+json",
          Authorization: `Bearer ${TOKEN}`,
          "User-Agent": "bluewhale-branch-protection-enforcer",
          "X-GitHub-Api-Version": "2022-11-28",
          ...(payload
            ? {
                "Content-Type": "application/json",
                "Content-Length": Buffer.byteLength(payload),
              }
            : {}),
        },
      },
      (res) => {
        let data = "";
        res.on("data", (chunk) => {
          data += chunk;
        });
        res.on("end", () => {
          if (res.statusCode >= 200 && res.statusCode < 300) {
            resolve({ statusCode: res.statusCode, body: data });
          } else {
            reject(
              new Error(
                `GitHub API ${method} ${path} failed (${res.statusCode}): ${data}`
              )
            );
          }
        });
      }
    );
    req.on("error", reject);
    if (payload) req.write(payload);
    req.end();
  });
}

async function main() {
  const protectionPath = `/repos/${REPO}/branches/${BRANCH}/protection`;
  const signaturesPath = `/repos/${REPO}/branches/${BRANCH}/protection/required_signatures`;

  if (DRY_RUN) {
    console.log(
      `[dry-run] Would PUT ${protectionPath} with payload:\n` +
        JSON.stringify(BRANCH_PROTECTION_PAYLOAD, null, 2)
    );
    console.log(
      `[dry-run] Would POST ${signaturesPath} to enforce signed commits.`
    );
    return;
  }

  console.log(`Applying branch protection rules to ${REPO}@${BRANCH}...`);
  await githubRequest("PUT", protectionPath, BRANCH_PROTECTION_PAYLOAD);
  console.log("Branch protection rules updated.");

  console.log(`Enforcing signed commits on ${REPO}@${BRANCH}...`);
  await githubRequest("POST", signaturesPath);
  console.log("Signed commit requirement enabled.");
}

if (require.main === module) {
  main().catch((err) => {
    console.error(err.message || err);
    process.exit(1);
  });
}

module.exports = {
  BRANCH_PROTECTION_PAYLOAD,
  githubRequest,
};
