import { describe, expect, it, vi } from "vitest";

// parse() only ever throws AddressParseError; force anything else through to
// prove extractRouting does not swallow unexpected failures.
vi.mock("../address/parse", () => ({
  parse: () => {
    throw new TypeError("unexpected");
  },
}));

const { extractRouting } = await import("../routing/extract");

describe("extractRouting() unexpected parse failures", () => {
  it("rethrows errors that are not AddressParseError", () => {
    expect(() =>
      extractRouting({ destination: "GABC", memoType: "none", memoValue: null, sourceAccount: null })
    ).toThrow(TypeError);
  });
});
