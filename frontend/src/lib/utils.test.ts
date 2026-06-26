import { describe, it, expect } from "vitest";
import { cn } from "./utils";

describe("cn", () => {
  it("joins class names", () => {
    expect(cn("a", "b")).toBe("a b");
  });

  it("drops falsy values", () => {
    expect(cn("a", false, undefined, null, "c")).toBe("a c");
  });

  it("keeps truthy conditional class names", () => {
    const includeB: string | false = "b";
    expect(cn("a", includeB && "b", "c")).toBe("a b c");
  });

  it("resolves tailwind conflicts (last wins)", () => {
    expect(cn("p-2", "p-4")).toBe("p-4");
  });
});
