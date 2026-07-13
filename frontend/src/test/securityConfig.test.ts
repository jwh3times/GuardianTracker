import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

describe("frontend nginx security policy", () => {
  const config = readFileSync(resolve(process.cwd(), "nginx.conf"), "utf8");

  it("allows bundled scripts without allowing inline scripts", () => {
    expect(config).toContain("script-src 'self';");
    expect(config).not.toContain("script-src 'self' 'unsafe-inline'");
  });

  it("allows the configured Google Fonts and denies risky embedding defaults", () => {
    expect(config).toContain(
      "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com;",
    );
    expect(config).toContain("font-src 'self' https://fonts.gstatic.com;");
    expect(config).toContain("object-src 'none';");
    expect(config).toContain("base-uri 'self';");
    expect(config).toContain("frame-ancestors 'self';");
  });
});
