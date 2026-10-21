/**
 * Unit tests for domain validation:
 * Validates RFC 1035 compliance and rejection of malicious and malformed domain inputs.
 */

import { describe, it, expect } from "vitest";
import {
  validateDomain,
  isValidDomain,
  isIpLiteral,
} from "../domain/validator";

describe("Domain Validator – Malformed Inputs", () => {
  it("rejects empty, null, or undefined domain inputs", () => {
    expect(isValidDomain("")).toBe(false);
    expect(isValidDomain("   ")).toBe(false);
    // @ts-expect-error testing invalid type input
    expect(isValidDomain(null)).toBe(false);
    // @ts-expect-error testing invalid type input
    expect(isValidDomain(undefined)).toBe(false);

    const result = validateDomain("");
    expect(result.valid).toBe(false);
    expect(result.code).toBe("EMPTY_DOMAIN");
  });

  it("rejects domains exceeding 253 characters", () => {
    // 60 + 1 + 60 + 1 + 60 + 1 + 60 + 1 + 10 = 254 chars
    const overlyLong = `${"a".repeat(60)}.${"b".repeat(60)}.${"c".repeat(60)}.${"d".repeat(60)}.${"e".repeat(10)}`;
    expect(overlyLong.length).toBeGreaterThan(253);

    const result = validateDomain(overlyLong);
    expect(result.valid).toBe(false);
    expect(result.code).toBe("DOMAIN_TOO_LONG");
  });

  it("rejects labels exceeding 63 characters", () => {
    const longLabel = `${"a".repeat(64)}.stellar.org`;
    const result = validateDomain(longLabel);
    expect(result.valid).toBe(false);
    expect(result.code).toBe("LABEL_TOO_LONG");
  });

  it("rejects empty labels and consecutive dots", () => {
    expect(isValidDomain("stellar..org")).toBe(false);
    expect(isValidDomain(".stellar.org")).toBe(false);
    expect(isValidDomain("stellar.org..")).toBe(false);

    const result = validateDomain("stellar..org");
    expect(result.valid).toBe(false);
    expect(result.code).toBe("LABEL_EMPTY");
  });

  it("rejects labels with leading or trailing hyphens", () => {
    const leadingHyphen = "-stellar.org";
    const resLeading = validateDomain(leadingHyphen);
    expect(resLeading.valid).toBe(false);
    expect(resLeading.code).toBe("LABEL_INVALID_START");

    const trailingHyphen = "stellar-.org";
    const resTrailing = validateDomain(trailingHyphen);
    expect(resTrailing.valid).toBe(false);
    expect(resTrailing.code).toBe("LABEL_INVALID_END");
  });

  it("rejects domains missing a TLD when requireTLD is true", () => {
    const result = validateDomain("localhost", { requireTLD: true });
    expect(result.valid).toBe(false);
    expect(result.code).toBe("MISSING_TLD");

    // But allows it when requireTLD is false
    const allowed = validateDomain("localhost", { requireTLD: false });
    expect(allowed.valid).toBe(true);
  });

  it("rejects invalid symbols and whitespace in labels", () => {
    const invalidInputs = [
      "stellar organ.org",
      "stellar_anchor.org",
      "stellar@anchor.org",
      "stellar!corp.com",
      "stellar#anchor.org",
      "stellar$wallet.com",
      "stellar%20.org",
    ];

    for (const input of invalidInputs) {
      expect(isValidDomain(input)).toBe(false);
      const res = validateDomain(input);
      expect(res.valid).toBe(false);
    }
  });
});

describe("Domain Validator – Malicious Inputs", () => {
  it("rejects path traversal attempts", () => {
    const maliciousPaths = [
      "../../etc/passwd",
      "stellar.org/../../secret",
      "..\\..\\windows\\system32",
      "example.com/api/v1",
      "example.com?query=param",
      "example.com#anchor",
    ];

    for (const path of maliciousPaths) {
      expect(isValidDomain(path)).toBe(false);
    }
  });

  it("rejects CRLF and header injection attempts", () => {
    const crlfInputs = [
      "stellar.org\r\nX-Injected: true",
      "stellar.org\nSet-Cookie: stolen=true",
      "stellar.org\r\n",
    ];

    for (const input of crlfInputs) {
      expect(isValidDomain(input)).toBe(false);
    }
  });

  it("rejects null byte and control character injection", () => {
    const nullByteInputs = [
      "stellar.org\0.evil.com",
      "evil.com\x00.stellar.org",
      "stellar\x1b[31m.org",
    ];

    for (const input of nullByteInputs) {
      expect(isValidDomain(input)).toBe(false);
    }
  });

  it("rejects XSS and script injection payloads", () => {
    const xssInputs = [
      "<script>alert(1)</script>.org",
      "javascript:alert(1)",
      "data:text/html,<html>",
      "stellar.org<svg/onload=alert(1)>",
    ];

    for (const input of xssInputs) {
      expect(isValidDomain(input)).toBe(false);
    }
  });

  it("rejects IPv4 address literals unless explicitly enabled", () => {
    const ipv4Targets = [
      "127.0.0.1",
      "192.168.1.1",
      "10.0.0.1",
      "169.254.169.254",
      "0.0.0.0",
      "255.255.255.255",
    ];

    for (const ip of ipv4Targets) {
      expect(isIpLiteral(ip)).toBe(true);

      const resDefault = validateDomain(ip);
      expect(resDefault.valid).toBe(false);
      expect(resDefault.code).toBe("IP_LITERAL_DISALLOWED");

      const resExplicit = validateDomain(ip, { allowIpLiterals: true });
      expect(resExplicit.valid).toBe(true);
    }
  });

  it("rejects IPv6 address literals unless explicitly enabled", () => {
    const ipv6Targets = [
      "::1",
      "[::1]",
      "2001:db8::1",
      "fe80::1",
      "[2001:db8::1]",
    ];

    for (const ip of ipv6Targets) {
      expect(isIpLiteral(ip)).toBe(true);

      const resDefault = validateDomain(ip);
      expect(resDefault.valid).toBe(false);
      expect(resDefault.code).toBe("IP_LITERAL_DISALLOWED");

      const resExplicit = validateDomain(ip, { allowIpLiterals: true });
      expect(resExplicit.valid).toBe(true);
    }
  });

  it("is resilient to catastrophic backtracking (ReDoS)", () => {
    const start = Date.now();
    const maliciousReDoS = `a${"-".repeat(200)}b.stellar.org`;
    const res = validateDomain(maliciousReDoS);
    const duration = Date.now() - start;

    expect(duration).toBeLessThan(100); // Must resolve in < 100ms
    expect(res.valid).toBe(false);
  });
});

describe("Domain Validator – Valid Domains", () => {
  it("accepts valid domain names and normalizes casing", () => {
    const validDomains = [
      "stellar.org",
      "ANCHOR.STELLAR.ORG",
      "sub-domain.example-domain.co.uk",
      "test-123.stellar.org",
      "a.co",
    ];

    for (const domain of validDomains) {
      const res = validateDomain(domain);
      expect(res.valid).toBe(true);
      expect(res.normalized).toBe(domain.toLowerCase());
    }
  });

  it("handles valid FQDNs with trailing dots", () => {
    const res = validateDomain("stellar.org.");
    expect(res.valid).toBe(true);
    expect(res.normalized).toBe("stellar.org");
  });
});
