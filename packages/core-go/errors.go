// Package corego holds sentinel errors shared by the core-go sub-packages.
//
// Each sentinel matches, under errors.Is, any error from address, muxed or
// routing that carries the same code, however deeply it has been wrapped with
// fmt.Errorf("%w", err):
//
//	_, err := address.Parse(input)
//	if errors.Is(err, corego.ErrInvalidChecksum) {
//		// ...
//	}
//
// Matching is by code, so callers do not need to know which sub-package
// produced the error. To read the details (input, message, cause), use
// errors.As with the concrete type: address.RoutingError or
// *muxed.AddressError.
package corego

// Error is a sentinel identified by a stable code.
type Error struct {
	code    string
	message string
}

func (e *Error) Error() string { return e.message }

// ErrorCode returns the sentinel's code. Errors in the sub-packages expose the
// same method, which is how they match each other under errors.Is.
func (e *Error) ErrorCode() string { return e.code }

// Is reports whether target carries the same code, which makes matching
// symmetric: errors.Is(corego.ErrInvalidLength, err) works as well.
func (e *Error) Is(target error) bool {
	t, ok := target.(interface{ ErrorCode() string })
	return ok && t.ErrorCode() == e.code
}

var (
	// ErrInvalidChecksum matches an address whose CRC16 checksum does not
	// verify (address.ErrInvalidChecksum).
	ErrInvalidChecksum = &Error{code: "INVALID_CHECKSUM", message: "invalid checksum"}

	// ErrInvalidLength matches an address or muxed payload of the wrong length
	// (address.ErrInvalidLength and muxed.ErrInvalidLength).
	ErrInvalidLength = &Error{code: "INVALID_LENGTH", message: "invalid length"}

	// ErrUnsupportedScheme matches a payment URI that does not use the
	// web+stellar: scheme (routing.ErrInvalidScheme).
	ErrUnsupportedScheme = &Error{code: "UNSUPPORTED_SCHEME", message: "unsupported URI scheme"}
)
