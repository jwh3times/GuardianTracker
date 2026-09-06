import { test } from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import {
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  readdirSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";

const root = join(import.meta.dirname, "..");
const secretMarker = "GUARDIAN_SYNTHETIC_BUILD_CONTEXT_SECRET";

// Docker's matcher is the oracle. The only workspace contents read by this
// harness are the two .dockerignore policies. Everything copied by the probe,
// including the source-shaped files, is synthetic and contains no credentials.
const excluded = [
  ".env",
  ".env.local",
  ".env.production",
  ".env.development",
  ".env.test",
  ".env.production.local",
  ".env.production.backup",
  ".env.example.local",
  "production.env",
  "production.env.backup",
  ".envrc",
  "config/.env",
  "config/nested/.env.production",
  "config/nested/production.env",
  "config/nested/.env.example",
  "token.pem",
  "token.key",
  "certificate.crt",
  "certificate.cer",
  "certificate.der",
  "certificate.p12",
  "certificate.pfx",
  "certificate.jks",
  "certificate.keystore",
  "config/nested/token.key",
  "config/id_rsa",
  "config/id_ed25519.pub",
  "id_ecdsa",
  "config/nested/id_dsa",
  ".ssh/config",
  ".aws/credentials",
  ".azure/accessTokens.json",
  ".kube/config",
  "secrets/credentials.json",
  ".secrets/credentials.json",
  "config/app.secrets.yaml",
  "certs/local.txt",
  "certificates/local.txt",
  "private/notes.md",
  ".private-workspace/repository.env.ref",
  "data/manifest.json",
  "config/nested/data/.env",
  "manifest.db",
  "manifest.db-wal",
  "manifest.db-shm",
  "manifest.sqlite",
  "manifest.sqlite-wal",
  "manifest.sqlite3",
  "manifest.sqlite3-shm",
  "logs/access.txt",
  "access.log",
  "access.log.1",
  "npm-debug.log.1",
  "yarn-error.log.1",
  ".cache/snapshot.json",
  "coverage",
  "report.cover",
  "report.coverprofile",
  "node_modules/fixture/index.js",
  "dist/bundle.js",
  "build/binary",
  "tmp/working.txt",
  "temp/working.txt",
  ".tmp/working.txt",
  "local.tmp",
  "local.bak",
  "coverage.out",
  "service.test",
  "service.exe",
  "tsconfig.tsbuildinfo",
  ".git/config",
  ".gitignore",
  ".idea/workspace.xml",
  ".vscode/settings.json",
  ".DS_Store",
  "Thumbs.db",
  "main.go.swp",
  "main.go.swo",
  "main.go~",
];

const contexts = [
  {
    directory: "backend/api-service",
    previousPolicy: "",
    required: [
      ".env.example",
      "go.mod",
      "go.sum",
      "main.go",
      "api/router.go",
      "auth/crypto.go",
      "db/migrate.go",
      "db/migrations/0001_init.sql",
      "db/migrations/0007_onboarding.sql",
      "services/bungie/manifest.go",
      "services/data/source.go",
      "cmd/fake-bungie/main.go",
    ],
    excluded: [
      ".e2e/manifest.db",
      "vendor/module/source.go",
      ".air.toml",
      "main",
    ],
  },
  {
    directory: "frontend",
    previousPolicy: ".env\n.env.local\n.env.*.local\n",
    required: [
      ".env.example",
      ".npmrc",
      "package.json",
      "package-lock.json",
      "index.html",
      "nginx.conf",
      "vite.config.ts",
      "tsconfig.json",
      "postcss.config.cjs",
      "tailwind.config.cjs",
      "src/main.tsx",
      "src/data/wishlist.ts",
      "src/lib/browserSessionClient.ts",
      "public/fixture.svg",
      "e2e/fixtures.ts",
      "e2e/fixtures/source.json",
    ],
    excluded: [
      "playwright/.auth/user.json",
      "playwright-report/index.html",
      "test-results/result.json",
      "nested/playwright/.auth/user.json",
    ],
  },
];

function writeFixture(directory, path, content) {
  const target = join(directory, path);
  mkdirSync(dirname(target), { recursive: true });
  writeFileSync(target, content);
}

function assertNoSecretContent(directory) {
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) {
      assertNoSecretContent(path);
    } else {
      assert.equal(
        readFileSync(path).includes(Buffer.from(secretMarker)),
        false,
        `synthetic secret content reached exported context: ${path}`,
      );
    }
  }
}

function exportContext(input, output) {
  // scratch needs no image pull and executes no application code. The local
  // exporter exposes exactly what COPY could put into an application layer.
  const result = spawnSync(
    "docker",
    [
      "buildx",
      "build",
      "--progress=plain",
      "--file",
      "-",
      "--output",
      `type=local,dest=${output}`,
      input,
    ],
    {
      input: "FROM scratch\nCOPY . /context/\n",
      encoding: "utf8",
      timeout: 120_000,
      maxBuffer: 8 * 1024 * 1024,
    },
  );
  assert.equal(
    result.status,
    0,
    `Docker context probe failed: ${result.error ?? ""}\n${result.stdout}\n${result.stderr}`,
  );
}

for (const context of contexts) {
  test(`${context.directory} excludes local credentials and retains build inputs using Docker`, () => {
    // Dockerfile-specific policies override the context policy. Require an
    // explicit extension of these probes before introducing such an override.
    assert.deepEqual(
      readdirSync(join(root, context.directory)).filter(
        (name) => name !== ".dockerignore" && name.endsWith(".dockerignore"),
      ),
      [],
      "Dockerfile-specific ignore files need their own context isolation probe",
    );
    const temporary = mkdtempSync(join(tmpdir(), "guardian-docker-context-"));
    try {
      const input = join(temporary, "input");
      const output = join(temporary, "output");
      mkdirSync(input);
      writeFixture(
        input,
        ".dockerignore",
        readFileSync(join(root, context.directory, ".dockerignore")),
      );
      for (const path of context.required) {
        writeFixture(input, path, `synthetic build input: ${path}\n`);
      }
      const forbidden = [...excluded, ...context.excluded];
      for (const path of forbidden) {
        writeFixture(input, path, `${secretMarker}: ${path}\n`);
      }

      // These synthetic controls reproduce the missing backend policy and the
      // previous frontend env rules. A probe that never copies anything must
      // not masquerade as a successful security regression check.
      writeFixture(input, ".dockerignore", context.previousPolicy);
      const before = join(temporary, "before");
      exportContext(input, before);
      assert.ok(
        readFileSync(
          join(before, "context", ".env.production"),
          "utf8",
        ).includes(secretMarker),
        "pre-fix policy must expose the synthetic production env sentinel",
      );
      writeFixture(
        input,
        ".dockerignore",
        readFileSync(join(root, context.directory, ".dockerignore")),
      );
      exportContext(input, output);
      const copied = join(output, "context");
      for (const path of context.required) {
        assert.equal(
          readFileSync(join(copied, path), "utf8"),
          `synthetic build input: ${path}\n`,
          `required build input was excluded: ${path}`,
        );
      }
      for (const path of forbidden) {
        assert.equal(
          existsSync(join(copied, path)),
          false,
          `not excluded: ${path}`,
        );
      }
      assertNoSecretContent(copied);
    } finally {
      rmSync(temporary, { recursive: true, force: true });
    }
  });
}
