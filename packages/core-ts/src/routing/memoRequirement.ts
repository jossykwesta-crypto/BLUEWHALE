import type { MemoRequirementFetcher, RoutingInput, RoutingResult } from "./types";
import { extractRouting } from "./extract";

const DEFAULT_HORIZON_URL = "https://horizon.stellar.org";

/**
 * Validates domain format conforming to RFC 1035 rules (Issue #293).
 */
export function isValidRfc1035Domain(domain: string): boolean {
  if (!domain || domain.length > 253) return false;
  const labels = domain.split(".");
  if (labels.length < 2) return false;

  const labelRegex = /^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$/;
  return labels.every((label) => labelRegex.test(label));
}

export type MemoRequirement = {
  required: boolean;
};

export interface StellarTomlResolverOptions {
  allowHttp?: boolean;
  pinnedCertFingerprints?: string[];
}

function decodeDataValue(value: unknown): boolean {
  if (value === "1" || value === 1 || value === true) return true;
  if (typeof value !== "string") return false;
  try {
    return atob(value) === "1";
  } catch {
    return false;
  }
}

/** Fetches the SEP-0029 memo requirement from a Horizon account response. */
export async function fetchMemoRequirement(
  baseAccount: string,
  fetchImpl: typeof fetch = fetch,
  horizonUrl = DEFAULT_HORIZON_URL,
  options?: StellarTomlResolverOptions
): Promise<boolean> {
  // Enforce strict HTTPS (Issue #292)
  if (!options?.allowHttp && !horizonUrl.startsWith("https://")) {
    throw new Error("SEP-0029 requires strict HTTPS protocol for resolver endpoint");
  }
  const response = await fetchImpl(
    `${horizonUrl.replace(/\/$/, "")}/accounts/${encodeURIComponent(baseAccount)}`
  );
  if (!response.ok) return false;

  const account = (await response.json()) as { data?: Record<string, unknown> };
  return decodeDataValue(account.data?.["config.memo_required"]) ||
    decodeDataValue(account.data?.["config.requiring_memo"]);
}

/**
 * Performs normal routing extraction and optionally checks SEP-0029 account data.
 * A failed network lookup is treated as unknown and does not block parsing.
 */
export async function extractRoutingAsync(
  input: RoutingInput,
  requirementFetcher: MemoRequirementFetcher = fetchMemoRequirement
): Promise<RoutingResult> {
  const result = extractRouting(input);
  if (!result.destinationBaseAccount || result.routingId !== null || result.destinationError) {
    return result;
  }

  try {
    if (await requirementFetcher(result.destinationBaseAccount)) {
      result.warnings = [
        ...result.warnings,
        {
          code: "MISSING_REQUIRED_MEMO",
          severity: "error",
          message: "Destination account requires a memo, but no routing ID was provided.",
        },
      ];
    }
  } catch {
    // Network/configuration failures must not change the synchronous result.
  }
  return result;
}
