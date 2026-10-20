/**
 * RFC 1035 Domain Name Validator & IP Address Literal Target Rejector.
 *
 * Implements domain name syntax validation with strict IP address literal rejection:
 * - Total domain length must not exceed 253 ASCII characters.
 * - Labels are separated by periods (dots).
 * - Each label must be between 1 and 63 characters in length.
 * - Labels may only contain alphanumeric characters (a-z, A-Z, 0-9) and hyphens (-).
 * - Rejects IPv4 and IPv6 address literals unless explicitly enabled via allowIpLiterals: true.
 */

export interface DomainValidationOptions {
  /**
   * If true, allows a single trailing dot indicating a fully qualified domain name (FQDN).
   * Defaults to true.
   */
  allowTrailingDot?: boolean;

  /**
   * If true, requires at least two labels (domain + TLD).
   * Defaults to true.
   */
  requireTLD?: boolean;

  /**
   * If true, permits IPv4 and IPv6 address literals as targets.
   * Defaults to false (IP literals are rejected).
   */
  allowIpLiterals?: boolean;
}

export interface DomainValidationResult {
  valid: boolean;
  normalized?: string;
  error?: string;
  code?:
    | "EMPTY_DOMAIN"
    | "DOMAIN_TOO_LONG"
    | "LABEL_EMPTY"
    | "LABEL_TOO_LONG"
    | "LABEL_INVALID_START"
    | "LABEL_INVALID_END"
    | "LABEL_INVALID_CHARS"
    | "MISSING_TLD"
    | "IP_LITERAL_DISALLOWED";
}

const RFC1035_LABEL_REGEX = /^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$/;
const MAX_DOMAIN_LENGTH = 253;
const MAX_LABEL_LENGTH = 63;

// Regular expression to identify IPv4 dotted-decimal literals (0.0.0.0 to 255.255.255.255)
const IPV4_REGEX =
  /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/;

// Regular expression to identify IPv6 address literals (standard or bracketed [::1])
const IPV6_REGEX =
  /^\s*(?:\[?(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\]?|\[?(?:[0-9a-fA-F]{1,4}:){1,7}:\]?|\[?(?:[0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}\]?|\[?::(?:ffff:)?(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\]?|\[?::1\]?|\[?::\]?)\s*$/;

/**
 * Checks whether a given host string is an IPv4 or IPv6 address literal.
 */
export function isIpLiteral(target: string): boolean {
  if (typeof target !== "string" || target.trim() === "") {
    return false;
  }
  const trimmed = target.trim();
  const unbracketed =
    trimmed.startsWith("[") && trimmed.endsWith("]")
      ? trimmed.slice(1, -1)
      : trimmed;

  if (IPV4_REGEX.test(unbracketed)) {
    return true;
  }
  if (trimmed.includes(":") && IPV6_REGEX.test(trimmed)) {
    return true;
  }
  return false;
}

/**
 * Validates a domain name according to RFC 1035 / RFC 1123 specifications,
 * rejecting IP address literals unless explicitly configured.
 *
 * @param domain The domain string to validate.
 * @param options Optional configuration parameters.
 * @returns DomainValidationResult with valid boolean and diagnostic error/code.
 */
export function validateDomain(
  domain: string,
  options: DomainValidationOptions = {}
): DomainValidationResult {
  const {
    allowTrailingDot = true,
    requireTLD = true,
    allowIpLiterals = false,
  } = options;

  if (typeof domain !== "string" || domain.trim() === "") {
    return {
      valid: false,
      error: "Domain name cannot be empty",
      code: "EMPTY_DOMAIN",
    };
  }

  let cleaned = domain.trim();

  // Check for IP address literals
  if (isIpLiteral(cleaned)) {
    if (!allowIpLiterals) {
      return {
        valid: false,
        error: "IP address literals are rejected as domain targets unless explicitly configured",
        code: "IP_LITERAL_DISALLOWED",
      };
    }
    return {
      valid: true,
      normalized: cleaned.toLowerCase(),
    };
  }

  // Handle trailing dot for root label (FQDN)
  if (cleaned.endsWith(".")) {
    if (!allowTrailingDot) {
      return {
        valid: false,
        error: "Trailing dot is not allowed",
        code: "LABEL_EMPTY",
      };
    }
    cleaned = cleaned.slice(0, -1);
  }

  if (cleaned.length === 0) {
    return {
      valid: false,
      error: "Domain name cannot be empty or solely a dot",
      code: "EMPTY_DOMAIN",
    };
  }

  if (cleaned.length > MAX_DOMAIN_LENGTH) {
    return {
      valid: false,
      error: `Domain name length (${cleaned.length}) exceeds maximum of ${MAX_DOMAIN_LENGTH} characters`,
      code: "DOMAIN_TOO_LONG",
    };
  }

  const labels = cleaned.split(".");

  if (requireTLD && labels.length < 2) {
    return {
      valid: false,
      error: "Domain name must contain at least two labels (including TLD)",
      code: "MISSING_TLD",
    };
  }

  for (let i = 0; i < labels.length; i++) {
    const label = labels[i];

    if (label.length === 0) {
      return {
        valid: false,
        error: `Empty label detected at position ${i + 1} (consecutive or leading dots)`,
        code: "LABEL_EMPTY",
      };
    }

    if (label.length > MAX_LABEL_LENGTH) {
      return {
        valid: false,
        error: `Label "${label}" length (${label.length}) exceeds maximum of ${MAX_LABEL_LENGTH} characters`,
        code: "LABEL_TOO_LONG",
      };
    }

    if (label.startsWith("-")) {
      return {
        valid: false,
        error: `Label "${label}" must not start with a hyphen`,
        code: "LABEL_INVALID_START",
      };
    }

    if (label.endsWith("-")) {
      return {
        valid: false,
        error: `Label "${label}" must not end with a hyphen`,
        code: "LABEL_INVALID_END",
      };
    }

    if (!RFC1035_LABEL_REGEX.test(label)) {
      return {
        valid: false,
        error: `Label "${label}" contains invalid characters; only ASCII alphanumeric and hyphens are permitted`,
        code: "LABEL_INVALID_CHARS",
      };
    }
  }

  return {
    valid: true,
    normalized: cleaned.toLowerCase(),
  };
}

/**
 * Returns true if the given domain string is valid according to RFC 1035 and IP literal policy.
 */
export function isValidDomain(
  domain: string,
  options?: DomainValidationOptions
): boolean {
  return validateDomain(domain, options).valid;
}
