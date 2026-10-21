import 'package:bluewhale_core/bluewhale_core.dart';
import 'package:test/test.dart';

void main() {
  const baseG = 'GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI';
  const muxedAddress =
      'MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACABAAAAAAAAAAEVIG';

  group('contract sender (C... source account)', () {
    const contractSource =
        'CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQVU2HHGCYSC';

    void expectContractSenderCleared(RoutingResult result) {
      expect(result.destinationBaseAccount, isNull);
      expect(result.id, isNull);
      expect(result.source, RoutingSource.none);
      expect(result.destinationError, isNull);
      expect(result.warnings, hasLength(1));
      expect(result.warnings.first.code, WarningCode.contractSenderDetected);
      expect(result.warnings.first.code, 'CONTRACT_SENDER_DETECTED');
      expect(result.warnings.first.severity, 'info');
      expect(result.warnings.first.message,
          'Contract source detected. Routing state cleared.');
    }

    test('clears routing state for a G destination with a memo', () {
      expectContractSenderCleared(extractRoutingSync(RoutingInput(
        destination: baseG,
        memoType: 'id',
        memoValue: '100',
        sourceAccount: contractSource,
      )));
    });

    test('clears routing state for an M destination', () {
      expectContractSenderCleared(extractRoutingSync(RoutingInput(
        destination: muxedAddress,
        memoType: 'none',
        sourceAccount: contractSource,
      )));
    });

    test('does not trigger for a G source account', () {
      final result = extractRoutingSync(RoutingInput(
        destination: baseG,
        memoType: 'id',
        memoValue: '100',
        sourceAccount: baseG,
      ));
      expect(result.source, RoutingSource.memo);
      expect(result.id, BigInt.from(100));
    });
  });

  group('extractRoutingSync', () {
    test('decodes muxed routing when no external memo is present', () {
      final result = extractRoutingSync(
        RoutingInput(destination: muxedAddress, memoType: 'none'),
      );

      expect(result.destinationBaseAccount, baseG);
      expect(result.id, BigInt.parse('9007199254740993'));
      expect(result.source, RoutingSource.muxed);
      expect(result.warnings, isEmpty);
      expect(result.destinationError, isNull);
    });

    test('prefers external memo over muxed routing and emits memo-ignored warning', () {
      final result = extractRoutingSync(
        RoutingInput(
          destination: muxedAddress,
          memoType: 'id',
          memoValue: '42',
        ),
      );

      expect(result.destinationBaseAccount, baseG);
      expect(result.id, BigInt.from(42));
      expect(result.source, RoutingSource.memo);
      expect(result.destinationError, isNull);
      expect(result.warnings, hasLength(1));
      expect(result.warnings.first.code, WarningCode.memoIgnoredForMuxed);
    });

    test('keeps muxed decode valid when external memo is unroutable', () {
      final result = extractRoutingSync(
        RoutingInput(
          destination: muxedAddress,
          memoType: 'text',
          memoValue: 'not-a-routing-id',
        ),
      );

      expect(result.destinationBaseAccount, baseG);
      expect(result.id, isNull);
      expect(result.source, RoutingSource.none);
      expect(result.destinationError, isNull);
      expect(
        result.warnings.map((warning) => warning.code),
        [WarningCode.memoIgnoredForMuxed, WarningCode.memoTextUnroutable],
      );
    });

    test('preserves existing non-muxed memo routing behavior', () {
      final result = extractRoutingSync(
        RoutingInput(
          destination: baseG,
          memoType: 'id',
          memoValue: '100',
        ),
      );

      expect(result.destinationBaseAccount, baseG);
      expect(result.id, BigInt.from(100));
      expect(result.source, RoutingSource.memo);
      expect(result.warnings, isEmpty);
      expect(result.destinationError, isNull);
    });

    test('returns INVALID_DESTINATION warning for C-addresses without throwing', () {
      const cAddress = 'CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQVU2HHGCYSC';
      final result =
          extractRoutingSync(RoutingInput(destination: cAddress, memoType: 'none'));

      expect(result.source, RoutingSource.none);
      expect(result.id, isNull);
      expect(result.destinationBaseAccount, isNull);
      expect(result.destinationError, isNull);
      expect(result.warnings, [RoutingWarning.invalidDestination]);
    });

    test('returns structured destinationError for empty destination (#77)', () {
      final result = extractRoutingSync(
        RoutingInput(destination: '', memoType: 'none'),
      );

      expect(result.source, RoutingSource.none);
      expect(result.destinationBaseAccount, isNull);
      expect(result.id, isNull);
      expect(result.destinationError, isNotNull);
    });
  });

  group('extractRoutingAsync', () {
    test('decodes muxed routing when no external memo is present', () async {
      await expectLater(
        extractRoutingAsync(RoutingInput(destination: muxedAddress, memoType: 'none')),
        completion(predicate((RoutingResult result) =>
            result.destinationBaseAccount == baseG &&
            result.id == BigInt.parse('9007199254740993') &&
            result.source == RoutingSource.muxed &&
            result.warnings.isEmpty &&
            result.destinationError == null)),
      );
    });

    test('prefers external memo over muxed routing and emits memo-ignored warning', () async {
      await expectLater(
        extractRoutingAsync(RoutingInput(
          destination: muxedAddress,
          memoType: 'id',
          memoValue: '42',
        )),
        completion(predicate((RoutingResult result) =>
            result.destinationBaseAccount == baseG &&
            result.id == BigInt.from(42) &&
            result.source == RoutingSource.memo &&
            result.destinationError == null &&
            result.warnings.length == 1 &&
            result.warnings.first.code == WarningCode.memoIgnoredForMuxed)),
      );
    });

    test('keeps muxed decode valid when external memo is unroutable', () async {
      await expectLater(
        extractRoutingAsync(RoutingInput(
          destination: muxedAddress,
          memoType: 'text',
          memoValue: 'not-a-routing-id',
        )),
        completion(predicate((RoutingResult result) =>
            result.destinationBaseAccount == baseG &&
            result.id == null &&
            result.source == RoutingSource.none &&
            result.destinationError == null &&
            result.warnings.length == 2 &&
            result.warnings[0].code == WarningCode.memoIgnoredForMuxed &&
            result.warnings[1].code == WarningCode.memoTextUnroutable)),
      );
    });

    test('preserves existing non-muxed memo routing behavior', () async {
      await expectLater(
        extractRoutingAsync(RoutingInput(
          destination: baseG,
          memoType: 'id',
          memoValue: '100',
        )),
        completion(predicate((RoutingResult result) =>
            result.destinationBaseAccount == baseG &&
            result.id == BigInt.from(100) &&
            result.source == RoutingSource.memo &&
            result.warnings.isEmpty &&
            result.destinationError == null)),
      );
    });

    test('completes with INVALID_DESTINATION warning for C-addresses', () async {
      const cAddress = 'CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQVU2HHGCYSC';
      final result =
          await extractRoutingAsync(RoutingInput(destination: cAddress, memoType: 'none'));

      expect(result.source, RoutingSource.none);
      expect(result.warnings, [RoutingWarning.invalidDestination]);
    });

    test('returns structured destinationError for empty destination (#77)', () async {
      final result = await extractRoutingAsync(
        RoutingInput(destination: '', memoType: 'none'),
      );

      expect(result.source, RoutingSource.none);
      expect(result.destinationBaseAccount, isNull);
      expect(result.id, isNull);
      expect(result.destinationError, isNotNull);
    });
  });
}
