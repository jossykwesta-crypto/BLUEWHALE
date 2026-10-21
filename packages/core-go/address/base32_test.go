package address

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

var base32ParityAddresses = []string{
	"GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI",
	"MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACAAAAAAAAAAAAD672",
	"MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQAD7777777777774OFW",
	"MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACABAAAAAAAAAAEVIG",
	"CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQVU2HHGCYSC",
}

// base32ParityInputs returns the valid addresses plus malformed variants of
// each: lowercase and mixed case, a corrupted checksum, a non-alphabet
// character, every truncation (which covers the impossible lengths 1, 3 and
// 6 mod 8 and the non-zero trailing bit cases), and an unknown prefix.
func base32ParityInputs() []string {
	inputs := []string{"", "G", "GA", "GAA", "SAAA", "G=======", "Gé"}
	for _, addr := range base32ParityAddresses {
		inputs = append(inputs,
			addr,
			strings.ToLower(addr),
			addr[:1]+strings.ToLower(addr[1:20])+addr[20:],
			addr[:len(addr)-1]+string(addr[len(addr)-1]^1),
			addr[:10]+"1"+addr[11:],
			addr[:10]+"0"+addr[11:],
			addr[:10]+"="+addr[11:],
			"S"+addr[1:],
			"X"+addr[1:],
			addr+"A",
		)
		for n := 1; n < len(addr); n++ {
			inputs = append(inputs, addr[:n])
		}
		// Flip every character to its neighbour in the alphabet so trailing
		// bits and checksum bytes are both exercised.
		for i := 1; i < len(addr); i++ {
			j := strings.IndexByte(strkeyAlphabet, addr[i])
			flipped := addr[:i] + string(strkeyAlphabet[(j+1)%32]) + addr[i+1:]
			inputs = append(inputs, flipped)
		}
	}
	return inputs
}

// TestDecodeStrKeyIntoMatchesDecodeStrKey checks that the allocation-free
// decoder accepts exactly the inputs DecodeStrKey accepts, returns the same
// version byte and payload, and fails with the same error otherwise.
func TestDecodeStrKeyIntoMatchesDecodeStrKey(t *testing.T) {
	buf := make([]byte, 128)
	for _, in := range base32ParityInputs() {
		wantVB, wantPayload, wantErr := DecodeStrKey(in)
		gotVB, gotPayload, gotErr := DecodeStrKeyInto(buf, in)

		if (wantErr == nil) != (gotErr == nil) {
			t.Fatalf("%q: DecodeStrKey err = %v, DecodeStrKeyInto err = %v", in, wantErr, gotErr)
		}
		if wantErr != nil {
			if !errors.Is(gotErr, wantErr) || gotErr.Error() != wantErr.Error() {
				t.Fatalf("%q: DecodeStrKey err = %v, DecodeStrKeyInto err = %v", in, wantErr, gotErr)
			}
			continue
		}
		if gotVB != wantVB || !bytes.Equal(gotPayload, wantPayload) {
			t.Fatalf("%q: got (0x%02x, %x), want (0x%02x, %x)", in, gotVB, gotPayload, wantVB, wantPayload)
		}
	}
}

func TestDecodeStrKeyIntoPayloadAliasesDst(t *testing.T) {
	const addr = "MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACABAAAAAAAAAAEVIG"
	dst := make([]byte, DecodedStrKeyLen(len(addr)))
	if len(dst) != 43 {
		t.Fatalf("DecodedStrKeyLen(69) = %d, want 43", len(dst))
	}

	_, payload, err := DecodeStrKeyInto(dst, addr)
	if err != nil {
		t.Fatalf("DecodeStrKeyInto: %v", err)
	}
	if len(payload) != 40 || &payload[0] != &dst[1] {
		t.Fatalf("payload should be dst[1:41]; len=%d", len(payload))
	}
}

func TestDecodeStrKeyIntoRejectsShortBuffer(t *testing.T) {
	const addr = "GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI"
	_, _, err := DecodeStrKeyInto(make([]byte, 34), addr)
	if !errors.Is(err, ErrInvalidLengthError) {
		t.Fatalf("err = %v, want invalid length", err)
	}
	if _, _, err := DecodeStrKeyInto(make([]byte, 35), addr); err != nil {
		t.Fatalf("exact-size buffer: %v", err)
	}
}

func TestDecodeStrKeyIntoDoesNotAllocate(t *testing.T) {
	buf := make([]byte, 64)
	inputs := []string{
		"GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI",
		"maycuyt553c5lhve2xpw5gmejt4bxgm7ahmjwlapzp53kjo7eiqacabaaaaaaaaaaevig",
		"GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSJ", // bad checksum
	}
	for _, in := range inputs {
		allocs := testing.AllocsPerRun(100, func() {
			_, _, _ = DecodeStrKeyInto(buf, in)
		})
		if allocs != 0 {
			t.Fatalf("%q: %v allocs/op, want 0", in, allocs)
		}
	}
}

var decodeIntoSink byte

func BenchmarkDecodeStrKeyInto(b *testing.B) {
	const addr = "MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACABAAAAAAAAAAEVIG"
	var buf [43]byte
	b.ReportAllocs()
	var vb byte
	for i := 0; i < b.N; i++ {
		vb, _, _ = DecodeStrKeyInto(buf[:], addr)
	}
	decodeIntoSink = vb
}

func BenchmarkDecodeStrKey(b *testing.B) {
	const addr = "MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACABAAAAAAAAAAEVIG"
	b.ReportAllocs()
	var vb byte
	for i := 0; i < b.N; i++ {
		vb, _, _ = DecodeStrKey(addr)
	}
	decodeIntoSink = vb
}
