package address

// strkeyAlphabet is the RFC 4648 base32 alphabet used by every strkey.
const strkeyAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"

// invalidBase32 marks bytes outside the strkey alphabet in base32DecodeMap.
const invalidBase32 = 0xff

// base32DecodeMap maps an input byte to its 5-bit value. Lowercase letters
// decode like their uppercase forms, matching DecodeStrKey's
// case-insensitivity without an uppercased copy of the input.
var base32DecodeMap = func() (m [256]byte) {
	for i := range m {
		m[i] = invalidBase32
	}
	for i := 0; i < len(strkeyAlphabet); i++ {
		c := strkeyAlphabet[i]
		m[c] = byte(i)
		if c >= 'A' && c <= 'Z' {
			m[c+('a'-'A')] = byte(i)
		}
	}
	return m
}()

// DecodedStrKeyLen returns the number of raw bytes (version byte, payload and
// checksum) that a strkey of encodedLen characters decodes to.
func DecodedStrKeyLen(encodedLen int) int {
	return encodedLen * 5 / 8
}

// DecodeStrKeyInto is DecodeStrKey for callers that supply the decoding
// buffer. It decodes address into dst and returns the version byte and the
// payload, which aliases dst. It does not allocate, so a caller that reuses
// dst (for example from a sync.Pool) decodes with zero heap allocations.
//
// dst must hold at least DecodedStrKeyLen(len(address)) bytes; a shorter dst
// is reported as an invalid length. Like DecodeStrKey it is case-insensitive
// and rejects unknown prefixes, non-alphabet characters, impossible lengths,
// non-zero trailing bits, unknown version bytes and checksum mismatches, with
// the same errors.
//
// The one difference is that only ASCII input is accepted: DecodeStrKey
// uppercases with strings.ToUpper, which also folds a few non-ASCII runes
// (such as U+017F, "ſ") into ASCII letters. Callers that must reproduce that
// exactly can fall back to DecodeStrKey when this function returns an error.
func DecodeStrKeyInto(dst []byte, address string) (versionByte byte, payload []byte, err error) {
	if len(address) < 3 {
		return 0, nil, errInvalidLength
	}

	switch address[0] {
	case 'G', 'g', 'M', 'm', 'C', 'c':
	default:
		return 0, nil, errUnknownPrefix
	}

	if len(dst) < DecodedStrKeyLen(len(address)) {
		return 0, nil, errInvalidLength
	}

	// Accumulate 5 bits per character and emit a byte whenever 8 are
	// available. Only the low bits of acc are ever read, so older bits are
	// allowed to shift out.
	var acc uint32
	var bits uint
	n := 0
	for i := 0; i < len(address); i++ {
		v := base32DecodeMap[address[i]]
		if v == invalidBase32 {
			return 0, nil, errInvalidBase32
		}
		acc = acc<<5 | uint32(v)
		bits += 5
		if bits >= 8 {
			bits -= 8
			dst[n] = byte(acc >> bits)
			n++
		}
	}

	// Unpadded base32 leaves 0-4 spare bits. Five or more means the length
	// is 1, 3 or 6 modulo 8, which no byte string encodes to; spare bits that
	// are set would not survive a re-encode. encoding/base32 rejects both,
	// and so does the round-trip check in DecodeStrKey.
	if bits >= 5 || acc&(1<<bits-1) != 0 {
		return 0, nil, errInvalidBase32
	}

	// Minimum length: version byte (1) + payload (at least 1) + checksum (2).
	if n < 4 {
		return 0, nil, errInvalidLength
	}

	versionByte = dst[0]
	switch versionByte {
	case VersionByteG, VersionByteM, VersionByteC:
	default:
		return 0, nil, errUnknownVersionByte
	}

	crc := CalculateCRC16(dst[:n-2])
	if dst[n-2] != byte(crc) || dst[n-1] != byte(crc>>8) {
		return 0, nil, errInvalidChecksum
	}

	return versionByte, dst[1 : n-2], nil
}
