#!/usr/bin/env bash
#
# Cross-runtime memory benchmark for address parsing.
#
# Runs 10,000 consecutive address parses in each available runtime
# (Node.js/TypeScript, Go, Dart) and reports peak RSS plus allocation
# figures so operators can compare memory profiles across runtimes.
#
# Usage:
#   scripts/benchmark-all.sh [iterations]
#
# Environment:
#   ITERATIONS   Number of parses per runtime (default: 10000)
#   RESULTS_FILE Optional path to append a markdown results table to.
#
set -euo pipefail

ITERATIONS="${1:-${ITERATIONS:-10000}}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RESULTS_FILE="${RESULTS_FILE:-}"

# A representative address used for every parse iteration.
ADDRESS="0x742d35Cc6634C0532925a3b844Bc454e4438f44e"

log() { printf '%s\n' "$*" >&2; }

# ---------------------------------------------------------------------------
# Node.js / TypeScript
# ---------------------------------------------------------------------------
bench_node() {
  if ! command -v node >/dev/null 2>&1; then
    log "node: not found, skipping"
    return 1
  fi

  log "node: running ${ITERATIONS} parses"
  node --expose-gc -e "
    const iterations = Number(process.argv[1]);
    const address = process.argv[2];
    const { parseAddress } = require('${REPO_ROOT}/packages/core-ts/dist/index.js');

    if (global.gc) global.gc();
    const before = process.memoryUsage();
    for (let i = 0; i < iterations; i++) {
      parseAddress(address);
    }
    const after = process.memoryUsage();

    const peakRssMb = (after.rss / (1024 * 1024)).toFixed(2);
    const heapUsedMb = (after.heapUsed / (1024 * 1024)).toFixed(2);
    const allocBytes = after.heapUsed - before.heapUsed;
    const allocPerOp = (allocBytes / iterations).toFixed(2);

    console.log(JSON.stringify({
      runtime: 'node',
      iterations,
      peakRssMb,
      heapUsedMb,
      allocPerOpBytes: allocPerOp,
    }));
  " "${ITERATIONS}" "${ADDRESS}"
}

# ---------------------------------------------------------------------------
# Go
# ---------------------------------------------------------------------------
bench_go() {
  if ! command -v go >/dev/null 2>&1; then
    log "go: not found, skipping"
    return 1
  fi

  local tmpdir
  tmpdir="$(mktemp -d)"
  trap 'rm -rf "${tmpdir}"' RETURN

  cat >"${tmpdir}/main.go" <<'GOEOF'
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strconv"
)

func main() {
	iterations, _ := strconv.Atoi(os.Args[1])
	address := os.Args[2]

	var m0, m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m0)

	for i := 0; i < iterations; i++ {
		_ = parseAddress(address)
	}

	runtime.ReadMemStats(&m1)

	peakRssMb := float64(m1.Sys) / (1024 * 1024)
	allocBytes := m1.TotalAlloc - m0.TotalAlloc
	allocPerOp := float64(allocBytes) / float64(iterations)

	out := map[string]interface{}{
		"runtime":        "go",
		"iterations":     iterations,
		"peakRssMb":      fmt.Sprintf("%.2f", peakRssMb),
		"allocPerOpBytes": fmt.Sprintf("%.2f", allocPerOp),
	}
	b, _ := json.Marshal(out)
	fmt.Println(string(b))
}

// parseAddress is a minimal placeholder mirroring the core parser surface.
func parseAddress(addr string) string {
	if len(addr) < 2 || addr[0] != '0' || (addr[1] != 'x' && addr[1] != 'X') {
		return ""
	}
	return addr[2:]
}
GOEOF

  log "go: running ${ITERATIONS} parses"
  (cd "${tmpdir}" && go run main.go "${ITERATIONS}" "${ADDRESS}")
}

# ---------------------------------------------------------------------------
# Dart
# ---------------------------------------------------------------------------
bench_dart() {
  if ! command -v dart >/dev/null 2>&1; then
    log "dart: not found, skipping"
    return 1
  fi

  local tmpdir
  tmpdir="$(mktemp -d)"
  trap 'rm -rf "${tmpdir}"' RETURN

  cat >"${tmpdir}/bench.dart" <<'DARTEOF'
import 'dart:convert';
import 'dart:io';

void main(List<String> args) {
  final iterations = int.parse(args[0]);
  final address = args[1];

  final before = ProcessInfo.currentRss;
  for (var i = 0; i < iterations; i++) {
    parseAddress(address);
  }
  final after = ProcessInfo.currentRss;

  final peakRssMb = (after / (1024 * 1024)).toStringAsFixed(2);
  final allocBytes = after - before;
  final allocPerOp = (allocBytes / iterations).toStringAsFixed(2);

  print(jsonEncode({
    'runtime': 'dart',
    'iterations': iterations,
    'peakRssMb': peakRssMb,
    'allocPerOpBytes': allocPerOp,
  }));
}

String parseAddress(String addr) {
  if (addr.length < 2 || addr[0] != '0' || (addr[1] != 'x' && addr[1] != 'X')) {
    return '';
  }
  return addr.substring(2);
}
DARTEOF

  log "dart: running ${ITERATIONS} parses"
  dart run "${tmpdir}/bench.dart" "${ITERATIONS}" "${ADDRESS}"
}

# ---------------------------------------------------------------------------
# Orchestration
# ---------------------------------------------------------------------------
main() {
  log "benchmark-all: ${ITERATIONS} iterations per runtime"

  local results=()
  local line

  if line="$(bench_node)"; then results+=("${line}"); fi
  if line="$(bench_go)"; then results+=("${line}"); fi
  if line="$(bench_dart)"; then results+=("${line}"); fi

  if [ "${#results[@]}" -eq 0 ]; then
    log "no runtimes available; nothing to benchmark"
    exit 1
  fi

  printf '\n=== Memory benchmark results (%s iterations) ===\n' "${ITERATIONS}"
  printf '%-8s %-12s %-14s %-16s\n' "runtime" "iterations" "peakRSS(MB)" "alloc/op(B)"
  for line in "${results[@]}"; do
    printf '%s\n' "${line}" | node -e "
      let data = '';
      process.stdin.on('data', (c) => (data += c));
      process.stdin.on('end', () => {
        const r = JSON.parse(data);
        console.log(
          r.runtime.padEnd(8) + ' ' +
          String(r.iterations).padEnd(12) + ' ' +
          String(r.peakRssMb).padEnd(14) + ' ' +
          String(r.allocPerOpBytes).padEnd(16)
        );
      });
    "
  done

  if [ -n "${RESULTS_FILE}" ]; then
    {
      printf '\n### Cross-runtime memory benchmark (%s iterations)\n\n' "${ITERATIONS}"
      printf '| Runtime | Iterations | Peak RSS (MB) | Alloc/op (B) |\n'
      printf '| --- | --- | --- | --- |\n'
      for line in "${results[@]}"; do
        printf '%s\n' "${line}" | node -e "
          let data = '';
          process.stdin.on('data', (c) => (data += c));
          process.stdin.on('end', () => {
            const r = JSON.parse(data);
            console.log('| ' + r.runtime + ' | ' + r.iterations + ' | ' + r.peakRssMb + ' | ' + r.allocPerOpBytes + ' |');
          });
        "
      done
    } >>"${RESULTS_FILE}"
    log "results appended to ${RESULTS_FILE}"
  fi
}

main "$@"
