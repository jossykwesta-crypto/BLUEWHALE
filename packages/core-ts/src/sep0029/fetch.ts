import { extractRouting } from "./extract";
import type { ExtractRoutingAsyncOptions, RoutingResult } from "../types";

/**
 * Fetches and parses a SEP-0029 `stellar.toml` file from the given domain,
 * resolving the routing information asynchronously.
 *
 * @param domain - The domain to fetch the `stellar.toml` from.
 * @param options - Optional configuration, including an `AbortSignal` to cancel the request.
 * @returns A promise resolving to the parsed routing result.
 */
export async function extractRoutingAsync(
  domain: string,
  options: ExtractRoutingAsyncOptions = {},
): Promise<RoutingResult> {
  const { signal } = options;
  const url = `https://${domain}/.well-known/stellar.toml`;

  let response: Response;
  try {
    response = await fetch(url, { signal });
  } catch (error) {
    if (isAbortError(error)) {
      throw new Error(
        `SEP-0029 stellar.toml resolution for "${domain}" was aborted`,
      );
    }
    throw error;
  }

  if (!response.ok) {
    throw new Error(
      `Failed to fetch stellar.toml from "${domain}": ${response.status} ${response.statusText}`,
    );
  }

  const toml = await response.text();
  return extractRouting(toml);
}

/**
 * Determines whether the given error represents an aborted fetch request.
 */
function isAbortError(error: unknown): boolean {
  return (
    typeof error === "object" &&
    error !== null &&
    "name" in error &&
    (error as { name?: unknown }).name === "AbortError"
  );
}
