import { describe, it, expect } from "vitest";
import { validate } from "../address/validate";
import { detect } from "../address/detect";
import { parse } from "../address/parse";

const VALID_G = "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H";

describe("malformed address strings containing terminal/control escape codes", () => {
  const escapePayloads = [
    `\x1b[31m${VALID_G}\x1b[0m`,
    `${VALID_G}\x1b[2J\x1b[H`,
    `\u001b]8;;https://evil.example\u0007${VALID_G}\u001b]8;;\u0007`,
    `${VALID_G.slice(0, 28)}\r\n${VALID_G.slice(28)}`,
    `${VALID_G}\x00`,
    `\x07${VALID_G}`,
  ];

  it("rejects addresses containing ANSI CSI/OSC or control characters in validate()", () => {
    for (const payload of escapePayloads) {
      expect(validate(payload)).toBe(false);
      expect(validate(payload, "G")).toBe(false);
    }
  });

  it('returns "invalid" in detect() for escape-injected address strings', () => {
    for (const payload of escapePayloads) {
      expect(detect(payload)).toBe("invalid");
    }
  });

  it("never throws when parsing malformed escape-sequence addresses", () => {
    for (const payload of escapePayloads) {
      const result = parse(payload);
      expect(result.valid).toBe(false);
    }
  });
});
