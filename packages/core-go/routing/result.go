package routing

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

// ErrNoRoutingID is returned by RoutingID.Uint64 when the RoutingID is nil
// or holds no value.
var ErrNoRoutingID = errors.New("routing id: no value")

var (
	_ json.Marshaler   = RoutingID{}
	_ json.Unmarshaler = (*RoutingID)(nil)
)

type MemoType string

const (
	MemoTypeNone   MemoType = "none"
	MemoTypeID     MemoType = "id"
	MemoTypeText   MemoType = "text"
	MemoTypeHash   MemoType = "hash"
	MemoTypeReturn MemoType = "return"
)

// RoutingID is a wrapper around a numeric string representing a 64-bit unsigned integer.
//
// The zero value (and a nil *RoutingID) holds no ID. All accessors are
// nil-safe. JSON encoding uses a decimal string ("123") so values above
// 2^53 survive JavaScript consumers; decoding accepts a string or a number.
type RoutingID struct {
	raw string
}

// MarshalJSON encodes the ID as a JSON decimal string, or null when empty.
func (r RoutingID) MarshalJSON() ([]byte, error) {
	if r.raw == "" {
		return []byte("null"), nil
	}
	return json.Marshal(r.raw)
}

// UnmarshalJSON accepts a JSON number, a quoted decimal string, or null.
// Values are canonicalized (leading zeros removed) and must fit in uint64.
func (r *RoutingID) UnmarshalJSON(data []byte) error {
	if r == nil {
		return fmt.Errorf("routing id: UnmarshalJSON on nil receiver")
	}

	if string(data) == "null" {
		r.raw = ""
		return nil
	}

	var id string
	if len(data) > 0 && data[0] == '"' {
		if err := json.Unmarshal(data, &id); err != nil {
			return fmt.Errorf("routing id: invalid quoted value: %w", err)
		}
	} else {
		id = string(data)
	}

	parsed, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return fmt.Errorf("routing id: invalid uint64 value %q: %w", id, err)
	}

	r.raw = strconv.FormatUint(parsed, 10)
	return nil
}

// String returns the decimal representation of the ID, or "" when r is nil
// or empty.
func (r *RoutingID) String() string {
	if r == nil {
		return ""
	}
	return r.raw
}

// Uint64 returns the ID as a uint64. It returns ErrNoRoutingID when r is nil
// or empty, and a *strconv.NumError when the value is not a valid uint64.
func (r *RoutingID) Uint64() (uint64, error) {
	if r.IsZero() {
		return 0, ErrNoRoutingID
	}
	return strconv.ParseUint(r.raw, 10, 64)
}

// IsZero reports whether r is nil or holds no ID. A RoutingID holding the
// value "0" is not zero: 0 is a valid routing ID.
func (r *RoutingID) IsZero() bool {
	return r == nil || r.raw == ""
}

func NewRoutingID(s string) *RoutingID {
	return &RoutingID{raw: s}
}

// RoutingResult is the outcome of a routing decision. It is safe to cache in
// Redis/BoltDB or send over gRPC via its JSON and binary marshalers.
type RoutingResult struct {
	ID       *RoutingID
	MemoType MemoType
	Memo     string
}

// MarshalJSON encodes the result as a JSON object. A nil ID is encoded as
// null; MemoType and Memo are always present.
func (r RoutingResult) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID       *RoutingID `json:"id"`
		MemoType MemoType   `json:"memoType"`
		Memo     string     `json:"memo"`
	}{
		ID:       r.ID,
		MemoType: r.MemoType,
		Memo:     r.Memo,
	})
}

// UnmarshalJSON decodes a JSON object produced by MarshalJSON.
func (r *RoutingResult) UnmarshalJSON(data []byte) error {
	if r == nil {
		return fmt.Errorf("routing result: UnmarshalJSON on nil receiver")
	}

	var v struct {
		ID       *RoutingID `json:"id"`
		MemoType MemoType   `json:"memoType"`
		Memo     string     `json:"memo"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("routing result: invalid JSON: %w", err)
	}

	r.ID = v.ID
	r.MemoType = v.MemoType
	r.Memo = v.Memo
	return nil
}

// MarshalBinary encodes the result into a compact, length-prefixed binary
// form suitable for caching. Layout:
//
//	[1 byte flags][2 byte memoType len][memoType][4 byte memo len][memo][8 byte id]
//
// The ID is only present when the flag bit is set.
func (r RoutingResult) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer

	var flags byte
	if !r.ID.IsZero() {
		flags |= 1
	}
	buf.WriteByte(flags)

	if err := writeString(&buf, string(r.MemoType)); err != nil {
		return nil, err
	}
	if err := writeString(&buf, r.Memo); err != nil {
		return nil, err
	}

	if flags&1 != 0 {
		id, err := r.ID.Uint64()
		if err != nil {
			return nil, fmt.Errorf("routing result: invalid id: %w", err)
		}
		if err := binary.Write(&buf, binary.BigEndian, id); err != nil {
			return nil, fmt.Errorf("routing result: write id: %w", err)
		}
	}

	return buf.Bytes(), nil
}

// UnmarshalBinary decodes data produced by MarshalBinary.
func (r *RoutingResult) UnmarshalBinary(data []byte) error {
	if r == nil {
		return fmt.Errorf("routing result: UnmarshalBinary on nil receiver")
	}

	buf := bytes.NewReader(data)

	flags, err := buf.ReadByte()
	if err != nil {
		return fmt.Errorf("routing result: read flags: %w", err)
	}

	memoType, err := readString(buf)
	if err != nil {
		return err
	}
	memo, err := readString(buf)
	if err != nil {
		return err
	}

	var id *RoutingID
	if flags&1 != 0 {
		var raw uint64
		if err := binary.Read(buf, binary.BigEndian, &raw); err != nil {
			return fmt.Errorf("routing result: read id: %w", err)
		}
		id = NewRoutingID(strconv.FormatUint(raw, 10))
	}

	r.ID = id
	r.MemoType = MemoType(memoType)
	r.Memo = memo
	return nil
}

func writeString(buf *bytes.Buffer, s string) error {
	if len(s) > 0xffff {
		return fmt.Errorf("routing result: string too long: %d", len(s))
	}
	if err := binary.Write(buf, binary.BigEndian, uint16(len(s))); err != nil {
		return fmt.Errorf("routing result: write length: %w", err)
	}
	buf.WriteString(s)
	return nil
}

func readString(buf *bytes.Reader) (string, error) {
	var n uint16
	if err := binary.Read(buf, binary.BigEndian, &n); err != nil {
		return "", fmt.Errorf("routing result: read length: %w", err)
	}
	b := make([]byte, n)
	if _, err := buf.Read(b); err != nil {
		return "", fmt.Errorf("routing result: read string: %w", err)
	}
	return string(b), nil
}
