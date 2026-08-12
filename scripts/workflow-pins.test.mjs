import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";

const root = join(import.meta.dirname, "..");
const workflowsDirectory = join(root, ".github", "workflows");
const workflowFiles = readdirSync(workflowsDirectory)
  .filter((file) => /\.ya?ml$/.test(file))
  .sort();

test("third-party workflow actions use immutable SHAs with release comments", () => {
  const failures = [];

  for (const file of workflowFiles) {
    const contents = readFileSync(join(workflowsDirectory, file), "utf8");

    for (const match of contents.matchAll(
      /^\s*uses:\s*([^\s#]+)(?:\s+#\s*(.+?))?\s*$/gm,
    )) {
      const [, action, comment = ""] = match;
      if (action.startsWith("./") || action.startsWith("docker://")) continue;

      if (!/^[^@\s]+@[0-9a-f]{40}$/.test(action)) {
        failures.push(`${file}: ${action} is not pinned to a full commit SHA`);
      }
      if (!/^v\d+\.\d+\.\d+$/.test(comment)) {
        failures.push(
          `${file}: ${action} needs a readable '# vX.Y.Z' release comment`,
        );
      }
    }
  }

  assert.deepEqual(failures, []);
});

test("the Go vulnerability scanner is a Dependabot-managed versioned tool", () => {
  const workflow = readFileSync(join(workflowsDirectory, "ci-cd.yml"), "utf8");
  const goMod = readFileSync(
    join(root, "backend", "api-service", "go.mod"),
    "utf8",
  );

  assert.doesNotMatch(workflow, /govulncheck@latest/);
  assert.match(workflow, /go tool govulncheck \.\/\.\.\./);
  assert.match(goMod, /^tool golang\.org\/x\/vuln\/cmd\/govulncheck$/m);
  assert.match(
    goMod,
    /^\s*golang\.org\/x\/vuln v\d+\.\d+\.\d+(?: \/\/ indirect)?\s*$/m,
  );
});

test("Dependabot monitors workflow actions and declared Go tools", () => {
  const dependabot = readFileSync(
    join(root, ".github", "dependabot.yml"),
    "utf8",
  );

  assert.match(
    dependabot,
    /package-ecosystem: "github-actions"\s+directory: "\/"/,
  );
  assert.match(
    dependabot,
    /package-ecosystem: "gomod"\s+directory: "\/backend\/api-service"/,
  );
});
