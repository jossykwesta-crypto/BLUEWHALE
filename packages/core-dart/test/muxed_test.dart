import 'package:test/test.dart';
import 'package:bluewhale_core/bluewhale_core.dart';

void main() {
  group('MuxedAddress.encode', () {
    // Shared base G address used across encoding tests
    const baseG = 'GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI';

    test('encodes id=0 (minimum uint64 boundary)', () {
      const expected =
          'MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACAAAAAAAAAAAAD672';

      final result = MuxedAddress.encode(baseG: baseG, id: BigInt.zero);

      expect(result, equals(expected));
    });

    test('encodes id above 2^53 (9007199254740993) without truncation', () {
      // ID = 2^53 + 1 = 9007199254740993, which exceeds JavaScript's
      // safe integer limit (2^53 - 1). BigInt must be used to avoid
      // floating-point precision loss when encoding to uint64.
      const expected =
          'MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACABAAAAAAAAAAEVIG';

      final result = MuxedAddress.encode(
        baseG: baseG,
        id: BigInt.parse('9007199254740993'),
      );

      expect(result, equals(expected));
    });

    test('encodes id=max uint64 (18446744073709551615) boundary', () {
      // Maximum uint64 value must round-trip without overflow on both
      // the Dart native VM and dart2js.
      final maxUint64 = BigInt.parse('18446744073709551615');

      final encoded = MuxedAddress.encode(baseG: baseG, id: maxUint64);
      final decoded = MuxedAddress.decode(encoded);

      expect(decoded.id, equals(maxUint64));
    });

    test('rejects id above max uint64 with INVALID_ROUTING_ID', () {
      final aboveMax = BigInt.parse('18446744073709551616'); // 2^64

      expect(
        () => MuxedAddress.encode(baseG: baseG, id: aboveMax),
        throwsA(
          isA<MuxedAddressException>().having(
            (e) => e.code,
            'code',
            equals('INVALID_ROUTING_ID'),
          ),
        ),
      );
    });

    test('rejects negative id with INVALID_ROUTING_ID', () {
      expect(
        () => MuxedAddress.encode(baseG: baseG, id: BigInt.from(-1)),
        throwsA(
          isA<MuxedAddressException>().having(
            (e) => e.code,
            'code',
            equals('INVALID_ROUTING_ID'),
          ),
        ),
      );
    });
  });
}
