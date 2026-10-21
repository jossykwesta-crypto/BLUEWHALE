import { describe, it, expect } from "vitest";
import {
  CRC16_XMODEM_CHECK_VALUE,
  CRC16_XMODEM_POLYNOMIAL,
  crc16ChecksumBytes,
  crc16SelfTest,
  crc16Xmodem,
  verifyCrc16Checksum,
} from "../strkey/crc16";
import { detect } from "../address/detect";

const BASE32 = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";

function hex(s: string): Uint8Array {
  const out = new Uint8Array(s.length / 2);
  for (let i = 0; i < out.length; i++) out[i] = parseInt(s.slice(i * 2, i * 2 + 2), 16);
  return out;
}

function base32Decode(s: string): Uint8Array {
  const out: number[] = [];
  let buffer = 0;
  let bits = 0;
  for (const ch of s) {
    buffer = (buffer << 5) | BASE32.indexOf(ch);
    bits += 5;
    if (bits >= 8) {
      out.push((buffer >> (bits - 8)) & 0xff);
      bits -= 8;
      buffer &= (1 << bits) - 1;
    }
  }
  return Uint8Array.from(out);
}

function base32Encode(bytes: Uint8Array): string {
  let out = "";
  let buffer = 0;
  let bits = 0;
  for (const b of bytes) {
    buffer = (buffer << 8) | b;
    bits += 8;
    while (bits >= 5) {
      out += BASE32[(buffer >> (bits - 5)) & 31];
      bits -= 5;
    }
    buffer &= (1 << bits) - 1;
  }
  if (bits > 0) out += BASE32[(buffer << (5 - bits)) & 31];
  return out;
}

/** CRC16 with a configurable initial value, to contrast XModem with CCITT-FALSE. */
function crc16WithInit(bytes: Uint8Array, init: number): number {
  let crc = init;
  for (const byte of bytes) {
    crc ^= byte << 8;
    for (let i = 0; i < 8; i++) {
      crc = crc & 0x8000 ? ((crc << 1) ^ 0x1021) & 0xffff : (crc << 1) & 0xffff;
    }
  }
  return crc;
}

/**
 * 50 known CRC16-XModem pairs (hex input, checksum), computed independently
 * with Python's binascii.crc_hqx(data, 0).
 */
const KNOWN_PAIRS: Array<[string, number]> = [
  ["", 0x0000],
  ["61", 0x7c87],
  ["41", 0x58e5],
  ["30", 0x3653],
  ["31", 0x2672],
  ["39", 0xa77a],
  ["6162", 0x74ff],
  ["616263", 0x9dd6],
  ["61626364", 0xa836],
  ["313233", 0x9752],
  ["31323334", 0xd789],
  ["3132333435", 0x546c],
  ["313233343536", 0x20e4],
  ["31323334353637", 0x86d6],
  ["3132333435363738", 0x9015],
  ["313233343536373839", 0x31c3],
  ["5374656c6c6172", 0xd587],
  ["7374656c6c6172", 0xbb71],
  ["426c75657768616c65", 0x9c32],
  ["626c75657768616c65", 0x3bce],
  ["54686520717569636b2062726f776e20666f78206a756d7073206f76657220746865206c617a7920646f67", 0xf0c8],
  ["68656c6c6f", 0xc362],
  ["68656c6c6f20776f726c64", 0x3be4],
  ["48656c6c6f2c20576f726c6421", 0x4fd6],
  ["00", 0x0000],
  ["0000", 0x0000],
  ["ff", 0x1ef0],
  ["ffff", 0x1d0f],
  ["4741594355595435353343354c4856453258505735474d454a54344258474d3741484d4a574c41505a5035334b4a4f374549514144525349", 0x1e3d],
  ["6d656d6f", 0xf0f0],
  ["4d454d4f5f4944", 0x597c],
  ["7765622b7374656c6c61723a706179", 0x7744],
  ["30313233343536373839616263646566", 0xe8a5],
  ["4142434445464748494a4b4c4d4e4f505152535455565758595a323334353637", 0xaedd],
  ["7a7a7a7a7a7a7a7a7a7a", 0xccb2],
  ["20202020202020202020", 0x25df],
  ["0a", 0xa14a],
  ["0d0a", 0xd716],
  ["09", 0x9129],
  ["7e21402324255e262a2829", 0x3a13],
  ["00", 0x0000],
  ["0001", 0x1021],
  ["00010203", 0x6131],
  ["0001020304050607", 0x26b3],
  ["000102030405060708090a0b0c0d0e0f", 0x513d],
  ["000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f", 0xd2ff],
  ["000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f", 0x2bf5],
  ["000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f404142434445464748494a4b4c4d4e4f505152535455565758595a5b5c5d5e5f606162636465666768696a6b6c6d6e6f707172737475767778797a7b7c7d7e7f", 0xe80a],
  ["000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f404142434445464748494a4b4c4d4e4f505152535455565758595a5b5c5d5e5f606162636465666768696a6b6c6d6e6f707172737475767778797a7b7c7d7e7f808182838485868788898a8b8c8d8e8f909192939495969798999a9b9c9d9e9fa0a1a2a3a4a5a6a7a8a9aaabacadaeafb0b1b2b3b4b5b6b7b8b9babbbcbdbebfc0c1c2c3c4c5c6c7c8c9cacbcccdcecfd0d1d2d3d4d5d6d7d8d9dadbdcdddedfe0e1e2e3e4e5e6e7e8e9eaebecedeeeff0f1f2f3f4f5f6f7f8f9fafbfcfdfe", 0x0530],
  ["000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f404142434445464748494a4b4c4d4e4f505152535455565758595a5b5c5d5e5f606162636465666768696a6b6c6d6e6f707172737475767778797a7b7c7d7e7f808182838485868788898a8b8c8d8e8f909192939495969798999a9b9c9d9e9fa0a1a2a3a4a5a6a7a8a9aaabacadaeafb0b1b2b3b4b5b6b7b8b9babbbcbdbebfc0c1c2c3c4c5c6c7c8c9cacbcccdcecfd0d1d2d3d4d5d6d7d8d9dadbdcdddedfe0e1e2e3e4e5e6e7e8e9eaebecedeeeff0f1f2f3f4f5f6f7f8f9fafbfcfdfeff", 0x7e55],
];

const encoder = new TextEncoder();

describe("crc16Xmodem", () => {
  it("passes its self-test", () => {
    expect(() => crc16SelfTest()).not.toThrow();
  });

  it("matches the standard check value for \"123456789\"", () => {
    expect(crc16Xmodem(encoder.encode("123456789"))).toBe(CRC16_XMODEM_CHECK_VALUE);
    expect(CRC16_XMODEM_CHECK_VALUE).toBe(0x31c3);
    expect(CRC16_XMODEM_POLYNOMIAL).toBe(0x1021);
  });

  it("is XModem, not CCITT-FALSE (0x29B1 uses init 0xFFFF)", () => {
    const input = encoder.encode("123456789");
    expect(crc16WithInit(input, 0xffff)).toBe(0x29b1);
    expect(crc16WithInit(input, 0x0000)).toBe(crc16Xmodem(input));
    expect(crc16Xmodem(input)).not.toBe(0x29b1);
  });

  it("has 50 known pairs", () => {
    expect(KNOWN_PAIRS).toHaveLength(50);
  });

  it.each(KNOWN_PAIRS)("crc16(%s) = %i", (input, expected) => {
    expect(crc16Xmodem(hex(input))).toBe(expected);
  });

  it("agrees with the bitwise reference on every byte value", () => {
    for (let i = 0; i < 256; i++) {
      const bytes = Uint8Array.of(i, 255 - i, i ^ 0x5a);
      expect(crc16Xmodem(bytes)).toBe(crc16WithInit(bytes, 0));
    }
  });

  it("always returns a 16-bit value", () => {
    for (const [input] of KNOWN_PAIRS) {
      const crc = crc16Xmodem(hex(input));
      expect(Number.isInteger(crc) && crc >= 0 && crc <= 0xffff).toBe(true);
    }
  });
});

describe("strkey checksum byte order", () => {
  const addresses = [
    "GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI",
    "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
    "MBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OAAAAAAAAAAAPOGVY",
    "CAAQCAIBAEAQCAIBAEAQCAIBAEAQCAIBAEAQCAIBAEAQCAIBAEAQC526",
  ];

  it.each(addresses)("stores the checksum of %s little-endian", (address) => {
    const raw = base32Decode(address);
    const payload = raw.subarray(0, raw.length - 2);
    const stored = raw.subarray(raw.length - 2);

    expect(Array.from(crc16ChecksumBytes(payload))).toEqual(Array.from(stored));
    expect(verifyCrc16Checksum(payload, stored)).toBe(true);
    // Big-endian order must not verify.
    expect(verifyCrc16Checksum(payload, Uint8Array.of(stored[1], stored[0]))).toBe(
      stored[0] === stored[1],
    );
  });

  it("emits low byte first", () => {
    expect(Array.from(crc16ChecksumBytes(encoder.encode("123456789")))).toEqual([0xc3, 0x31]);
  });

  it("rejects checksums that are not exactly two bytes", () => {
    const payload = encoder.encode("123456789");
    expect(verifyCrc16Checksum(payload, Uint8Array.of(0xc3))).toBe(false);
    expect(verifyCrc16Checksum(payload, Uint8Array.of(0xc3, 0x31, 0x00))).toBe(false);
    expect(verifyCrc16Checksum(payload, new Uint8Array(0))).toBe(false);
  });

  it("round-trips: re-encoding a payload with its checksum gives a valid address", () => {
    const raw = base32Decode(addresses[2]);
    const payload = raw.slice(0, raw.length - 2);
    const rebuilt = new Uint8Array(raw.length);
    rebuilt.set(payload);
    rebuilt.set(crc16ChecksumBytes(payload), payload.length);

    expect(base32Encode(rebuilt)).toBe(addresses[2]);
    expect(detect(base32Encode(rebuilt))).toBe("M");
  });
});
