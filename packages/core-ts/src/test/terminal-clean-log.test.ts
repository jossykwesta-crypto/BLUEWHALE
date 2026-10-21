import { describe, it, expect, vi } from "vitest";
import { validate } from "../address/validate";
import { detect } from "../address/detect";
import { parse } from "../address/parse";

describe("clean terminal log output and zero-throw guarantee on invalid inputs", () => {
  it("does not emit console output or raw ANSI escape codes during malformed address validation", () => {
    const logSpy = vi.spyOn(console, "log").mockImplementation(() => {});
    const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});

    const inputs = [
      "",
      "\x1b[31mMALFORMED_ESCAPE\x1b[0m",
      "G_INVALID_BASE32_!@#$%^&*()",
      "M12345",
      "C99999",
    ];

    for (const input of inputs) {
      expect(() => validate(input)).not.toThrow();
      expect(() => detect(input)).not.toThrow();
      expect(() => parse(input)).not.toThrow();
    }

    expect(logSpy).not.toHaveBeenCalled();
    expect(warnSpy).not.toHaveBeenCalled();
    expect(errorSpy).not.toHaveBeenCalled();

    logSpy.mockRestore();
    warnSpy.mockRestore();
    errorSpy.mockRestore();
  });
});
