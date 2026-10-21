# Automated Security Scan Workflow (`.github/workflows/security-scan.yml`)

The `.github/workflows/security-scan.yml` workflow enforces automated supply-chain and static security verification across all Bluewhale workspace packages:

1. **TypeScript / npm (`audit-npm`)**: Runs `pnpm audit --audit-level=high` across `@redishfish/bluewhale-core` and `@redishfish/bluewhale`.
2. **Go (`audit-go`)**: Runs `govulncheck ./...` inside `packages/core-go`.
3. **Rust (`audit-rust-prism` & `audit-rust-fuzzer`)**: Runs `cargo audit` on `examples/prism-core` and `examples/rust-address-fuzzer`.
4. **Cross-Language Error & Warning Parity (`#240`, `#241`, `#242`, `#243`)**:
   - Ensures identical warning code enum serialization across TypeScript, Go, and Dart (`#242`).
   - Harmonizes base32 StrKey error message formatting (`#243`) and accessible `aria-invalid` error state styling in `AddressInput` (`#240`, `#241`).
