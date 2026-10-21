package muxed

import (
	"errors"
	"strings"
	"testing"

	"github.com/REDISHFISH/BLUEWHALE/packages/core-go/address"
)

const (
	poolTestGAddr = "GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI"
	poolTestMAddr = "MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACABAAAAAAAAAAEVIG"
	poolTestMMax  = "MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQAD7777777777774OFW"
	poolTestMZero = "MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACAAAAAAAAAAAAD672"
)

func TestDecodeMuxedRaw(t *testing.T) {
	tests := []struct {
		in string
		id uint64
	}{
		{poolTestMAddr, 9007199254740993},
		{poolTestMMax, ^uint64(0)},
		{poolTestMZero, 0},
		{strings.ToLower(poolTestMAddr), 9007199254740993},
	}

	_, wantKey, err := address.DecodeStrKey(poolTestGAddr)
	if err != nil {
		t.Fatalf("DecodeStrKey(G): %v", err)
	}

	for _, tt := range tests {
		pubkey, id, err := DecodeMuxedRaw(tt.in)
		if err != nil {
			t.Fatalf("DecodeMuxedRaw(%s): %v", tt.in, err)
		}
		if id != tt.id {
			t.Fatalf("DecodeMuxedRaw(%s) id = %d, want %d", tt.in, id, tt.id)
		}
		if string(pubkey[:]) != string(wantKey) {
			t.Fatalf("DecodeMuxedRaw(%s) pubkey = %x, want %x", tt.in, pubkey, wantKey)
		}

		baseG, gotID, err := DecodeMuxed(tt.in)
		if err != nil || baseG != poolTestGAddr || gotID != tt.id {
			t.Fatalf("DecodeMuxed(%s) = (%s, %d, %v), want (%s, %d, nil)", tt.in, baseG, gotID, err, poolTestGAddr, tt.id)
		}
	}
}

// TestDecodeMuxedErrorsUnchanged checks that every malformed input still
// fails with the error the pre-pool decoder (decodeMuxedChecked) reports.
func TestDecodeMuxedErrorsUnchanged(t *testing.T) {
	inputs := []string{
		"",
		"M",
		poolTestGAddr, // valid strkey, wrong version byte
		poolTestMAddr[:len(poolTestMAddr)-1] + "E",    // checksum mismatch
		poolTestMAddr[:20] + "1" + poolTestMAddr[21:], // not base32
		poolTestMAddr + "A",                           // impossible length
		poolTestMAddr[:len(poolTestMAddr)-8],          // truncated
		"S" + poolTestMAddr[1:],                       // unknown prefix
		"CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQVU2HHGCYSC",
	}

	for _, in := range inputs {
		_, _, wantErr := decodeMuxedChecked(in)
		if wantErr == nil {
			t.Fatalf("%q: expected the checked decoder to fail", in)
		}

		_, _, gotErr := DecodeMuxed(in)
		var want, got *AddressError
		if !errors.As(wantErr, &want) || !errors.As(gotErr, &got) {
			t.Fatalf("%q: errors are not *AddressError: %v / %v", in, wantErr, gotErr)
		}
		if got.Code != want.Code || got.Error() != want.Error() || got.Input != in {
			t.Fatalf("%q: DecodeMuxed err = %+v, want %+v", in, got, want)
		}

		if _, _, rawErr := DecodeMuxedRaw(in); rawErr == nil || rawErr.Error() != want.Error() {
			t.Fatalf("%q: DecodeMuxedRaw err = %v, want %v", in, rawErr, want)
		}
	}
}

// TestDecodeMuxedPoolBuffersAreCleared checks that buffers go back to the
// pool zeroed, so a later Get never sees a previous address's bytes.
func TestDecodeMuxedPoolBuffersAreCleared(t *testing.T) {
	for i := 0; i < 8; i++ {
		if _, _, err := DecodeMuxedRaw(poolTestMMax); err != nil {
			t.Fatalf("DecodeMuxedRaw: %v", err)
		}
	}

	bufs := make([]*[muxedRawLen]byte, 0, 8)
	for i := 0; i < 8; i++ {
		buf := muxedBufPool.Get().(*[muxedRawLen]byte)
		if *buf != ([muxedRawLen]byte{}) {
			t.Fatalf("pooled buffer was not cleared: %x", buf[:])
		}
		bufs = append(bufs, buf)
	}
	for _, buf := range bufs {
		muxedBufPool.Put(buf)
	}
}

// TestDecodeMuxedRawDoesNotAllocate is the hot-path guarantee: decoding a
// well-formed M-address costs zero heap allocations once the pool is warm.
//
// The race detector deliberately drops a share of sync.Pool puts, so the
// allocation count is only meaningful without -race.
func TestDecodeMuxedRawDoesNotAllocate(t *testing.T) {
	if raceEnabled {
		t.Skip("sync.Pool drops items at random under the race detector")
	}

	for _, in := range []string{poolTestMAddr, poolTestMMax, strings.ToLower(poolTestMZero)} {
		allocs := testing.AllocsPerRun(1000, func() {
			_, _, _ = DecodeMuxedRaw(in)
		})
		if allocs != 0 {
			t.Fatalf("DecodeMuxedRaw(%s): %v allocs/op, want 0", in, allocs)
		}
	}
}

// TestDecodeMuxedAllocatesOnlyTheResult pins DecodeMuxed at one allocation:
// the returned G-address string.
func TestDecodeMuxedAllocatesOnlyTheResult(t *testing.T) {
	if raceEnabled {
		t.Skip("sync.Pool drops items at random under the race detector")
	}

	allocs := testing.AllocsPerRun(1000, func() {
		_, _, _ = DecodeMuxed(poolTestMAddr)
	})
	if allocs > 1 {
		t.Fatalf("DecodeMuxed: %v allocs/op, want at most 1", allocs)
	}
}

var benchRawSink [32]byte

// BenchmarkDecodeMuxedRaw measures the pooled hot path. It reports
// 0 allocs/op:
//
//	go test ./muxed -run '^$' -bench 'DecodeMuxed' -benchmem
func BenchmarkDecodeMuxedRaw(b *testing.B) {
	if _, _, err := DecodeMuxedRaw(poolTestMAddr); err != nil {
		b.Fatalf("DecodeMuxedRaw unexpected error: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	var pubkey [32]byte
	var id uint64
	for i := 0; i < b.N; i++ {
		pubkey, id, _ = DecodeMuxedRaw(poolTestMAddr)
	}
	benchRawSink = pubkey
	benchDecodeSink = id
}

// BenchmarkDecodeMuxedParallel runs the pooled decoder from many goroutines,
// the shape of a webhook handler under load.
func BenchmarkDecodeMuxedParallel(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, _, err := DecodeMuxedRaw(poolTestMAddr); err != nil {
				b.Error(err)
				return
			}
		}
	})
}
