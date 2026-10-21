package address

import (
	"crypto/subtle"
	"encoding/base32"
	"strings"
)

const (
	VersionByteG = 6 << 3
	VersionByteM = 12 << 3
	VersionByteC = 2 << 3

	VersionByteAccountID    = VersionByteG
	VersionByteMuxedAccount = VersionByteM
	VersionByteContract     = VersionByteC
)

// strkeyEncoding is shared so decoding and encoding do not allocate a new
// *base32.Encoding on every call.
var strkeyEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// Boxed copies of the standard validation errors. Converting a RoutingError
// struct to the error interface allocates, so the hot path returns these
// preallocated values instead. They compare equal to the exported
// Err*Error values under errors.Is.
var (
	errInvalidLength      error = ErrInvalidLengthError
	errInvalidBase32      error = ErrInvalidBase32Error
	errInvalidChecksum    error = ErrInvalidChecksumError
	errUnknownPrefix      error = ErrUnknownPrefixError
	errUnknownVersionByte error = ErrUnknownVersionByteError
)

// stackBufLen bounds the scratch buffers kept on the stack during decode and
// encode. The longest supported strkey (M-address) is 69 characters.
const stackBufLen = 128

// DecodeStrKey decodes a strkey address and returns version byte and payload.
// It is case-insensitive; Parse is the entry point that reports case
// normalization via WarnNonCanonicalAddress.
func DecodeStrKey(address string) (versionByte byte, payload []byte, err error) {
	versionByte, payload, _, err = decodeStrKey(address)
	return versionByte, payload, err
}

// decodeStrKey is DecodeStrKey that also returns the canonical (uppercase)
// form of address. canonical aliases address when it is already uppercase.
func decodeStrKey(address string) (versionByte byte, payload []byte, canonical string, err error) {
	if len(address) < 3 {
		return 0, nil, "", errInvalidLength
	}

	// Reject unsupported prefixes with a byte comparison before doing any
	// base32 work. The first character fixes the top five bits of the
	// version byte, so this is consistent with the version byte check below.
	switch address[0] {
	case 'G', 'g', 'M', 'm', 'C', 'c':
	default:
		return 0, nil, "", errUnknownPrefix
	}

	// strings.ToUpper returns its input without allocating when it is
	// already uppercase ASCII.
	canonical = strings.ToUpper(address)

	// DecodeString decodes in place into a single buffer; Decode would
	// allocate an extra internal copy of its input.
	decoded, err := strkeyEncoding.DecodeString(canonical)
	if err != nil {
		return 0, nil, "", errInvalidBase32
	}
	n := len(decoded)

	// Verify round-trip encoding to reject non-zero trailing bits.
	var reBuf [stackBufLen]byte
	var re []byte
	if encLen := strkeyEncoding.EncodedLen(n); encLen <= len(reBuf) {
		re = reBuf[:encLen]
	} else {
		re = make([]byte, encLen)
	}
	strkeyEncoding.Encode(re, decoded)
	if string(re) != canonical {
		return 0, nil, "", errInvalidBase32
	}

	// Minimum length: version byte (1) + payload (at least 1) + checksum (2).
	if n < 4 {
		return 0, nil, "", errInvalidLength
	}

	versionByte = decoded[0]
	switch versionByte {
	case VersionByteG, VersionByteM, VersionByteC:
	default:
		return 0, nil, "", errUnknownVersionByte
	}

	crc := CalculateCRC16(decoded[:n-2])
	expectedChecksum := [2]byte{byte(crc), byte(crc >> 8)}
	if subtle.ConstantTimeCompare(decoded[n-2:], expectedChecksum[:]) != 1 {
		return 0, nil, "", errInvalidChecksum
	}

	// Return payload without version byte or checksum.
	return versionByte, decoded[1 : n-2], canonical, nil
}

// EncodeStrKey encodes a payload with a version byte and checksum.
func EncodeStrKey(versionByte byte, payload []byte) (string, error) {
	rawLen := 1 + len(payload) + 2

	var rawBuf [stackBufLen]byte
	var raw []byte
	if rawLen <= len(rawBuf) {
		raw = rawBuf[:rawLen]
	} else {
		raw = make([]byte, rawLen)
	}
	raw[0] = versionByte
	copy(raw[1:], payload)

	crc := CalculateCRC16(raw[:rawLen-2])
	raw[rawLen-2] = byte(crc)
	raw[rawLen-1] = byte(crc >> 8)

	var outBuf [stackBufLen]byte
	var out []byte
	if encLen := strkeyEncoding.EncodedLen(rawLen); encLen <= len(outBuf) {
		out = outBuf[:encLen]
	} else {
		out = make([]byte, encLen)
	}
	strkeyEncoding.Encode(out, raw)
	return string(out), nil
}

// kindForVersionByte maps a version byte validated by decodeStrKey to its
// AddressKind.
func kindForVersionByte(versionByte byte) (AddressKind, error) {
	switch versionByte {
	case VersionByteG:
		return KindG, nil
	case VersionByteM:
		return KindM, nil
	case VersionByteC:
		return KindC, nil
	default:
		return "", errUnknownVersionByte
	}
}
