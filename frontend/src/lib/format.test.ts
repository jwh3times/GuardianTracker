import { describe, it, expect, vi, afterEach } from "vitest";
import { relTime } from "./format";

afterEach(() => {
  vi.useRealTimers();
});

describe("relTime", () => {
  const at = (msAgo: number) => new Date(Date.now() - msAgo).toISOString();

  it("returns 'just now' under a minute", () => {
    expect(relTime(at(30_000))).toBe("just now");
  });
  it("returns minutes under an hour", () => {
    expect(relTime(at(5 * 60_000))).toBe("5m ago");
    expect(relTime(at(59 * 60_000))).toBe("59m ago");
  });
  it("returns hours under a day", () => {
    expect(relTime(at(3 * 3_600_000))).toBe("3h ago");
  });
  it("returns days under a month", () => {
    expect(relTime(at(4 * 86_400_000))).toBe("4d ago");
  });
  it("returns months beyond 30 days", () => {
    expect(relTime(at(65 * 86_400_000))).toBe("2mo ago");
  });

  it("reports unknown rather than nonsense for unusable input", () => {
    expect(relTime("")).toBe("unknown");
    expect(relTime("not a date")).toBe("unknown");
    // Go's zero time, which would otherwise render as "24168mo ago".
    expect(relTime("0001-01-01T00:00:00Z")).toBe("unknown");
  });
});
