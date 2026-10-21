import { detect } from "./address/detect";

/**
 * A string checked to be a valid G (Ed25519 public key) address. The brand
 * only exists at compile time; at runtime these are ordinary strings.
 *
 * Like {@link detect}, the predicates accept lowercase input, so a narrowed
 * value is valid but not necessarily in canonical uppercase form.
 */
export type PublicKeyAddress = string & { readonly __stellarAddressKind: "G" };
/** A string checked to be a valid SEP-0023 muxed (M) address. */
export type MuxedAddress = string & { readonly __stellarAddressKind: "M" };
/** A string checked to be a valid Soroban contract (C) address. */
export type ContractAddress = string & { readonly __stellarAddressKind: "C" };

/**
 * Returns `true` if `addr` is a valid G (Ed25519 public key) address.
 * Never throws: non-strings and malformed input return `false`.
 *
 * @example
 * if (isPublicKeyAddress(input)) {
 *   accept(input); // input: PublicKeyAddress
 * }
 */
export function isPublicKeyAddress(addr: unknown): addr is PublicKeyAddress {
  return typeof addr === "string" && detect(addr) === "G";
}

/**
 * Returns `true` if `addr` is a valid M (SEP-0023 muxed) address.
 * Never throws: non-strings and malformed input return `false`.
 */
export function isMuxedAddress(addr: unknown): addr is MuxedAddress {
  return typeof addr === "string" && detect(addr) === "M";
}

/**
 * Returns `true` if `addr` is a valid C (contract) address.
 * Never throws: non-strings and malformed input return `false`.
 */
export function isContractAddress(addr: unknown): addr is ContractAddress {
  return typeof addr === "string" && detect(addr) === "C";
}
