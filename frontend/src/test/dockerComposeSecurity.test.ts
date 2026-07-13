import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

// PR 1 ("Security and Authentication") requires the development backing
// services — Postgres and pgAdmin, plus the throwaway test/e2e Postgres
// instances — to publish their host ports on the loopback interface only, so
// they are never reachable from other machines on the network. api-service
// and frontend are intentionally excluded: they are meant to listen broadly.
const LOOPBACK_ONLY_SERVICES = [
  "postgres",
  "pgadmin",
  "test-postgres",
  "e2e-postgres",
] as const;

/**
 * Returns the raw text of one top-level `services.<name>` block from a
 * Compose file, i.e. everything between the `  <name>:` line and the next
 * line that starts a new top-level service (two-space indent) or a new
 * top-level document key (no indent, e.g. `volumes:`).
 *
 * Intentionally anchored on exact two-space indentation so it does not match
 * same-named keys nested deeper in the file (e.g. `depends_on:\n  postgres:`
 * under another service, which is indented six spaces).
 */
function getServiceBlock(compose: string, serviceName: string): string {
  const startPattern = new RegExp(`^ {2}${serviceName}:[ \\t]*$`, "m");
  const startMatch = startPattern.exec(compose);
  if (!startMatch) {
    throw new Error(`service "${serviceName}" not found in docker-compose.yml`);
  }

  const afterStart = compose.slice(startMatch.index + startMatch[0].length);
  const nextTopLevelKey = /^(?: {2}\S|\S)/m.exec(afterStart);
  return nextTopLevelKey
    ? afterStart.slice(0, nextTopLevelKey.index)
    : afterStart;
}

/**
 * Extracts the list of `ports:` mapping entries from a service block (e.g.
 * `127.0.0.1:${POSTGRES_PORT:-5532}:5432`), stripping YAML list/quote
 * syntax. Resilient to comment lines interleaved in the list and to single-,
 * double-, or unquoted mapping strings.
 */
function getPortMappings(serviceBlock: string): string[] {
  const portsKey = /^( *)ports:[ \t]*$/m.exec(serviceBlock);
  if (!portsKey) {
    throw new Error("ports: key not found in service block");
  }

  const keyIndent = portsKey[1].length;
  const afterKey = serviceBlock.slice(portsKey.index + portsKey[0].length);
  const mappings: string[] = [];

  for (const line of afterKey.split("\n")) {
    if (line.trim() === "") continue;
    const lineIndent = line.length - line.trimStart().length;
    if (lineIndent <= keyIndent) break; // dedented out of the ports list

    const trimmed = line.trim();
    if (trimmed.startsWith("#")) continue;
    if (!trimmed.startsWith("-")) continue;

    const raw = trimmed.slice(1).trim();
    mappings.push(raw.replace(/^["']|["']$/g, ""));
  }

  return mappings;
}

describe("docker-compose backing-service port bindings", () => {
  const compose = readFileSync(
    resolve(process.cwd(), "..", "docker-compose.yml"),
    "utf8",
  );

  it.each(LOOPBACK_ONLY_SERVICES)(
    "publishes %s's port bound to 127.0.0.1 only",
    (serviceName) => {
      const mappings = getPortMappings(getServiceBlock(compose, serviceName));

      expect(mappings.length).toBeGreaterThan(0);
      for (const mapping of mappings) {
        expect(mapping.startsWith("127.0.0.1:")).toBe(true);
      }
    },
  );

  // Proves the assertion above is meaningful rather than a substring search
  // that would pass regardless of content: feed the extractor a service
  // block whose mapping has lost the loopback prefix (the exact regression
  // this test exists to catch) and confirm it reports a non-loopback
  // mapping, which would fail the `.each` assertions above.
  it("flags a port mapping that is missing the 127.0.0.1 prefix", () => {
    const regressed = `  postgres:\n    ports:\n      - "5532:5432"\n  pgadmin:\n`;

    const [mapping] = getPortMappings(getServiceBlock(regressed, "postgres"));

    expect(mapping).toBe("5532:5432");
    expect(mapping.startsWith("127.0.0.1:")).toBe(false);
  });

  it("does not require api-service or frontend to bind loopback-only", () => {
    for (const serviceName of ["api-service", "frontend"]) {
      const mappings = getPortMappings(getServiceBlock(compose, serviceName));
      expect(mappings.length).toBeGreaterThan(0);
      for (const mapping of mappings) {
        expect(mapping.startsWith("127.0.0.1:")).toBe(false);
      }
    }
  });
});
