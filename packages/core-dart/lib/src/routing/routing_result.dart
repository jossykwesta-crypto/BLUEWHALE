/// Routing result types with web-safe routing ID accessors.
///
/// Exposes the [SafeRoutingId] wrapper so browser contexts (Flutter Web)
/// never have to push 64-bit routing IDs through a JS `Number`.
library;

import '../address/codes.dart' show WarningCode, WarningContext, WarningSeverity;
import 'safe_routing_id.dart';

/// Identifies the mechanism used to resolve a routing ID.
enum RoutingSource {
  /// The routing ID was extracted directly from a Muxed address (M-address).
  muxed,

  /// The routing ID was extracted from the transaction's MEMO field (ID or TEXT).
  memo,

  /// No routing ID could be resolved.
  none;

  /// Returns a human-friendly description of the routing source.
  String toDisplayString() {
    switch (this) {
      case RoutingSource.muxed:
        return 'Routed via muxed address (M-address)';
      case RoutingSource.memo:
        return 'Routed via memo ID';
      case RoutingSource.none:
        return 'No routing source detected';
    }
  }
}

/// Represents a non-blocking notification emitted during routing resolution.
class RoutingWarning {
  /// The unique code identifying the warning type.
  ///
  /// Always one of the normative [WarningCode] strings, so it serializes
  /// byte-for-byte identically to core-ts and core-go.
  final String code;

  /// The severity of the warning (`info`, `warn` or `error`).
  final String severity;

  /// A descriptive message explaining the warning.
  final String message;

  /// Optional extra detail required by some codes.
  ///
  /// `INVALID_DESTINATION` requires `destinationKind: 'C'` and
  /// `UNSUPPORTED_MEMO_TYPE` requires `memoType` to be one of `hash`,
  /// `return` or `unknown`; `spec/schema.json` rejects those warnings when
  /// [context] is missing. Every other code must leave it `null`, because
  /// the schema forbids additional properties on a generic warning.
  final WarningContext? context;

  /// Creates a warning. See [WarningSeverity] for valid [severity] values.
  const RoutingWarning({
    required this.code,
    required this.severity,
    required this.message,
    this.context,
  });

  /// Emitted when a memo is present but ignored because the destination is a muxed address.
  static const memoIgnored = RoutingWarning(
    code: WarningCode.memoIgnoredForMuxed,
    severity: WarningSeverity.info,
    message: 'Memo present with M-address. Any potential routing ID in memo is ignored.',
  );

  /// Emitted when both an M-address and a routable memo were supplied; the
  /// M-address ID takes precedence.
  static const memoPresentWithMuxed = RoutingWarning(
    code: WarningCode.memoPresentWithMuxed,
    severity: WarningSeverity.warn,
    message:
        'Routing ID found in both M-address and Memo. M-address ID takes precedence.',
  );

  /// Emitted when the transaction sender is detected as a smart contract.
  static const contractSender = RoutingWarning(
    code: WarningCode.contractSenderDetected,
    severity: WarningSeverity.info,
    message: 'Contract source detected. Routing state cleared.',
  );

  /// Emitted when SEP-0029 requires a memo but no routing ID was supplied.
  static const missingRequiredMemo = RoutingWarning(
    code: WarningCode.missingRequiredMemo,
    severity: WarningSeverity.error,
    message: 'Destination account requires a memo, but no routing ID was provided.',
  );

  /// Emitted when the destination is a contract (C) address, which cannot
  /// receive classic payments.
  static const invalidDestination = RoutingWarning(
    code: WarningCode.invalidDestination,
    severity: WarningSeverity.error,
    message: 'C address is not a valid destination',
    context: WarningContext(destinationKind: 'C'),
  );

  /// Deserializes a warning written by [toJson].
  ///
  /// Accepts any [Map] with string keys, including the `Map<dynamic, dynamic>`
  /// that Hive and some platform channels hand back, not just the
  /// `Map<String, dynamic>` produced by `jsonDecode`. Missing fields fall back
  /// to empty strings and `info` severity rather than throwing, so a cached
  /// result written by an older version still loads.
  factory RoutingWarning.fromJson(Map<Object?, Object?> json) {
    return RoutingWarning(
      code: json['code'] as String? ?? '',
      severity: json['severity'] as String? ?? WarningSeverity.info,
      message: json['message'] as String? ?? '',
      context: _warningContextFromJson(json['context']),
    );
  }

  /// The parsed [WarningSeverity] of this warning, or `null` if [severity]
  /// is not a recognized level.
  String? get severityLevel => WarningSeverity.tryParse(severity);

  /// Serializes this warning to the JSON shape defined by
  /// `spec/schema.json`.
  ///
  /// The `context` key is present only when [context] is non-null, which is
  /// exactly when the schema's `oneOf` branch for [code] requires it.
  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'code': code,
      'severity': severity,
      'message': message,
      if (context != null) 'context': context!.toJson(),
    };
  }

  @override
  String toString() => '[$severity] $code: $message';

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is RoutingWarning &&
          runtimeType == other.runtimeType &&
          code == other.code &&
          severity == other.severity &&
          message == other.message &&
          context == other.context;

  @override
  int get hashCode => Object.hash(code, severity, message, context);
}

/// Details of a terminal error encountered during destination account parsing.
class DestinationError {
  /// The `ErrorCode` identifying the failure reason.
  final String code;

  /// A human-readable error message.
  final String message;

  /// Creates a destination error.
  DestinationError({required this.code, required this.message});

  /// Deserializes an error written by [toJson]. Accepts any [Map] with string
  /// keys (see [RoutingWarning.fromJson]).
  factory DestinationError.fromJson(Map<Object?, Object?> json) {
    return DestinationError(
      code: json['code'] as String? ?? '',
      message: json['message'] as String? ?? '',
    );
  }

  /// Serializes this error as `{code, message}`, the shape used under
  /// `destinationError` in [RoutingResult.toJson].
  Map<String, dynamic> toJson() => <String, dynamic>{
        'code': code,
        'message': message,
      };

  @override
  String toString() => 'DestinationError($code: $message)';

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DestinationError && code == other.code && message == other.message;

  @override
  int get hashCode => Object.hash(code, message);
}

/// Exception thrown when the routing input is fundamentally malformed.
///
/// `extractRoutingSync` and `extractRouting` throw this for an empty
/// destination or one whose prefix is not `G` or `M`. Structurally invalid
/// `G`/`M` addresses do **not** throw; they are reported through
/// [RoutingResult.destinationError].
///
/// ```dart
/// try {
///   extractRoutingSync(RoutingInput(destination: '', memoType: 'none'));
/// } on ExtractRoutingException catch (e) {
///   print(e.message);
/// }
/// ```
class ExtractRoutingException implements Exception {
  /// A human-readable description of what was wrong with the input.
  final String message;

  /// Creates an exception carrying [message].
  const ExtractRoutingException(this.message);

  @override
  String toString() => 'ExtractRoutingException: $message';
}

/// The set of parameters required to resolve a deposit route.
class RoutingInput {
  /// The destination address (G, M, or C) from the payment operation.
  final String destination;

  /// The type of memo attached to the transaction (none, id, text, hash, return).
  final String memoType;

  /// The raw value of the memo field, if any.
  final String? memoValue;

  /// The source account address of the transaction.
  final String? sourceAccount;

  /// Minimum severity (`info`, `warn` or `error`) of warnings to include in
  /// the result. Defaults to `info` (all warnings are returned).
  final String? minSeverityLevel;

  /// Creates an input. [memoValue], [sourceAccount] and [minSeverityLevel]
  /// are optional; see [RoutingResult] for how the result is produced.
  RoutingInput({
    required this.destination,
    required this.memoType,
    this.memoValue,
    this.sourceAccount,
    this.minSeverityLevel,
  });
}

/// Immutable result object returned from routing resolution.
///
/// Holds the [source] tag indicating how the route was resolved,
/// an optional numeric [id] extracted from the address or memo,
/// and any [warnings] emitted during resolution.
///
/// ## Flutter Web precision safety
///
/// [id] is a [BigInt], which is exact on every Dart target — including
/// Flutter Web, where plain `int` values compile to JS `Number`s and lose
/// precision above `2^53 - 1`. In browser contexts, serialize the routing
/// ID via [idString] or the [safeId] wrapper instead of ever converting it
/// to `int`/`num`.
final class RoutingResult {
  /// The mechanism that successfully resolved the routing ID.
  final RoutingSource source;

  /// The numeric routing identifier (e.g., User ID), or null if none was found.
  final BigInt? id;

  /// A list of non-blocking warnings encountered during resolution.
  final List<RoutingWarning> warnings;

  /// The classic 'G' address of the destination, even if an 'M' address was provided.
  final String? destinationBaseAccount;

  /// Details of the error if the destination address was unparseable.
  final DestinationError? destinationError;

  /// Creates a routing result. [warnings] is copied into an unmodifiable
  /// list.
  RoutingResult({
    required this.source,
    this.id,
    List<RoutingWarning>? warnings,
    this.destinationBaseAccount,
    this.destinationError,
  }) : warnings = List.unmodifiable(warnings ?? const []);

  /// Serializes this result to a JSON-compatible map.
  ///
  /// JSON key names are harmonized with the Go and TypeScript implementations:
  /// - [source]                → `"routingSource"`
  /// - [id]                   → `"routingId"` (decimal string, or null)
  /// - [destinationBaseAccount] → `"destinationBaseAccount"`
  /// - [warnings]             → `"warnings"`
  /// - [destinationError]     → `"destinationError"` (or absent when null)
  ///
  /// Warning objects serialize as `code`/`severity`/`message` plus a
  /// `context` object when — and only when — the code requires one
  /// (`INVALID_DESTINATION`, `UNSUPPORTED_MEMO_TYPE`), matching the
  /// `oneOf` branches in `spec/schema.json`.
  ///
  /// The routing ID is always serialized as a **decimal string** — never as a
  /// JS `Number` — so the exact uint64 value survives across isolate boundaries
  /// and platform channels on Flutter Web.
  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'destinationBaseAccount': destinationBaseAccount,
      'routingId': id?.toString(),
      'routingSource': source.name,
      'warnings': warnings.map((w) => w.toJson()).toList(),
      if (destinationError != null) 'destinationError': destinationError!.toJson(),
    };
  }

  /// Deserializes a [RoutingResult] from a JSON-compatible map.
  ///
  /// Expects the canonical cross-language JSON keys produced by [toJson]:
  /// `routingSource`, `routingId`, `destinationBaseAccount`, `warnings`,
  /// and optionally `destinationError`.
  ///
  /// The routing ID is parsed from a **decimal string** using [SafeRoutingId]
  /// to guarantee bit-exact values on Flutter Web.
  ///
  /// [json] may be any [Map] with string keys. `jsonDecode` returns
  /// `Map<String, dynamic>`, but Hive and some platform channels return
  /// `Map<dynamic, dynamic>` for the result and for every nested warning and
  /// error; both shapes load identically.
  ///
  /// ```dart
  /// await box.put('lastRouting', result.toJson());
  /// final cached = RoutingResult.fromJson(box.get('lastRouting') as Map);
  /// ```
  factory RoutingResult.fromJson(Map<Object?, Object?> json) {
    // Parse routingSource enum
    final sourceName = json['routingSource'] as String? ?? 'none';
    final source = RoutingSource.values.firstWhere(
      (e) => e.name == sourceName,
      orElse: () => RoutingSource.none,
    );

    // Parse routingId as a decimal string to avoid JS Number precision loss
    BigInt? id;
    final rawId = json['routingId'];
    if (rawId != null) {
      final idStr = rawId.toString();
      final safe = SafeRoutingId.tryParse(idStr);
      id = safe?.toBigInt;
    }

    // Parse warnings
    final rawWarnings = json['warnings'];
    final warnings = <RoutingWarning>[];
    if (rawWarnings is List) {
      for (final w in rawWarnings) {
        if (w is Map) warnings.add(RoutingWarning.fromJson(w));
      }
    }

    // Parse optional destinationError
    final rawError = json['destinationError'];
    final destinationError =
        rawError is Map ? DestinationError.fromJson(rawError) : null;

    return RoutingResult(
      source: source,
      id: id,
      destinationBaseAccount: json['destinationBaseAccount'] as String?,
      warnings: warnings,
      destinationError: destinationError,
    );
  }

  /// The routing ID as an exact, canonical decimal string, or `null` when
  /// no ID was resolved.
  ///
  /// This is the **web-safe** way to consume [id]: strings never pass
  /// through a JS `Number`, so routing IDs above
  /// `Number.MAX_SAFE_INTEGER` (`9007199254740991`) — up to the uint64
  /// ceiling `18446744073709551615` — keep full precision when running in
  /// a browser context (Flutter Web). Prefer this over `id.toInt()`-style
  /// conversions, which silently truncate there.
  String? get idString => id?.toString();

  /// The routing ID wrapped in a [SafeRoutingId], or `null` when no ID was
  /// resolved.
  ///
  /// [SafeRoutingId] is the BigInt-backed wrapper that guarantees the exact
  /// value survives parsing, comparison, and JSON serialization on all
  /// platforms, including Flutter Web.
  SafeRoutingId? get safeId => id == null ? null : SafeRoutingId.fromBigInt(id!);

  /// Returns a one-line human-readable summary of the result, such as
  /// `Muxed routing: ID 42 -> GA…`.
  String toDisplayString() {
    switch (source) {
      case RoutingSource.muxed:
        final idStr = id?.toString() ?? 'unknown';
        final base = destinationBaseAccount ?? 'unknown';
        return 'Muxed routing: ID $idStr -> $base';
      case RoutingSource.memo:
        final idStr = id?.toString() ?? 'unknown';
        return 'Memo routing: ID $idStr';
      case RoutingSource.none:
        return 'No routing detected';
    }
  }

  @override
  String toString() =>
      'RoutingResult(source: $source, id: $id, warnings: $warnings, destinationBaseAccount: $destinationBaseAccount, destinationError: $destinationError)';

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is RoutingResult &&
          source == other.source &&
          id == other.id &&
          destinationBaseAccount == other.destinationBaseAccount &&
          destinationError == other.destinationError &&
          _listEquals(warnings, other.warnings);

  @override
  int get hashCode => Object.hash(source, id, destinationBaseAccount,
      destinationError, Object.hashAll(warnings));

  static bool _listEquals(List<RoutingWarning> a, List<RoutingWarning> b) {
    if (a.length != b.length) return false;
    for (var i = 0; i < a.length; i++) {
      if (a[i] != b[i]) return false;
    }
    return true;
  }
}

/// Reads a `context` object out of a deserialized warning.
///
/// Returns `null` for a missing or empty object so a warning without context
/// round-trips through [RoutingResult.toJson] unchanged.
WarningContext? _warningContextFromJson(Object? raw) {
  if (raw is! Map) return null;
  final destinationKind = raw['destinationKind'];
  final memoType = raw['memoType'];
  if (destinationKind == null && memoType == null) return null;
  return WarningContext(
    destinationKind: destinationKind as String?,
    memoType: memoType as String?,
  );
}
