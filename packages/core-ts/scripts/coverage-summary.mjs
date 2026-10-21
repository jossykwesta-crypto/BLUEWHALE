#!/usr/bin/env node
// Renders coverage/coverage-summary.json (vitest's json-summary reporter) and
// the thresholds from vitest.config.ts as a Markdown table on stdout.
// Used by ci-ts.yml for the job summary and the PR comment.
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const MARKER = "<!-- core-ts-coverage -->";
const METRICS = ["statements", "branches", "functions", "lines"];

let summary;
try {
  summary = JSON.parse(readFileSync(join(root, "coverage", "coverage-summary.json"), "utf8"));
} catch {
  console.log(`${MARKER}\n### core-ts coverage\n\nNo coverage report was produced; see the test step log.`);
  process.exit(0);
}

// Read thresholds from the config text so the table can't drift from it.
const config = readFileSync(join(root, "vitest.config.ts"), "utf8");
const block = config.match(/thresholds:\s*{([^}]*)}/)?.[1] ?? "";
const thresholds = Object.fromEntries(
  [...block.matchAll(/(\w+):\s*(\d+(?:\.\d+)?)/g)].map(([, k, v]) => [k, Number(v)])
);

const rows = METRICS.map((metric) => {
  const { pct, covered, total } = summary.total[metric];
  const min = thresholds[metric];
  const status = min === undefined ? "–" : pct >= min ? "✅" : "❌";
  return `| ${metric} | ${pct.toFixed(2)}% (${covered}/${total}) | ${min === undefined ? "–" : `${min}%`} | ${status} |`;
});

console.log(
  [
    MARKER,
    "### core-ts coverage",
    "",
    "| Metric | Covered | Threshold | |",
    "| --- | --- | --- | --- |",
    ...rows,
  ].join("\n")
);
