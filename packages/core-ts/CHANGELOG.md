# Changelog — @redishfish/bluewhale-core

All notable changes to this package are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Releases are coordinated with `spec/vectors.json` `spec_version`: see
[Changelogs & release versioning](../../CONTRIBUTING.md#changelogs--release-versioning).

## [Unreleased]

### Added

- `isPublicKeyAddress`, `isMuxedAddress` and `isContractAddress` predicates,
  narrowing to the branded `PublicKeyAddress`, `MuxedAddress` and
  `ContractAddress` types.
- A cross-language parity test (`src/spec/warning-codes.test.ts`) that reads
  `spec/schema.json` and asserts the `WarningCode` union matches it.
- `JSON.stringify(extractRouting(...))` no longer throws on a `bigint`
  `routingId`: results carry a non-enumerable `toJSON()` that writes the ID
  as a decimal string. Also adds `stringifyRoutingResult`,
  `toSerializableRoutingResult`, `routingIdToJSON` and the
  `SerializedRoutingResult` type (#201).

### Fixed

- **Breaking for consumers matching on warning codes:** `extractRouting`
  reported `MEMO_TEXT_UNROUTABLE` for a `hash` or `return` memo and for any
  unrecognized memo type. It now reports `UNSUPPORTED_MEMO_TYPE` with the
  `context.memoType` that `spec/schema.json` requires and that core-go and
  core-dart already emitted (#76). `MEMO_TEXT_UNROUTABLE` is now reserved for
  a `MEMO_TEXT` value that is not a numeric uint64.
- `@stellar/stellar-sdk` v17 removed its default export, so
  `import StellarSdk from "@stellar/stellar-sdk"` left `StrKey` undefined and
  12 test files failed to load. Switched to the named import.
- `extractFromURI.ts` declared `sanitizeSep7UriForLogging` twice (a bad merge),
  which is a hard esbuild error. Kept the documented implementation.
- The spec runner did not account for the legacy 50-character placeholder
  addresses in `spec/vectors.json`, which no implementation can parse.
  core-dart already substitutes canonical addresses and core-go skips those
  vectors; core-ts now substitutes identically. It also stopped asserting
  that a C-destination throws, which contradicted the vector's expected
  `INVALID_DESTINATION` result.
- `parse` indexed into inputs too short to hold a version prefix and threw a
  `TypeError` for non-strings. Inputs under 4 characters and non-strings now
  fail with a structured `AddressParseError` (`INVALID_LENGTH`), which
  `extractRouting` reports as a `destinationError` (#199).

### Changed

- CRC16 moved to `src/strkey/crc16.ts` as a table-driven CRC16-XModem with a
  lazy runtime self-test (`crc16SelfTest`, check value `0x31C3`, little-endian
  checksum bytes) and new exports `crc16Xmodem`, `crc16ChecksumBytes`,
  `verifyCrc16Checksum` and `Crc16IntegrityError`; `detect` uses it (#196).
- `package.json` now has an `exports` map with explicit `bun` and `deno`
  conditions (#200) and declares `"sideEffects": false`, with a test that
  importing only `extractRouting` stays under 5 KB gzipped (#194).

## [1.2.0] - 2026-09-24

Implements spec `1.2.0`.

### Added

- `extractRouting` returns `CONTRACT_SENDER_DETECTED` (severity `info`) and
  clears routing state when `sourceAccount` is a Soroban contract (`C...`),
  matching core-go and core-dart.
- Exported `SEVERITY_ORDER`, `severityWeight` and `filterBySeverity` with the
  normative weights `info = 0`, `warn = 1`, `error = 2`.

### Fixed

- An unknown `minSeverityLevel` no longer drops every warning; unknown
  severities now weigh as `info`, identically to Go and Dart.

## [1.0.1] - 2026-04-24

### Changed

- Synced package documentation for the registry release.

## [1.0.0] - 2026-04-23

### Added

- Initial release: G / M / C address detection, parsing and validation,
  SEP-23 muxed encode/decode, and deposit routing extraction.
