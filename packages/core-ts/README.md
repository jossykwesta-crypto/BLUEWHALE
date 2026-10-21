# @redishfish/bluewhale-core (TypeScript)

The TypeScript reference implementation of the Bluewhale for secure deposit routing and address interop.

```bash
npm install @redishfish/bluewhale-core
```

Part of a multi-language suite also available in **[Go](https://github.com/REDISHFISH/BLUEWHALE/tree/main/packages/core-go)** and **[Dart](https://github.com/REDISHFISH/BLUEWHALE/tree/main/packages/core-dart)**.

---

### 📖 Documentation & Guides
- [TypeScript: Reconciling Deposits with Missing Memos](https://github.com/REDISHFISH/BLUEWHALE/blob/main/docs/guides/reconciling-deposits-missing-memo.md)
- [TypeScript: Pooled Accounts & Muxed Deposits](https://github.com/REDISHFISH/BLUEWHALE/blob/main/docs/guides/pooled-accounts-muxed-deposits.md)
- [General: Compatibility Reference](https://github.com/REDISHFISH/BLUEWHALE/blob/main/docs/guides/compatibility-reference.md)

---

## Quick Start

```typescript
import { extractRouting } from '@redishfish/bluewhale-core';

const result = extractRouting({
  destination: 'MA7QYNF7SOWQ3GLR2B6RS22TBGZAOR6KLYH4PA5ZAM73A3H4K2HZZSQUVRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRRTD6',
  memoType: 'none',
  memoValue: null,
  sourceAccount: 'GA...'
});

if (!result.destinationError) {
  console.log('Routing ID:', result.routingId); // "123456789"
  console.log('Source:', result.routingSource); // "muxed"
}
```

## Bun and Deno

The package ships explicit `bun` and `deno` export conditions, both pointing
at the ESM build, so no extra configuration is needed.

```bash
# Bun
bun add @redishfish/bluewhale-core @stellar/stellar-sdk

# Deno (2.x)
deno add npm:@redishfish/bluewhale-core npm:@stellar/stellar-sdk
```

```typescript
// Bun: plain package import
import { extractRouting } from '@redishfish/bluewhale-core';

// Deno: npm specifier (or the bare name once added with `deno add`)
import { extractRouting } from 'npm:@redishfish/bluewhale-core';
```

`@stellar/stellar-sdk` is a peer dependency and must be installed alongside.
To run the runtime smoke tests from a checkout, build first:

```bash
pnpm --filter @redishfish/bluewhale-core build
pnpm --filter @redishfish/bluewhale-core test:bun
pnpm --filter @redishfish/bluewhale-core test:deno
```

## Tree-shaking

`package.json` declares `"sideEffects": false`, so bundlers (webpack, Rollup,
esbuild, Vite, Next.js) drop every module you do not import. Importing only
`extractRouting` stays under 5 KB gzipped (excluding the
`@stellar/stellar-sdk` peer); `src/test/bundle-size.test.ts` enforces this.

## Documentation

For full guides, integration examples, and deep dives into the routing logic, see our [comprehensive Guides](https://github.com/REDISHFISH/BLUEWHALE/tree/main/docs/guides).

## License

MIT
