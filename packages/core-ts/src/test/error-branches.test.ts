import { afterEach, describe, expect, it, vi } from "vitest";
import { detect } from "../address/detect";
import { parse } from "../address/parse";
import { AddressParseError } from "../address/errors";
import { extractRouting, ExtractRoutingError } from "../routing/extract";
import { extractRoutingFromURI, sanitizeSep7UriForLogging } from "../routing/extractFromURI";
import { WARNING_MESSAGES } from "../routing/warnings";

/**
 * Error-handling and fallback branches that the behaviour-focused suites
 * don't reach. Each case states the input that drives the branch.
 */

const G = "GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI";
const M = "MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACAAAAAAAAAAAFLKXQ";
/** M with the final (padding) bit set: CRC still matches, StrKey rejects it. */
const M_NON_CANONICAL = "MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACAAAAAAAAAAAFLKXR";
/** M with a corrupted checksum character. */
const M_BAD_CHECKSUM = "MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACAAAAAAAAAAAFLKAQ";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("detect() base32 fallback", () => {
  it("returns M for a muxed address whose only defect is a set padding bit", () => {
    // StrKey rejects non-canonical encodings; the fallback decoder only checks
    // version byte, length, and CRC.
    expect(detect(M_NON_CANONICAL)).toBe("M");
  });

  it("returns invalid when the muxed checksum does not match", () => {
    expect(detect(M_BAD_CHECKSUM)).toBe("invalid");
  });

  it("returns invalid for an M-prefixed string with non-base32 characters", () => {
    expect(detect("M1")).toBe("invalid");
  });
});

describe("parse() muxed decode failures", () => {
  it("raises INVALID_CHECKSUM when detect() accepts but the muxed decode fails", () => {
    expect(() => parse(M_NON_CANONICAL)).toThrow(AddressParseError);
    try {
      parse(M_NON_CANONICAL);
    } catch (error) {
      expect((error as AddressParseError).code).toBe("INVALID_CHECKSUM");
    }
  });
});

describe("extractRouting() guard and error paths", () => {
  it("throws ExtractRoutingError for a non-string destination", () => {
    const input = { destination: 42 as unknown as string, memoType: "none", memoValue: null, sourceAccount: null };
    expect(() => extractRouting(input)).toThrow(ExtractRoutingError);
    try {
      extractRouting(input);
    } catch (error) {
      expect(error).toBeInstanceOf(Error);
      expect((error as Error).name).toBe("ExtractRoutingError");
    }
  });

  it.each([
    ["an unknown prefix", "XNOTANADDRESS", "UNKNOWN_PREFIX"],
    ["a bad G checksum", G.slice(0, -1) + "A", "INVALID_CHECKSUM"],
    ["a non-canonical muxed address", M_NON_CANONICAL, "INVALID_CHECKSUM"],
  ])("returns a destinationError for %s", (_label, destination, code) => {
    const result = extractRouting({ destination, memoType: "none", memoValue: null, sourceAccount: null });
    expect(result).toMatchObject({
      destinationBaseAccount: null,
      routingId: null,
      routingSource: "none",
      destinationError: { code },
    });
  });

  it("routes MEMO_TEXT numerically for a G-address", () => {
    expect(
      extractRouting({ destination: G, memoType: "text", memoValue: "0042", sourceAccount: null })
    ).toMatchObject({ routingId: "42", routingSource: "memo" });
  });

  it("flags an unrecognized memo type", () => {
    const result = extractRouting({ destination: G, memoType: "bogus", memoValue: "1", sourceAccount: null });
    expect(result.warnings).toEqual([
      expect.objectContaining({ code: "UNSUPPORTED_MEMO_TYPE", context: { memoType: "unknown" } }),
    ]);
  });

  it("filters muxed warnings by minimum severity", () => {
    const result = extractRouting({
      destination: M,
      memoType: "hash",
      memoValue: "ab",
      sourceAccount: null,
      minSeverityLevel: "warn",
    });
    expect(result.warnings).toEqual([]);
  });
});

describe("extractRoutingFromURI() memo types and failures", () => {
  it.each([
    ["MEMO_TEXT", "text", "42", "42"],
    ["memo_text", "text", "42", "42"],
    ["MEMO_HASH", "hash", "ab", null],
    ["MEMO_RETURN", "return", "ab", null],
    ["MEMO_BOGUS", "none", "42", null],
  ])("maps memo_type=%s", (sep7Type, _internal, memo, routingId) => {
    const result = extractRoutingFromURI(`web+stellar:pay?destination=${G}&memo=${memo}&memo_type=${sep7Type}`);
    expect(result.success).toBe(true);
    if (result.success) expect(result.routing.routingId).toBe(routingId);
  });

  it("returns INVALID_ENCODING when the query string cannot be parsed", () => {
    vi.stubGlobal(
      "URLSearchParams",
      class {
        constructor() {
          throw new TypeError("malformed");
        }
      }
    );
    expect(extractRoutingFromURI(`web+stellar:pay?destination=${G}`)).toEqual({
      success: false,
      error: "Failed to parse URI query parameters",
      code: "INVALID_ENCODING",
    });
  });

  it("redacts a sensitive key that has no value", () => {
    expect(sanitizeSep7UriForLogging("web+stellar:pay?signature&destination=G")).toBe(
      "web+stellar:pay?signature=[REDACTED]&destination=G"
    );
  });
});

describe("WARNING_MESSAGES", () => {
  it("matches the messages extractRouting emits", () => {
    const c = "CAAQCAIBAEAQCAIBAEAQCAIBAEAQCAIBAEAQCAIBAEAQCAIBAEAQC526";
    const result = extractRouting({ destination: c, memoType: "none", memoValue: null, sourceAccount: null });
    expect(result.warnings[0].message).toBe(WARNING_MESSAGES.INVALID_DESTINATION);
  });
});
