@TestOn('browser')
library;

import 'package:bluewhale_core/bluewhale_core.dart';
import 'package:test/test.dart';

void main() {
  group('Dart WASM Compilation & SafeRoutingId Verification (Issue #62)', () {
    test('SafeRoutingId compiles and preserves exact 64-bit bounds under WASM', () {
      final idZero = SafeRoutingId.fromString('0');
      expect(idZero.toBigInt(), BigInt.zero);
      expect(idZero.toDecimalString(), '0');

      // Beyond JS safe integer range: 2^53 + 1
      final idAboveSafe = SafeRoutingId.fromString('9007199254740993');
      expect(idAboveSafe.toDecimalString(), '9007199254740993');
      expect(idAboveSafe.toBigInt(), BigInt.parse('9007199254740993'));

      // uint64 maximum value
      final idMax = SafeRoutingId.fromString('18446744073709551615');
      expect(idMax.toDecimalString(), '18446744073709551615');
      expect(idMax.toBigInt(), BigInt.parse('18446744073709551615'));
    });

    test('Muxed address parsing and decoding compiles cleanly under Dart WASM', () {
      const baseG = 'GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI';
      final parsed = parse(baseG);
      expect(parsed.kind, AddressKind.g);
      expect(parsed.address, baseG);
    });
  });
}
