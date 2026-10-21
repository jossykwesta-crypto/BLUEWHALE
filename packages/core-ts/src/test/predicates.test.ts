import { describe, it, expect, expectTypeOf } from "vitest";
import {
  isContractAddress,
  isMuxedAddress,
  isPublicKeyAddress,
  type ContractAddress,
  type MuxedAddress,
  type PublicKeyAddress,
} from "../index";

const G = "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H";
const M = "MBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OAAAAAAAAAAAPOGVY";
const C = "CAAQCAIBAEAQCAIBAEAQCAIBAEAQCAIBAEAQCAIBAEAQCAIBAEAQC526";
const G_BAD_CHECKSUM = "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2X";

const predicates = {
  G: isPublicKeyAddress,
  M: isMuxedAddress,
  C: isContractAddress,
} as const;

const valid = { G, M, C } as const;

describe("address predicates", () => {
  for (const [kind, predicate] of Object.entries(predicates)) {
    describe(predicate.name, () => {
      it(`accepts a valid ${kind} address`, () => {
        expect(predicate(valid[kind as keyof typeof valid])).toBe(true);
      });

      it("accepts lowercase input, like detect()", () => {
        expect(predicate(valid[kind as keyof typeof valid].toLowerCase())).toBe(true);
      });

      it("rejects the other address kinds", () => {
        for (const [other, addr] of Object.entries(valid)) {
          if (other !== kind) expect(predicate(addr)).toBe(false);
        }
      });

      it("rejects malformed input without throwing", () => {
        for (const input of [
          "",
          " ",
          kind,
          G_BAD_CHECKSUM,
          `${valid[kind as keyof typeof valid]} `,
          valid[kind as keyof typeof valid].slice(0, -1),
          "bob*example.com",
          "\u0000",
        ]) {
          expect(predicate(input)).toBe(false);
        }
      });

      it("rejects non-string values", () => {
        for (const input of [undefined, null, 0, 1n, {}, [], [valid.G], new String(G)]) {
          expect(predicate(input)).toBe(false);
        }
      });
    });
  }

  it("narrows to the branded types", () => {
    const input: unknown = G;
    if (isPublicKeyAddress(input)) expectTypeOf(input).toEqualTypeOf<PublicKeyAddress>();
    if (isMuxedAddress(input)) expectTypeOf(input).toEqualTypeOf<MuxedAddress>();
    if (isContractAddress(input)) expectTypeOf(input).toEqualTypeOf<ContractAddress>();

    // Branded values are still usable wherever a string is expected.
    expectTypeOf<PublicKeyAddress>().toMatchTypeOf<string>();
  });
});
