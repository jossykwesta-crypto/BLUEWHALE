import { describe, expect, it } from "vitest";
import { MAX_SAFE_INTEGER, SafeRoutingId, UINT64_MAX } from "../routing/safeRoutingId";
import { routingIdAsBigInt } from "../routing/types";

describe("SafeRoutingId", () => {
  it.each([
    ["a string", "42"],
    ["a padded string", " 42 "],
    ["a bigint", 42n],
    ["a number", 42],
  ])("accepts %s", (_label, input) => {
    const id = new SafeRoutingId(input);
    expect(id.toString()).toBe("42");
    expect(id.toJSON()).toBe("42");
    expect(id.toBigInt()).toBe(42n);
  });

  it("keeps uint64 max exact", () => {
    const id = SafeRoutingId.parse("18446744073709551615");
    expect(id.toBigInt()).toBe(UINT64_MAX);
    expect(id.isSafeInteger()).toBe(false);
    expect(JSON.stringify({ id })).toBe('{"id":"18446744073709551615"}');
  });

  it("reports whether the value fits in a JS number", () => {
    expect(SafeRoutingId.from(MAX_SAFE_INTEGER).isSafeInteger()).toBe(true);
    expect(SafeRoutingId.from(MAX_SAFE_INTEGER + 1n).isSafeInteger()).toBe(false);
  });

  it.each([
    ["a negative bigint", -1n, RangeError],
    ["a bigint above uint64", UINT64_MAX + 1n, RangeError],
    ["a negative number", -1, RangeError],
    ["a fractional number", 1.5, RangeError],
    ["a number above MAX_SAFE_INTEGER", Number.MAX_SAFE_INTEGER + 2, RangeError],
    ["a non-numeric string", "12a", TypeError],
    ["a signed string", "-1", TypeError],
    ["a string above uint64", "18446744073709551616", RangeError],
    ["an unsupported type", {} as unknown as string, TypeError],
  ])("rejects %s", (_label, input, errorType) => {
    expect(() => new SafeRoutingId(input)).toThrow(errorType);
  });

  it("tryParse returns null instead of throwing", () => {
    expect(SafeRoutingId.tryParse(null)).toBeNull();
    expect(SafeRoutingId.tryParse(undefined)).toBeNull();
    expect(SafeRoutingId.tryParse("nope")).toBeNull();
    expect(SafeRoutingId.tryParse("7")?.toString()).toBe("7");
  });
});

describe("routingIdAsBigInt", () => {
  it.each([
    ["null", null, null],
    ["a SafeRoutingId", new SafeRoutingId("9"), 9n],
    ["a bigint", 9n, 9n],
    ["a decimal string", "9", 9n],
  ])("converts %s", (_label, input, expected) => {
    expect(routingIdAsBigInt(input)).toBe(expected);
  });
});
