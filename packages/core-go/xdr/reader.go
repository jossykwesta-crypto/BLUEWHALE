package xdr

import (
	"encoding/binary"
	"fmt"
)

// reader is a minimal XDR (RFC 4506) stream decoder over a byte slice.
//
// The first failure is sticky: once err is set every read returns a zero
// value, so decoding code can read a whole structure and check err once.
type reader struct {
	buf []byte
	off int
	err error
}

func (r *reader) fail(format string, args ...any) {
	if r.err == nil {
		r.err = fmt.Errorf("%w: %s at byte %d", ErrInvalidXDR, fmt.Sprintf(format, args...), r.off)
	}
}

// take returns the next n bytes, or nil once the input is exhausted.
func (r *reader) take(n int) []byte {
	if r.err != nil {
		return nil
	}
	if n < 0 || len(r.buf)-r.off < n {
		r.fail("unexpected end of input reading %d bytes", n)
		return nil
	}
	b := r.buf[r.off : r.off+n]
	r.off += n
	return b
}

func (r *reader) uint32() uint32 {
	b := r.take(4)
	if b == nil {
		return 0
	}
	return binary.BigEndian.Uint32(b)
}

func (r *reader) int32() int32 {
	return int32(r.uint32())
}

func (r *reader) uint64() uint64 {
	b := r.take(8)
	if b == nil {
		return 0
	}
	return binary.BigEndian.Uint64(b)
}

// bool decodes an XDR boolean, which is also the presence flag of an
// optional value ("T*"). Anything but 0 or 1 is malformed.
func (r *reader) bool() bool {
	v := r.uint32()
	if v > 1 {
		r.fail("invalid boolean %d", v)
		return false
	}
	return v == 1
}

// fixed reads fixed-length opaque data (opaque[n]), padded to 4 bytes.
func (r *reader) fixed(n int) []byte {
	b := r.take(n)
	r.padding(n)
	return b
}

// variable reads variable-length opaque data or a string (opaque<max>),
// rejecting lengths above max.
func (r *reader) variable(max uint32) []byte {
	n := r.uint32()
	if r.err != nil {
		return nil
	}
	if n > max {
		r.fail("length %d exceeds maximum %d", n, max)
		return nil
	}
	b := r.take(int(n))
	r.padding(int(n))
	return b
}

// count reads an array length, rejecting lengths above max.
func (r *reader) count(max uint32) int {
	n := r.uint32()
	if r.err == nil && n > max {
		r.fail("array length %d exceeds maximum %d", n, max)
		return 0
	}
	return int(n)
}

// padding consumes the zero bytes that align an n-byte item to 4 bytes.
func (r *reader) padding(n int) {
	for _, p := range r.take((4 - n%4) % 4) {
		if p != 0 {
			r.fail("non-zero padding")
			return
		}
	}
}

// done reports an error if decoding failed or left trailing bytes.
func (r *reader) done() error {
	if r.err == nil && r.off != len(r.buf) {
		r.fail("%d trailing bytes", len(r.buf)-r.off)
	}
	return r.err
}
