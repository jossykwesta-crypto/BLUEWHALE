import 'package:core_dart/src/warnings.dart';
import 'package:test/test.dart';

void main() {
  group('DestinationWarning sealed hierarchy', () {
    test('NonCanonicalAddress is a DestinationWarning', () {
      const DestinationWarning warning = NonCanonicalAddress(
        address: 'GABC',
        canonical: 'GDEF',
      );
      expect(warning, isA<NonCanonicalAddress>());
    });

    test('MissingRequiredMemo is a DestinationWarning', () {
      const DestinationWarning warning = MissingRequiredMemo(
        address: 'GABC',
        memoType: 'MEMO_ID',
      );
      expect(warning, isA<MissingRequiredMemo>());
    });

    test('ContractDestination is a DestinationWarning', () {
      const DestinationWarning warning = ContractDestination(
        address: 'CABC',
      );
      expect(warning, isA<ContractDestination>());
    });

    test('exhaustive switch over warning types', () {
      String describe(DestinationWarning warning) {
        return switch (warning) {
          NonCanonicalAddress(:final address, :final canonical) =>
            'non-canonical: $address -> $canonical',
          MissingRequiredMemo(:final address, :final memoType) =>
            'missing memo ($memoType): $address',
          ContractDestination(:final address) => 'contract: $address',
        };
      }

      expect(
        describe(const NonCanonicalAddress(address: 'GABC', canonical: 'GDEF')),
        'non-canonical: GABC -> GDEF',
      );
      expect(
        describe(const MissingRequiredMemo(address: 'GABC', memoType: 'MEMO_ID')),
        'missing memo (MEMO_ID): GABC',
      );
      expect(
        describe(const ContractDestination(address: 'CABC')),
        'contract: CABC',
      );
    });

    test('pattern matching with guard clauses', () {
      bool isBlocking(DestinationWarning warning) {
        return switch (warning) {
          MissingRequiredMemo() => true,
          ContractDestination() => true,
          NonCanonicalAddress() => false,
        };
      }

      expect(isBlocking(const MissingRequiredMemo(address: 'GABC', memoType: 'MEMO_ID')), isTrue);
      expect(isBlocking(const ContractDestination(address: 'CABC')), isTrue);
      expect(isBlocking(const NonCanonicalAddress(address: 'GABC', canonical: 'GDEF')), isFalse);
    });
  });
}
