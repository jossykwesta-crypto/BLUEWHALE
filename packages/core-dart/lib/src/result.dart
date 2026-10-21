import 'warnings.dart';

/// The outcome of resolving a routing address.
///
/// [RoutingResult] is a `sealed` class so that consumers can exhaustively
/// `switch` over its variants using Dart 3 pattern matching.
sealed class RoutingResult {
  const RoutingResult();
}

/// A successfully resolved routing address.
final class RoutingSuccess extends RoutingResult {
  /// The canonical address that was resolved.
  final String address;

  /// Non-fatal warnings produced while resolving [address].
  final List<DestinationWarning> warnings;

  const RoutingSuccess(this.address, {this.warnings = const []});

  /// Whether resolution produced any warnings.
  bool get hasWarnings => warnings.isNotEmpty;
}

/// A failed routing resolution.
final class RoutingFailure extends RoutingResult {
  /// A human-readable description of why resolution failed.
  final String message;

  /// Non-fatal warnings produced before resolution failed.
  final List<DestinationWarning> warnings;

  const RoutingFailure(this.message, {this.warnings = const []});

  /// Whether resolution produced any warnings.
  bool get hasWarnings => warnings.isNotEmpty;
}
