import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";

const root = join(import.meta.dirname, "..");
const read = (...parts) => readFileSync(join(root, ...parts), "utf8");

const SHIM = ["frontend", "src", "test", "jest-dom.d.ts"];
const LINT_OVERRIDE_GLOB = "src/test/*.d.ts";
const UPSTREAM_TYPES = [
  "frontend",
  "node_modules",
  "@testing-library",
  "jest-dom",
  "types",
  "vitest.d.ts",
];

// The shape jest-dom 7.0.1 ships: a single-type-parameter augmentation written
// for Vitest 4. It cannot merge with Vitest 5's `Assertion<R, T>`, and
// `skipLibCheck` hides the arity mismatch, so every matcher silently loses its
// type. frontend/src/test/jest-dom.d.ts exists only to cover that gap.
const VITEST_4_SHAPE = /interface\s+Assertion\s*<\s*T\s*=\s*any\s*>/;

test("the jest-dom matcher shim is present exactly while jest-dom needs it", () => {
  const shimPresent = existsSync(join(root, ...SHIM));

  // The shim's empty interface body is what performs the declaration merge, so
  // it needs the scoped lint exemption. Removing either half alone breaks the
  // build, so they travel together.
  const overridePresent = JSON.parse(
    read("frontend", ".oxlintrc.json"),
  ).overrides.some((entry) => (entry.files ?? []).includes(LINT_OVERRIDE_GLOB));

  assert.equal(
    shimPresent,
    overridePresent,
    shimPresent
      ? `${SHIM.join("/")} exists but frontend/.oxlintrc.json has no "${LINT_OVERRIDE_GLOB}" override for typescript/no-empty-object-type; lint will fail on the shim`
      : `frontend/.oxlintrc.json still carries the "${LINT_OVERRIDE_GLOB}" override after ${SHIM.join("/")} was deleted; drop the override too`,
  );

  const upstreamTypes = join(root, ...UPSTREAM_TYPES);
  if (!existsSync(upstreamTypes)) {
    // Frontend dependencies are not installed. CI always installs them before
    // this suite runs; locally, the consistency check above still applies.
    return;
  }

  const upstreamIsStillVitest4 = VITEST_4_SHAPE.test(
    readFileSync(upstreamTypes, "utf8"),
  );

  if (upstreamIsStillVitest4) {
    assert.ok(
      shimPresent,
      `@testing-library/jest-dom still augments the Vitest 4 assertion shape, so ${SHIM.join("/")} is required for the matcher types to resolve`,
    );
    return;
  }

  // Any other shape means upstream changed these types — the release this item
  // was waiting on, or a rewrite that needs re-reading. Nothing else in CI
  // notices: a redundant augmentation keeps working and reddens no check.
  assert.fail(
    `@testing-library/jest-dom no longer ships the Vitest 4 assertion shape this shim compensates for (upstream issue #738).\n` +
      `Delete ${SHIM.join("/")} and the "${LINT_OVERRIDE_GLOB}" override in frontend/.oxlintrc.json, then confirm "npm run type-check" and "npm run lint" stay green in frontend/.\n` +
      `If the matchers still fail to type without the shim, upstream's new shape is incompatible in some other way: update ${VITEST_4_SHAPE} in scripts/jest-dom-shim-policy.test.mjs to match what they now ship, and record why.`,
  );
});
