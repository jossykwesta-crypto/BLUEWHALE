package routing

import (
	"errors"
	"strings"

	"github.com/REDISHFISH/BLUEWHALE/packages/core-go/address"
)

// uint64MaxStr is the decimal string representation of math.MaxUint64.
const uint64MaxStr = "18446744073709551615"

// maxMemoTextBytes is the maximum byte length allowed for a MEMO_TEXT value.
const maxMemoTextBytes = 28

// ErrMemoTooLong is returned when a memo string exceeds the 28-byte MEMO_TEXT limit.
var ErrMemoTooLong = errors.New("memo text exceeds 28 bytes")

type NormalizeResult struct {
	Normalized string
	Warnings   []address.Warning
}

// isAllDigits reports whether s is non-empty and contains only ASCII decimal digits.
// This is used instead of a compiled regexp to avoid heap allocations on the hot path.
func isAllDigits(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// fitsUint64 reports whether a canonical (no leading zeros) decimal string is within
// the uint64 range. Uses lexicographic comparison to avoid big.Int allocation.
func fitsUint64(s string) bool {
	if len(s) < len(uint64MaxStr) {
		return true
	}
	if len(s) > len(uint64MaxStr) {
		return false
	}
	return s <= uint64MaxStr
}

// NormalizeMemoText validates and normalizes a MEMO_TEXT value. It enforces the
// strict 28-byte limit (counting UTF-8 bytes via len(s), not runes) before any
// numeric parsing is attempted, returning ErrMemoTooLong for oversized inputs.
func NormalizeMemoText(s string) (string, error) {
	if len(s) > maxMemoTextBytes {
		return "", ErrMemoTooLong
	}
	return s, nil
}

func NormalizeMemoTextID(s string) NormalizeResult {
	if s == "" || !isAllDigits(s) {
		return NormalizeResult{}
	}

	// Strip leading zeros; strings.TrimLeft returns a sub-slice (no allocation).
	normalized := strings.TrimLeft(s, "0")
	if normalized == "" {
		normalized = "0"
	}

	var warnings []address.Warning
	if normalized != s {
		warnings = []address.Warning{{
			Code:     address.WarnNonCanonicalRoutingID,
			Severity: "warn",
			Message:  "Memo routing ID had leading zeros. Normalized to canonical decimal.",
			Normalization: &address.Normalization{
				Original:   s,
				Normalized: normalized,
			},
		}}
	}

	if !fitsUint64(normalized) {
		return NormalizeResult{Warnings: warnings}
	}

	return NormalizeResult{Normalized: normalized, Warnings: warnings}
}
