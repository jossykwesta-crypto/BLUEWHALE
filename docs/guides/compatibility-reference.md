# Compatibility Reference

This guide documents cross-runtime compatibility for the address parsing
implementations shipped in this repository, including behavioral parity and
runtime resource characteristics.

## Runtime Support Matrix

| Runtime | Package | Status |
| --- | --- | --- |
| Node.js / TypeScript | `@stellar/address-core` | Supported |
| Go | `github.com/stellar/address-core-go` | Supported |
| Dart VM | `address_core` | Supported |

All three implementations are validated against the shared vectors in
`spec/vectors.json` and the schema in `spec/schema.json`.

## Memory Benchmark: 10,000 Consecutive Address Parses

To help high-scale backend operators choose an appropriate runtime, the
repository ships `scripts/benchmark-all.sh`, which executes 10,000 consecutive
address parses in each language and reports peak RSS memory and allocation
counts.

### Running the benchmark

```sh
./scripts/benchmark-all.sh
```

The script requires a Node.js toolchain, a Go toolchain, and a Dart SDK on
`PATH`. Each runtime is measured in a fresh process so that peak RSS reflects
only the parse workload.

### Methodology

- Each runtime parses the same set of valid address vectors 10,000 times.
- Peak RSS is sampled from the operating system for the benchmark process.
- Allocation counts are reported where the runtime exposes them
  (Node.js via `process.memoryUsage()`, Go via `runtime.MemStats`).
- Results are indicative, not normative: absolute numbers vary by machine,
  OS, and toolchain version.

### Comparative Results

| Runtime | Peak RSS (10k parses) | Allocations |
| --- | --- | --- |
| Node.js / TypeScript | see `scripts/benchmark-all.sh` output | see output |
| Go | see `scripts/benchmark-all.sh` output | see output |
| Dart VM | see `scripts/benchmark-all.sh` output | see output |

Run the script on your target hardware to obtain numbers representative of
your deployment environment. The table above is intentionally populated from
the script output rather than hard-coded, since memory profiles are
environment-dependent.

## Behavioral Parity

Beyond resource usage, the runtimes must agree on parse results. The shared
vector suite covers:

- Valid SEP-0023 / SEP-0029 address encodings.
- Negative vectors that must be rejected.
- SEP-0007 memo vectors.

Any divergence between runtimes is treated as a compatibility bug and should
be reported with the offending vector.
