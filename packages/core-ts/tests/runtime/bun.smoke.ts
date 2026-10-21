// Bun smoke test for the published entry point (#200). Run after building:
//   pnpm build && pnpm test:bun
// Loads the built file the "bun" export condition points at; which file each
// condition selects is checked in src/test/package-exports.test.ts.
import { describe, expect, it } from "bun:test";
import { detect, extractRouting, parse } from "../../dist/index.mjs";

const G = "GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI";
const M = "MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACABAAAAAAAAAAEVIG";

describe("bluewhale-core under Bun", () => {
  it("detects and parses addresses", () => {
    expect(detect(G)).toBe("G");
    expect(parse(M).kind).toBe("M");
  });

  it("extracts routing from a muxed destination", () => {
    const result = extractRouting({ destination: M, memoType: "none", memoValue: null, sourceAccount: null });
    expect(result.destinationBaseAccount).toBe(G);
    expect(result.routingSource).toBe("muxed");
  });
});
