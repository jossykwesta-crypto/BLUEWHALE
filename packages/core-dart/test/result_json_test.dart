import 'dart:convert';

import 'package:bluewhale_core/bluewhale_core.dart';
import 'package:test/test.dart';

/// Round-trip and storage-shape tests for `RoutingResult` JSON serialization.
///
/// Flutter apps cache routing results in Hive, SharedPreferences or Isar.
/// SharedPreferences stores a `jsonEncode`d string, which decodes back to
/// `Map<String, dynamic>`; Hive hands back `Map<dynamic, dynamic>` for the
/// result and every nested map. Both have to load into an equal result.
void main() {
  const baseG = 'GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI';
  const muxedAddress =
      'MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACABAAAAAAAAAAEVIG';

  /// Rebuilds [value] with `Map<dynamic, dynamic>` at every level, the way
  /// Hive returns a stored map.
  Object? hiveShaped(Object? value) {
    if (value is Map) {
      return <dynamic, dynamic>{
        for (final entry in value.entries) entry.key: hiveShaped(entry.value),
      };
    }
    if (value is List) return <dynamic>[for (final v in value) hiveShaped(v)];
    return value;
  }

  RoutingResult viaJsonString(RoutingResult result) =>
      RoutingResult.fromJson(jsonDecode(jsonEncode(result.toJson())) as Map);

  RoutingResult viaHive(RoutingResult result) =>
      RoutingResult.fromJson(hiveShaped(result.toJson()) as Map);

  final fullResult = RoutingResult(
    source: RoutingSource.memo,
    id: BigInt.parse('18446744073709551615'),
    destinationBaseAccount: baseG,
    warnings: const [
      RoutingWarning.memoPresentWithMuxed,
      RoutingWarning.invalidDestination,
      RoutingWarning(
        code: WarningCode.unsupportedMemoType,
        severity: WarningSeverity.warn,
        message: 'Memo type hash cannot carry a routing ID.',
        context: WarningContext(memoType: 'hash'),
      ),
    ],
  );

  final errorResult = RoutingResult(
    source: RoutingSource.none,
    destinationError:
        DestinationError(code: 'INVALID_CHECKSUM', message: 'invalid checksum'),
  );

  group('round-trip through a JSON string (SharedPreferences, Isar)', () {
    test('preserves every field, including warning context', () {
      final restored = viaJsonString(fullResult);
      expect(restored, fullResult);
      expect(restored.warnings[1].context?.destinationKind, 'C');
      expect(restored.warnings[2].context?.memoType, 'hash');
    });

    test('keeps the uint64 ceiling exact', () {
      expect(viaJsonString(fullResult).idString, '18446744073709551615');
    });

    test('preserves destinationError code and message', () {
      final restored = viaJsonString(errorResult);
      expect(restored, errorResult);
      expect(restored.destinationError?.message, 'invalid checksum');
    });

    test('round-trips real extractRouting output', () {
      for (final input in [
        RoutingInput(destination: muxedAddress, memoType: 'none'),
        RoutingInput(
            destination: muxedAddress, memoType: 'id', memoValue: '42'),
        RoutingInput(destination: baseG, memoType: 'text', memoValue: 'hello'),
        RoutingInput(destination: 'GBAD', memoType: 'none'),
      ]) {
        final result = extractRouting(input);
        expect(viaJsonString(result), result, reason: input.destination);
      }
    });

    test('toJson output is stable across a second round trip', () {
      final once = viaJsonString(fullResult).toJson();
      final twice = viaJsonString(RoutingResult.fromJson(once)).toJson();
      expect(jsonEncode(twice), jsonEncode(once));
    });
  });

  group('Map<dynamic, dynamic> input (Hive)', () {
    test('loads every nested warning and the destination error', () {
      expect(viaHive(fullResult), fullResult);
      expect(viaHive(errorResult), errorResult);
    });

    test('nested maps are not silently dropped', () {
      final restored = viaHive(fullResult);
      expect(restored.warnings, hasLength(3));
      expect(viaHive(errorResult).destinationError, isNotNull);
    });
  });

  group('equality', () {
    test('destinationError messages are compared, not just codes', () {
      final a = RoutingResult(
        source: RoutingSource.none,
        destinationError: DestinationError(code: 'X', message: 'one'),
      );
      final b = RoutingResult(
        source: RoutingSource.none,
        destinationError: DestinationError(code: 'X', message: 'two'),
      );
      expect(a == b, isFalse);
    });

    test('equal results have equal hash codes', () {
      expect(viaJsonString(fullResult).hashCode, fullResult.hashCode);
      expect(viaHive(errorResult).hashCode, errorResult.hashCode);
    });
  });

  group('component serialization', () {
    test('RoutingWarning.fromJson inverts toJson', () {
      for (final warning in fullResult.warnings) {
        expect(RoutingWarning.fromJson(warning.toJson()), warning);
      }
    });

    test('DestinationError.fromJson inverts toJson', () {
      final error =
          DestinationError(code: 'INVALID_LENGTH', message: 'too short');
      expect(
          error.toJson(), {'code': 'INVALID_LENGTH', 'message': 'too short'});
      expect(DestinationError.fromJson(error.toJson()), error);
    });

    test('a result without destinationError omits the key', () {
      expect(fullResult.toJson().containsKey('destinationError'), isFalse);
    });
  });
}
