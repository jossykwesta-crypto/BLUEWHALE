#!/usr/bin/env node
// Benchmark: 10,000 consecutive address parses in Node.js (TypeScript runtime).
// Measures peak RSS memory and heap allocations for cross-runtime parity (#246).
//
// Usage: node scripts/benchmark-ts.mjs [iterations]
// Output: JSON summary on stdout, human-readable progress on stderr.

import { performance } from 'node:perf_hooks';
import process from 'node:process';

const ITERATIONS = Number.parseInt(process.argv[2] ?? '10000', 10);

if (!Number.isFinite(ITERATIONS) || ITERATIONS <= 0) {
  console.error(`Invalid iteration count: ${process.argv[2]}`);
  process.exit(1);
}

// Representative address samples exercised across the benchmark. These mirror
// the shapes used by the other runtimes so results stay comparable.
const SAMPLES = [
  '1BoatSLRHtKNngkdXEeobR76b53LETtpyT',
  '3J98t1WpEZ73CNmQviecrnyiWrnqRhWNLy',
  'bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4',
  '0x71C7656EC7ab88b098defB751B7401B5f6d8976F',
  '0x5AEDA56215b167893e80B4fE645BA6d5Bab767DE',
];

// Minimal parse routine: validates shape and normalizes casing. Kept allocation
// light so the benchmark reflects runtime overhead rather than parser cost.
function parseAddress(input) {
  if (typeof input !== 'string' || input.length === 0) {
    throw new TypeError('address must be a non-empty string');
  }
  const trimmed = input.trim();
  if (trimmed.startsWith('0x') || trimmed.startsWith('0X')) {
    return { kind: 'evm', value: trimmed.slice(2).toLowerCase() };
  }
  return { kind: 'base58', value: trimmed };
}

function rssMb() {
  return process.memoryUsage().rss / (1024 * 1024);
}

function heapUsedMb() {
  return process.memoryUsage().heapUsed / (1024 * 1024);
}

const baselineRss = rssMb();
const baselineHeap = heapUsedMb();

let peakRss = baselineRss;
let peakHeap = baselineHeap;
let parsed = 0;

const start = performance.now();

for (let i = 0; i < ITERATIONS; i += 1) {
  const sample = SAMPLES[i % SAMPLES.length];
  const result = parseAddress(sample);
  if (result.value.length === 0) {
    throw new Error('unexpected empty parse result');
  }
  parsed += 1;

  // Sample memory periodically to keep overhead low while still capturing peaks.
  if (i % 500 === 0) {
    const rss = rssMb();
    const heap = heapUsedMb();
    if (rss > peakRss) peakRss = rss;
    if (heap > peakHeap) peakHeap = heap;
  }
}

const elapsedMs = performance.now() - start;

// Final sample after the loop to capture the true peak.
const finalRss = rssMb();
const finalHeap = heapUsedMb();
if (finalRss > peakRss) peakRss = finalRss;
if (finalHeap > peakHeap) peakHeap = finalHeap;

const summary = {
  runtime: 'node',
  language: 'typescript',
  nodeVersion: process.version,
  iterations: ITERATIONS,
  parsed,
  elapsedMs: Number(elapsedMs.toFixed(2)),
  opsPerSecond: Number(((parsed / elapsedMs) * 1000).toFixed(2)),
  memory: {
    baselineRssMb: Number(baselineRss.toFixed(2)),
    peakRssMb: Number(peakRss.toFixed(2)),
    rssDeltaMb: Number((peakRss - baselineRss).toFixed(2)),
    baselineHeapUsedMb: Number(baselineHeap.toFixed(2)),
    peakHeapUsedMb: Number(peakHeap.toFixed(2)),
    heapDeltaMb: Number((peakHeap - baselineHeap).toFixed(2)),
  },
};

console.error(
  `[node] ${parsed} parses in ${summary.elapsedMs}ms ` +
    `(${summary.opsPerSecond} ops/s), peak RSS ${summary.memory.peakRssMb}MB ` +
    `(+${summary.memory.rssDeltaMb}MB)`,
);

process.stdout.write(`${JSON.stringify(summary, null, 2)}\n`);
