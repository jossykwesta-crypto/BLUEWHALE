import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    globals: true,
    environment: "node",
    include: ["src/**/*.test.ts", "tests/**/*.test.ts"],
    coverage: {
      provider: "v8",
      // Measure shipped source only, so a stray script or a test helper can't
      // move the numbers either way.
      include: ["src/**/*.ts"],
      exclude: ["src/**/*.test.ts", "src/test/**", "src/spec/**"],
      reporter: ["text", "json-summary", "lcov"],
      // Report even when a test fails, so CI shows where coverage stands.
      reportOnFailure: true,
      // Parsing and routing logic must stay fully exercised. CI fails if any
      // of these drop; raise them as coverage improves, never lower them.
      thresholds: {
        statements: 95,
        branches: 95,
        functions: 95,
        lines: 98,
      },
    },
  },
});
