import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

/**
 * The CLI parses `process.argv` and prints when its module is evaluated, so
 * each case sets argv and imports a fresh copy of the module.
 */
const G = "GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI";
const M = "MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACAAAAAAAAAAAFLKXQ";

const originalArgv = process.argv;
let log: ReturnType<typeof vi.spyOn>;
let error: ReturnType<typeof vi.spyOn>;

async function runCli(...args: string[]): Promise<void> {
  process.argv = ["node", "stellar-route", ...args];
  vi.resetModules();
  await import("../cli");
}

beforeEach(() => {
  log = vi.spyOn(console, "log").mockImplementation(() => {});
  error = vi.spyOn(console, "error").mockImplementation(() => {});
  process.exitCode = undefined;
});

afterEach(() => {
  process.argv = originalArgv;
  process.exitCode = undefined;
  vi.restoreAllMocks();
  vi.doUnmock("../routing/extract");
});

describe("stellar-route CLI", () => {
  it("prints the routing result for a G-address and MEMO_ID", async () => {
    await runCli("--dest", G, "--memo", "42", "--type", "id");
    const printed = JSON.parse(log.mock.calls[0][0] as string);
    expect(printed).toMatchObject({ destinationBaseAccount: G, routingId: "42", routingSource: "memo" });
    expect(process.exitCode).toBeUndefined();
  });

  it("serializes bigint muxed IDs as decimal strings", async () => {
    await runCli("--dest", M);
    const printed = JSON.parse(log.mock.calls[0][0] as string);
    expect(printed).toMatchObject({ destinationBaseAccount: G, routingId: "42", routingSource: "muxed" });
  });

  it("defaults the memo type to none", async () => {
    await runCli("--dest", G, "--memo", "42");
    const printed = JSON.parse(log.mock.calls[0][0] as string);
    expect(printed).toMatchObject({ routingId: null, routingSource: "none" });
  });

  it("reports an Error thrown by extractRouting and sets exit code 1", async () => {
    vi.doMock("../routing/extract", () => ({
      extractRouting: () => {
        throw new Error("boom");
      },
    }));
    await runCli("--dest", G);
    expect(error).toHaveBeenCalledWith("Error: boom");
    expect(process.exitCode).toBe(1);
  });

  it("reports a non-Error throw generically", async () => {
    vi.doMock("../routing/extract", () => ({
      extractRouting: () => {
        throw "boom";
      },
    }));
    await runCli("--dest", G);
    expect(error).toHaveBeenCalledWith("An unknown error occurred.");
    expect(process.exitCode).toBe(1);
  });
});
