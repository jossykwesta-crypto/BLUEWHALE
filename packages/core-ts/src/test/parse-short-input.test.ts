/**
 * parse() on truncated input (#199): strings shorter than MIN_ADDRESS_LENGTH
 * and non-string values fail with a structured AddressParseError, never a
 * TypeError, and extractRouting reports short destinations as a
 * destinationError.
 */
import { describe, it, expect } from "vitest";
import { parse, MIN_ADDRESS_LENGTH } from "../address/parse";
import { AddressParseError } from "../address/errors";
import { extractRouting } from "../routing/extract";

function parseError(input: unknown): AddressParseError {
  try {
    parse(input as string);
  } catch (error) {
    expect(error).toBeInstanceOf(AddressParseError);
    return error as AddressParseError;
  }
  throw new Error(`parse(${JSON.stringify(input)}) did not throw`);
}

describe("parse() short input guard", () => {
  it("requires at least 4 characters", () => {
    expect(MIN_ADDRESS_LENGTH).toBe(4);
  });

  it.each(["", "G", "GA", "GAY", "M", "MA", "C", "CAA", "S", "?", "  "])(
    "rejects %j with INVALID_LENGTH",
    (input) => {
      const error = parseError(input);
      expect(error.code).toBe("INVALID_LENGTH");
      expect(error.input).toBe(input);
      expect(error.message).toBe(`Address is too short (${input.length} characters)`);
    }
  );

  it.each([undefined, null, 42, {}, ["G"]])("rejects non-string %j without a TypeError", (input) => {
    const error = parseError(input);
    expect(error.code).toBe("INVALID_LENGTH");
    expect(error.input).toBe("");
    expect(error.message).toBe("Address must be a string");
  });

  it("keeps the existing errors for inputs of 4 or more characters", () => {
    expect(parseError("GAYC").code).toBe("INVALID_CHECKSUM");
    expect(parseError("ABCD").code).toBe("UNKNOWN_PREFIX");
  });

  it("still parses valid addresses", () => {
    const g = "GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI";
    expect(parse(g)).toEqual({ kind: "G", address: g, warnings: [] });
  });
});

describe("extractRouting() with truncated destinations", () => {
  it.each(["", "G", "GA", "GAY"])("returns a destinationError for %j", (destination) => {
    const result = extractRouting({
      destination,
      memoType: "none",
      memoValue: null,
      sourceAccount: null,
    });

    expect(result.destinationBaseAccount).toBeNull();
    expect(result.routingId).toBeNull();
    expect(result.destinationError).toEqual({
      code: "INVALID_LENGTH",
      message: `Address is too short (${destination.length} characters)`,
    });
  });
});
