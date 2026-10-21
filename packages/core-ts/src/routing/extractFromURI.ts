/**
 * SEP-0007 URI Parser
 *
 * Parses `web+stellar:pay?...` URIs (commonly from QR code scanners)
 * and delegates to the core `extractRouting` logic to produce canonical
 * routing information.
 *
 * @see https://github.com/stellar/stellar-protocol/blob/master/ecosystem/sep-0007.md
 */

import { extractRouting } from "./extract";
import type { RoutingInput, RoutingResult } from "./types";

export interface SEP7PayParams {
  destination: string;
  amount?: string;
  assetCode?: string;
  assetIssuer?: string;
  memo?: string;
  memoType?: string;
  callback?: string;
  msg?: string;
  networkPassphrase?: string;
  originDomain?: string;
  signature?: string;
}

export type ExtractRoutingFromURIResult =
  | {
      success: true;
      routing: RoutingResult;
      rawParams: SEP7PayParams;
    }
  | {
      success: false;
      error: string;
      code: "INVALID_URI" | "UNSUPPORTED_OPERATION" | "MISSING_DESTINATION" | "INVALID_ENCODING";
    };

/**
 * Query keys whose values must never appear in error messages or logs.
 * SEP-0007 `signature`/`callback` can carry secrets; `memo`/`msg` can
 * carry personal data.
 */
const SENSITIVE_QUERY_KEYS = new Set(["signature", "callback", "memo", "msg"]);

const REDACTED = "[REDACTED]";

/**
 * Return a log-safe copy of a SEP-0007 URI with sensitive query values
 * redacted. Unknown/unparseable input is returned unchanged when it holds
 * no query string, otherwise with its values redacted.
 *
 * Use this whenever a URI (or a string derived from one) ends up in an
 * error description or log line.
 */
export function sanitizeSep7UriForLogging(uriString: string): string {
  const queryIndex = uriString.indexOf("?");
  if (queryIndex === -1) return uriString;
  const prefix = uriString.slice(0, queryIndex + 1);
  const query = uriString.slice(queryIndex + 1);
  const redacted = query
    .split("&")
    .map((pair) => {
      const eq = pair.indexOf("=");
      const key = eq === -1 ? pair : pair.slice(0, eq);
      if (SENSITIVE_QUERY_KEYS.has(key.toLowerCase())) {
        return `${key}=${REDACTED}`;
      }
      return pair;
    })
    .join("&");
  return prefix + redacted;
}

/**
 * Keys that would reach `Object.prototype` (or a constructor) if written
 * onto a plain object. They are never valid SEP-0007 parameters.
 */
const FORBIDDEN_QUERY_KEYS: ReadonlySet<string> = new Set(["__proto__", "constructor", "prototype"]);

/** Query parameters as a prototype-less record. */
export type QueryParams = Record<string, string | undefined>;

/**
 * Parse a URI query string into a prototype-less record.
 *
 * The accumulator is created with `Object.create(null)`, so no key can
 * resolve to an inherited `Object.prototype` member, and `__proto__`,
 * `constructor`, and `prototype` are dropped outright. When a key repeats,
 * the first value wins, matching `URLSearchParams#get`.
 *
 * @throws if the query string cannot be parsed by `URLSearchParams`.
 */
export function parseQueryParams(queryString: string): QueryParams {
  const params: QueryParams = Object.create(null);
  for (const [key, value] of new URLSearchParams(queryString)) {
    if (FORBIDDEN_QUERY_KEYS.has(key)) continue;
    if (Object.prototype.hasOwnProperty.call(params, key)) continue;
    params[key] = value;
  }
  return params;
}

/**
 * Map SEP-0007 memo_type values to internal KnownMemoType.
 */
function mapMemoType(sep7MemoType: string | undefined): RoutingInput["memoType"] {
  if (!sep7MemoType) return "none";

  const upper = sep7MemoType.toUpperCase();
  switch (upper) {
    case "MEMO_ID":
      return "id";
    case "MEMO_TEXT":
      return "text";
    case "MEMO_HASH":
      return "hash";
    case "MEMO_RETURN":
      return "return";
    default:
      return "none";
  }
}

/**
 * Parse a SEP-0007 URI and extract canonical routing information.
 *
 * Supported format:
 *   web+stellar:pay?destination=<G...>&memo=<value>&memo_type=<MEMO_TEXT|MEMO_ID|MEMO_HASH|MEMO_RETURN>
 *
 * The function safely decodes URL-encoded parameters, validates the scheme,
 * and passes `destination` + `memo` into `extractRouting()` for canonical
 * output (handling G-addresses, M-addresses, C-addresses, etc.).
 *
 * @param uriString - The raw URI string from a QR code scanner or deeplink
 * @returns ExtractRoutingFromURIResult
 *
 * @example
 * ```ts
 * const result = extractRoutingFromURI(
 *   "web+stellar:pay?destination=GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI&memo=123&memo_type=MEMO_ID"
 * );
 * if (result.success) {
 *   console.log(result.routing.destinationBaseAccount); // "GAYC..."
 *   console.log(result.routing.routingId);            // "123"
 * }
 * ```
 */

export function extractRoutingFromURI(uriString: string): ExtractRoutingFromURIResult {
  // 1. Validate scheme
  if (!uriString.startsWith("web+stellar:")) {
    return {
      success: false,
      error: "URI must use 'web+stellar:' scheme",
      code: "INVALID_URI",
    };
  }

  // 2. Split operation and query string
  const withoutScheme = uriString.slice("web+stellar:".length);
  const [operation, queryString] = withoutScheme.includes("?")
    ? withoutScheme.split("?", 2)
    : [withoutScheme, ""];

  // 3. Only 'pay' operation is supported for routing extraction.
  // The operation is URI-derived text, so it passes through the query
  // sanitizer before reaching the error description.
  if (operation !== "pay") {
    const sanitizedOp = sanitizeSep7UriForLogging(operation);
    return {
      success: false,
      error: `Unsupported operation: '${sanitizeSep7UriForLogging(operation)}'. Only 'pay' is supported for routing extraction.`,
      code: "UNSUPPORTED_OPERATION",
    };
  }

  // 4. Parse query parameters safely
  let params: QueryParams;
  try {
    params = parseQueryParams(queryString);
  } catch {
    return {
      success: false,
      error: "Failed to parse URI query parameters",
      code: "INVALID_ENCODING",
    };
  }

  // 5. Extract required destination
  const destination = params.destination ?? null;
  if (!destination || destination.trim() === "") {
    return {
      success: false,
      error: "Missing required 'destination' parameter",
      code: "MISSING_DESTINATION",
    };
  }

  // 6. Extract optional parameters with safe decoding
  const rawParams: SEP7PayParams = {
    destination: safelyDecode(destination.trim()),
    amount: safelyDecode(params.amount ?? null),
    assetCode: safelyDecode(params.asset_code ?? null),
    assetIssuer: safelyDecode(params.asset_issuer ?? null),
    memo: safelyDecode(params.memo ?? null),
    memoType: safelyDecode(params.memo_type ?? null),
    callback: safelyDecode(params.callback ?? null),
    msg: safelyDecode(params.msg ?? null),
    networkPassphrase: safelyDecode(params.network_passphrase ?? null),
    originDomain: safelyDecode(params.origin_domain ?? null),
    signature: safelyDecode(params.signature ?? null),
  };

  // 7. Build RoutingInput for extractRouting
  const routingInput: RoutingInput = {
    destination: rawParams.destination,
    memoType: mapMemoType(rawParams.memoType),
    memoValue: rawParams.memo ?? null,
    sourceAccount: null,
  };

  // 8. Delegate to core extractRouting logic
  const routingResult = extractRouting(routingInput);

  // 9. Return combined result
  return {
    success: true,
    routing: routingResult,
    rawParams,
  };
}

/**
 * Safely decode a URL-encoded value, returning undefined for null/empty.
 */
function safelyDecode(value: string | null): string | undefined {
  if (value === null || value === "") {
    return undefined;
  }
  try {
    return decodeURIComponent(value);
  } catch {
    return value; // Return raw if decoding fails
  }
}

/**
 * Type guard for successful URI parsing.
 */
export function isSuccessfulURIResult(
  result: ExtractRoutingFromURIResult
): result is ExtractRoutingFromURIResult & { success: true } {
  return result.success === true;
}