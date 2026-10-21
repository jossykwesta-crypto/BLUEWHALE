// Deno smoke test for the published entry point (#200). Run after building:
//   pnpm build && pnpm test:deno
// Loads the built file the "deno" export condition points at; which file each
// condition selects is checked in src/test/package-exports.test.ts. Its
// @stellar/stellar-sdk import is resolved from node_modules (pnpm install).
import { assertEquals } from "jsr:@std/assert@1";
import { detect, extractRouting, parse } from "../../dist/index.mjs";

const G = "GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI";
const M = "MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACABAAAAAAAAAAEVIG";

Deno.test("detects and parses addresses", () => {
  assertEquals(detect(G), "G");
  assertEquals(parse(M).kind, "M");
});

Deno.test("extracts routing from a muxed destination", () => {
  const result = extractRouting({ destination: M, memoType: "none", memoValue: null, sourceAccount: null });
  assertEquals(result.destinationBaseAccount, G);
  assertEquals(result.routingSource, "muxed");
});
