/**
 * CRC16-XModem, the checksum used by Stellar strkeys (SEP-0023).
 *
 * Parameters: polynomial 0x1021, initial value 0x0000, no reflection, no
 * final XOR. The standard check value is crc16("123456789") = 0x31C3.
 *
 * This is not CRC16-CCITT-FALSE, which uses initial value 0xFFFF and gives
 * 0x29B1 for the same input; using that variant would reject every valid
 * address. A strkey stores the checksum little-endian: low byte first.
 */

/** Generator polynomial of CRC16-XModem. */
export const CRC16_XMODEM_POLYNOMIAL = 0x1021;

/** crc16Xmodem of the ASCII bytes "123456789". */
export const CRC16_XMODEM_CHECK_VALUE = 0x31c3;

const CHECK_INPUT = Uint8Array.of(0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39);

/** Thrown when the CRC16 implementation fails its runtime self-test. */
export class Crc16IntegrityError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "Crc16IntegrityError";
    Object.setPrototypeOf(this, Crc16IntegrityError.prototype);
  }
}

/** Bit-at-a-time reference implementation, used to verify the table. */
function crc16Bitwise(bytes: Uint8Array): number {
  let crc = 0;
  for (const byte of bytes) {
    crc ^= byte << 8;
    for (let i = 0; i < 8; i++) {
      crc = crc & 0x8000 ? ((crc << 1) ^ CRC16_XMODEM_POLYNOMIAL) & 0xffff : (crc << 1) & 0xffff;
    }
  }
  return crc;
}

function buildTable(): Uint16Array {
  const table = new Uint16Array(256);
  for (let i = 0; i < 256; i++) {
    table[i] = crc16Bitwise(Uint8Array.of(i));
  }
  return table;
}

// Built and verified lazily, on first use, so importing this module has no
// side effects and tree-shakes away when unused.
let table: Uint16Array | null = null;

function tableCrc(t: Uint16Array, bytes: Uint8Array): number {
  let crc = 0;
  for (let i = 0; i < bytes.length; i++) {
    crc = ((crc << 8) ^ t[((crc >> 8) ^ bytes[i]) & 0xff]) & 0xffff;
  }
  return crc;
}

/**
 * Verifies the CRC16 implementation: the lookup table matches the bitwise
 * reference for every byte, entry 1 equals the polynomial, the standard
 * check value holds, and checksum bytes come out little-endian.
 *
 * Runs automatically before the first checksum; call it directly to verify
 * the runtime at startup.
 *
 * @throws {Crc16IntegrityError} if any invariant fails
 */
export function crc16SelfTest(): void {
  const t = table ?? buildTable();

  if (t[0] !== 0 || t[1] !== CRC16_XMODEM_POLYNOMIAL) {
    throw new Crc16IntegrityError("CRC16 table does not match polynomial 0x1021");
  }
  for (let i = 0; i < 256; i++) {
    if (t[i] !== crc16Bitwise(Uint8Array.of(i))) {
      throw new Crc16IntegrityError(`CRC16 table entry ${i} is corrupt`);
    }
  }

  const check = tableCrc(t, CHECK_INPUT);
  if (check !== CRC16_XMODEM_CHECK_VALUE) {
    throw new Crc16IntegrityError(
      `CRC16 check value is 0x${check.toString(16)}, expected 0x31c3 (XModem)`,
    );
  }

  const [lo, hi] = toLittleEndian(check);
  if (lo !== 0xc3 || hi !== 0x31) {
    throw new Crc16IntegrityError("CRC16 checksum bytes are not little-endian");
  }

  table = t;
}

function toLittleEndian(crc: number): [number, number] {
  return [crc & 0xff, (crc >> 8) & 0xff];
}

/**
 * Computes the CRC16-XModem checksum of `bytes` as a number in 0..0xFFFF.
 *
 * @throws {Crc16IntegrityError} on the first call if the self-test fails
 */
export function crc16Xmodem(bytes: Uint8Array): number {
  if (table === null) crc16SelfTest();
  return tableCrc(table as Uint16Array, bytes);
}

/** The two checksum bytes a strkey appends to `payload`, low byte first. */
export function crc16ChecksumBytes(payload: Uint8Array): Uint8Array {
  return Uint8Array.from(toLittleEndian(crc16Xmodem(payload)));
}

/**
 * True if `checksum` holds the little-endian CRC16-XModem of `payload`.
 * Returns false for anything other than exactly two checksum bytes.
 */
export function verifyCrc16Checksum(payload: Uint8Array, checksum: Uint8Array): boolean {
  if (checksum.length !== 2) return false;
  const crc = crc16Xmodem(payload);
  return checksum[0] === (crc & 0xff) && checksum[1] === crc >> 8;
}
