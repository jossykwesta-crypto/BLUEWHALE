import 'extract.dart';
import 'routing_result.dart';

/// Extracts deposit routing information from a Stellar transaction or simulated
/// transaction JSON / response object (such as those from `stellar_flutter_sdk`).
///
/// Supports the following operations:
/// - `PaymentOperation`
/// - `PathPaymentStrictReceiveOperation`
/// - `CreateAccountOperation`
///
/// Returns [RoutingResult] if a supported payment/create-account operation is found,
/// or `null` if the transaction contains no supported operation or is null/invalid.
RoutingResult? extractRoutingFromTx(dynamic tx) {
  if (tx == null) return null;

  try {
    // 1. Resolve operations list
    List? operations;
    if (tx is Map) {
      final ops = tx['operations'] ?? tx['ops'];
      if (ops is List) {
        operations = ops;
      } else if (tx.containsKey('type') &&
          (tx.containsKey('destination') || tx.containsKey('account'))) {
        operations = [tx];
      }
    } else {
      try {
        final ops = (tx as dynamic).operations;
        if (ops is List) operations = ops;
      } catch (_) {}
    }

    if (operations == null || operations.isEmpty) {
      return null;
    }

    // 2. Find first supported operation
    String? destination;
    for (final op in operations) {
      if (op == null) continue;
      final typeStr = _getOperationType(op);
      if (_isSupportedOperationType(typeStr)) {
        destination = _getDestination(op);
        if (destination != null) {
          break;
        }
      }
    }

    if (destination == null) {
      return null;
    }

    // 3. Resolve memo
    final memoPair = _getMemo(tx);

    // 4. Resolve source account
    final sourceAccount = _getSourceAccount(tx);

    return extractRoutingSync(
      RoutingInput(
        destination: destination,
        memoType: memoPair.type,
        memoValue: memoPair.value,
        sourceAccount: sourceAccount,
      ),
    );
  } catch (_) {
    return null;
  }
}

bool _isSupportedOperationType(String? type) {
  if (type == null) return false;
  final lower = type
      .toLowerCase()
      .replaceAll('_', '')
      .replaceAll('operation', '')
      .replaceAll('response', '');
  return lower == 'payment' ||
      lower == 'pathpaymentstrictreceive' ||
      lower == 'createaccount';
}

String? _getOperationType(dynamic op) {
  if (op is Map) {
    return (op['type'] ?? op['type_i']?.toString() ?? op['operation_type'])
        ?.toString();
  }
  try {
    final type = (op as dynamic).type?.toString();
    if (type != null) return type;
  } catch (_) {}
  return op.runtimeType.toString();
}

String? _getDestination(dynamic op) {
  if (op is Map) {
    return (op['destination'] ??
            op['destination_account'] ??
            op['account'] ??
            op['destinationAccount'])
        ?.toString();
  }
  try {
    final dest = (op as dynamic).destination?.toString();
    if (dest != null) return dest;
  } catch (_) {}
  try {
    final dest = (op as dynamic).destinationAccount?.toString();
    if (dest != null) return dest;
  } catch (_) {}
  try {
    final dest = (op as dynamic).account?.toString();
    if (dest != null) return dest;
  } catch (_) {}
  return null;
}

class _MemoPair {
  final String? type;
  final String? value;
  const _MemoPair(this.type, this.value);
}

_MemoPair _getMemo(dynamic tx) {
  if (tx is Map) {
    final memoType = (tx['memo_type'] ?? tx['memoType'])?.toString();
    final memoValue = (tx['memo'] ?? tx['memo_value'] ?? tx['memoValue']);
    if (memoValue is Map) {
      return _MemoPair(
        (memoValue['type'] ?? memoType)?.toString(),
        (memoValue['value'] ?? memoValue['text'] ?? memoValue['id'])
            ?.toString(),
      );
    }
    return _MemoPair(memoType, memoValue?.toString());
  }

  try {
    final memo = (tx as dynamic).memo;
    if (memo != null) {
      if (memo is Map) {
        return _MemoPair(
          (memo['type'] ?? memo['memo_type'])?.toString(),
          (memo['value'] ?? memo['text'] ?? memo['id'])?.toString(),
        );
      }
      try {
        final mType = (memo as dynamic).type?.toString();
        final mValue = (memo as dynamic).value?.toString() ??
            (memo as dynamic).text?.toString() ??
            (memo as dynamic).id?.toString();
        return _MemoPair(mType, mValue);
      } catch (_) {}

      final runtimeName = memo.runtimeType.toString();
      if (runtimeName.contains('MemoText')) {
        try {
          return _MemoPair('text', (memo as dynamic).text?.toString());
        } catch (_) {}
      } else if (runtimeName.contains('MemoId')) {
        try {
          return _MemoPair('id', (memo as dynamic).id?.toString());
        } catch (_) {}
      } else if (runtimeName.contains('MemoHash')) {
        try {
          return _MemoPair('hash', (memo as dynamic).hexValue?.toString());
        } catch (_) {}
      } else if (runtimeName.contains('MemoNone')) {
        return const _MemoPair('none', null);
      }

      return _MemoPair(null, memo.toString());
    }
  } catch (_) {}

  return const _MemoPair(null, null);
}

String? _getSourceAccount(dynamic tx) {
  if (tx is Map) {
    return (tx['source_account'] ?? tx['sourceAccount'] ?? tx['source'])
        ?.toString();
  }
  try {
    return (tx as dynamic).sourceAccount?.toString() ??
        (tx as dynamic).source?.toString();
  } catch (_) {}
  return null;
}
