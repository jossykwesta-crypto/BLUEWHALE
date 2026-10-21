import type { RoutingResult, SerializedRoutingResult } from "./types";
import { SafeRoutingId } from "./safeRoutingId";

/**
 * Returns a routing ID as a canonical decimal string, or `null`. A `bigint`
 * cannot go through `JSON.stringify` (it throws a TypeError), and a JS number
 * would lose precision above 2^53 - 1, so the wire form is always a string.
 */
export function routingIdToJSON(
  routingId: RoutingResult["routingId"] | undefined
): string | null {
  if (routingId === null || routingId === undefined) return null;
  if (routingId instanceof SafeRoutingId) return routingId.toString();
  return typeof routingId === "bigint" ? routingId.toString() : String(routingId);
}

/**
 * Returns a JSON-safe plain copy of `result` with `routingId` as a string.
 * Works for any RoutingResult, including copies made with object spread,
 * which drop the non-enumerable `toJSON` that `extractRouting` attaches.
 */
export function toSerializableRoutingResult(result: RoutingResult): SerializedRoutingResult {
  const copy = { ...result, routingId: routingIdToJSON(result.routingId) };
  // Object spread skips the non-enumerable toJSON from extractRouting; drop
  // an enumerable one too, so the copy is plain data.
  delete (copy as { toJSON?: unknown }).toJSON;
  return copy as SerializedRoutingResult;
}

/**
 * Serializes a RoutingResult to JSON without throwing on `bigint` routing IDs
 * and without precision loss: `routingId` is written as a decimal string
 * (`"18446744073709551615"`) or `null`. Parse it back with `BigInt()` or
 * `SafeRoutingId.parse()`.
 */
export function stringifyRoutingResult(result: RoutingResult, space?: string | number): string {
  return JSON.stringify(toSerializableRoutingResult(result), null, space);
}

/**
 * Gives `result` a `toJSON()` method so a plain `JSON.stringify(result)` (or
 * any logger or HTTP framework that calls it) serializes `routingId` as a
 * string instead of throwing on a `bigint`.
 *
 * The method is non-enumerable: it does not show up in `Object.keys`, object
 * spread or deep-equality assertions, so results compare exactly as before.
 */
export function withRoutingResultJSON<T extends RoutingResult>(result: T): T {
  Object.defineProperty(result, "toJSON", {
    value: function toJSON(this: RoutingResult): SerializedRoutingResult {
      return toSerializableRoutingResult(this);
    },
    enumerable: false,
    configurable: true,
    writable: true,
  });
  return result;
}
