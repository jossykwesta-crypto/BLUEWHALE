import { describe, expect, it, vi } from "vitest";
import { extractRoutingAsync, fetchMemoRequirement } from "../routing/memoRequirement";

const G = "GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI";
const M = "MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACAAAAAAAAAAAFLKXQ";

/** A fetch stub that answers with `data` as the account's data entries. */
function horizon(data: Record<string, unknown> | undefined, ok = true) {
  return vi.fn(async (_url: string) => ({ ok, json: async () => ({ data }) })) as unknown as typeof fetch;
}

describe("fetchMemoRequirement", () => {
  it("queries the account endpoint, trimming a trailing slash", async () => {
    const fetchImpl = horizon({});
    await fetchMemoRequirement(G, fetchImpl, "https://horizon-testnet.stellar.org/");
    expect(fetchImpl).toHaveBeenCalledWith(`https://horizon-testnet.stellar.org/accounts/${G}`);
  });

  it("defaults to the public Horizon URL", async () => {
    const fetchImpl = horizon({});
    await fetchMemoRequirement(G, fetchImpl);
    expect(fetchImpl).toHaveBeenCalledWith(`https://horizon.stellar.org/accounts/${G}`);
  });

  it.each([
    ["base64 '1'", { "config.memo_required": btoa("1") }, true],
    ["literal '1'", { "config.memo_required": "1" }, true],
    ["number 1", { "config.memo_required": 1 }, true],
    ["boolean true", { "config.memo_required": true }, true],
    ["legacy config.requiring_memo key", { "config.requiring_memo": btoa("1") }, true],
    ["base64 '0'", { "config.memo_required": btoa("0") }, false],
    ["invalid base64", { "config.memo_required": "%%%" }, false],
    ["a non-string value", { "config.memo_required": { nested: 1 } }, false],
    ["no entry", {}, false],
    ["no data at all", undefined, false],
  ])("reads %s", async (_label, data, expected) => {
    await expect(fetchMemoRequirement(G, horizon(data))).resolves.toBe(expected);
  });

  it("treats a non-OK response as not required", async () => {
    await expect(fetchMemoRequirement(G, horizon({ "config.memo_required": "1" }, false))).resolves.toBe(false);
  });
});

describe("extractRoutingAsync", () => {
  const input = { destination: G, memoType: "none", memoValue: null, sourceAccount: null };

  it("adds MISSING_REQUIRED_MEMO when the destination requires a memo", async () => {
    const result = await extractRoutingAsync(input, async () => true);
    expect(result.warnings.map((w) => w.code)).toEqual(["MISSING_REQUIRED_MEMO"]);
  });

  it("leaves the result unchanged when no memo is required", async () => {
    const result = await extractRoutingAsync(input, async () => false);
    expect(result.warnings).toEqual([]);
  });

  it("fails open when the lookup throws", async () => {
    const result = await extractRoutingAsync(input, async () => {
      throw new Error("network down");
    });
    expect(result.warnings).toEqual([]);
    expect(result.destinationBaseAccount).toBe(G);
  });

  it.each([
    ["a routing ID is already present", { ...input, memoType: "id", memoValue: "7" }],
    ["the destination is muxed", { ...input, destination: M }],
    ["the destination is invalid", { ...input, destination: "XNOTANADDRESS" }],
  ])("skips the lookup when %s", async (_label, routingInput) => {
    const fetcher = vi.fn(async () => true);
    await extractRoutingAsync(routingInput, fetcher);
    expect(fetcher).not.toHaveBeenCalled();
  });

  it("uses fetchMemoRequirement by default", async () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch").mockRejectedValue(new Error("offline"));
    const result = await extractRoutingAsync(input);
    expect(fetchSpy).toHaveBeenCalledOnce();
    expect(result.warnings).toEqual([]);
    fetchSpy.mockRestore();
  });
});
