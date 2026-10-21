package corego_test

import (
	"errors"
	"fmt"
	"testing"

	corego "github.com/REDISHFISH/BLUEWHALE/packages/core-go"
	"github.com/REDISHFISH/BLUEWHALE/packages/core-go/address"
	"github.com/REDISHFISH/BLUEWHALE/packages/core-go/muxed"
	"github.com/REDISHFISH/BLUEWHALE/packages/core-go/routing"
)

const validG = "GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI"

// badChecksumG is validG with its last character changed, so the payload still
// decodes but the CRC16 no longer verifies.
const badChecksumG = "GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSA"

// shortMuxed is a well-formed strkey with the muxed version byte but a 32-byte
// payload instead of 40.
func shortMuxed(t *testing.T) string {
	t.Helper()
	m, err := address.EncodeStrKey(address.VersionByteMuxedAccount, make([]byte, 32))
	if err != nil {
		t.Fatalf("encoding test input: %v", err)
	}
	return m
}

func TestErrorsIs_InvalidChecksum(t *testing.T) {
	_, err := address.Parse(badChecksumG)
	if err == nil {
		t.Fatal("expected an error for a bad checksum")
	}
	if !errors.Is(err, corego.ErrInvalidChecksum) {
		t.Errorf("errors.Is(err, corego.ErrInvalidChecksum) = false for %v", err)
	}
	if !errors.Is(err, address.ErrInvalidChecksumError) {
		t.Errorf("errors.Is(err, address.ErrInvalidChecksumError) = false for %v", err)
	}
	if errors.Is(err, corego.ErrInvalidLength) {
		t.Errorf("a checksum error must not match corego.ErrInvalidLength")
	}
}

func TestErrorsIs_InvalidLength(t *testing.T) {
	_, err := address.Parse(shortMuxed(t))
	if !errors.Is(err, corego.ErrInvalidLength) {
		t.Errorf("errors.Is(err, corego.ErrInvalidLength) = false for %v", err)
	}
	if !errors.Is(err, address.ErrInvalidLengthError) {
		t.Errorf("errors.Is(err, address.ErrInvalidLengthError) = false for %v", err)
	}
	if errors.Is(err, corego.ErrInvalidChecksum) {
		t.Errorf("a length error must not match corego.ErrInvalidChecksum")
	}
}

func TestErrorsIs_MuxedInvalidLength(t *testing.T) {
	_, _, err := muxed.DecodeMuxed(shortMuxed(t))
	if !errors.Is(err, corego.ErrInvalidLength) {
		t.Errorf("errors.Is(err, corego.ErrInvalidLength) = false for %v", err)
	}
	// Matching by code, not by pointer: DecodeMuxed builds a fresh
	// *AddressError, which previously never matched the exported sentinel.
	if !errors.Is(err, muxed.ErrInvalidLengthError) {
		t.Errorf("errors.Is(err, muxed.ErrInvalidLengthError) = false for %v", err)
	}
}

func TestErrorsIs_MuxedWrapsAddressCause(t *testing.T) {
	_, _, err := muxed.DecodeMuxed(badChecksumG)
	if err == nil {
		t.Fatal("expected an error")
	}
	// The muxed error reports INVALID_G_ADDRESS and wraps the address error,
	// so both codes are reachable through the chain.
	if !errors.Is(err, muxed.ErrInvalidGAddressError) {
		t.Errorf("expected the muxed code to match, got %v", err)
	}
	if !errors.Is(err, corego.ErrInvalidChecksum) {
		t.Errorf("expected the wrapped checksum error to match, got %v", err)
	}
}

func TestErrorsIs_UnsupportedScheme(t *testing.T) {
	_, _, err := routing.ExtractRoutingFromURI("https://example.com/pay?destination=" + validG)
	if !errors.Is(err, corego.ErrUnsupportedScheme) {
		t.Errorf("errors.Is(err, corego.ErrUnsupportedScheme) = false for %v", err)
	}
	if !errors.Is(err, routing.ErrInvalidScheme) {
		t.Errorf("errors.Is(err, routing.ErrInvalidScheme) = false for %v", err)
	}
	if err != routing.ErrInvalidScheme {
		t.Errorf("direct comparison with routing.ErrInvalidScheme must keep working")
	}
}

func TestErrorsIs_ThroughFmtWrapping(t *testing.T) {
	_, err := address.Parse(badChecksumG)
	wrapped := fmt.Errorf("validating deposit: %w", fmt.Errorf("parsing destination: %w", err))

	if !errors.Is(wrapped, corego.ErrInvalidChecksum) {
		t.Errorf("errors.Is must see through fmt.Errorf %%w wrapping: %v", wrapped)
	}

	_, _, uriErr := routing.ExtractRoutingFromURI("ftp://nope")
	if !errors.Is(fmt.Errorf("handling callback: %w", uriErr), corego.ErrUnsupportedScheme) {
		t.Errorf("wrapped scheme error did not match")
	}
}

func TestErrorsIs_Symmetric(t *testing.T) {
	_, err := address.Parse(badChecksumG)
	if !errors.Is(corego.ErrInvalidChecksum, err) {
		t.Errorf("errors.Is(sentinel, err) should also match by code")
	}
}

func TestErrorsIs_PointerTarget(t *testing.T) {
	_, err := address.Parse(badChecksumG)
	target := &address.RoutingError{Code: address.ErrInvalidChecksum}
	if !errors.Is(err, target) {
		t.Errorf("a *RoutingError target should match by code")
	}
}

func TestErrorsAs_ConcreteTypes(t *testing.T) {
	_, err := address.Parse(badChecksumG)
	wrapped := fmt.Errorf("outer: %w", err)

	var routingErr address.RoutingError
	if !errors.As(wrapped, &routingErr) {
		t.Fatalf("errors.As to address.RoutingError failed for %v", wrapped)
	}
	if routingErr.Code != address.ErrInvalidChecksum || routingErr.Input != badChecksumG {
		t.Errorf("unexpected RoutingError: %+v", routingErr)
	}

	_, _, muxErr := muxed.DecodeMuxed(shortMuxed(t))
	var addrErr *muxed.AddressError
	if !errors.As(fmt.Errorf("outer: %w", muxErr), &addrErr) {
		t.Fatalf("errors.As to *muxed.AddressError failed for %v", muxErr)
	}
	if addrErr.Code != muxed.ErrInvalidLength {
		t.Errorf("unexpected AddressError code: %s", addrErr.Code)
	}

	var sentinel *corego.Error
	if errors.As(wrapped, &sentinel) {
		t.Errorf("errors.As must not turn a RoutingError into a sentinel")
	}
}

func TestSentinelCodes(t *testing.T) {
	cases := []struct {
		sentinel *corego.Error
		code     string
	}{
		{corego.ErrInvalidChecksum, string(address.ErrInvalidChecksum)},
		{corego.ErrInvalidLength, string(address.ErrInvalidLength)},
		{corego.ErrInvalidLength, string(muxed.ErrInvalidLength)},
		{corego.ErrUnsupportedScheme, "UNSUPPORTED_SCHEME"},
	}
	for _, c := range cases {
		if c.sentinel.ErrorCode() != c.code {
			t.Errorf("%v: code %q, want %q", c.sentinel, c.sentinel.ErrorCode(), c.code)
		}
		if c.sentinel.Error() == "" {
			t.Errorf("%s: empty message", c.code)
		}
	}
}
