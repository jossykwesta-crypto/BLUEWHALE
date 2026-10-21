/// Safe handling of Stellar G, M, and C addresses and deposit routing.
///
/// This is the only library consumers should import; everything under
/// `lib/src/` is an implementation detail and may change without notice.
///
/// ```dart
/// import 'package:bluewhale_core/bluewhale_core.dart';
///
/// final result = extractRoutingSync(RoutingInput(
///   destination: 'GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI',
///   memoType: 'id',
///   memoValue: '9007199254740993',
/// ));
/// print(result.source);   // RoutingSource.memo
/// print(result.idString); // 9007199254740993 (exact, even on Flutter Web)
/// ```
///
/// ## Public API
///
/// - **Addresses:** [detect], [validate], [parse], [StellarAddress],
///   [AddressKind], [ParseResult], [AddressError].
/// - **Muxed accounts:** [MuxedAddress], [DecodedMuxedAddress], and the
///   low-level [MuxedEncoder] / [MuxedDecoder].
/// - **Routing:** [extractRouting], [extractRoutingAsync], [extractRoutingSync], [RoutingInput],
///   [RoutingResult], [RoutingSource], [RoutingWarning], [DestinationError],
///   [SafeRoutingId], [normalizeMemoId], [normalizeMemoTextId],
///   [extractRoutingFromUriString], [UriRoutingResult].
/// - **Codes:** [ErrorCode], [WarningCode], [WarningSeverity], [Warning].
/// - **Exceptions:** [StellarAddressException], [ExtractRoutingException].
/// - **Platform:** [isWebJsRuntime].
library;

// Addresses
export 'src/address/codes.dart';
export 'src/address/detect.dart';
export 'src/address/parse.dart';
export 'src/address/stellar_address.dart';
export 'src/address/validate.dart';

// Muxed accounts
export 'src/muxed/decode.dart';
export 'src/muxed/decoded_muxed_address.dart';
export 'src/muxed/encode.dart';
export 'src/muxed/muxed_address.dart';

// Routing. `uint64Max` and `digitsOnly` are internal helpers of the memo
// normalizers; use `SafeRoutingId.uint64Max` instead.
export 'src/routing/extract.dart';
export 'src/routing/memo.dart' hide uint64Max, digitsOnly;
export 'src/routing/routing_result.dart';
export 'src/routing/safe_routing_id.dart';
export 'src/routing/severity.dart';
export 'src/routing/tx.dart';
export 'src/routing/uri.dart';
export 'src/exceptions.dart';

// Platform
export 'src/util/web_platform.dart';
