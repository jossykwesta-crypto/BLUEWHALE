package routing

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/REDISHFISH/BLUEWHALE/packages/core-go/address"
)

// normalizeUnsupportedMemoType canonicalizes a memo type string by lower-casing it
// and stripping underscores and hyphens, then maps it to a known unsupported type.
// Uses allocation-free byte comparisons: exact-match fast paths for canonical types, then a folded compare.
func normalizeUnsupportedMemoType(memoType string) string {
	switch memoType {
	case "hash", "return":
		return memoType
	}

	switch memoType {
	case "none", "id", "text":
		return ""
	}

	switch {
	case foldedMemoTypeEquals(memoType, "memohash"):
		return "hash"
	case foldedMemoTypeEquals(memoType, "memoreturn"):
		return "return"
	default:
		return ""
	}
}

// ExtractRouting identifies the deposit routing destination and identifier from a Stellar
// payment input. It implements the standard priority policy where M-address identifiers
// take precedence over any provided memo. Returns a RoutingResult with the decoded
// state and applicable warnings, filtered by input.MinSeverityLevel.
func ExtractRouting(input RoutingInput) RoutingResult {
	result := extractRouting(input)
	result.Warnings = FilterBySeverity(result.Warnings, input.MinSeverityLevel)
	return result
}

func extractRouting(input RoutingInput) RoutingResult {
	if input.SourceAccount != "" {
		source, err := address.Parse(input.SourceAccount)
		if err == nil && source.Kind == address.KindC {
			return RoutingResult{
				RoutingSource: "none",
				Warnings: []address.Warning{{
					Code:     address.WarnContractSenderDetected,
					Severity: "info",
					Message:  "Contract source detected. Routing state cleared.",
				}},
			}
		}
	}

	parsed, err := address.Parse(input.Destination)
	if err != nil {
		return RoutingResult{
			RoutingSource: "none",
			Warnings:      []address.Warning{},
			DestinationError: &DestinationError{
				Code:    address.ErrUnknownPrefix,
				Message: err.Error(),
			},
		}
	}

	if parsed.Kind == address.KindC {
		return RoutingResult{
			RoutingSource: "none",
			Warnings: []address.Warning{{
				Code:     address.WarnInvalidDestination,
				Severity: "error",
				Message:  "C address is not a valid destination",
				Context: &address.WarningContext{
					DestinationKind: "C",
				},
			}},
		}
	}

	if parsed.Kind == address.KindM {
		// address.Parse already decoded the muxed payload; reuse it rather
		// than decoding the M-address a second time.
		baseG, id := parsed.BaseG, parsed.MuxedID

		// Pre-allocate with capacity for existing warnings plus at most one more.
		warnings := make([]address.Warning, 0, len(parsed.Warnings)+1)
		warnings = append(warnings, parsed.Warnings...)
		memoValue := stringValue(input.MemoValue)

		// isAllDigits replaces the regex match to avoid heap allocation.
		if input.MemoType == "id" || (input.MemoType == "text" && isAllDigits(memoValue)) {
			warnings = append(warnings, address.Warning{
				Code:     address.WarnMemoPresentWithMuxed,
				Severity: "warn",
				Message:  "Routing ID found in both M-address and Memo. M-address ID takes precedence.",
			})
		} else if input.MemoType != "none" {
			warnings = append(warnings, address.Warning{
				Code:     address.WarnMemoIgnoredForMuxed,
				Severity: "info",
				Message:  "Memo present with M-address. Any potential routing ID in memo is ignored.",
			})
		}

		return RoutingResult{
			DestinationBaseAccount: baseG,
			RoutingID:              NewRoutingID(strconv.FormatUint(id, 10)),
			RoutingSource:          "muxed",
			Warnings:               warnings,
		}
	}

	var routingID *RoutingID
	routingSource := "none"
	// Pre-allocate with capacity for existing address warnings plus at most two memo warnings.
	warnings := make([]address.Warning, 0, len(parsed.Warnings)+2)
	warnings = append(warnings, parsed.Warnings...)
	memoValue := stringValue(input.MemoValue)

	if input.MemoType == "id" {
		norm := NormalizeMemoTextID(memoValue)
		if norm.Normalized != "" {
			routingID = NewRoutingID(norm.Normalized)
			routingSource = "memo"
		}
		warnings = append(warnings, norm.Warnings...)

		if norm.Normalized == "" {
			warnings = append(warnings, address.Warning{
				Code:     address.WarnMemoIDInvalidFormat,
				Severity: "warn",
				Message:  "MEMO_ID was empty, non-numeric, or exceeded uint64 max.",
			})
		}
	} else if input.MemoType == "text" && memoValue != "" {
		norm := NormalizeMemoTextID(memoValue)
		if norm.Normalized != "" {
			routingID = NewRoutingID(norm.Normalized)
			routingSource = "memo"
			warnings = append(warnings, norm.Warnings...)
		} else {
			warnings = append(warnings, address.Warning{
				Code:     address.WarnMemoTextUnroutable,
				Severity: "warn",
				Message:  "MEMO_TEXT was not a valid numeric uint64.",
			})
		}
	} else if unsupportedMemoType := normalizeUnsupportedMemoType(input.MemoType); unsupportedMemoType != "" {
		// Use pre-computed string literals for known memo types to avoid string concatenation.
		var msg string
		switch unsupportedMemoType {
		case "hash":
			msg = "Memo type hash is not supported for routing."
		case "return":
			msg = "Memo type return is not supported for routing."
		}
		warnings = append(warnings, address.Warning{
			Code:     address.WarnUnsupportedMemoType,
			Severity: "warn",
			Message:  msg,
			Context: &address.WarningContext{
				MemoType: unsupportedMemoType,
			},
		})
	} else if input.MemoType != "none" {
		// Use strings.Builder to avoid the two intermediate allocations from "prefix" + var concatenation.
		var sb strings.Builder
		sb.Grow(len("Unrecognized memo type: ") + len(input.MemoType))
		sb.WriteString("Unrecognized memo type: ")
		sb.WriteString(input.MemoType)
		warnings = append(warnings, address.Warning{
			Code:     address.WarnUnsupportedMemoType,
			Severity: "warn",
			Message:  sb.String(),
			Context: &address.WarningContext{
				MemoType: "unknown",
			},
		})
	}

	return RoutingResult{
		DestinationBaseAccount: parsed.Raw,
		RoutingID:              routingID,
		RoutingSource:          routingSource,
		Warnings:               warnings,
	}
}

// MemoRequirementFetcher retrieves whether a destination account requires a
// routing memo. Implementations can use Horizon, an indexer, or a cached source.
//
// Deprecated: Use [ContextMemoRequirementFetcher] and
// [ExtractRoutingWithContext] instead, which accept a [context.Context] for
// cancellation and deadline propagation.
type MemoRequirementFetcher func(baseAccount string) (bool, error)

// ContextMemoRequirementFetcher is the context-aware form of
// [MemoRequirementFetcher]. Implementations should honour ctx cancellation so
// an in-flight Horizon or indexer lookup does not outlive the caller's
// deadline.
type ContextMemoRequirementFetcher func(ctx context.Context, baseAccount string) (bool, error)

// ExtractRoutingWithContext performs normal routing extraction and
// optionally adds the SEP-0029 error when a classic destination requires a
// memo but no routing ID was supplied. Fetch failures fail open so callers
// retain the result of the synchronous parser.
//
// ctx is forwarded to fetch, allowing a cancelled or expired context to abort
// the lookup; a fetcher that returns an error contributes no warning.
func ExtractRoutingWithContext(ctx context.Context, input RoutingInput, fetch ContextMemoRequirementFetcher) RoutingResult {
	result := ExtractRouting(input)
	if fetch == nil || result.DestinationBaseAccount == "" || result.RoutingID != nil || result.DestinationError != nil {
		return result
	}
	if err := ctx.Err(); err != nil {
		return result
	}
	required, err := fetch(ctx, result.DestinationBaseAccount)
	if err == nil && required {
		result.Warnings = append(result.Warnings, address.Warning{
			Code:     address.WarnMissingRequiredMemo,
			Severity: "error",
			Message:  "Destination account requires a memo, but no routing ID was provided.",
		})
	}
	return result
}

// ExtractRoutingWithMemoRequirement performs normal routing extraction and
// optionally adds the SEP-0029 error when a classic destination requires a
// memo but no routing ID was supplied. Fetch failures fail open so callers
// retain the result of the synchronous parser.
//
// Deprecated: Use [ExtractRoutingWithContext] with a [ContextMemoRequirementFetcher]
// to enable request cancellation and deadline propagation.
func ExtractRoutingWithMemoRequirement(input RoutingInput, fetch MemoRequirementFetcher) RoutingResult {
	if fetch == nil {
		return ExtractRoutingWithContext(context.Background(), input, nil)
	}
	// Wrap the legacy fetcher so it satisfies ContextMemoRequirementFetcher.
	// The context is intentionally not forwarded to the wrapped function
	// (it has no context parameter), but callers can migrate to
	// ExtractRoutingWithContext when they are ready.
	return ExtractRoutingWithContext(context.Background(), input, func(_ context.Context, baseAccount string) (bool, error) {
		return fetch(baseAccount)
	})
}

func stringValue(s string) string {
	return s
}

// ValidateHostAntiSSRF validates that host / IP is not a loopback, private or link-local address
// to protect against Server-Side Request Forgery (SSRF) when resolving stellar.toml (Issue #212).
func ValidateHostAntiSSRF(hostOrIP string) error {
	ip := net.ParseIP(hostOrIP)
	if ip == nil {
		ips, err := net.LookupIP(hostOrIP)
		if err != nil {
			return err
		}
		if len(ips) == 0 {
			return errors.New("no IP addresses resolved for host")
		}
		ip = ips[0]
	}

	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return fmt.Errorf("SSRF protection: prohibited address %s", ip.String())
	}
	return nil
}

