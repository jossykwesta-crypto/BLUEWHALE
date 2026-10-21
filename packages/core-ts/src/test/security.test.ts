import { afterEach, describe, expect, it } from "vitest";
import { extractRoutingFromURI, parseQueryParams } from "../routing/extractFromURI";

const G = "GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI";

/** Fail loudly if anything leaked onto the shared prototype. */
function assertPrototypeClean(): void {
  const probe: Record<string, unknown> = {};
  expect(probe.polluted).toBeUndefined();
  expect(probe.isAdmin).toBeUndefined();
  expect(Object.prototype).not.toHaveProperty("polluted");
  expect(Object.prototype).not.toHaveProperty("isAdmin");
}

afterEach(() => {
  // Belt and braces: a failing test must not poison the ones after it.
  delete (Object.prototype as Record<string, unknown>).polluted;
  delete (Object.prototype as Record<string, unknown>).isAdmin;
  delete (Object.prototype as Record<string, unknown>).destination;
});

const POLLUTION_PAYLOADS = [
  "__proto__[polluted]=yes",
  "__proto__=polluted",
  "__proto__.polluted=yes",
  "constructor[prototype][polluted]=yes",
  "constructor.prototype.polluted=yes",
  "constructor=polluted",
  "prototype=polluted",
  "%5F%5Fproto%5F%5F=polluted",
  "__proto__%5Bpolluted%5D=yes",
  "__proto__[isAdmin]=true&__proto__[isAdmin]=true",
];

describe("parseQueryParams", () => {
  it("returns a prototype-less record", () => {
    const params = parseQueryParams("destination=x");
    expect(Object.getPrototypeOf(params)).toBeNull();
    expect(params.destination).toBe("x");
  });

  it("does not resolve inherited Object.prototype members", () => {
    const params = parseQueryParams("destination=x");
    expect(params.toString).toBeUndefined();
    expect(params.hasOwnProperty).toBeUndefined();
    expect(params.constructor).toBeUndefined();
  });

  it.each(["__proto__", "constructor", "prototype"])("drops the %s key", (key) => {
    const params = parseQueryParams(`${key}=polluted&destination=x`);
    expect(Object.keys(params)).toEqual(["destination"]);
    expect(Object.getPrototypeOf(params)).toBeNull();
  });

  it("drops percent-encoded forbidden keys once decoded", () => {
    const params = parseQueryParams("%5F%5Fproto%5F%5F=polluted&%63onstructor=x");
    expect(Object.keys(params)).toEqual([]);
  });

  it("keeps the first value of a repeated key, like URLSearchParams#get", () => {
    expect(parseQueryParams("memo=1&memo=2").memo).toBe("1");
  });

  it.each(POLLUTION_PAYLOADS)("leaves Object.prototype untouched for %s", (payload) => {
    const params = parseQueryParams(payload);
    expect(Object.getPrototypeOf(params)).toBeNull();
    assertPrototypeClean();
  });
});

describe("extractRoutingFromURI prototype pollution", () => {
  it.each(POLLUTION_PAYLOADS)("routes normally and leaves Object.prototype untouched for %s", (payload) => {
    const result = extractRoutingFromURI(
      `web+stellar:pay?${payload}&destination=${G}&memo=42&memo_type=MEMO_ID`
    );
    expect(result.success).toBe(true);
    if (!result.success) return;
    expect(result.routing.destinationBaseAccount).toBe(G);
    expect(result.routing.routingId).toBe("42");
    assertPrototypeClean();
  });

  it("does not take the destination from an inherited property", () => {
    // With a plain `{}` accumulator this would inherit G as the destination.
    (Object.prototype as Record<string, unknown>).destination = G;
    const result = extractRoutingFromURI("web+stellar:pay?memo=1");
    expect(result).toMatchObject({ success: false, code: "MISSING_DESTINATION" });
  });

  it("cannot be tricked into a destination via __proto__", () => {
    const result = extractRoutingFromURI(`web+stellar:pay?__proto__[destination]=${G}`);
    expect(result).toMatchObject({ success: false, code: "MISSING_DESTINATION" });
    assertPrototypeClean();
  });

  it("exposes no forbidden keys in rawParams", () => {
    const result = extractRoutingFromURI(
      `web+stellar:pay?destination=${G}&__proto__=x&constructor=y&prototype=z`
    );
    expect(result.success).toBe(true);
    if (!result.success) return;
    expect(Object.prototype.hasOwnProperty.call(result.rawParams, "__proto__")).toBe(false);
    expect(Object.prototype.hasOwnProperty.call(result.rawParams, "prototype")).toBe(false);
    expect(Object.getPrototypeOf(result.rawParams)).toBe(Object.prototype);
  });
});
