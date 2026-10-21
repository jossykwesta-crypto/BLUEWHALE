package address

import (
	"encoding/binary"
	"errors"
	"strings"
)

// Parse parses a Stellar address string into an Address struct.
//
// Input is normalized to uppercase. When the input contained lowercase
// characters, a WarnNonCanonicalAddress warning carrying the original and
// normalized strings is appended to Address.Warnings.
func Parse(input string) (*Address, error) {
	if len(input) > 1024 {
		return nil, RoutingError{
			Code:    ErrInvalidLength,
			Input:   input[:32],
			Message: "address input exceeds maximum allowed length of 1024 bytes",
		}
	}
	kind, err := Detect(input)
	if err != nil {
		var code ErrorCode = ErrUnknownPrefix

		switch {
		case errors.Is(err, ErrInvalidChecksumError):
			code = ErrInvalidChecksum
		case errors.Is(err, ErrInvalidBase32Error):
			code = ErrInvalidBase32
		case errors.Is(err, ErrInvalidLengthError):
			code = ErrInvalidLength
		}

		return nil, RoutingError{
			Code:    code,
			Input:   input,
			Message: err.Error(),
		}
	}

	raw := strings.ToUpper(input)
	addr := &Address{
		Kind: kind,
		Raw:  raw,
	}

	if raw != input {
		addr.Warnings = append(addr.Warnings, nonCanonicalAddressWarning(input, raw))
	}

	if kind == KindM {
		versionByte, payload, err := DecodeStrKey(raw)
		if err != nil {
			return nil, err
		}
		if versionByte != VersionByteM {
			return nil, ErrUnknownPrefixError
		}
		if len(payload) != 40 {
			return nil, ErrInvalidLengthError
		}
		pubkey := payload[:32]
		id := binary.BigEndian.Uint64(payload[32:40])
		baseG, err := EncodeStrKey(VersionByteG, pubkey)
		if err != nil {
			return nil, err
		}
		addr.BaseG = baseG
		addr.MuxedID = id
	}

	return addr, nil
}

// nonCanonicalAddressWarning reports that original was normalized to its
// canonical uppercase form.
func nonCanonicalAddressWarning(original, normalized string) Warning {
	return Warning{
		Code:     WarnNonCanonicalAddress,
		Severity: "warn",
		Message:  "lowercase address was normalized to uppercase",
		Normalization: &Normalization{
			Original:   original,
			Normalized: normalized,
		},
	}
}
