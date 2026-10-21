package address

import (
	"bytes"
	"encoding/base32"
	"strconv"
	"strings"
	"testing"
)

func TestCalculateCRC16(t *testing.T) {
	tests := []struct {
		name    string
		address string
	}{
		{
			name:    "Valid G address",
			address: "GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI",
		},
		{
			name:    "Valid M address with id=0",
			address: "MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACAAAAAAAAAAAAD672",
		},
		{
			name:    "Valid M address with id=1",
			address: "MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACAAAAAAAAAAAAHOO2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Decode the address to extract payload and checksum
			decoder := base32.StdEncoding.WithPadding(base32.NoPadding)
			decoded, err := decoder.DecodeString(strings.ToUpper(tt.address))
			if err != nil {
				t.Fatalf("failed to decode address: %v", err)
			}

			// Extract payload (everything except the last 2 checksum bytes)
			payload := decoded[:len(decoded)-2]

			// Extract the checksum bytes from the address
			checksumBytes := decoded[len(decoded)-2:]

			// Calculate CRC16 for the payload
			calculatedCRC := CalculateCRC16(payload)

			// Reconstruct the checksum as it is stored (little-endian: LSB first, then MSB)
			storedChecksum := uint16(checksumBytes[0]) | (uint16(checksumBytes[1]) << 8)

			// Verify the calculated CRC matches the stored checksum in the address
			if calculatedCRC != storedChecksum {
				t.Errorf("CRC16 mismatch for address %s:\n  calculated: 0x%04x\n  stored:      0x%04x",
					tt.address, calculatedCRC, storedChecksum)
			}
		})
	}
}

// TestCalculateCRC16WithKnownVectors verifies CRC16 calculation against known test vectors.
// These vectors represent the payload (version byte + data) that gets CRC-checked.
func TestCalculateCRC16WithKnownVectors(t *testing.T) {
	tests := []struct {
		name     string
		payload  []byte
		expected uint16
	}{
		{
			name:     "Empty payload",
			payload:  []byte{},
			expected: 0x0000,
		},
		{
			name:     "Single byte version G (0x30)",
			payload:  []byte{VersionByteG}, // 0x30
			expected: 0x3653,
		},
		{
			name:     "Single byte version M (0x60)",
			payload:  []byte{VersionByteM}, // 0x60
			expected: 0x6ca6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateCRC16(tt.payload)
			if result != tt.expected {
				t.Errorf("CRC16 mismatch:\n  payload:    %x\n  calculated: 0x%04x\n  expected:   0x%04x",
					tt.payload, result, tt.expected)
			}
		})
	}
}

// crc16Bytewise is the byte-at-a-time CRC-16/XMODEM loop that
// CalculateCRC16 replaced. It is the reference for the equivalence test and
// the baseline for the benchmarks below.
func crc16Bytewise(data []byte) uint16 {
	crc := uint16(0)
	for _, b := range data {
		crc = (crc << 8) ^ crc16Table[(crc>>8)^uint16(b)]
	}
	return crc
}

// crc16TestData returns n deterministic pseudo-random bytes (xorshift32), so
// a failure is reproducible.
func crc16TestData(n int) []byte {
	data := make([]byte, n)
	state := uint32(0x9e3779b9)
	for i := range data {
		state ^= state << 13
		state ^= state >> 17
		state ^= state << 5
		data[i] = byte(state)
	}
	return data
}

// TestCalculateCRC16Check uses the published CRC-16/XMODEM check value: the
// checksum of the ASCII string "123456789" is 0x31C3.
func TestCalculateCRC16Check(t *testing.T) {
	if got := CalculateCRC16([]byte("123456789")); got != 0x31c3 {
		t.Fatalf("CRC16(\"123456789\") = 0x%04x, want 0x31c3", got)
	}
}

// TestCalculateCRC16MatchesBytewise covers every length from 0 to 300, so
// each remainder of the eight-byte unrolled loop runs both with and without
// full blocks before it, plus slices that do not start at the beginning of
// their backing array.
func TestCalculateCRC16MatchesBytewise(t *testing.T) {
	data := crc16TestData(300 + 8)
	for n := 0; n <= 300; n++ {
		for _, offset := range []int{0, 1, 7} {
			in := data[offset : offset+n]
			if got, want := CalculateCRC16(in), crc16Bytewise(in); got != want {
				t.Fatalf("len=%d offset=%d: CalculateCRC16 = 0x%04x, bytewise = 0x%04x", n, offset, got, want)
			}
		}
	}

	allOnes := bytes.Repeat([]byte{0xff}, 64)
	if got, want := CalculateCRC16(allOnes), crc16Bytewise(allOnes); got != want {
		t.Fatalf("0xff x 64: CalculateCRC16 = 0x%04x, bytewise = 0x%04x", got, want)
	}
}

// TestCalculateCRC16StrKeyChecksumBytes checks that the checksum is still
// stored little-endian: re-encoding a known address's payload with
// EncodeStrKey must reproduce the address exactly.
func TestCalculateCRC16StrKeyChecksumBytes(t *testing.T) {
	for _, addr := range []string{
		"GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI",
		"MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACAAAAAAAAAAAAD672",
		"MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQAD7777777777774OFW",
		"CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQVU2HHGCYSC",
	} {
		versionByte, payload, err := DecodeStrKey(addr)
		if err != nil {
			t.Fatalf("DecodeStrKey(%s): %v", addr, err)
		}
		reencoded, err := EncodeStrKey(versionByte, payload)
		if err != nil {
			t.Fatalf("EncodeStrKey(%s): %v", addr, err)
		}
		if reencoded != addr {
			t.Fatalf("re-encoded %s as %s", addr, reencoded)
		}
	}
}

var crc16Sink uint16

// benchmarkCRC16 runs fn over the buffer sizes that matter in practice: a
// G-address checksum input (33 bytes: version byte and key), an M-address
// input (41 bytes) and a larger block.
func benchmarkCRC16(b *testing.B, fn func([]byte) uint16) {
	for _, size := range []int{33, 41, 1024} {
		data := crc16TestData(size)
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ReportAllocs()
			var crc uint16
			for i := 0; i < b.N; i++ {
				crc = fn(data)
			}
			crc16Sink = crc
		})
	}
}

// BenchmarkCalculateCRC16 is the unrolled implementation. Compare it with
// BenchmarkCalculateCRC16Bytewise:
//
//	go test ./address -run '^$' -bench 'CalculateCRC16' -benchmem
func BenchmarkCalculateCRC16(b *testing.B) {
	benchmarkCRC16(b, CalculateCRC16)
}

// BenchmarkCalculateCRC16Bytewise is the previous byte-at-a-time loop.
func BenchmarkCalculateCRC16Bytewise(b *testing.B) {
	benchmarkCRC16(b, crc16Bytewise)
}
