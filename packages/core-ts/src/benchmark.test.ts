import { describe, expect, it } from "vitest";
import { decodeBase32 } from "./strkey/base32";

/**
 * Throughput benchmark for the base32 decoder used by `detect()`.
 *
 * Output parity with the previous `indexOf`-based decoder is always checked.
 * Timing is noisy on shared CI runners, so the speedup is only asserted when
 * `BENCHMARK=1` is set:
 *
 *   BENCHMARK=1 pnpm --filter @redishfish/bluewhale-core test -- src/benchmark.test.ts
 */

const ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";

/** The decoder as it was before the lookup table, kept as the baseline. */
function decodeBase32IndexOf(input: string): Uint8Array {
  const s = input.toUpperCase().replace(/=+$/, "");
  const byteCount = Math.floor((s.length * 5) / 8);
  const result = new Uint8Array(byteCount);
  let buffer = 0;
  let bitsLeft = 0;
  let byteIndex = 0;
  for (const ch of s) {
    const value = ALPHABET.indexOf(ch);
    if (value === -1) throw new Error(`Invalid base32 character: ${ch}`);
    buffer = (buffer << 5) | value;
    bitsLeft += 5;
    if (bitsLeft >= 8) {
      if (byteIndex < byteCount) {
        result[byteIndex++] = (buffer >> (bitsLeft - 8)) & 0xff;
      }
      bitsLeft -= 8;
      buffer &= (1 << bitsLeft) - 1;
    }
  }
  return result;
}

const M_ADDRESS =
  "MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACAAAAAAAAAAAFLKXQ";
const G_ADDRESS = "GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI";

/** Deterministic pseudo-random base32 strings of every length 0..80. */
function corpus(): string[] {
  let seed = 0x5eed;
  const next = () => (seed = (seed * 1103515245 + 12345) & 0x7fffffff);
  const out: string[] = [M_ADDRESS, G_ADDRESS, M_ADDRESS.toLowerCase(), "MA======", ""];
  for (let len = 0; len <= 80; len++) {
    let s = "";
    for (let i = 0; i < len; i++) s += ALPHABET[next() % 32];
    out.push(s, s.toLowerCase());
  }
  return out;
}

/** Operations per second of `fn(input)`, best of several rounds. */
function opsPerSecond(fn: (s: string) => Uint8Array, input: string, iterations: number): number {
  let sink = 0;
  for (let i = 0; i < iterations / 10; i++) sink ^= fn(input)[0] ?? 0; // warm-up
  let best = 0;
  for (let round = 0; round < 5; round++) {
    const start = performance.now();
    for (let i = 0; i < iterations; i++) sink ^= fn(input)[0] ?? 0;
    const elapsed = performance.now() - start;
    best = Math.max(best, iterations / (elapsed / 1000));
  }
  expect(Number.isFinite(sink)).toBe(true);
  return best;
}

describe("base32 decoder", () => {
  it("matches the indexOf decoder byte for byte", () => {
    for (const input of corpus()) {
      expect(decodeBase32(input)).toEqual(decodeBase32IndexOf(input));
    }
  });

  it.each(["MA1", "MA!", "MA=A", "M A", "MAé", "MA😀"])("rejects %j like the indexOf decoder", (input) => {
    expect(() => decodeBase32IndexOf(input)).toThrow(/Invalid base32 character/);
    expect(() => decodeBase32(input)).toThrow(/Invalid base32 character/);
  });

  const benchmark = process.env.BENCHMARK === "1" ? it : it.skip;

  benchmark("decodes M-addresses at least 30% faster than the indexOf decoder", () => {
    const iterations = 200_000;
    const before = opsPerSecond(decodeBase32IndexOf, M_ADDRESS, iterations);
    const after = opsPerSecond(decodeBase32, M_ADDRESS, iterations);
    const speedup = after / before - 1;
    console.log(
      `base32 decode: indexOf ${Math.round(before).toLocaleString()} ops/s, ` +
        `lookup ${Math.round(after).toLocaleString()} ops/s (+${(speedup * 100).toFixed(0)}%)`
    );
    expect(speedup).toBeGreaterThan(0.3);
  });
});
