import { StrKey } from "@stellar/stellar-sdk";
import * as crypto from "crypto";

/**
 * Constant-time byte array comparison using crypto.timingSafeEqual with portable fallback.
 */
export function constantTimeByteEqual(a: Uint8Array, b: Uint8Array): boolean {
  if (a.length !== b.length) {
    return false;
  }
  if (typeof crypto !== "undefined" && typeof crypto.timingSafeEqual === "function") {
    try {
      return crypto.timingSafeEqual(Buffer.from(a), Buffer.from(b));
    } catch {
      // Fallback to constant-time loop
    }
  }
  let diff = 0;
  for (let i = 0; i < a.length; i++) {
    diff |= a[i] ^ b[i];
  }
  return diff === 0;
}

const BASE32_CHARS = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";

/**
 * Decodes a Base32-encoded string into binary data.
 * @internal
 */
function decodeBase32(input: string): Uint8Array {
  const s = input.toUpperCase().replace(/=+$/, "");
  const byteCount = Math.floor((s.length * 5) / 8);
  const result = new Uint8Array(byteCount);
  let buffer = 0;
  let bitsLeft = 0;
  let byteIndex = 0;
  for (const ch of s) {
    const value = BASE32_CHARS.indexOf(ch);
    if (value === -1) throw new Error(`Invalid base32 character: ${ch}`);
    buffer = (buffer << 5) | value;
    bitsLeft += 5;
    if (bitsLeft >= 8) {
      if (byteIndex < byteCount) {
        result[byteIndex++] = (buffer >> (bitsLeft - 8)) & 0xff;
      }
      bitsLeft -= 8;
      buffer &= (1 << bitsLeft) - 1;
    }
  }
  return result;
}

/**
 * Computes a 16-bit CRC (CCITT-FALSE) for binary validation.
 * @internal
 */
function crc16(bytes: Uint8Array): number {
  let crc = 0;
  for (const byte of bytes) {
    crc ^= byte << 8;
    for (let i = 0; i < 8; i++) {
      if (crc & 0x8000) {
        crc = (crc << 1) ^ 0x1021;
      } else {
        crc <<= 1;
      }
      crc &= 0xffff;
    }
  }
  return crc;
}

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
      if (decoded.length === 43 && decoded[0] === 0x60) {
        const data = decoded.slice(0, decoded.length - 2);
        const expectedCrc = crc16(data);
        const expectedBytes = new Uint8Array([
          expectedCrc & 0xff,
          (expectedCrc >> 8) & 0xff,
        ]);
        const actualBytes = decoded.slice(decoded.length - 2);
        if (constantTimeByteEqual(actualBytes, expectedBytes)) {
          return "M";
        }
      }
    }
  } catch {
    // Return "invalid" on any decoding failure.
  }

  return "invalid";
}
