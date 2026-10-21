import 'package:test/test.dart';
import 'package:bluewhale_core/bluewhale_core.dart';

void main() {
  const destinationG =
      'GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI';
  const destinationM =
      'MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACAAAAAAAAAAAAD672';

  group('extractRoutingFromTx', () {
    test('returns null for null or empty input', () {
      expect(extractRoutingFromTx(null), isNull);
      expect(extractRoutingFromTx({}), isNull);
      expect(extractRoutingFromTx({'operations': []}), isNull);
    });

    test('extracts routing from PaymentOperation with memo ID', () {
      final tx = {
        'source_account': 'GBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB',
        'memo_type': 'id',
        'memo': '123456789',
        'operations': [
          {
            'type': 'payment',
            'destination': destinationG,
            'amount': '100.0',
            'asset': 'native',
          }
        ],
      };

      final result = extractRoutingFromTx(tx);
      expect(result, isNotNull);
      expect(result!.source, equals(RoutingSource.memo));
      expect(result.idString, equals('123456789'));
      expect(result.destinationBaseAccount, equals(destinationG));
    });

    test('extracts routing from PaymentOperation with M-address', () {
      final tx = {
        'operations': [
          {
            'type': 'payment',
            'destination': destinationM,
            'amount': '50.0',
          }
        ],
      };

      final result = extractRoutingFromTx(tx);
      expect(result, isNotNull);
      expect(result!.source, equals(RoutingSource.muxed));
      expect(result.idString, equals('0'));
      expect(result.destinationBaseAccount, equals(destinationG));
    });

    test(
        'extracts routing from PathPaymentStrictReceiveOperation with memo text',
        () {
      final tx = {
        'memo': {
          'type': 'text',
          'value': 'order-9988',
        },
        'operations': [
          {
            'type': 'path_payment_strict_receive',
            'destination': destinationG,
            'dest_amount': '25.0',
          }
        ],
      };

      final result = extractRoutingFromTx(tx);
      expect(result, isNotNull);
      expect(result!.destinationBaseAccount, equals(destinationG));
    });

    test('extracts routing from CreateAccountOperation', () {
      final tx = {
        'memo_type': 'id',
        'memo_value': '445566',
        'operations': [
          {
            'type': 'create_account',
            'destination': destinationG,
            'starting_balance': '10.0',
          }
        ],
      };

      final result = extractRoutingFromTx(tx);
      expect(result, isNotNull);
      expect(result!.source, equals(RoutingSource.memo));
      expect(result.idString, equals('445566'));
      expect(result.destinationBaseAccount, equals(destinationG));
    });

    test('returns null for unsupported operation type', () {
      final tx = {
        'memo_type': 'text',
        'memo': 'test',
        'operations': [
          {
            'type': 'manage_data',
            'name': 'config',
            'value': '1234',
          }
        ],
      };

      expect(extractRoutingFromTx(tx), isNull);
    });
  });
}

