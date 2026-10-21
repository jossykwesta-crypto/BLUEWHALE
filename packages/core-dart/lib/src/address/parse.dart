import 'detect.dart';
import 'codes.dart';

/// Parses a [String] into a [ParseResult].
/// Normalizes the input to uppercase and returns any applicable warnings
/// or errors if the format is unrecognized.
///
/// Never throws. On failure, [ParseResult.kind] is `null` and
/// [ParseResult.error] explains why.
///
/// ```dart
/// final ok = parse('gaycuyt553c5lhve2xpw5gmejt4bxgm7ahmjwlapzp53kjo7eiqadrsi');
/// ok.kind;              // AddressKind.g
/// ok.address;           // the uppercase canonical form
/// ok.warnings.single.code; // WarningCode.nonCanonicalAddress
///
/// final bad = parse('not-an-address');
/// bad.kind;             // null
/// bad.error!.code;      // ErrorCode.unknownPrefix
/// ```
ParseResult parse(String input) {
  final kind = detect(input);
  if (kind == null) {
    return ParseResult(
      kind: null,
      address: input,
      warnings: [],
      error: AddressError(
        code: ErrorCode.unknownPrefix,
        input: input,
        message: 'Invalid address',
      ),
    );
  }

  final warnings = <Warning>[];
  final normalized = input.trim().toUpperCase();
  if (input != normalized) {
    warnings.add(
      Warning(
        code: WarningCode.nonCanonicalAddress,
        severity: WarningSeverity.warn,
        message: 'Address normalized to uppercase',
        normalization: Normalization(
          original: input,
          normalized: normalized,
        ),
      ),
    );
  }

  return ParseResult(
    kind: kind,
    address: normalized,
    warnings: warnings,
  );
}
