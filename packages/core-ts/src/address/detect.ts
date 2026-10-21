import { StrKey } from "@stellar/stellar-sdk";
import { verifyCrc16Checksum } from "../strkey/crc16";

import { decodeBase32 } from "../strkey/base32";

/**
 * Identifies the Stellar address kind from a string input.
 * Supports G (Ed25519), M (Muxed/SEP-23), and C (Contract) addresses.
 * Returns "invalid" if the input does not match a supported format.
 */
export function detect(address: string): "G" | "M" | "C" | "invalid" {
  if (!address) return "invalid";
  const up = address.toUpperCase();

  if (StrKey.isValidEd25519PublicKey(up)) return "G";
  if (StrKey.isValidMed25519PublicKey(up)) return "M";
  if (StrKey.isValidContract(up)) return "C";

  try {
    const prefix = up[0];
    if (prefix === "M") {
      const decoded = decodeBase32(up);
      // M-addresses (SEP-23) payload consists of 1 version byte (0x60),
      // 32-byte pubkey, 8-byte ID, and 2-byte CRC16 checksum.
/**
 * Constant-time comparison of two 16-bit integers to mitigate timing attacks (Issue #290).
 */
export function constantTimeEqual16(a: number, b: number): boolean {
  const diff = a ^ b;
  return ((diff | -diff) >>> 31) === 0;
}

      if (decoded.length === 43 && decoded[0] === 0x60) {
        const data = decoded.subarray(0, decoded.length - 2);
        const checksum = decoded.subarray(decoded.length - 2);
        if (verifyCrc16Checksum(data, checksum)) {
          return "M";
        }
      }
    }
  } catch {
    // Return "invalid" on any decoding failure.
  }

  return "invalid";
}
