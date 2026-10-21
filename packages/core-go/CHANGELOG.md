# Changelog — core-go

All notable changes to this package are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Releases are coordinated with `spec/vectors.json` `spec_version`: see
[Changelogs & release versioning](../../CONTRIBUTING.md#changelogs--release-versioning).

Go module versions are published as git tags of the form
`packages/core-go/vX.Y.Z`.

## [Unreleased]

### Added

- Sentinel errors in the root `corego` package (`errors.go`):
  `ErrInvalidChecksum`, `ErrInvalidLength` and `ErrUnsupportedScheme`. Each
  matches, under `errors.Is`, any error from `address`, `muxed` or `routing`
  carrying the same code, including through `fmt.Errorf("%w", err)` wrapping.
- `ErrorCode()` on `address.RoutingError` and `*muxed.AddressError`.

- `address.AllWarningCodes`, `WarningCode.String` and `WarningCode.IsKnown`,
  plus a cross-language parity test (`spec/warning_codes_test.go`) that reads
  `spec/schema.json` and asserts this package's code list matches it.
- `xdr.ExtractRoutingFromXDR`: routing results straight from a base64
  `TransactionEnvelope` (v0, v1 or fee bump) for create account, payment and
  path payment operations, including muxed destinations. It uses its own XDR
  reader, so it adds no dependency (#203).
- `address.DecodeStrKeyInto` and `muxed.DecodeMuxedRaw`: decode into a
  caller-supplied or pooled buffer with zero heap allocations (#202).

### Changed

- `address.RoutingError.Is` also matches a `*RoutingError` target.
- `*muxed.AddressError` now implements `Is` and matches by code. Previously
  `errors.Is(err, muxed.ErrInvalidLengthError)` was false for errors built by
  `DecodeMuxed`, because they are new pointers.
- `routing.ErrInvalidScheme` is now a comparable custom error value (same
  message), so `err == routing.ErrInvalidScheme` keeps working and it also
  matches `corego.ErrUnsupportedScheme`.

- **Breaking:** removed the unused legacy multi-chain fields
  `RoutingInput.SourceAddress`, `RoutingInput.TargetChains` and
  `RoutingResult.ResolvedAddresses`. Nothing in this module, its tests or
  `examples/go-payment-listener` ever read or wrote them, and they are
  meaningless for a Stellar-only router (#42).
- Documented every field of `RoutingInput` and `RoutingResult`, and promoted
  `Destination`, `MemoType`, `MemoValue` and `SourceAccount` from
  "backward-compatible" to primary. `Success` and `ErrorMessage` are now
  documented as belonging to the SEP-0007 URI entry point, which is the only
  place that sets them.
- `RoutingInput`'s JSON is now uniformly `omitempty`; the removed fields were
  the only ones without it, so an empty `RoutingInput` marshals to `{}`
  instead of `{"sourceAddress":"","targetChains":null}` (#42).

### Fixed

- `address.DecodeStrKey` had been left unterminated by a bad merge, with
  `decodeStrKey`, `EncodeStrKey` and `kindForVersionByte` nested inside it,
  so the package did not compile.
- `ExtractRoutingWithMemoRequirement` was declared twice, the context-aware
  copy was mis-typed as `MemoRequirementFetcher`, and it called an undefined
  `ExtractRoutingWithContext`. Split into `ContextMemoRequirementFetcher` +
  `ExtractRoutingWithContext` (which forwards the context and short-circuits
  on an already-cancelled one), with the no-context entry point kept as the
  deprecated wrapper.

### Performance

- `muxed.DecodeMuxed` decodes into a `sync.Pool` buffer that is cleared
  before reuse, leaving the returned G-address string as its only
  allocation (#202).
- `address.CalculateCRC16` processes eight bytes per loop iteration; results
  are unchanged (#207).

## [1.2.0] - 2026-09-24

Implements spec `1.2.0`. First versioned release of the Go module; earlier
changes are available in the git history.

### Added

- `RoutingInput.MinSeverityLevel` and `routing.SeverityWeight` /
  `routing.FilterBySeverity` (plus `SeverityInfo`, `SeverityWarn`,
  `SeverityError`) using the normative weights `info = 0`, `warn = 1`,
  `error = 2`. `ExtractRouting` filters warnings by the threshold (default
  `info`).
- The spec vector runner now executes `extract_routing` vectors that carry a
  `sourceAccount` (contract-sender policy).
