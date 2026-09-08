// jest-dom's own Vitest entry point still augments the Vitest 4 assertion shape
// (`interface Assertion<T = any>`), which no longer merges with Vitest 5's
// `Assertion<R, T>` — TypeScript requires identical type parameter lists, and
// `skipLibCheck` hides the mismatch, so every matcher silently reads as missing.
// The matchers register correctly at runtime; only their types were lost.
//
// This is Vitest's documented extension point for third-party Jest matcher
// libraries: augment `Matchers`, which both `Assertion` and
// `AsymmetricMatchersContaining` extend. Remove this file once
// @testing-library/jest-dom ships Vitest 5 support (upstream issue #738).
import "vitest";
import type { TestingLibraryMatchers } from "@testing-library/jest-dom/matchers";

declare module "vitest" {
  interface Matchers<R, T> extends TestingLibraryMatchers<unknown, R> {}
}
