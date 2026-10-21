# @redishfish/bluewhale-core (Go)

The Go implementation of the Bluewhale for high-performance deposit routing and address interop.

```bash
go get github.com/REDISHFISH/BLUEWHALE/packages/core-go
```

Part of a multi-language suite also available in **[TypeScript](https://github.com/REDISHFISH/BLUEWHALE/tree/main/packages/core-ts)** and **[Dart](https://github.com/REDISHFISH/BLUEWHALE/tree/main/packages/core-dart)**.

---

### 📖 Documentation & Guides
- [Go: Deposit Routing Service Integration](https://github.com/REDISHFISH/BLUEWHALE/blob/main/docs/guides/go-deposit-routing-service.md)
- [Go: Running the Spec Validator](https://github.com/REDISHFISH/BLUEWHALE/blob/main/docs/guides/go-running-spec-validator.md)
- [General: Compatibility Reference](https://github.com/REDISHFISH/BLUEWHALE/blob/main/docs/guides/compatibility-reference.md)

---

## Zero External Dependencies

`core-go` is committed to **zero external dependencies**. Its `go.mod` must
never declare a third-party `require` directive — only the Go standard library
is allowed. This keeps the package auditable, supply-chain safe, and trivially
embeddable.

CI enforces this commitment on every change:

- `go list -m all` must list only the module itself.
- Any external `require` directive in `packages/core-go/go.mod` fails the build.

If you need functionality that would otherwise pull in a dependency, implement
it against the standard library instead.

---

## Quick Start

```go
package main

import (
	"fmt"
	addresskit "github.com/REDISHFISH/BLUEWHALE/packages/core-go/address"
	"github.com/REDISHFISH/BLUEWHALE/packages/core-go/routing"
)

func main() {
	// Validate and detect address types
	addr := "GA7QYNF7SOWQ3GLR2B6RS22TBGZAOR6KLYH4PA5ZAM73A3H4K2HZZSQU"
	if addresskit.IsValid(addr) {
		kind := addresskit.Detect(addr)
		fmt.Printf("Address kind: %s\n", kind)
	}

	// Extract routing information from an incoming payment
	result := routing.ExtractRouting(routing.RoutingInput{
		Destination: "MA7QYNF7SOWQ3GLR2B6RS22TBGZAOR6KLYH4PA5ZAM73A3H4K2HZZSQU...",
		MemoType:    "none",
	})
	
	fmt.Printf("Routing ID: %s\n", result.RoutingID)
}
```

## Performance

Run the benchmarks with:

```bash
go test ./address ./muxed ./routing -run '^$' -bench . -benchmem
```

Baseline numbers from Go 1.22.12 on an Intel Core i7-1185G7 (8 threads,
Windows, amd64). A single run, so expect some noise:

| Benchmark | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkDetect_G` | 278 | 64 | 1 |
| `BenchmarkDetect_M` | 339 | 80 | 1 |
| `BenchmarkDetect_C` | 285 | 64 | 1 |
| `BenchmarkParse_G` | 319 | 144 | 2 |
| `BenchmarkParse_M` | 497 | 224 | 3 |
| `BenchmarkParse_C` | 325 | 144 | 2 |
| `BenchmarkParse_LowercaseG` | 426 | 208 | 3 |
| `BenchmarkParse_BadChecksum` | 317 | 112 | 2 |
| `BenchmarkParse_UnknownPrefix` | 39 | 48 | 1 |
| `BenchmarkDecodeMuxed` | 464 | 144 | 2 |
| `BenchmarkExtractRouting_GAddr_MemoID` | 435 | 288 | 4 |
| `BenchmarkExtractRouting_GAddr_NoMemo` | 417 | 276 | 4 |
| `BenchmarkExtractRouting_MAddr` | 767 | 320 | 6 |
| `BenchmarkExtractRouting_GAddr_MemoID_Parallel` | 187 | 288 | 4 |
| `BenchmarkExtractRouting_MAddr_Parallel` | 377 | 320 | 6 |

Before the allocation work in `address/strkey.go` and `address/parse.go`,
the same machine measured:

| Benchmark | ns/op | allocs/op |
| --- | ---: | ---: |
| `BenchmarkDetect_G` | 349 | 3 |
| `BenchmarkParse_G` | 444 | 4 |
| `BenchmarkParse_M` | 1185 | 11 |
| `BenchmarkParse_UnknownPrefix` | 532 | 8 |
| `BenchmarkDecodeMuxed` | 623 | 7 |
| `BenchmarkExtractRouting_GAddr_MemoID` | 560 | 6 |
| `BenchmarkExtractRouting_MAddr` | 2304 | 21 |

## Examples

### Complete Payment Listener
For a production-ready example of a background worker that listens for payments on the Stellar network and routes them using this kit, see the [go-payment-listener](../../examples/go-payment-listener) in the root repository.

## Documentation

For integration guides and detailed Go examples, see the [Go Guides](https://github.com/REDISHFISH/BLUEWHALE/tree/main/docs/guides).

## License

MIT
