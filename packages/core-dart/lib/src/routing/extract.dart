import '../address/codes.dart' as codes;
import '../address/codes.dart' show WarningContext;
import '../address/parse.dart';
import '../muxed/decode.dart';
import 'routing_result.dart';
import 'memo.dart';
import 'safe_routing_id.dart';
import 'severity.dart';

/// Extracts deposit routing information from a Stellar payment input.
/// Following the standard priority policy, M-address identifiers take
/// precedence over any provided memo.
///
/// Zero-throw policy: this function never throws for any valid string input.
/// C-addresses, empty destinations, and unrecognized prefixes are returned as
/// a structured [RoutingResult] with either an `INVALID_DESTINATION` warning
/// or a [DestinationError], rather than raising an [ExtractRoutingException].
/// The only remaining exception case is when [RoutingInput.destination] is
/// truly invalid at the type level (which cannot happen in typed Dart code).
///
/// Web safety: routing IDs are resolved through [SafeRoutingId], which
/// parses the canonical decimal **string** exactly and never converts
/// through `int`/JS `Number`. Combined with the `BigInt`-backed
/// [RoutingResult.id], [RoutingResult.idString], and
/// [RoutingResult.safeId] accessors, MEMO_IDs and muxed IDs up to the
/// uint64 ceiling survive Flutter Web without truncation.
///
/// Deprecated synchronous alias of [extractRouting]. For async network checks
/// (Federation, SEP-0029), use [extractRoutingAsync].
///
/// Warnings below [RoutingInput.minSeverityLevel] are filtered out using the
/// shared severity ordering (info = 0, warn = 1, error = 2).
RoutingResult extractRoutingSync(RoutingInput input) {
  return _filterResultBySeverity(_extractRoutingUnfiltered(input), input.minSeverityLevel);
}

/// Extracts deposit routing synchronously (pure string parsing).
///
/// For async network checks (Federation, SEP-0029) use [extractRoutingAsync].
RoutingResult extractRouting(RoutingInput input) => extractRoutingSync(input);

/// Re-applies [RoutingInput.minSeverityLevel] to an already-built result.
///
/// Returns [result] unchanged when the threshold is `info` (or unrecognized),
/// which is the common path and avoids re-allocating the warning list.
RoutingResult _filterResultBySeverity(RoutingResult result, String? minSeverityLevel) {
  if (severityWeight(minSeverityLevel) == 0) return result;
  return RoutingResult(
    source: result.source,
    id: result.id,
    destinationBaseAccount: result.destinationBaseAccount,
    destinationError: result.destinationError,
    warnings: filterBySeverity(result.warnings, minSeverityLevel),
  );
}

RoutingResult _extractRoutingUnfiltered(RoutingInput input) {
  final trimmed = input.destination.trim();

  // Empty destination → return structured destinationError (zero-throw policy).
  if (trimmed.isEmpty) {
    return RoutingResult(
      source: RoutingSource.none,
      warnings: [],
      destinationError: DestinationError(
        code: codes.ErrorCode.unknownPrefix,
        message: 'Invalid input: destination must be a non-empty string.',
      ),
    );
  }

  final parsed = parse(input.destination);

  if (parsed.kind == codes.AddressKind.c) {
    return RoutingResult(
      source: RoutingSource.none,
      warnings: [
        for (final w in parsed.warnings)
          RoutingWarning(code: w.code, severity: w.severity, message: w.message),
        RoutingWarning.invalidDestination,
      ],
    );
  }

  if (input.sourceAccount != null && input.sourceAccount!.isNotEmpty) {
    try {
      final source = parse(input.sourceAccount!);
      if (source.kind == codes.AddressKind.c) {
        return RoutingResult(
          source: RoutingSource.none,
          warnings: [RoutingWarning.contractSender],
        );
      }
    } catch (_) {
      // Ignore source account parsing errors for routing extraction
    }
  }

  if (parsed.kind == null) {
    return RoutingResult(
      source: RoutingSource.none,
      warnings: [],
      destinationError: parsed.error != null
          ? DestinationError(
              code: parsed.error!.code,
              message: parsed.error!.message,
            )
          : null,
    );
  }

  final warnings = <RoutingWarning>[];
  for (final w in parsed.warnings) {
    warnings.add(RoutingWarning(
      code: w.code,
      severity: w.severity,
      message: w.message,
    ));
  }

  if (parsed.kind == codes.AddressKind.m) {
    final decoded = MuxedDecoder.decodeMuxedString(parsed.address);
    final baseG = decoded.baseG;
    final muxedId = decoded.id;

    if (input.memoType == 'none') {
      return RoutingResult(
        destinationBaseAccount: baseG,
        id: muxedId,
        source: RoutingSource.muxed,
        warnings: warnings,
      );
    }

    BigInt? routingId;
    RoutingSource routingSource = RoutingSource.none;

    warnings.add(RoutingWarning.memoIgnored);

    if (input.memoType == 'id') {
      final norm = normalizeMemoId(input.memoValue ?? '');
      if (norm.normalized != null) {
        routingId = SafeRoutingId.tryParse(norm.normalized!)?.toBigInt;
        routingSource = RoutingSource.memo;
      } else {
        warnings.add(
          const RoutingWarning(
            code: codes.WarningCode.memoIdInvalidFormat,
            severity: codes.WarningSeverity.warn,
            message: 'MEMO_ID was empty, non-numeric, or exceeded uint64 max.',
          ),
        );
      }
      for (final w in norm.warnings) {
        warnings.add(RoutingWarning(
          code: w.code,
          severity: w.severity,
          message: w.message,
        ));
      }
    } else if (input.memoType == 'text' && input.memoValue != null) {
      final norm = normalizeMemoTextId(input.memoValue!);
      if (norm.normalized != null) {
        routingId = SafeRoutingId.tryParse(norm.normalized!)?.toBigInt;
        routingSource = RoutingSource.memo;
      } else {
        warnings.add(
          const RoutingWarning(
            code: codes.WarningCode.memoTextUnroutable,
            severity: codes.WarningSeverity.warn,
            message: 'MEMO_TEXT was not a valid numeric uint64.',
          ),
        );
      }
      for (final w in norm.warnings) {
        warnings.add(RoutingWarning(
          code: w.code,
          severity: w.severity,
          message: w.message,
        ));
      }
    } else if (input.memoType == 'hash' || input.memoType == 'return') {
      warnings.add(
        RoutingWarning(
          code: codes.WarningCode.unsupportedMemoType,
          severity: codes.WarningSeverity.warn,
          message: 'Memo type ${input.memoType} is not supported for routing.',
          context: WarningContext(memoType: input.memoType),
        ),
      );
    } else {
      warnings.add(
        const RoutingWarning(
          code: codes.WarningCode.unsupportedMemoType,
          severity: codes.WarningSeverity.warn,
          message: 'Unrecognized memo type: unknown',
          context: WarningContext(memoType: 'unknown'),
        ),
      );
    }

    return RoutingResult(
      destinationBaseAccount: baseG,
      id: routingId,
      source: routingSource,
      warnings: warnings,
    );
  }

  BigInt? routingId;
  RoutingSource routingSource = RoutingSource.none;

  if (input.memoType == 'id') {
    final norm = normalizeMemoId(input.memoValue ?? '');
    if (norm.normalized != null) {
      routingId = SafeRoutingId.tryParse(norm.normalized!)?.toBigInt;
      routingSource = RoutingSource.memo;
    } else {
      warnings.add(
        const RoutingWarning(
          code: codes.WarningCode.memoIdInvalidFormat,
          severity: codes.WarningSeverity.warn,
          message: 'MEMO_ID was empty, non-numeric, or exceeded uint64 max.',
        ),
      );
    }
    for (final w in norm.warnings) {
      warnings.add(RoutingWarning(
        code: w.code,
        severity: w.severity,
        message: w.message,
      ));
    }
  } else if (input.memoType == 'text' && input.memoValue != null) {
    final norm = normalizeMemoTextId(input.memoValue!);
    if (norm.normalized != null) {
      routingId = SafeRoutingId.tryParse(norm.normalized!)?.toBigInt;
      routingSource = RoutingSource.memo;
    } else {
      warnings.add(
        const RoutingWarning(
          code: codes.WarningCode.memoTextUnroutable,
          severity: codes.WarningSeverity.warn,
          message: 'MEMO_TEXT was not a valid numeric uint64.',
        ),
      );
    }
    for (final w in norm.warnings) {
      warnings.add(RoutingWarning(
        code: w.code,
        severity: w.severity,
        message: w.message,
      ));
    }
  } else if (input.memoType == 'hash' || input.memoType == 'return') {
    warnings.add(
      RoutingWarning(
        code: codes.WarningCode.unsupportedMemoType,
        severity: codes.WarningSeverity.warn,
        message: 'Memo type ${input.memoType} is not supported for routing.',
        context: WarningContext(memoType: input.memoType),
      ),
    );
  } else if (input.memoType != 'none') {
    warnings.add(
      const RoutingWarning(
        code: codes.WarningCode.unsupportedMemoType,
        severity: codes.WarningSeverity.warn,
        message: 'Unrecognized memo type: unknown',
        context: WarningContext(memoType: 'unknown'),
      ),
    );
  }

  return RoutingResult(
    destinationBaseAccount: parsed.address,
    id: routingId,
    source: routingSource,
    warnings: warnings,
  );
}

/// Looks up whether [baseAccount] requires a memo on incoming payments
/// (SEP-0029, the `config.memo_required` account data entry).
///
/// Supplied by the caller so this package stays free of network code.
typedef MemoRequirementFetcher = Future<bool> Function(String baseAccount);

/// Extracts deposit routing information, optionally checking the
/// destination's SEP-0029 memo requirement.
///
/// Runs [extractRouting] first. If [fetchMemoRequirement] is given and
/// the result has a destination account but no routing ID, the fetcher is
/// called; when it returns `true`, [RoutingWarning.missingRequiredMemo] is
/// appended. Fetch failures fail open: the synchronous result is returned
/// unchanged.
///
/// ```dart
/// final result = await extractRoutingAsync(
///   RoutingInput(
///     destination: 'GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI',
///     memoType: 'none',
///   ),
///   fetchMemoRequirement: (account) async => horizon.requiresMemo(account),
/// );
/// if (result.warnings.contains(RoutingWarning.missingRequiredMemo)) {
///   // Hold the deposit for manual review.
/// }
/// ```
Future<RoutingResult> extractRoutingAsync(
  RoutingInput input, {
  MemoRequirementFetcher? fetchMemoRequirement,
}) async {
  final result = extractRouting(input);
  if (fetchMemoRequirement == null ||
      result.destinationBaseAccount == null ||
      result.id != null ||
      result.destinationError != null) {
    return result;
  }

  try {
    if (await fetchMemoRequirement(result.destinationBaseAccount!)) {
      final withMemoWarning = RoutingResult(
        source: result.source,
        id: result.id,
        destinationBaseAccount: result.destinationBaseAccount,
        destinationError: result.destinationError,
        warnings: [...result.warnings, RoutingWarning.missingRequiredMemo],
      );
      return _filterResultBySeverity(withMemoWarning, input.minSeverityLevel);
    }
  } catch (_) {
    // Network/configuration failures must not change the synchronous result.
  }
  return result;
}

/// Looks up whether [baseAccount] requires a memo over HTTP by querying Horizon account data (SEP-0029).
/// Issue #61
Future<bool> httpFetchMemoRequirement(
  String baseAccount, {
  String horizonUrl = 'https://horizon.stellar.org',
  dynamic httpClient,
}) async {
  try {
    final sanitizedUrl = horizonUrl.replaceAll(RegExp(r'/+$'), '');
    final uri = Uri.parse('$sanitizedUrl/accounts/$baseAccount');
    if (httpClient != null) {
      final res = await httpClient.get(uri);
      final body = res?.body?.toString() ?? '';
      return body.contains('config.memo_required') || body.contains('config.requiring_memo');
    }
  } catch (_) {}
  return false;
}

