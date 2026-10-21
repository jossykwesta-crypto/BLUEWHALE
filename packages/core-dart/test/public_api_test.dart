// Guards the public surface of `package:bluewhale_core/bluewhale_core.dart`.
//
// Every symbol here must be reachable through the single library entrypoint
// (no `src/` imports). If an export is dropped, this file stops compiling.
library;

import 'package:bluewhale_core/bluewhale_core.dart';
import 'package:test/test.dart';

void main() {
  const baseG = 'GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI';

  test('types are exported from the library entrypoint', () {
    const types = <Type>[
      // Addresses
      AddressKind, ErrorCode, WarningCode, WarningSeverity, Warning,
      Normalization, WarningContext, ParseResult, AddressError,
      StellarAddress,
      // Muxed accounts
      MuxedAddress, DecodedMuxedAddress, MuxedEncoder, MuxedDecoder,
      // Routing
      RoutingInput, RoutingResult, RoutingSource, RoutingWarning,
      DestinationError, SafeRoutingId, NormalizeResult,
      MemoRequirementFetcher,
      // Exceptions
      StellarAddressException, ExtractRoutingException,
    ];
    expect(types, hasLength(24));
  });

  test('top-level functions are exported from the library entrypoint', () {
    expect(detect(baseG), AddressKind.g);
    expect(validate(baseG), isTrue);
    expect(parse(baseG).kind, AddressKind.g);
    expect(normalizeMemoId('1').normalized, '1');
    expect(normalizeMemoTextId('1').normalized, '1');
    expect(isWebJsRuntime, isA<bool>());

    final input = RoutingInput(destination: baseG, memoType: 'none');
    expect(extractRoutingSync(input).source, RoutingSource.none);
    expect(extractRouting(input).source, RoutingSource.none);
    expect(extractRoutingAsync(input), completion(isA<RoutingResult>()));
  });

  test('exception types are catchable by their exported names', () {
    expect(() => StellarAddress.parse(''),
        throwsA(isA<StellarAddressException>()));
    // extractRoutingSync follows the zero-throw policy (issue #77): an empty
    // destination is reported through `destinationError`, not an exception.
    final result =
        extractRoutingSync(RoutingInput(destination: '', memoType: 'none'));
    expect(result.destinationError, isNotNull);
  });

  test('WarningSeverity matches the severities emitted by the library', () {
    expect(WarningSeverity.values, ['info', 'warn', 'error']);
    expect(RoutingWarning.memoIgnored.severity, WarningSeverity.info);
    expect(RoutingWarning.missingRequiredMemo.severity, WarningSeverity.error);

    final lower = parse(baseG.toLowerCase());
    expect(lower.warnings.single.severity, WarningSeverity.warn);
  });
}
