package muxed

import (
	"encoding/binary"
	"sync"

	"github.com/REDISHFISH/BLUEWHALE/packages/core-go/address"
)

const (
	// muxedPayloadLen is the M-address payload: a 32-byte Ed25519 public
	// key followed by the 8-byte big-endian ID.
	muxedPayloadLen = 32 + 8

	// muxedRawLen is the fully decoded M-address: version byte, payload and
	// 2-byte checksum.
	muxedRawLen = 1 + muxedPayloadLen + 2

	// muxedEncodedLen is the length of every M-address (69 characters).
	muxedEncodedLen = (muxedRawLen*8 + 4) / 5
)

// muxedBufPool recycles the decoding buffers used by DecodeMuxedRaw, so
// decoding a steady stream of M-addresses does not allocate. It holds
// pointers to arrays: putting a slice into a sync.Pool would allocate to box
// the slice header.
var muxedBufPool = sync.Pool{
	New: func() any { return new([muxedRawLen]byte) },
}

// DecodeMuxed extracts the base G-address and uint64 ID from a muxed
// M-address. The 64-bit ID is parsed directly from the big-endian payload
// using binary.BigEndian.Uint64, matching the Stellar protocol's native
// byte-aligned serialization.
//
// The returned G-address string is its only heap allocation. Use
// DecodeMuxedRaw when the raw public key is enough.
func DecodeMuxed(mAddress string) (string, uint64, error) {
	pubkey, id, err := DecodeMuxedRaw(mAddress)
	if err != nil {
		return "", 0, err
	}

	baseG, err := address.EncodeStrKey(address.VersionByteAccountID, pubkey[:])
	if err != nil {
		return "", 0, &AddressError{
			Code:    ErrInvalidGAddress,
			Input:   mAddress,
			Message: "failed to encode base G-address",
			Cause:   err,
		}
	}

	return baseG, id, nil
}

// DecodeMuxedRaw extracts the raw Ed25519 public key and uint64 ID from a
// muxed M-address. A well-formed address is decoded into a pooled buffer
// with zero heap allocations; the buffer is cleared before it is returned to
// the pool, so a recycled buffer never carries a previous address's bytes.
//
// It accepts and rejects exactly the inputs DecodeMuxed does, with the same
// errors.
func DecodeMuxedRaw(mAddress string) (pubkey [32]byte, id uint64, err error) {
	if key, muxID, ok := decodeMuxedPooled(mAddress); ok {
		return key, muxID, nil
	}
	return decodeMuxedChecked(mAddress)
}

// decodeMuxedPooled is the allocation-free fast path. It reports ok only for
// a well-formed M-address; anything else is left to decodeMuxedChecked,
// which produces the detailed error.
func decodeMuxedPooled(mAddress string) (pubkey [32]byte, id uint64, ok bool) {
	if len(mAddress) != muxedEncodedLen {
		return pubkey, 0, false
	}

	buf := muxedBufPool.Get().(*[muxedRawLen]byte)
	versionByte, payload, err := address.DecodeStrKeyInto(buf[:], mAddress)
	if err == nil && versionByte == address.VersionByteMuxedAccount && len(payload) == muxedPayloadLen {
		copy(pubkey[:], payload[:32])
		id = binary.BigEndian.Uint64(payload[32:muxedPayloadLen])
		ok = true
	}
	clear(buf[:])
	muxedBufPool.Put(buf)

	return pubkey, id, ok
}

// decodeMuxedChecked decodes mAddress through address.DecodeStrKey and maps
// each failure to an AddressError.
func decodeMuxedChecked(mAddress string) (pubkey [32]byte, id uint64, err error) {
	versionByte, payload, err := address.DecodeStrKey(mAddress)
	if err != nil {
		return pubkey, 0, &AddressError{
			Code:    ErrInvalidGAddress,
			Input:   mAddress,
			Message: "failed to decode muxed address",
			Cause:   err,
		}
	}

	if versionByte != address.VersionByteMuxedAccount {
		return pubkey, 0, &AddressError{
			Code:    ErrUnknownVersionByte,
			Input:   mAddress,
			Message: "expected muxed account version byte",
		}
	}

	if len(payload) != muxedPayloadLen {
		return pubkey, 0, &AddressError{
			Code:    ErrInvalidLength,
			Input:   mAddress,
			Message: "muxed payload must be 40 bytes",
		}
	}

	copy(pubkey[:], payload[:32])
	return pubkey, binary.BigEndian.Uint64(payload[32:muxedPayloadLen]), nil
}
