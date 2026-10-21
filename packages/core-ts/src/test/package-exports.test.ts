/**
 * package.json entry points (#200, #194): explicit "bun" and "deno" export
 * conditions, CJS/ESM/types entries that match tsup's output names, and the
 * "sideEffects": false flag bundlers need for tree-shaking.
 */
import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

const pkg = JSON.parse(readFileSync(resolve(__dirname, "../../package.json"), "utf8"));
const root = pkg.exports["."];

/**
 * Resolves the "." export the way Node, Bun and Deno do: the first key, in
 * object order, that is one of the active conditions or "default".
 */
function resolveExport(conditions: string[]): string | undefined {
  for (const [key, target] of Object.entries(root)) {
    if (key === "default" || conditions.includes(key)) {
      return typeof target === "string" ? target : undefined;
    }
  }
  return undefined;
}

describe("package.json exports", () => {
  it("declares bun and deno conditions", () => {
    expect(root.bun).toBe("./dist/index.mjs");
    expect(root.deno).toBe("./dist/index.mjs");
  });

  it.each([
    ["Bun", ["bun", "import", "node"], "./dist/index.mjs"],
    ["Bun (require)", ["bun", "require", "node"], "./dist/index.mjs"],
    ["Deno", ["deno", "import", "node"], "./dist/index.mjs"],
    ["Node ESM", ["node", "import"], "./dist/index.mjs"],
    ["Node CJS", ["node", "require"], "./dist/index.js"],
    ["bundler", ["import", "module", "browser"], "./dist/index.mjs"],
    ["unknown runtime", ["worker"], "./dist/index.mjs"],
  ])("resolves %s to %s", (_, conditions, expected) => {
    expect(resolveExport(conditions as string[])).toBe(expected);
  });

  it("lists types first and default last, as resolution order requires", () => {
    const keys = Object.keys(root);
    expect(keys[0]).toBe("types");
    expect(keys[keys.length - 1]).toBe("default");
    expect(keys.indexOf("bun")).toBeLessThan(keys.indexOf("import"));
    expect(keys.indexOf("deno")).toBeLessThan(keys.indexOf("import"));
  });

  it("keeps the legacy fields in sync with the exports map", () => {
    expect(`./${pkg.main}`).toBe(root.require);
    expect(`./${pkg.module}`).toBe(root.import);
    expect(`./${pkg.types}`).toBe(root.types);
  });

  it("points only at files tsup emits for src/index.ts", () => {
    // `tsup src/index.ts --format cjs,esm --dts` without "type": "module"
    // writes index.js (CJS), index.mjs (ESM) and index.d.ts.
    expect(pkg.type).toBeUndefined();
    expect(pkg.scripts.build).toContain("--format cjs,esm");
    expect(pkg.scripts.build).toContain("--dts");
    for (const target of [root.types, root.import, root.require]) {
      expect(target).toMatch(/^\.\/dist\/index\.(js|mjs|d\.ts)$/);
    }
  });

  it("exposes package.json for tooling", () => {
    expect(pkg.exports["./package.json"]).toBe("./package.json");
  });
});

describe("sideEffects flag", () => {
  it('declares "sideEffects": false', () => {
    expect(pkg.sideEffects).toBe(false);
  });
});
