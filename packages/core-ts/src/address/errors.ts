export type ErrorCode =
  | "INVALID_CHECKSUM"
  | "INVALID_LENGTH"
  | "INVALID_BASE32"
  | "REJECTED_SEED_KEY"
  | "REJECTED_PREAUTH"
  | "REJECTED_HASH_X"
  | "FEDERATION_ADDRESS_NOT_SUPPORTED"
  | "MAX_INPUT_EXCEEDED"
  | "UNKNOWN_PREFIX";

/**
 * Sanitizes control characters and ANSI escape sequences in error messages (Issue #294).
 */
export function sanitizeErrorMessage(msg: string): string {
  return msg
    .replace(/\x1b\[[0-9;]*[a-zA-Z]/g, "")
    .replace(/[\x00-\x08\x0B-\x1F\x7F-\x9F]/g, "");
}

/**
 * Represents an error encountered during the parsing of a Stellar address.
 * Includes a machine-readable ErrorCode and the original input string.
 */
export class AddressParseError extends Error {
  code: ErrorCode;
  readonly input: string;

  constructor(code: ErrorCode, input: string, message: string) {
    super(sanitizeErrorMessage(message));
    this.name = "AddressParseError";
    this.code = code;
    this.input = sanitizeErrorMessage(input);
    Object.setPrototypeOf(this, AddressParseError.prototype);
  }
}
