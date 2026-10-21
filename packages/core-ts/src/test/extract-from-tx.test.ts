import { describe, expect, it } from "vitest";
import { extractRoutingFromTx } from "../routing/extractFromTx";

const G = "GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI";
const C = "CAAQCAIBAEAQCAIBAEAQCAIBAEAQCAIBAEAQCAIBAEAQCAIBAEAQC526";

describe("extractRoutingFromTx", () => {
  it("returns null for a transaction with no operations", () => {
    expect(extractRoutingFromTx({ operations: [], memo: { type: "none" } })).toBeNull();
  });

  it("returns null when the first operation is not a payment", () => {
    const tx = { operations: [{ type: "createAccount", destination: G }], memo: { type: "none" } };
    expect(extractRoutingFromTx(tx)).toBeNull();
  });

  it("routes a payment by its MEMO_ID", () => {
    const tx = {
      operations: [{ type: "payment", destination: G }],
      memo: { type: "id", value: 42n },
    };
    expect(extractRoutingFromTx(tx)).toMatchObject({
      destinationBaseAccount: G,
      routingId: "42",
      routingSource: "memo",
    });
  });

  it("passes a missing memo value through as null", () => {
    const tx = { operations: [{ type: "payment", destination: G }], memo: { type: "none" } };
    expect(extractRoutingFromTx(tx)).toMatchObject({ routingId: null, routingSource: "none" });
  });

  it("uses the transaction source to detect contract senders", () => {
    const tx = {
      source: C,
      operations: [{ type: "payment", destination: G }],
      memo: { type: "id", value: "42" },
    };
    expect(extractRoutingFromTx(tx)?.warnings.map((w) => w.code)).toEqual(["CONTRACT_SENDER_DETECTED"]);
  });
});
