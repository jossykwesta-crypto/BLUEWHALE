/**
 * RoutingResult JSON serialization (#201): JSON.stringify must not throw on
 * bigint routing IDs, and IDs above 2^53 - 1 must survive as exact strings.
 */
import { describe, it, expect } from "vitest";
import { extractRouting } from "../routing/extract";
import { extractRoutingFromTx } from "../routing/extractFromTx";
import {
  routingIdToJSON,
  stringifyRoutingResult,
  toSerializableRoutingResult,
  withRoutingResultJSON,
} from "../routing/result";
import { SafeRoutingId } from "../routing/safeRoutingId";
import { encodeMuxed } from "../muxed/encode";
import type { RoutingResult } from "../routing/types";

const G_ADDRESS = "GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI";
const UINT64_MAX = 18446744073709551615n;
const ABOVE_SAFE = 9007199254740993n;

function muxedInput(id: bigint) {
  return {
    destination: encodeMuxed(G_ADDRESS, id),
    memoType: "none",
    memoValue: null,
    sourceAccount: null,
  };
}

describe("JSON.stringify(extractRouting(...))", () => {
  it("does not throw for a muxed result whose routingId is a bigint", () => {
    const result = extractRouting(muxedInput(ABOVE_SAFE));
    expect(typeof result.routingId).toBe("bigint");
    expect(() => JSON.stringify(result)).not.toThrow();
  });

  it("keeps uint64 max exact as a decimal string", () => {
    const parsed = JSON.parse(JSON.stringify(extractRouting(muxedInput(UINT64_MAX))));
    expect(parsed.routingId).toBe("18446744073709551615");
    expect(BigInt(parsed.routingId)).toBe(UINT64_MAX);
    expect(parsed.destinationBaseAccount).toBe(G_ADDRESS);
    expect(parsed.routingSource).toBe("muxed");
  });

  it("serializes memo routing IDs and null routing IDs", () => {
    const memo = extractRouting({
      destination: G_ADDRESS,
      memoType: "id",
      memoValue: "9007199254740993",
      sourceAccount: null,
    });
    expect(JSON.parse(JSON.stringify(memo)).routingId).toBe("9007199254740993");

    const none = extractRouting({ destination: G_ADDRESS, memoType: "none", memoValue: null, sourceAccount: null });
    expect(JSON.stringify(none)).toContain('"routingId":null');
  });

  it("serializes error results and nested results", () => {
    const bad = extractRouting({ destination: "XNOTANADDRESS", memoType: "none", memoValue: null, sourceAccount: null });
    expect(() => JSON.stringify({ results: [bad, extractRouting(muxedInput(1n))] })).not.toThrow();
  });

  it("works for extractRoutingFromTx results", () => {
    const result = extractRoutingFromTx({
      operations: [{ type: "payment", destination: encodeMuxed(G_ADDRESS, ABOVE_SAFE) }],
      memo: { type: "none", value: null },
    });
    expect(JSON.parse(JSON.stringify(result)).routingId).toBe(ABOVE_SAFE.toString());
  });

  it("does not change the result's enumerable shape", () => {
    const result = extractRouting(muxedInput(42n));
    expect(Object.keys(result)).not.toContain("toJSON");
    expect(result).toEqual({
      destinationBaseAccount: G_ADDRESS,
      routingId: 42n,
      routingSource: "muxed",
      warnings: [],
    });
  });
});

describe("stringifyRoutingResult", () => {
  it("serializes hand-built and spread results that have no toJSON", () => {
    const original = extractRouting(muxedInput(UINT64_MAX));
    const spread: RoutingResult = { ...original };
    expect(() => JSON.stringify(spread)).toThrow(TypeError);

    expect(JSON.parse(stringifyRoutingResult(spread)).routingId).toBe(UINT64_MAX.toString());
  });

  it("matches JSON.stringify of an extractRouting result", () => {
    const result = extractRouting(muxedInput(ABOVE_SAFE));
    expect(stringifyRoutingResult(result)).toBe(JSON.stringify(result));
  });

  it("supports indentation", () => {
    const out = stringifyRoutingResult(extractRouting(muxedInput(1n)), 2);
    expect(out).toContain('\n  "routingId": "1"');
  });
});

describe("routingIdToJSON / toSerializableRoutingResult", () => {
  it("normalises every routingId representation to a string or null", () => {
    expect(routingIdToJSON(UINT64_MAX)).toBe("18446744073709551615");
    expect(routingIdToJSON("123")).toBe("123");
    expect(routingIdToJSON(new SafeRoutingId(ABOVE_SAFE))).toBe("9007199254740993");
    expect(routingIdToJSON(null)).toBeNull();
    expect(routingIdToJSON(undefined)).toBeNull();
  });

  it("returns a plain copy and leaves the input untouched", () => {
    const result = withRoutingResultJSON<RoutingResult>({
      destinationBaseAccount: G_ADDRESS,
      routingId: 7n,
      routingSource: "muxed",
      warnings: [],
    });
    const copy = toSerializableRoutingResult(result);
    expect(copy).toEqual({ destinationBaseAccount: G_ADDRESS, routingId: "7", routingSource: "muxed", warnings: [] });
    expect("toJSON" in copy).toBe(false);
    expect(result.routingId).toBe(7n);
  });
});
