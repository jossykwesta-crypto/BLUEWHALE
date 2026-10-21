import { describe, expect, it, vi } from "vitest";
import { AddressParseError } from "../address/errors";

// decodeMuxed() wraps the SDK and normally throws a plain Error. If it ever
// raises an AddressParseError, parse() must pass it through untouched rather
// than rewrapping it as INVALID_CHECKSUM.
vi.mock("../muxed/decode", () => ({
  decodeMuxed: () => {
    throw new AddressParseError("INVALID_LENGTH", "M...", "from decodeMuxed");
  },
}));

const { parse } = await import("../address/parse");

const M = "MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACAAAAAAAAAAAFLKXQ";

describe("parse() muxed decode errors", () => {
  it("rethrows an AddressParseError from decodeMuxed unchanged", () => {
    expect(() => parse(M)).toThrow(expect.objectContaining({ code: "INVALID_LENGTH", message: "from decodeMuxed" }));
  });
});
