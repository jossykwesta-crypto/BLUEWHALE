package routing

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/REDISHFISH/BLUEWHALE/packages/core-go/address"
)

// ─── MarshalJSON ──────────────────────────────────────────────────────────────
//
// routingId MUST serialize as a JSON quoted decimal string, never as a raw
// integer literal.  JavaScript and Flutter Web represent numbers as IEEE-754
// doubles, which silently truncate integers above 2^53-1
// (Number.MAX_SAFE_INTEGER = 9007199254740991).  Muxed-account IDs are uint64
// and may legally reach 2^64-1, so a bare number would corrupt large IDs on
// those platforms.

func TestRoutingIDMarshalJSONEmitsQuotedDecimalString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		id   RoutingID
		want string
	}{
		{
			name: "zero",
			id:   RoutingID{raw: "0"},
			want: `"0"`,
		},
		{
			name: "small value",
			id:   RoutingID{raw: "12345"},
			want: `"12345"`,
		},
		{
			name: "above JS MAX_SAFE_INTEGER (2^53+1)",
			id:   RoutingID{raw: "9007199254740993"},
			want: `"9007199254740993"`,
		},
		{
			name: "uint64 max",
			id:   RoutingID{raw: "18446744073709551615"},
			want: `"18446744073709551615"`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.id.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON() error = %v", err)
			}
			if string(got) != tc.want {
				t.Errorf("MarshalJSON() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestRoutingIDMarshalJSONEmptyRawIsNull(t *testing.T) {
	t.Parallel()

	id := RoutingID{}
	got, err := id.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	if string(got) != "null" {
		t.Errorf("MarshalJSON() on empty RoutingID = %s, want null", got)
	}
}

func TestRoutingResultMarshalJSONRoutingIDIsQuotedString(t *testing.T) {
	t.Parallel()

	// uint64 max — would lose precision if serialized as a JS Number.
	const largeID = "18446744073709551615"
	result := RoutingResult{
		Success:                true,
		DestinationBaseAccount: "GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLT7AV7Y6S33Z6S3CHBAAAAAAAAAAAAABQD2",
		RoutingID:              NewRoutingID(largeID),
		RoutingSource:          "muxed",
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal(RoutingResult) error = %v", err)
	}

	// Decode into a generic map so we can inspect the raw JSON type.
	var raw map[string]any
	if err := json.Unmarshal(encoded, &raw); err != nil {
		t.Fatalf("json.Unmarshal error = %v", err)
	}

	routingIdValue, ok := raw["routingId"]
	if !ok {
		t.Fatalf("routingId field missing from JSON: %s", encoded)
	}

	// In JSON the value must decode as a Go string, not a float64.
	// If it were a bare integer the decoder would give us a float64.
	if _, isString := routingIdValue.(string); !isString {
		t.Errorf(
			"routingId decoded as %T (%v), want string — "+
				"bare integer literals lose precision above 2^53 in JS/Dart-Web",
			routingIdValue, routingIdValue,
		)
	}

	if routingIdValue != largeID {
		t.Errorf("routingId = %v, want %q", routingIdValue, largeID)
	}
}

func TestRoutingResultMarshalJSONRoundtrip(t *testing.T) {
	t.Parallel()

	original := RoutingResult{
		Success:                true,
		DestinationBaseAccount: "GABC",
		RoutingID:              NewRoutingID("9007199254740993"),
		RoutingSource:          "memo",
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal error = %v", err)
	}

	var decoded RoutingResult
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error = %v", err)
	}

	if decoded.RoutingID == nil {
		t.Fatal("decoded RoutingID is nil")
	}
	if decoded.RoutingID.String() != "9007199254740993" {
		t.Errorf("decoded RoutingID = %q, want %q", decoded.RoutingID.String(), "9007199254740993")
	}
}

// ─── UnmarshalJSON ─────────────────────────────────────────────────────────────

func TestRoutingIDUnmarshalJSONPreservesUint64Number(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"id":18446744073709551615}`)
	var body struct {
		ID RoutingID `json:"id"`
	}

	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got := body.ID.String(); got != "18446744073709551615" {
		t.Fatalf("RoutingID.String() = %q, want %q", got, "18446744073709551615")
	}

	gotUint64, err := body.ID.Uint64()
	if err != nil {
		t.Fatalf("RoutingID.Uint64() error = %v", err)
	}

	if gotUint64 != ^uint64(0) {
		t.Fatalf("RoutingID.Uint64() = %d, want %d", gotUint64, ^uint64(0))
	}
}

func TestRoutingIDUnmarshalJSONAcceptsQuotedDecimalString(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"id":"18446744073709551615"}`)
	var body struct {
		ID RoutingID `json:"id"`
	}

	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got := body.ID.String(); got != "18446744073709551615" {
		t.Fatalf("RoutingID.String() = %q, want %q", got, "18446744073709551615")
	}
}

func TestRoutingIDUnmarshalJSONRejectsInvalidNumbers(t *testing.T) {
	t.Parallel()

	testCases := []string{
		`{"id":18446744073709551616}`,
		`{"id":-1}`,
		`{"id":1.5}`,
		`{"id":"not-a-number"}`,
	}

	for _, payload := range testCases {
		payload := payload
		t.Run(payload, func(t *testing.T) {
			t.Parallel()

			var body struct {
				ID RoutingID `json:"id"`
			}

			if err := json.Unmarshal([]byte(payload), &body); err == nil {
				t.Fatalf("json.Unmarshal(%s) error = nil, want non-nil", payload)
			}
		})
	}
}

// RoutingResult is the JSON shape the other SDKs and the spec vectors read, so
// its empty form is part of the contract: fields that were never set must not
// come back as empty strings.
func TestRoutingResultMarshalsEmptyFieldsAway(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(RoutingResult{})
	if err != nil {
		t.Fatalf("json.Marshal(RoutingResult{}) error = %v", err)
	}

	const want = `{"success":false}`
	if got := string(encoded); got != want {
		t.Errorf("json.Marshal(RoutingResult{}) = %s, want %s", got, want)
	}
}

func TestDestinationErrorOmitsEmptyFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		result         RoutingResult
		wantCode       bool
		wantMessage    bool
		wantErrorField bool
	}{
		{
			name:           "no error at all",
			result:         RoutingResult{Success: true},
			wantErrorField: false,
		},
		{
			name: "code without a message",
			result: RoutingResult{
				DestinationError: &DestinationError{Code: address.ErrUnknownPrefix},
			},
			wantErrorField: true,
			wantCode:       true,
			wantMessage:    false,
		},
		{
			name: "code and message",
			result: RoutingResult{
				DestinationError: &DestinationError{
					Code:    address.ErrUnknownPrefix,
					Message: "unrecognised destination prefix",
				},
			},
			wantErrorField: true,
			wantCode:       true,
			wantMessage:    true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := json.Marshal(tt.result)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			var decoded map[string]any
			if err := json.Unmarshal(encoded, &decoded); err != nil {
				t.Fatalf("json.Unmarshal(%s) error = %v", encoded, err)
			}

			raw, present := decoded["destinationError"]
			if present != tt.wantErrorField {
				t.Fatalf("destinationError present = %v, want %v (json: %s)", present, tt.wantErrorField, encoded)
			}
			if !tt.wantErrorField {
				return
			}

			errFields, ok := raw.(map[string]any)
			if !ok {
				t.Fatalf("destinationError is %T, want an object (json: %s)", raw, encoded)
			}

			if _, ok := errFields["code"]; ok != tt.wantCode {
				t.Errorf("destinationError.code present = %v, want %v (json: %s)", ok, tt.wantCode, encoded)
			}
			if _, ok := errFields["message"]; ok != tt.wantMessage {
				t.Errorf("destinationError.message present = %v, want %v (json: %s)", ok, tt.wantMessage, encoded)
			}
		})
	}
}

// ─── MarshalBinary / UnmarshalBinary ───────────────────────────────────────────
//
// Binary serialization is used for efficient caching (Redis, BoltDB) and gRPC
// transport.  The round-trip must preserve every field, including the optional
// DestinationError pointer and the RoutingID.

func TestRoutingResultBinaryRoundtrip(t *testing.T) {
	t.Parallel()

	original := RoutingResult{
		Success:                true,
		DestinationBaseAccount: "GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLT7AV7Y6S33Z6S3CHBAAAAAAAAAAAAABQD2",
		RoutingID:              NewRoutingID("18446744073709551615"),
		RoutingSource:          "muxed",
		DestinationError: &DestinationError{
			Code:    address.ErrUnknownPrefix,
			Message: "unrecognised destination prefix",
		},
	}

	encoded, err := original.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v", err)
	}

	var decoded RoutingResult
	if err := decoded.UnmarshalBinary(encoded); err != nil {
		t.Fatalf("UnmarshalBinary() error = %v", err)
	}

	if decoded.Success != original.Success {
		t.Errorf("Success = %v, want %v", decoded.Success, original.Success)
	}
	if decoded.DestinationBaseAccount != original.DestinationBaseAccount {
		t.Errorf("DestinationBaseAccount = %q, want %q", decoded.DestinationBaseAccount, original.DestinationBaseAccount)
	}
	if decoded.RoutingSource != original.RoutingSource {
		t.Errorf("RoutingSource = %q, want %q", decoded.RoutingSource, original.RoutingSource)
	}
	if decoded.RoutingID == nil {
		t.Fatal("decoded RoutingID is nil")
	}
	if decoded.RoutingID.String() != original.RoutingID.String() {
		t.Errorf("RoutingID = %q, want %q", decoded.RoutingID.String(), original.RoutingID.String())
	}
	if decoded.DestinationError == nil {
		t.Fatal("decoded DestinationError is nil")
	}
	if decoded.DestinationError.Code != original.DestinationError.Code {
		t.Errorf("DestinationError.Code = %v, want %v", decoded.DestinationError.Code, original.DestinationError.Code)
	}
	if decoded.DestinationError.Message != original.DestinationError.Message {
		t.Errorf("DestinationError.Message = %q, want %q", decoded.DestinationError.Message, original.DestinationError.Message)
	}
}

func TestRoutingResultBinaryRoundtripEmpty(t *testing.T) {
	t.Parallel()

	original := RoutingResult{}

	encoded, err := original.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v", err)
	}

	var decoded RoutingResult
	if err := decoded.UnmarshalBinary(encoded); err != nil {
		t.Fatalf("UnmarshalBinary() error = %v", err)
	}

	if decoded.Success != false {
		t.Errorf("Success = %v, want false", decoded.Success)
	}
	if decoded.DestinationBaseAccount != "" {
		t.Errorf("DestinationBaseAccount = %q, want empty", decoded.DestinationBaseAccount)
	}
	if decoded.RoutingID != nil {
		t.Errorf("RoutingID = %v, want nil", decoded.RoutingID)
	}
	if decoded.RoutingSource != "" {
		t.Errorf("RoutingSource = %q, want empty", decoded.RoutingSource)
	}
	if decoded.DestinationError != nil {
		t.Errorf("DestinationError = %v, want nil", decoded.DestinationError)
	}
}

func TestRoutingResultUnmarshalBinaryRejectsTruncatedInput(t *testing.T) {
	t.Parallel()

	original := RoutingResult{
		Success:                true,
		DestinationBaseAccount: "GABC",
		RoutingID:              NewRoutingID("12345"),
		RoutingSource:          "memo",
	}

	encoded, err := original.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v", err)
	}

	// Every strict prefix of a valid encoding must be rejected rather than
	// silently producing a partially-populated result.
	for i := 0; i < len(encoded); i++ {
		var decoded RoutingResult
		if err := decoded.UnmarshalBinary(encoded[:i]); err == nil {
			t.Errorf("UnmarshalBinary(truncated to %d bytes) error = nil, want non-nil", i)
		}
	}
}

func TestRoutingResultBinaryRoundtripIsStable(t *testing.T) {
	t.Parallel()

	original := RoutingResult{
		Success:                true,
		DestinationBaseAccount: "GABC",
		RoutingID:              NewRoutingID("9007199254740993"),
		RoutingSource:          "memo",
	}

	first, err := original.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v", err)
	}

	var decoded RoutingResult
	if err := decoded.UnmarshalBinary(first); err != nil {
		t.Fatalf("UnmarshalBinary() error = %v", err)
	}

	second, err := decoded.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v", err)
	}

	if !bytes.Equal(first, second) {
		t.Errorf("re-encoded bytes differ:\n first = %x\nsecond = %x", first, second)
	}
}
