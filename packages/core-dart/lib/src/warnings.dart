/// Warning hierarchy for routing destinations.
///
/// Modeled as a `sealed` class so that consumers can exhaustively
/// pattern-match over the possible warning variants using Dart 3
/// `switch` expressions.
sealed class DestinationWarning {
  const DestinationWarning();

  /// A short, human-readable description of the warning.
  String get message;
}

/// The destination address is not in its canonical (SEP-0023) form.
final class NonCanonicalAddress extends DestinationWarning {
  const NonCanonicalAddress({
    required this.address,
    this.canonical,
  });

  /// The address as provided by the caller.
  final String address;

  /// The canonical form of [address], when it could be computed.
  final String? canonical;

  @override
  String get message => canonical == null
      ? 'Address is not canonical: $address'
      : 'Address is not canonical: $address (expected $canonical)';
}

/// The destination requires a memo that was not supplied.
final class MissingRequiredMemo extends DestinationWarning {
  const MissingRequiredMemo({
    required this.address,
    this.memoType,
  });

  /// The destination address that requires a memo.
  final String address;

  /// The kind of memo required (e.g. `MEMO_TEXT`, `MEMO_ID`), if known.
  final String? memoType;

  @override
  String get message => memoType == null
      ? 'Missing required memo for destination: $address'
      : 'Missing required $memoType memo for destination: $address';
}

/// The destination is a contract address rather than an account.
final class ContractDestination extends DestinationWarning {
  const ContractDestination({
    required this.address,
  });

  /// The contract address that was supplied as a destination.
  final String address;

  @override
  String get message => 'Destination is a contract address: $address';
}
