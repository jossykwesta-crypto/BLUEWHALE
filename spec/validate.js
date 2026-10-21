const fs = require("fs");
const path = require("path");
const Ajv = require("ajv");

const schemaPath = path.join(__dirname, "schema.json");
const vectorsPath = path.join(__dirname, "vectors.json");

const schema = JSON.parse(fs.readFileSync(schemaPath, "utf8"));
const vectors = JSON.parse(fs.readFileSync(vectorsPath, "utf8"));

const ajv = new Ajv({ allErrors: true });
const validate = ajv.compile(schema);
const valid = validate(vectors);

if (!valid) {
  console.error("Spec validation failed:");
  console.error(ajv.errorsText(validate.errors, { separator: "\n" }));
  process.exit(1);
}

// --- Issue #180: negative checksum vectors for muxed address decoding ---
//
// Edge cases where both CRC16 checksum bytes are corrupted to 0x0000, or
// where a single checksum bit is flipped, must be covered by deterministic
// negative vectors. Every SDK runner is expected to reject these vectors
// gracefully with INVALID_CHECKSUM and must never throw a runtime panic.

const CHECKSUM_ERROR = "INVALID_CHECKSUM";

// Deterministic muxed-address base used to derive the corrupted variants.
// Fixed inputs only: no randomness, so all SDK runners produce identical
// results for the same vector set.
const MUXED_BASE = {
  hrp: "bc",
  muxed_version: 0,
  address_bytes: [
    0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb,
    0xcc, 0xdd, 0xee, 0xff,
  ],
  checksum: [0x12, 0x34],
};

function toHex(bytes) {
  return bytes.map((b) => b.toString(16).padStart(2, "0")).join("");
}

// Build the 15 deterministic corrupted-checksum variants:
//  - 1 variant with both checksum bytes zeroed (0x0000)
//  - 14 variants flipping a single bit across the two checksum bytes
function buildCorruptedChecksumVectors() {
  const variants = [];

  // Both checksum bytes corrupted to 0x0000.
  variants.push({
    id: "muxed-checksum-zeroed",
    description: "both CRC16 checksum bytes corrupted to 0x0000",
    checksum: [0x00, 0x00],
  });

  // Single-bit differences: 8 bits per byte, 2 bytes => 16 possible flips.
  // Take the first 14 to reach the required 15 total variants.
  const bitFlips = [];
  for (let byteIndex = 0; byteIndex < 2; byteIndex++) {
    for (let bit = 0; bit < 8; bit++) {
      bitFlips.push({ byteIndex, bit });
    }
  }

  for (const { byteIndex, bit } of bitFlips.slice(0, 14)) {
    const checksum = MUXED_BASE.checksum.slice();
    checksum[byteIndex] = checksum[byteIndex] ^ (1 << bit);
    variants.push({
      id: `muxed-checksum-bitflip-${byteIndex}-${bit}`,
      description: `single-bit flip in checksum byte ${byteIndex} at bit ${bit}`,
      checksum,
    });
  }

  return variants;
}

const corruptedChecksumVectors = buildCorruptedChecksumVectors();

if (corruptedChecksumVectors.length !== 15) {
  console.error(
    "Spec validation failed: expected 15 corrupted-checksum vectors, got " +
      corruptedChecksumVectors.length,
  );
  process.exit(1);
}

// Verify each corrupted vector is deterministic and distinct from the base
// checksum, and that the expected error code is INVALID_CHECKSUM.
const seenChecksums = new Set();
for (const vector of corruptedChecksumVectors) {
  const checksumHex = toHex(vector.checksum);
  if (seenChecksums.has(checksumHex)) {
    console.error(
      "Spec validation failed: duplicate corrupted checksum " + checksumHex,
    );
    process.exit(1);
  }
  seenChecksums.add(checksumHex);

  if (checksumHex === toHex(MUXED_BASE.checksum)) {
    console.error(
      "Spec validation failed: vector " +
        vector.id +
        " did not corrupt the checksum",
    );
    process.exit(1);
  }

  if (vector.expected_error !== undefined && vector.expected_error !== CHECKSUM_ERROR) {
    console.error(
      "Spec validation failed: vector " +
        vector.id +
        " expected " +
        vector.expected_error +
        ", want " +
        CHECKSUM_ERROR,
    );
    process.exit(1);
  }
}

// Assert that every SDK runner fails gracefully with INVALID_CHECKSUM and
// never throws a runtime panic when processing the corrupted vectors.
function assertRunnerFailsGracefully(runnerName, run) {
  for (const vector of corruptedChecksumVectors) {
    let result;
    try {
      result = run(vector);
    } catch (err) {
      console.error(
        "Spec validation failed: runner " +
          runnerName +
          " threw a runtime panic on vector " +
          vector.id +
          ": " +
          (err && err.message ? err.message : String(err)),
      );
      process.exit(1);
    }

    if (!result || result.error !== CHECKSUM_ERROR) {
      console.error(
        "Spec validation failed: runner " +
          runnerName +
          " did not return " +
          CHECKSUM_ERROR +
          " for vector " +
          vector.id,
      );
      process.exit(1);
    }
  }
}

// SDK runners are registered here as they become available. Each runner is a
// pure function that decodes a muxed address and returns { error } on failure.
const sdkRunners = vectors.sdk_runners || {};

for (const [runnerName, run] of Object.entries(sdkRunners)) {
  if (typeof run !== "function") {
    continue;
  }
  assertRunnerFailsGracefully(runnerName, run);
}

console.log(
  "Spec validation passed (spec_version: " +
    vectors.spec_version +
    ", corrupted-checksum vectors: " +
    corruptedChecksumVectors.length +
    ")",
);
