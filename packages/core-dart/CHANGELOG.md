# Changelog — bluewhale_core (Dart)

All notable changes to this package are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Releases are coordinated with `spec/vectors.json` `spec_version`: see
[Changelogs & release versioning](../../CONTRIBUTING.md#changelogs--release-versioning).

## [Unreleased]

### Added

- `RoutingWarning.fromJson`, plus `DestinationError.toJson` / `fromJson` and
  value equality, so every part of a `RoutingResult` round-trips on its own
  (#228).

- `WarningCode.values`, `WarningCode.isKnown` and `WarningCode.tryParse` for
  validating codes off the wire, plus the `MISSING_REQUIRED_MEMO` constant
  that `extractRouting` already emitted but `WarningCode` never declared
  (#76).
- `RoutingWarning.context` and `WarningContext.toJson`, wired through
  `RoutingResult.toJson` / `fromJson`. `INVALID_DESTINATION` and
  `UNSUPPORTED_MEMO_TYPE` require a `context` in `spec/schema.json`, so
  warnings carrying those codes could not previously be validated against the
  spec. `WarningContext` is now `const` with value equality so warnings
  round-trip.
- `WarningSeverity.tryParse`.
- A cross-language parity test (`test/warning_code_parity_test.dart`) that
  reads `spec/schema.json` and asserts `WarningCode.values` matches it.
- `test/strkey_test.dart`, covering the detailed Base32 diagnostics below
  (#72).

### Fixed

- `RoutingResult.fromJson` accepts any `Map` with string keys. Hive returns
  `Map<dynamic, dynamic>` for stored maps, and the nested `is Map<String,
  dynamic>` checks silently dropped every warning and the `destinationError`
  from such input (#228).
- `RoutingResult` equality now compares the `destinationError` message, not
  only its code.
- The library compiles again: `extractRouting` (the synchronous entry point)
  had been dropped in a merge while `extractRoutingAsync` still called it.

- **Breaking for consumers matching on warning codes:** a `hash`/`return` memo
  and an unrecognized memo type now emit `UNSUPPORTED_MEMO_TYPE` with
  `context.memoType`, matching core-go and core-ts and the spec (#76).
- `RoutingWarning.memoIgnored` carried the ad-hoc code `'memo-ignored'`, which
  is not in the spec. It is now `MEMO_IGNORED_FOR_MUXED`, and the message
  matches core-ts and core-go. `RoutingWarning.contractSender` and
  `missingRequiredMemo` reference `WarningCode` instead of hard-coded strings
  (#76).
- **Breaking:** `WarningSeverity` was declared three times — as a string
  constant class in `address/codes.dart`, again in `routing/severity.dart`,
  and as an enum in `routing/routing_result.dart`. The duplicate exports were
  a compile error and the enum made `WarningSeverity.info` an enum value where
  a `String` was required. Kept the single string-constant declaration, which
  matches core-ts's `"info" | "warn" | "error"` union and core-go's bare
  `string` (#76).
- The package did not compile: `extract.dart` called an undefined
  `_filterBySeverity`, `uri.dart` typed `minSeverityLevel` as the removed
  enum, and `bluewhale_core.dart` exported two libraries twice while never
  exporting `routing/uri.dart` (so `extractRoutingFromUriString` was
  unreachable despite `test/uri_test.dart` using it).
- `StrKeyUtil.decodeBase32` ignored unused bits in the final Base32 group, so
  one muxed payload had many valid string encodings. It now round-trips the
  decoded bytes and rejects a mismatch, as core-go and core-ts already did.
- `extractRoutingFromUriString` could never report
  `UriRoutingErrorCode.invalidEncoding` for a bad percent-escape, because
  `Uri.parse` normalizes `%zz` to `%25zz` before the query is read. Escapes are
  now validated against the raw string, so a malformed `memo` is rejected
  instead of silently half-decoded.
- `StrKeyUtil.decodeBase32` reported `Invalid Base32 character: !` with no
  position. The message now names the character, its code point and its index,
  and `FormatException.source` / `offset` are populated so tooling can point
  at the character. A character above U+FFFF is reported as a whole code point
  rather than a lone surrogate (#72).
- `dart analyze` is now clean across the package: `avoid_dynamic_calls` was
  added to `analysis_options.yaml` along with the missing community rules, and
  every finding was fixed (#71).

## [1.2.0] - 2026-09-24

Implements spec `1.2.0`.

### Added

- `RoutingInput.minSeverityLevel` plus `WarningSeverity`, `severityOrder`,
  `severityWeight` and `filterBySeverity` using the normative weights
  `info = 0`, `warn = 1`, `error = 2`. `extractRoutingSync` / `extractRouting`
  filter warnings by the threshold (default `info`).

### Fixed

- Contract senders (`C...` source account) now emit the spec warning code
  `CONTRACT_SENDER_DETECTED` instead of `contract-sender`.

## [1.1.0] - 2026-08-27

### Added

- **Flutter Web precision safety for 64-bit routing IDs.**
  - Added `SafeRoutingId`, a BigInt-backed wrapper that parses, validates,
    compares, and serializes MEMO_ID / muxed routing IDs as exact decimal
    strings — never through `int`/JS `Number` — so IDs above
    `Number.MAX_SAFE_INTEGER` (`2^53 - 1`) up to the uint64 ceiling
    (`2^64 - 1`) are never silently truncated in browser contexts.
  - Added conditional-compilation platform probe: `isWebJsRuntime` is
    `true` only when compiled to JavaScript (Flutter Web / dart2js / DDC).
  - Added web-safe accessors on `RoutingResult`: `idString` (exact canonical
    decimal string) and `safeId` (`SafeRoutingId` wrapper).
  - `SafeRoutingId.fromInt` refuses values above `Number.MAX_SAFE_INTEGER`
    on web builds instead of propagating an already-truncated JS `Number`.
  - MEMO_ID / MEMO_TEXT uint64 range validation now goes through the
    string-exact `SafeRoutingId` parser (behavior unchanged on all platforms).
  - New Flutter Web test vectors (`test/web_compat/routing_id_web_test.dart`,
    run with `dart test test/web_compat --platform chrome`) covering the
    2^53-1, 2^53, 2^53+1, 2^63-1, 2^63, and 2^64-1 boundary IDs through
    memo extraction, muxed decode, and JSON serialization.

## [1.0.0] - 2026-04-23

### Added

- Initial release of the Bluewhale for Dart and Flutter.
- Support for G, M, and C address detection and validation.
- Support for SEP-0023 Muxed Address encoding and decoding.
- Routing extraction logic for reconciling incoming payments.
