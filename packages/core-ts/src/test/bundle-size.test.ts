/**
 * Tree-shaking budget (#194): a consumer importing only extractRouting from
 * the package entry must get a small bundle. Relies on "sideEffects": false,
 * which lets the bundler drop every module extractRouting does not use.
 *
 * Bundles with tsup (esbuild), already a devDependency, so no new tooling.
 */
import { describe, it, expect, beforeAll, afterAll } from "vitest";
import { mkdtempSync, readFileSync, readdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { gzipSync } from "node:zlib";
import { build } from "tsup";

/** Gzipped size budget for `import { extractRouting }`, in bytes. */
const MAX_GZIP_BYTES = 5 * 1024;

const ENTRY_INDEX = resolve(__dirname, "../index.ts").replace(/\\/g, "/");

let workDir: string;

async function bundle(source: string): Promise<string> {
  const dir = mkdtempSync(join(workDir, "case-"));
  const entry = join(dir, "entry.ts");
  writeFileSync(entry, source);
  await build({
    config: false,
    entry: { bundle: entry },
    outDir: join(dir, "out"),
    format: ["esm"],
    platform: "neutral",
    target: "es2020",
    minify: true,
    bundle: true,
    splitting: false,
    sourcemap: false,
    dts: false,
    clean: false,
    silent: true,
    // A peer dependency: provided by the consumer, not part of our size.
    external: ["@stellar/stellar-sdk"],
  });
  const out = join(dir, "out");
  const file = readdirSync(out).find((f) => f.startsWith("bundle."));
  if (!file) throw new Error(`no bundle written to ${out}`);
  return readFileSync(join(out, file), "utf8");
}

const gzipBytes = (code: string) => gzipSync(code, { level: 9 }).length;

beforeAll(() => {
  workDir = mkdtempSync(join(tmpdir(), "bluewhale-bundle-"));
});

afterAll(() => {
  rmSync(workDir, { recursive: true, force: true });
});

describe("tree-shaken bundle size", () => {
  let onlyExtract: string;
  let everything: string;

  beforeAll(async () => {
    onlyExtract = await bundle(
      `import { extractRouting } from "${ENTRY_INDEX}";\nexport const run = extractRouting;\n`
    );
    everything = await bundle(`export * from "${ENTRY_INDEX}";\n`);
  }, 60_000);

  it(`keeps import { extractRouting } under ${MAX_GZIP_BYTES} bytes gzipped`, () => {
    const size = gzipBytes(onlyExtract);
    console.log(`[bundle-size] extractRouting only: ${size} B gzipped`);
    expect(size).toBeLessThan(MAX_GZIP_BYTES);
  });

  it("drops modules extractRouting does not use", () => {
    // String literals survive minification, so they reveal which modules
    // made it into the bundle.
    expect(onlyExtract).not.toContain("web+stellar:"); // extractFromURI
    expect(onlyExtract).not.toContain("config.memo_required"); // memoRequirement
    expect(everything).toContain("web+stellar:");
    expect(everything).toContain("config.memo_required");
  });

  it("is smaller than bundling the whole package", () => {
    expect(gzipBytes(onlyExtract)).toBeLessThan(gzipBytes(everything));
  });

  it("an empty import adds nothing", async () => {
    const code = await bundle(`import "${ENTRY_INDEX}";\nexport const x = 1;\n`);
    expect(code).not.toContain("web+stellar:");
    expect(code).not.toContain("extractRouting");
    expect(gzipBytes(code)).toBeLessThan(200);
  }, 60_000);
});
