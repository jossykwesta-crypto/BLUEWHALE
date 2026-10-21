import { RoutingInput, RoutingResult } from "./types";
import { Warning, WarningSeverity } from "../address/types";
import { parse } from "../address/parse";
import { detect } from "../address/detect";
import { AddressParseError } from "../address/errors";
import { normalizeMemoTextId } from "./memo";
import { withRoutingResultJSON } from "./result";

/**
 * Numeric weight of each warning severity. This ordering is normative and is
 * shared verbatim by core-go (`routing.SeverityWeight`) and core-dart
 * (`severityWeight`): info = 0, warn = 1, error = 2.
 */
export const SEVERITY_ORDER: Readonly<Record<WarningSeverity, number>> =
  Object.freeze({
    info: 0,
    warn: 1,
    error: 2,
  });

/**
 * Returns the numeric weight of a severity. Unknown severities weigh the same
 * as `info` (0) so that an unrecognized value never hides warnings.
 */
export function severityWeight(severity: string | null | undefined): number {
  return Object.prototype.hasOwnProperty.call(SEVERITY_ORDER, severity ?? "")
    ? SEVERITY_ORDER[severity as WarningSeverity]
    : SEVERITY_ORDER.info;
}

/**
 * Keeps only warnings whose severity weight is >= the weight of
 * `minSeverity`, preserving their original order.
 */
export function filterBySeverity<T extends { severity: string }>(
  warnings: T[],
  minSeverity: WarningSeverity | string | null | undefined
): T[] {
  const threshold = severityWeight(minSeverity);
  return warnings.filter((w) => severityWeight(w.severity) >= threshold);
}

export class ExtractRoutingError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "ExtractRoutingError";
    Object.setPrototypeOf(this, ExtractRoutingError.prototype);
  }
}

/**
 * Type guard: true when the string is structurally routable (a G or M
 * Stellar address). Only G-addresses and M-addresses are valid routing
 * targets. Never throws — non-strings and empty input return false.
 */
export function isRoutableAddress(address: unknown): address is string {
  if (typeof address !== "string") return false;
  const prefix = address.trim()[0]?.toUpperCase();
  return prefix === "G" || prefix === "M";
}

/**
 * Extracts deposit routing information from a Stellar address and memo.
 *
 * Zero-throw policy: this function never throws for any valid address string
 * input. C-addresses and addresses with unrecognized prefixes are returned as
 * a structured result with a `destinationError` field rather than raising an
 * exception. The only case where an exception is still thrown is when
 * `destination` is not a string at all (a programmer error / invalid input
 * type), which is reserved for strict type enforcement.
 *
 * Routing Policy:
 * 1. M-addresses: Routing ID is extracted from the address; any memo is ignored for routing.
 * 2. G-addresses: Routing ID is extracted from MEMO_ID or numeric MEMO_TEXT if valid.
 * 3. C-addresses: Returned as a structured result with an INVALID_DESTINATION warning.
 * 4. Unknown prefixes / malformed input: Returned with a destinationError.
 *
 * The result can be passed straight to `JSON.stringify`: it carries a
 * non-enumerable `toJSON()` that writes `routingId` as a decimal string.
 *
 * @param input - The destination address and optional memo components.
 * @returns A result containing the base account, routing ID, source, and any warnings.
 */
export function extractRouting(input: RoutingInput): RoutingResult {
  return withRoutingResultJSON(extractRoutingResult(input));
}

function extractRoutingResult(input: RoutingInput): RoutingResult {
  // Only throw for a non-string destination — this is a programmer error, not
  // an address-domain error, and cannot be meaningfully expressed as a
  // RoutingResult.
  if (typeof input.destination !== "string") {
    throw new ExtractRoutingError(
      "Invalid input: destination must be a non-empty string."
    );
  }

  const minSeverity = input.minSeverityLevel ?? "info";

  // Deposits sent by a Soroban contract (C... source) cannot be attributed to
  // a routing ID, so routing state is cleared. Mirrors core-go and core-dart.
  if (typeof input.sourceAccount === "string" && input.sourceAccount !== "") {
    if (detect(input.sourceAccount) === "C") {
      return {
        destinationBaseAccount: null,
        routingId: null,
        routingSource: "none",
        warnings: filterBySeverity(
          [
            {
              code: "CONTRACT_SENDER_DETECTED",
              severity: "info",
              message: "Contract source detected. Routing state cleared.",
            },
          ],
          minSeverity
        ),
      };
    }
  }

  let parsed;
  try {
    parsed = parse(input.destination);
  } catch (error) {
    if (error instanceof AddressParseError) {
      return {
        destinationBaseAccount: null,
        routingId: null,
        routingSource: "none",
        warnings: [],
        destinationError: {
          code: error.code,
          message: error.message,
        },
      };
    }
    throw error;
  }

  // parse() throws instead of returning kind "invalid"; this narrows the
  // type and guards against that changing. Excluded from coverage.
  /* v8 ignore next 8 */
  if (parsed.kind === "invalid") {
    return {
      destinationBaseAccount: null,
      routingId: null,
      routingSource: "none",
      warnings: [],
    };
  }

  if (parsed.kind === "C") {
    const warnings: Warning[] = [...parsed.warnings];

    warnings.push({
      code: "INVALID_DESTINATION",
      severity: "error",
      message: "C address is not a valid destination",
      context: {
        destinationKind: "C",
      },
    });

    return {
      destinationBaseAccount: null,
      routingId: null,
      routingSource: "none",
      warnings: filterBySeverity(warnings, minSeverity),
    };
  }

  if (parsed.kind === "M") {
    const warnings: Warning[] = [...parsed.warnings];

    if (
      input.memoType === "id" ||
      (input.memoType === "text" && /^\d+$/.test(input.memoValue ?? ""))
    ) {
      warnings.push({
        code: "MEMO_PRESENT_WITH_MUXED",
        severity: "warn",
        message:
          "Routing ID found in both M-address and Memo. M-address ID takes precedence.",
      });
    } else if (input.memoType !== "none") {
      warnings.push({
        code: "MEMO_IGNORED_FOR_MUXED",
        severity: "info",
        message:
          "Memo present with M-address. Any potential routing ID in memo is ignored.",
      });
    }

    return {
      destinationBaseAccount: parsed.baseG,
      routingId: parsed.muxedId,
      routingSource: "muxed",
      warnings: filterBySeverity(warnings, minSeverity),
    };
  }

  let routingId: string | bigint | null = null;
  let routingSource: "none" | "memo" = "none";
  const warnings: Warning[] = [...parsed.warnings];

  if (input.memoType === "id") {
    const rawValue = input.memoValue ?? "";
    const norm = normalizeMemoTextId(rawValue);

    if (norm.normalized) {
      // Explicit bigint parsing for MEMO_ID to avoid Number precision issues.
      try {
        const parsedMemoId = BigInt(norm.normalized);
        routingId = parsedMemoId.toString();
        routingSource = "memo";
        warnings.push(...norm.warnings);
        /* v8 ignore start */
      } catch {
        // normalizeMemoTextId only returns digits, so BigInt() cannot throw.
        routingSource = "none";
        warnings.push(...norm.warnings);
        warnings.push({
          code: "MEMO_ID_INVALID_FORMAT",
          severity: "warn",
          message: "MEMO_ID was empty, non-numeric, or exceeded uint64 max.",
        });
      }
      /* v8 ignore stop */
    } else {
      routingSource = "none";
      warnings.push(...norm.warnings);
      warnings.push({
        code: "MEMO_ID_INVALID_FORMAT",
        severity: "warn",
        message: "MEMO_ID was empty, non-numeric, or exceeded uint64 max.",
      });
    }
  } else if (input.memoType === "text" && input.memoValue) {
    const norm = normalizeMemoTextId(input.memoValue);
    if (norm.normalized) {
      routingId = norm.normalized;
      routingSource = "memo";
      warnings.push(...norm.warnings);
    } else {
      warnings.push({
        code: "MEMO_TEXT_UNROUTABLE",
        severity: "warn",
        message: "MEMO_TEXT was not a valid numeric uint64.",
      });
    }
  } else if (input.memoType === "hash" || input.memoType === "return") {
    // `UNSUPPORTED_MEMO_TYPE` is the normative code for a memo type that
    // cannot carry a routing ID, and spec/schema.json requires the matching
    // `context.memoType`. This branch previously reported
    // `MEMO_TEXT_UNROUTABLE`, a code core-go and core-dart never emit and one
    // the schema models as a plain generic warning with no context.
    warnings.push({
      code: "UNSUPPORTED_MEMO_TYPE",
      severity: "warn",
      message: `Memo type ${input.memoType} is not supported for routing.`,
      context: { memoType: input.memoType },
    });
  } else if (input.memoType !== "none") {
    warnings.push({
      code: "UNSUPPORTED_MEMO_TYPE",
      severity: "warn",
      message: `Unrecognized memo type: ${input.memoType}`,
      context: { memoType: "unknown" },
    });
  }

  return {
    destinationBaseAccount: parsed.address,
    routingId,
    routingSource,
    warnings: filterBySeverity(warnings, minSeverity),
  };
}