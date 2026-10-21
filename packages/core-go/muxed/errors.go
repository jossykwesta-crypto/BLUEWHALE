package muxed

import "fmt"

// ErrorCode represents a muxed address error code
type ErrorCode string

const (
	ErrInvalidGAddress    ErrorCode = "INVALID_G_ADDRESS"
	ErrUnknownVersionByte ErrorCode = "UNKNOWN_VERSION_BYTE"
	ErrInvalidLength      ErrorCode = "INVALID_LENGTH"
)

// AddressError represents a muxed address-related error
type AddressError struct {
	Code    ErrorCode
	Input   string
	Message string
	Cause   error
}

func (e *AddressError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *AddressError) Unwrap() error {
	return e.Cause
}

// ErrorCode returns the error's code as a plain string, which the core-go
// sentinels match on.
func (e *AddressError) ErrorCode() string {
	return string(e.Code)
}

// Is allows errors.Is to match by Code rather than by pointer identity, so a
// freshly built &AddressError{Code: ErrInvalidLength} matches
// ErrInvalidLengthError. Any error exposing the same code through ErrorCode()
// (such as corego.ErrInvalidLength) matches too.
func (e *AddressError) Is(target error) bool {
	if e == nil {
		return false
	}
	if t, ok := target.(interface{ ErrorCode() string }); ok {
		return string(e.Code) == t.ErrorCode()
	}
	return false
}

// Predefined muxed address errors
var (
	ErrInvalidGAddressError    = &AddressError{Code: ErrInvalidGAddress, Message: "invalid G address"}
	ErrUnknownVersionByteError = &AddressError{Code: ErrUnknownVersionByte, Message: "unknown version byte"}
	ErrInvalidLengthError      = &AddressError{Code: ErrInvalidLength, Message: "invalid length"}
)

// NewInvalidGAddressError creates an AddressError with a cause for invalid G address
func NewInvalidGAddressError(cause error) *AddressError {
	return &AddressError{
		Code:    ErrInvalidGAddress,
		Message: "invalid G address",
		Cause:   cause,
	}
}
