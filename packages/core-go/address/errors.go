package address

// ErrorCode represents specific error types for pattern matching
type ErrorCode string

const (
	ErrInvalidChecksum               ErrorCode = "INVALID_CHECKSUM"
	ErrInvalidLength                 ErrorCode = "INVALID_LENGTH"
	ErrInvalidBase32                 ErrorCode = "INVALID_BASE32"
	ErrUnknownPrefix                 ErrorCode = "UNKNOWN_PREFIX"
	ErrRejectedSeedKey               ErrorCode = "REJECTED_SEED_KEY"
	ErrRejectedPreauth               ErrorCode = "REJECTED_PREAUTH"
	ErrRejectedHashX                 ErrorCode = "REJECTED_HASH_X"
	ErrFederationAddressNotSupported ErrorCode = "FEDERATION_ADDRESS_NOT_SUPPORTED"
)

// RoutingError is the main custom error type
type RoutingError struct {
	Code    ErrorCode
	Input   string
	Message string
}

func (e RoutingError) Error() string {
	return e.Message
}

// ErrorCode returns the error's code as a plain string. It is what the
// package-level sentinels in core-go match on, so errors.Is works across
// packages without them importing each other.
func (e RoutingError) ErrorCode() string {
	return string(e.Code)
}

// Is allows errors.Is to match by Code. The target may be a RoutingError, a
// *RoutingError, or any error exposing the same code through ErrorCode()
// (such as corego.ErrInvalidChecksum).
func (e RoutingError) Is(target error) bool {
	switch t := target.(type) {
	case RoutingError:
		return e.Code == t.Code
	case *RoutingError:
		return t != nil && e.Code == t.Code
	case interface{ ErrorCode() string }:
		return string(e.Code) == t.ErrorCode()
	}
	return false
}

// Common error variables
var (
	ErrInvalidChecksumError               = RoutingError{Code: ErrInvalidChecksum, Message: "invalid checksum"}
	ErrInvalidBase32Error                 = RoutingError{Code: ErrInvalidBase32, Message: "invalid base32 encoding"}
	ErrInvalidLengthError                 = RoutingError{Code: ErrInvalidLength, Message: "invalid address length"}
	ErrUnknownPrefixError                 = RoutingError{Code: ErrUnknownPrefix, Message: "unknown address prefix"}
	ErrFederationAddressNotSupportedError = RoutingError{Code: ErrFederationAddressNotSupported, Message: "federation address not supported"}
)

// Legacy / additional error for version byte
var ErrUnknownVersionByteError = RoutingError{
	Code:    ErrUnknownPrefix,
	Message: "unknown version byte",
}
