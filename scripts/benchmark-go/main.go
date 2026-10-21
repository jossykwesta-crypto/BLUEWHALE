// Command benchmark-go measures peak RSS memory and allocations for 10,000
// consecutive address parses. It is invoked by scripts/benchmark-all.sh as the
// Go leg of the cross-runtime (TS/Node, Go, Dart) memory benchmark for issue #246.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"
)

const iterations = 10000

// sampleAddresses are valid Stellar-style addresses parsed in a round-robin
// fashion across the benchmark iterations.
var sampleAddresses = []string{
	"GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF5",
	"GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
	"GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUWDA",
	"GDNSSYSCSSJ76FER5WEEXME5G4MTCUBKDRQSKOYP36KUKVDB2VCMERS6",
}

// parseAddress performs a lightweight structural validation of an address.
// It mirrors the shape checks the TS and Dart benchmarks apply so the three
// runtimes measure comparable work per operation.
func parseAddress(addr string) (string, error) {
	if len(addr) != 56 {
		return "", fmt.Errorf("invalid address length: %d", len(addr))
	}
	if addr[0] != 'G' {
		return "", fmt.Errorf("invalid address prefix: %c", addr[0])
	}
	for i := 1; i < len(addr); i++ {
		c := addr[i]
		if !((c >= 'A' && c <= 'Z') || (c >= '2' && c <= '7')) {
			return "", fmt.Errorf("invalid base32 character at %d: %c", i, c)
		}
	}
	return addr, nil
}

func main() {
	var mStart, mEnd runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&mStart)

	start := time.Now()
	parsed := 0
	for i := 0; i < iterations; i++ {
		addr := sampleAddresses[i%len(sampleAddresses)]
		if _, err := parseAddress(addr); err != nil {
			fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
			os.Exit(1)
		}
		parsed++
	}
	duration := time.Since(start)

	runtime.ReadMemStats(&mEnd)

	result := map[string]interface{}{
		"runtime":          "go",
		"iterations":       iterations,
		"parsed":           parsed,
		"duration_ms":      float64(duration.Microseconds()) / 1000.0,
		"alloc_bytes":      mEnd.TotalAlloc - mStart.TotalAlloc,
		"alloc_objects":    mEnd.Mallocs - mStart.Mallocs,
		"peak_rss_bytes":   mEnd.Sys,
		"heap_inuse_bytes": mEnd.HeapInuse,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode result: %v\n", err)
		os.Exit(1)
	}
}
