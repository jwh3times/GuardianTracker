import { test } from "node:test";
import assert from "node:assert/strict";
import {
  chmodSync,
  copyFileSync,
  existsSync,
  mkdirSync,
  mkdtempSync,
  readdirSync,
  readFileSync,
  rmSync,
  symlinkSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { delimiter, join } from "node:path";
import { spawnSync } from "node:child_process";

const projectRoot = join(import.meta.dirname, "..");
const bootstrapScript = join(
  projectRoot,
  "scripts",
  "bootstrap-private-workspace.ps1",
);
const statusScript = join(projectRoot, "scripts", "workspace-status.ps1");
const restoreScript = join(
  projectRoot,
  "scripts",
  "restore-private-secrets.ps1",
);
const powershell =
  process.env.PORTABILITY_POWERSHELL ??
  (process.platform === "win32" ? "powershell.exe" : "pwsh");

function run(command, args, options = {}) {
  return spawnSync(command, args, {
    encoding: "utf8",
    ...options,
  });
}

function git(root, ...args) {
  const result = run("git", ["-c", "core.excludesFile=", "-C", root, ...args]);
  assert.equal(
    result.status,
    0,
    `git ${args.join(" ")} failed: ${result.stderr}`,
  );
  return result.stdout.trim();
}

function makePublicFixture() {
  const root = mkdtempSync(join(tmpdir(), "guardian-portability-"));
  git(root, "init");
  git(root, "symbolic-ref", "HEAD", "refs/heads/main");
  git(root, "config", "user.name", "Portability Test");
  git(root, "config", "user.email", "portability@example.invalid");
  writeFileSync(
    join(root, ".gitignore"),
    [
      ".env",
      ".env.*",
      "!.env.example",
      "k8s/*-secret.yaml",
      "private/",
      ".private-workspace/",
      ".private-bootstrap-*/",
      "",
    ].join("\n"),
  );
  writeFileSync(join(root, "README.md"), "fixture\n");
  mkdirSync(join(root, "backend", "api-service"), { recursive: true });
  mkdirSync(join(root, "frontend"), { recursive: true });
  mkdirSync(join(root, "k8s"), { recursive: true });
  git(root, "add", ".gitignore", "README.md");
  git(root, "commit", "-m", "fixture baseline");
  return root;
}

function runPowerShell(script, root, extraArgs = [], env = {}) {
  return run(
    powershell,
    [
      "-NoProfile",
      ...(process.platform === "win32" ? ["-ExecutionPolicy", "Bypass"] : []),
      "-File",
      script,
      "-RepositoryRoot",
      root,
      ...extraArgs,
    ],
    {
      env: {
        ...process.env,
        GIT_TRACE: "1",
        GIT_TRACE_CURL: "1",
        GIT_CURL_VERBOSE: "1",
        ...env,
      },
    },
  );
}

function combinedOutput(result) {
  return `${result.stdout}\n${result.stderr}`;
}

function createFailingOp(root, sentinel) {
  const privateConfig = join(root, ".private-workspace");
  mkdirSync(privateConfig, { recursive: true });
  if (process.platform === "win32") {
    const path = join(privateConfig, "fake-op.cmd");
    writeFileSync(path, `@echo off\r\necho ${sentinel} 1>&2\r\nexit /b 23\r\n`);
    return path;
  }

  const path = join(privateConfig, "fake-op");
  writeFileSync(path, `#!/bin/sh\necho '${sentinel}' >&2\nexit 23\n`);
  chmodSync(path, 0o700);
  return path;
}

function createUnavailableOp(root) {
  const executableDirectory = join(
    root,
    ".private-workspace",
    "unavailable-op-bin",
  );
  mkdirSync(executableDirectory, { recursive: true });
  if (process.platform === "win32") {
    writeFileSync(join(executableDirectory, "op.cmd"), "@exit /b 23\r\n");
  } else {
    const path = join(executableDirectory, "op");
    writeFileSync(path, "#!/bin/sh\nexit 23\n");
    chmodSync(path, 0o700);
  }
  return executableDirectory;
}

function createAuthorizationFailingOp(root) {
  const executableDirectory = join(
    root,
    ".private-workspace",
    "authorization-failing-op-bin",
  );
  mkdirSync(executableDirectory, { recursive: true });
  if (process.platform === "win32") {
    writeFileSync(
      join(executableDirectory, "op.cmd"),
      '@if "%1"=="--version" exit /b 0\r\n@exit /b 23\r\n',
    );
  } else {
    const path = join(executableDirectory, "op");
    writeFileSync(
      path,
      '#!/bin/sh\nif [ "$1" = "--version" ]; then exit 0; fi\nexit 23\n',
    );
    chmodSync(path, 0o700);
  }
  return executableDirectory;
}

function createMarkerOp(root) {
  const privateConfig = join(root, ".private-workspace");
  mkdirSync(privateConfig, { recursive: true });
  const markerPath = join(privateConfig, "op-invoked.marker");
  if (process.platform === "win32") {
    const path = join(privateConfig, "marker-op.cmd");
    writeFileSync(
      path,
      `@echo off\r\ntype nul > "${markerPath}"\r\nexit /b 23\r\n`,
    );
    return { path, markerPath };
  }

  const path = join(privateConfig, "marker-op");
  writeFileSync(path, `#!/bin/sh\n: > '${markerPath}'\nexit 23\n`);
  chmodSync(path, 0o700);
  return { path, markerPath };
}

function createInspectingOp(root) {
  const privateConfig = join(root, ".private-workspace");
  const executableDirectory = join(privateConfig, "fake-op-bin");
  const markerPath = join(privateConfig, "op-env-file.marker");
  mkdirSync(executableDirectory, { recursive: true });
  if (process.platform === "win32") {
    copyFileSync(process.execPath, join(executableDirectory, "op.exe"));
    writeFileSync(
      join(root, "run"),
      [
        'const { readFileSync, writeFileSync } = require("node:fs");',
        `const markerPath = ${JSON.stringify(markerPath)};`,
        "const args = process.argv.slice(2);",
        'const envFileIndex = args.indexOf("--env-file");',
        "if (envFileIndex < 0 || !args[envFileIndex + 1]) process.exit(24);",
        'writeFileSync(markerPath, readFileSync(args[envFileIndex + 1], "utf8"));',
        "process.exit(23);",
        "",
      ].join("\n"),
    );
  } else {
    const implementationPath = join(executableDirectory, "inspect-op.mjs");
    writeFileSync(
      implementationPath,
      [
        'import { readFileSync, writeFileSync } from "node:fs";',
        `const markerPath = ${JSON.stringify(markerPath)};`,
        "const args = process.argv.slice(2);",
        'if (args[0] === "--version") process.exit(0);',
        'const envFileIndex = args.indexOf("--env-file");',
        "if (envFileIndex < 0 || !args[envFileIndex + 1]) process.exit(24);",
        'writeFileSync(markerPath, readFileSync(args[envFileIndex + 1], "utf8"));',
        "process.exit(23);",
        "",
      ].join("\n"),
    );
    const executablePath = join(executableDirectory, "op");
    writeFileSync(
      executablePath,
      `#!/bin/sh\nexec '${process.execPath.replaceAll("'", "'\\''")}' "\${0%/*}/inspect-op.mjs" "$@"\n`,
    );
    chmodSync(executablePath, 0o700);
  }
  return { executableDirectory, markerPath };
}

function createFakeGit(root) {
  const privateConfig = join(root, ".private-workspace");
  mkdirSync(privateConfig, { recursive: true });
  const path = join(privateConfig, "fake-git.ps1");
  const logPath = join(privateConfig, "git-arguments.log");
  const environmentLogPath = join(privateConfig, "git-environment.log");
  const traceEnvironmentLogPath = join(
    privateConfig,
    "git-trace-environment.log",
  );
  const onePasswordEnvironmentLogPath = join(
    privateConfig,
    "git-onepassword-environment.log",
  );
  writeFileSync(
    path,
    [
      "$argumentsList = @($args)",
      `$logPath = '${logPath.replaceAll("'", "''")}'`,
      `$environmentLogPath = '${environmentLogPath.replaceAll("'", "''")}'`,
      `$traceEnvironmentLogPath = '${traceEnvironmentLogPath.replaceAll("'", "''")}'`,
      `$onePasswordEnvironmentLogPath = '${onePasswordEnvironmentLogPath.replaceAll("'", "''")}'`,
      "[IO.File]::AppendAllText($logPath, ($argumentsList -join [char]31) + [Environment]::NewLine)",
      "$environmentValue = [Environment]::GetEnvironmentVariable('GUARDIAN_PRIVATE_REPOSITORY_URL', 'Process')",
      "[IO.File]::AppendAllText($environmentLogPath, ([string]$environmentValue) + [Environment]::NewLine)",
      "$traceNames = @('GIT_TRACE2', 'GIT_TRACE2_EVENT', 'GIT_TRACE2_PERF')",
      "$traceValues = @($traceNames | ForEach-Object { [Environment]::GetEnvironmentVariable($_, 'Process') })",
      "[IO.File]::AppendAllText($traceEnvironmentLogPath, ($traceValues -join '|') + [Environment]::NewLine)",
      "$onePasswordNames = @('OP_SERVICE_ACCOUNT_TOKEN', 'OP_CONNECT_TOKEN', 'OP_SESSION_fixture')",
      "$onePasswordValues = @($onePasswordNames | ForEach-Object { if ([string]::IsNullOrEmpty([Environment]::GetEnvironmentVariable($_, 'Process'))) { 'missing' } else { 'present' } })",
      "[IO.File]::AppendAllText($onePasswordEnvironmentLogPath, ($onePasswordValues -join '|') + [Environment]::NewLine)",
      "if ($argumentsList -contains 'clone') {",
      "  $target = $argumentsList[$argumentsList.Count - 1]",
      "  [IO.Directory]::CreateDirectory((Join-Path $target '.git')) | Out-Null",
      '  [IO.File]::WriteAllText((Join-Path $target \'.git/config\'), "[core]`n`trepositoryformatversion = 0`n[remote `"origin`"]`n`turl = guardian-private:`n`tfetch = +refs/heads/*:refs/remotes/origin/*`n")',
      "} elseif ($argumentsList -contains '--is-inside-work-tree') {",
      "  Write-Output 'true'",
      "} elseif ($argumentsList -contains '--show-toplevel') {",
      "  $rootIndex = [Array]::IndexOf($argumentsList, '-C') + 1",
      "  Write-Output $argumentsList[$rootIndex]",
      "}",
      "$global:LASTEXITCODE = 0",
      "",
    ].join("\n"),
  );
  return {
    path,
    logPath,
    environmentLogPath,
    traceEnvironmentLogPath,
    onePasswordEnvironmentLogPath,
  };
}

function createForwardingOp(root, repositoryUrl, gitExecutable) {
  const privateConfig = join(root, ".private-workspace");
  mkdirSync(privateConfig, { recursive: true });
  const path = join(privateConfig, "forwarding-op.ps1");
  writeFileSync(
    path,
    [
      "$separator = -1",
      "for ($index = 0; $index -lt $args.Count; $index++) {",
      "  if ($args[$index] -eq '--') { $separator = $index; break }",
      "}",
      "$commandIndex = $separator + 1",
      "if ($separator -lt 0) {",
      "  for ($index = 0; $index + 2 -lt $args.Count; $index++) {",
      "    if ($args[$index] -eq '--env-file') { $commandIndex = $index + 2; break }",
      "  }",
      "}",
      "if ($commandIndex -le 0 -or $commandIndex -ge $args.Count) { $global:LASTEXITCODE = 24; return }",
      "$command = $args[$commandIndex]",
      "$commandArguments = @()",
      "for ($index = $commandIndex + 1; $index -lt $args.Count; $index++) {",
      "  $commandArguments += $args[$index]",
      "}",
      "for ($index = 0; $index + 1 -lt $commandArguments.Count; $index++) {",
      `  if ($commandArguments[$index] -eq '-GitExecutable') { $commandArguments[$index + 1] = '${gitExecutable.replaceAll("'", "''")}'; break }`,
      "}",
      `$env:GUARDIAN_PRIVATE_REPOSITORY_URL = '${repositoryUrl.replaceAll("'", "''")}'`,
      "& $command @commandArguments",
      "$childExitCode = $LASTEXITCODE",
      "[Environment]::SetEnvironmentVariable('GUARDIAN_PRIVATE_REPOSITORY_URL', $null, 'Process')",
      "$global:LASTEXITCODE = $childExitCode",
      "",
    ].join("\n"),
  );
  return path;
}

function createPartialInjectOp(root, noisySentinel) {
  const privateConfig = join(root, ".private-workspace");
  mkdirSync(privateConfig, { recursive: true });
  if (process.platform === "win32") {
    const path = join(privateConfig, "partial-op.cmd");
    writeFileSync(
      path,
      [
        "@echo off",
        ":scan",
        'if "%~1"=="" goto missing',
        'if "%~1"=="-o" goto output',
        "shift",
        "goto scan",
        ":output",
        "shift",
        "echo PARTIAL-SECRET-SENTINEL>%~1",
        `echo ${noisySentinel} 1>&2`,
        "exit /b 23",
        ":missing",
        "exit /b 24",
        "",
      ].join("\r\n"),
    );
    return path;
  }

  const path = join(privateConfig, "partial-op");
  writeFileSync(
    path,
    [
      "#!/bin/sh",
      'while [ "$#" -gt 0 ]; do',
      '  if [ "$1" = "-o" ]; then',
      "    shift",
      "    printf '%s\\n' 'PARTIAL-SECRET-SENTINEL' > \"$1\"",
      `    printf '%s\\n' '${noisySentinel}' >&2`,
      "    exit 23",
      "  fi",
      "  shift",
      "done",
      "exit 24",
      "",
    ].join("\n"),
  );
  chmodSync(path, 0o700);
  return path;
}

function createRootInjectOp(root, { malformed = false } = {}) {
  const privateConfig = join(root, ".private-workspace");
  mkdirSync(privateConfig, { recursive: true });
  const lines = malformed
    ? [
        'GO_ENV="development',
        "BUNGIE_API_KEY=value",
        "BUNGIE_CLIENT_ID=value",
        "JWT_SECRET=value",
        "POSTGRES_PASSWORD=value",
        "TOKEN_ENCRYPTION_KEY=value",
        "TOKEN_ENCRYPTION_KEY_VERSION=1",
      ]
    : [
        "GO_ENV=development",
        "BUNGIE_API_KEY=value",
        "BUNGIE_CLIENT_ID=value",
        "JWT_SECRET=value",
        "POSTGRES_PASSWORD=value",
        "TOKEN_ENCRYPTION_KEY=value",
        "TOKEN_ENCRYPTION_KEY_VERSION=1",
      ];

  if (process.platform === "win32") {
    const path = join(
      privateConfig,
      malformed ? "malformed-op.cmd" : "valid-op.cmd",
    );
    writeFileSync(
      path,
      [
        "@echo off",
        ":scan",
        'if "%~1"=="" goto missing',
        'if "%~1"=="-o" goto output',
        "shift",
        "goto scan",
        ":output",
        "shift",
        "(\r\n" +
          lines.map((line) => `echo ${line}`).join("\r\n") +
          "\r\n)>%~1",
        "exit /b 0",
        ":missing",
        "exit /b 24",
        "",
      ].join("\r\n"),
    );
    return path;
  }

  const path = join(privateConfig, malformed ? "malformed-op" : "valid-op");
  writeFileSync(
    path,
    [
      "#!/bin/sh",
      'while [ "$#" -gt 0 ]; do',
      '  if [ "$1" = "-o" ]; then',
      "    shift",
      "    {",
      ...lines.map((line) => `      printf '%s\\n' '${line}'`),
      '    } > "$1"',
      "    exit 0",
      "  fi",
      "  shift",
      "done",
      "exit 24",
      "",
    ].join("\n"),
  );
  chmodSync(path, 0o700);
  return path;
}

function createMalformedK8sOp(root) {
  const privateConfig = join(root, ".private-workspace");
  mkdirSync(privateConfig, { recursive: true });
  const lines = [
    "apiVersion: v1",
    "kind: Secret",
    "metadata:",
    "  name: api-service-secrets",
    "  namespace: default",
    "type: Opaque",
    "stringData:",
    "  BUNGIE_API_KEY: value",
    "  BUNGIE_CLIENT_ID: value",
    "  JWT_SECRET: value",
    "  TOKEN_ENCRYPTION_KEY: value",
    "[malformed-yaml-sentinel",
  ];

  if (process.platform === "win32") {
    const path = join(privateConfig, "malformed-k8s-op.cmd");
    writeFileSync(
      path,
      [
        "@echo off",
        ":scan",
        'if "%~1"=="" goto missing',
        'if "%~1"=="-o" goto output',
        "shift",
        "goto scan",
        ":output",
        "shift",
        "(\r\n" +
          lines.map((line) => `echo ${line}`).join("\r\n") +
          "\r\n)>%~1",
        "exit /b 0",
        ":missing",
        "exit /b 24",
        "",
      ].join("\r\n"),
    );
    return path;
  }

  const path = join(privateConfig, "malformed-k8s-op");
  writeFileSync(
    path,
    [
      "#!/bin/sh",
      'while [ "$#" -gt 0 ]; do',
      '  if [ "$1" = "-o" ]; then',
      "    shift",
      "    {",
      ...lines.map((line) => `      printf '%s\\n' '${line}'`),
      '    } > "$1"',
      "    exit 0",
      "  fi",
      "  shift",
      "done",
      "exit 24",
      "",
    ].join("\n"),
  );
  chmodSync(path, 0o700);
  return path;
}

function createK8sScalarOp(root, scalar) {
  const privateConfig = join(root, ".private-workspace");
  mkdirSync(privateConfig, { recursive: true });
  const lines = [
    "apiVersion: v1",
    "kind: Secret",
    "metadata:",
    "  name: api-service-secrets",
    "  namespace: default",
    "type: Opaque",
    "stringData:",
    `  BUNGIE_API_KEY: ${scalar}`,
    `  BUNGIE_CLIENT_ID: ${scalar}`,
    `  JWT_SECRET: ${scalar}`,
    `  TOKEN_ENCRYPTION_KEY: ${scalar}`,
  ];

  if (process.platform === "win32") {
    const path = join(privateConfig, "scalar-k8s-op.cmd");
    writeFileSync(
      path,
      [
        "@echo off",
        ":scan",
        'if "%~1"=="" goto missing',
        'if "%~1"=="-o" goto output',
        "shift",
        "goto scan",
        ":output",
        "shift",
        "(\r\n" +
          lines.map((line) => `echo ${line}`).join("\r\n") +
          "\r\n)>%~1",
        "exit /b 0",
        ":missing",
        "exit /b 24",
        "",
      ].join("\r\n"),
    );
    return path;
  }

  const path = join(privateConfig, "scalar-k8s-op");
  writeFileSync(
    path,
    [
      "#!/bin/sh",
      'while [ "$#" -gt 0 ]; do',
      '  if [ "$1" = "-o" ]; then',
      "    shift",
      "    {",
      ...lines.map((line) => `      printf '%s\\n' '${line}'`),
      '    } > "$1"',
      "    exit 0",
      "  fi",
      "  shift",
      "done",
      "exit 24",
      "",
    ].join("\n"),
  );
  chmodSync(path, 0o700);
  return path;
}

function initializePrivateTemplates(root, names = ["root.env.tpl"]) {
  const privateRoot = join(root, "private");
  const templateRoot = join(privateRoot, "bootstrap", "templates");
  mkdirSync(templateRoot, { recursive: true });
  git(privateRoot, "init");
  for (const name of names) {
    writeFileSync(join(templateRoot, name), "value-free template fixture\n");
  }
  return privateRoot;
}

test("bootstrap defaults to public-only and never requires private access", () => {
  const root = makePublicFixture();
  try {
    const result = runPowerShell(bootstrapScript, root);
    assert.equal(result.status, 0, combinedOutput(result));
    assert.match(result.stdout, /Public workspace is ready/);
    assert.equal(result.stderr, "");
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("bootstrap recognizes an existing independent private repository", () => {
  const root = makePublicFixture();
  try {
    const privateRoot = join(root, "private");
    mkdirSync(privateRoot);
    git(privateRoot, "init");
    const result = runPowerShell(bootstrapScript, root);
    assert.equal(result.status, 0, combinedOutput(result));
    assert.match(result.stdout, /clone already present/);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("bootstrap refuses a populated private directory without listing it", () => {
  const root = makePublicFixture();
  const sentinel = "PRIVATE-FILENAME-SENTINEL";
  try {
    mkdirSync(join(root, "private"));
    writeFileSync(join(root, "private", sentinel), "preserve me\n");
    const result = runPowerShell(
      bootstrapScript,
      root,
      ["-InternalCloneFromEnvironment"],
      {
        GUARDIAN_PRIVATE_REPOSITORY_URL:
          "https://github.com/example/companion.git",
      },
    );
    assert.notEqual(result.status, 0);
    assert.doesNotMatch(combinedOutput(result), new RegExp(sentinel));
    assert.equal(
      readFileSync(join(root, "private", sentinel), "utf8"),
      "preserve me\n",
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("failed 1Password authorization is sanitized and changes no files", () => {
  const root = makePublicFixture();
  const referenceSentinel = "op://PRIVATE-VAULT/PRIVATE-ITEM/PRIVATE-FIELD";
  const noisySentinel = "OP-ERROR-SECRET-SENTINEL";
  try {
    mkdirSync(join(root, ".private-workspace"));
    writeFileSync(
      join(root, ".private-workspace", "repository.env.ref"),
      `GUARDIAN_PRIVATE_REPOSITORY_URL=${referenceSentinel}\n`,
    );
    const fakeOp = createFailingOp(root, noisySentinel);
    const result = runPowerShell(bootstrapScript, root, [
      "-PrivateFromOnePassword",
      "-OpExecutable",
      fakeOp,
    ]);
    assert.notEqual(result.status, 0);
    const output = combinedOutput(result);
    assert.doesNotMatch(output, new RegExp(noisySentinel));
    assert.doesNotMatch(output, /PRIVATE-VAULT|PRIVATE-ITEM|PRIVATE-FIELD/);
    assert.equal(run("git", ["-C", root, "status", "--short"]).stdout, "");
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("bootstrap accepts a quoted whitespace-bearing 1Password reference", () => {
  const root = makePublicFixture();
  const referenceSentinel = "op://PRIVATE VAULT/PRIVATE ITEM/PRIVATE FIELD";
  try {
    mkdirSync(join(root, ".private-workspace"));
    writeFileSync(
      join(root, ".private-workspace", "repository.env.ref"),
      `GUARDIAN_PRIVATE_REPOSITORY_URL="${referenceSentinel}"\n`,
    );
    const fakeOp = createMarkerOp(root);
    const result = runPowerShell(bootstrapScript, root, [
      "-PrivateFromOnePassword",
      "-OpExecutable",
      fakeOp.path,
    ]);
    assert.notEqual(result.status, 0);
    assert.equal(existsSync(fakeOp.markerPath), true);
    assert.doesNotMatch(
      combinedOutput(result),
      /must contain exactly one approved variable mapping/,
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("Node bootstrap quotes a whitespace-bearing 1Password reference", () => {
  const root = makePublicFixture();
  const referenceSentinel = "op://PRIVATE VAULT/PRIVATE ITEM/PRIVATE FIELD";
  try {
    mkdirSync(join(root, "scripts"));
    writeFileSync(
      join(root, "scripts", "bootstrap-private.mjs"),
      readFileSync(join(projectRoot, "scripts", "bootstrap-private.mjs")),
    );
    const fakeOp = createInspectingOp(root);
    const result = run(
      process.execPath,
      [
        join(root, "scripts", "bootstrap-private.mjs"),
        "--op-reference",
        referenceSentinel,
      ],
      {
        cwd: root,
        env: {
          ...process.env,
          PATH: `${fakeOp.executableDirectory}${delimiter}${process.env.PATH}`,
        },
      },
    );
    assert.ifError(result.error);
    assert.notEqual(result.status, 0);
    assert.equal(existsSync(fakeOp.markerPath), true, combinedOutput(result));
    assert.equal(
      readFileSync(fakeOp.markerPath, "utf8"),
      `GUARDIAN_PRIVATE_REPOSITORY_URL="${referenceSentinel}"\n`,
    );
    assert.doesNotMatch(
      combinedOutput(result),
      /--op-reference must be a single/,
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("Node bootstrap removes its temporary reference after an unavailable 1Password CLI", () => {
  const root = makePublicFixture();
  const temporaryRoot = join(root, "temporary");
  try {
    mkdirSync(join(root, "scripts"));
    mkdirSync(temporaryRoot);
    writeFileSync(
      join(root, "scripts", "bootstrap-private.mjs"),
      readFileSync(join(projectRoot, "scripts", "bootstrap-private.mjs")),
    );
    const executableDirectory = createUnavailableOp(root);
    const result = run(
      process.execPath,
      [
        join(root, "scripts", "bootstrap-private.mjs"),
        "--op-reference",
        "op://PRIVATE-VAULT/PRIVATE-ITEM/PRIVATE-FIELD",
      ],
      {
        cwd: root,
        env: {
          ...process.env,
          PATH: `${executableDirectory}${delimiter}${process.env.PATH}`,
          TEMP: temporaryRoot,
          TMP: temporaryRoot,
          TMPDIR: temporaryRoot,
        },
      },
    );
    assert.ifError(result.error);
    assert.notEqual(result.status, 0);
    assert.match(combinedOutput(result), /1Password CLI is not available/);
    assert.deepEqual(readdirSync(temporaryRoot), []);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("Node bootstrap removes its temporary reference after failed 1Password authorization", () => {
  const root = makePublicFixture();
  const temporaryRoot = join(root, "temporary");
  try {
    mkdirSync(join(root, "scripts"));
    mkdirSync(temporaryRoot);
    writeFileSync(
      join(root, "scripts", "bootstrap-private.mjs"),
      readFileSync(join(projectRoot, "scripts", "bootstrap-private.mjs")),
    );
    const executableDirectory = createAuthorizationFailingOp(root);
    const result = run(
      process.execPath,
      [
        join(root, "scripts", "bootstrap-private.mjs"),
        "--op-reference",
        "op://PRIVATE-VAULT/PRIVATE-ITEM/PRIVATE-FIELD",
      ],
      {
        cwd: root,
        env: {
          ...process.env,
          PATH: `${executableDirectory}${delimiter}${process.env.PATH}`,
          TEMP: temporaryRoot,
          TMP: temporaryRoot,
          TMPDIR: temporaryRoot,
        },
      },
    );
    assert.ifError(result.error);
    assert.notEqual(result.status, 0);
    assert.match(
      combinedOutput(result),
      /1Password authorization or private workspace setup failed/,
    );
    assert.deepEqual(readdirSync(temporaryRoot), []);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("Node bootstrap accepts a quoted whitespace-bearing reference file", () => {
  const root = makePublicFixture();
  const referenceSentinel = "OP://PRIVATE VAULT/PRIVATE ITEM/PRIVATE FIELD";
  try {
    mkdirSync(join(root, "scripts"));
    writeFileSync(
      join(root, "scripts", "bootstrap-private.mjs"),
      readFileSync(join(projectRoot, "scripts", "bootstrap-private.mjs")),
    );
    mkdirSync(join(root, ".private-workspace"));
    const assignment = `GUARDIAN_PRIVATE_REPOSITORY_URL="${referenceSentinel}"\n`;
    writeFileSync(
      join(root, ".private-workspace", "repository.env.ref"),
      assignment,
    );
    const fakeOp = createInspectingOp(root);
    const result = run(
      process.execPath,
      [join(root, "scripts", "bootstrap-private.mjs")],
      {
        cwd: root,
        env: {
          ...process.env,
          PATH: `${fakeOp.executableDirectory}${delimiter}${process.env.PATH}`,
        },
      },
    );
    assert.ifError(result.error);
    assert.notEqual(result.status, 0);
    assert.equal(existsSync(fakeOp.markerPath), true, combinedOutput(result));
    assert.equal(readFileSync(fakeOp.markerPath, "utf8"), assignment);
    assert.doesNotMatch(
      combinedOutput(result),
      /reference file must contain exactly one approved variable mapping/,
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("bootstrap rejects extra 1Password environment mappings before authorization", () => {
  const root = makePublicFixture();
  const noisySentinel = "OP-MUST-NOT-RUN-EXTRA-MAPPING";
  try {
    mkdirSync(join(root, ".private-workspace"));
    writeFileSync(
      join(root, ".private-workspace", "repository.env.ref"),
      [
        "GUARDIAN_PRIVATE_REPOSITORY_URL=op://<vault>/<item>/<field>",
        "UNRELATED_SECRET=op://<vault>/<item>/<other-field>",
        "",
      ].join("\n"),
    );
    const fakeOp = createFailingOp(root, noisySentinel);
    const result = runPowerShell(bootstrapScript, root, [
      "-PrivateFromOnePassword",
      "-OpExecutable",
      fakeOp,
    ]);
    assert.notEqual(result.status, 0);
    assert.doesNotMatch(combinedOutput(result), new RegExp(noisySentinel));
    assert.equal(run("git", ["-C", root, "status", "--short"]).stdout, "");
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("bootstrap evaluates committed ignore negations before invoking 1Password", () => {
  const root = makePublicFixture();
  try {
    const ignorePath = join(root, ".gitignore");
    const committedIgnore = `${readFileSync(ignorePath, "utf8")}!.private-workspace/\n!.private-workspace/repository.env.ref\n`;
    writeFileSync(ignorePath, committedIgnore);
    git(root, "add", ".gitignore");
    git(root, "commit", "-m", "negate private reference protection");
    writeFileSync(
      ignorePath,
      committedIgnore.replace(
        "!.private-workspace/\n!.private-workspace/repository.env.ref\n",
        "",
      ),
    );
    mkdirSync(join(root, ".private-workspace"), { recursive: true });
    writeFileSync(
      join(root, ".private-workspace", "repository.env.ref"),
      "GUARDIAN_PRIVATE_REPOSITORY_URL=op://<vault>/<item>/<field>\n",
    );
    const fakeOp = createMarkerOp(root);
    const result = runPowerShell(bootstrapScript, root, [
      "-PrivateFromOnePassword",
      "-OpExecutable",
      fakeOp.path,
    ]);
    assert.notEqual(result.status, 0);
    assert.equal(existsSync(fakeOp.markerPath), false);
    assert.match(
      combinedOutput(result),
      /private repository reference file is not protected/,
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("bootstrap rejects unsafe repository transports before invoking Git", () => {
  const root = makePublicFixture();
  try {
    const result = runPowerShell(
      bootstrapScript,
      root,
      ["-InternalCloneFromEnvironment"],
      { GUARDIAN_PRIVATE_REPOSITORY_URL: "ext::PRIVATE-HELPER-SENTINEL" },
    );
    assert.notEqual(result.status, 0);
    assert.doesNotMatch(combinedOutput(result), /PRIVATE-HELPER-SENTINEL/);
    assert.equal(run("git", ["-C", root, "status", "--short"]).stdout, "");
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("bootstrap keeps a resolved URL out of the initial Git arguments", () => {
  const root = makePublicFixture();
  const repositoryUrl =
    "https://github.com/example/PRIVATE-REPOSITORY-SENTINEL.git";
  try {
    const fakeGit = createFakeGit(root);
    const result = runPowerShell(
      bootstrapScript,
      root,
      ["-InternalCloneFromEnvironment", "-GitExecutable", fakeGit.path],
      { GUARDIAN_PRIVATE_REPOSITORY_URL: repositoryUrl },
    );
    assert.equal(result.status, 0, combinedOutput(result));
    assert.doesNotMatch(combinedOutput(result), /PRIVATE-REPOSITORY-SENTINEL/);
    assert.doesNotMatch(
      readFileSync(fakeGit.logPath, "utf8"),
      /PRIVATE-REPOSITORY-SENTINEL/,
    );
    assert.doesNotMatch(
      readFileSync(fakeGit.environmentLogPath, "utf8"),
      /PRIVATE-REPOSITORY-SENTINEL/,
    );
    assert.match(
      readFileSync(join(root, "private", ".git", "config"), "utf8"),
      /PRIVATE-REPOSITORY-SENTINEL/,
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("bootstrap disables Trace2 for every Git invocation", () => {
  const root = makePublicFixture();
  const repositoryUrl = "https://github.com/example/private-workspace.git";
  try {
    const privateConfig = join(root, ".private-workspace");
    mkdirSync(privateConfig);
    const tracePath = join(privateConfig, "global-trace.json");
    const configPath = join(privateConfig, "gitconfig");
    writeFileSync(
      configPath,
      `[trace2]\n\teventTarget = ${tracePath.replaceAll("\\", "/")}\n`,
    );
    const fakeGit = createFakeGit(root);
    const result = runPowerShell(
      bootstrapScript,
      root,
      ["-InternalCloneFromEnvironment", "-GitExecutable", fakeGit.path],
      {
        GIT_CONFIG_GLOBAL: configPath,
        GUARDIAN_PRIVATE_REPOSITORY_URL: repositoryUrl,
      },
    );
    assert.equal(result.status, 0, combinedOutput(result));
    const traceLines = readFileSync(fakeGit.traceEnvironmentLogPath, "utf8")
      .trim()
      .split(/\r?\n/);
    assert.ok(traceLines.length > 0);
    assert.ok(traceLines.every((line) => line === "0|0|0"));
    assert.equal(existsSync(tracePath), false);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("bootstrap strips 1Password credentials from every Git invocation", () => {
  const root = makePublicFixture();
  const repositoryUrl = "https://github.com/example/private-workspace.git";
  try {
    const fakeGit = createFakeGit(root);
    const result = runPowerShell(
      bootstrapScript,
      root,
      ["-InternalCloneFromEnvironment", "-GitExecutable", fakeGit.path],
      {
        GUARDIAN_PRIVATE_REPOSITORY_URL: repositoryUrl,
        OP_SERVICE_ACCOUNT_TOKEN: "SERVICE-ACCOUNT-TOKEN-SENTINEL",
        OP_CONNECT_TOKEN: "CONNECT-TOKEN-SENTINEL",
        OP_SESSION_fixture: "SESSION-TOKEN-SENTINEL",
      },
    );
    assert.equal(result.status, 0, combinedOutput(result));
    const environmentLines = readFileSync(
      fakeGit.onePasswordEnvironmentLogPath,
      "utf8",
    )
      .trim()
      .split(/\r?\n/);
    assert.ok(environmentLines.length > 0);
    assert.ok(
      environmentLines.every((line) => line === "missing|missing|missing"),
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("1Password mode forwards only the resolved URL into the internal bootstrap", () => {
  const root = makePublicFixture();
  const repositoryUrl =
    "https://github.com/example/PRIVATE-OP-FORWARD-SENTINEL.git";
  try {
    mkdirSync(join(root, ".private-workspace"));
    writeFileSync(
      join(root, ".private-workspace", "repository.env.ref"),
      'GUARDIAN_PRIVATE_REPOSITORY_URL="op://TEST VAULT/TEST ITEM/TEST FIELD"\n',
    );
    const fakeGit = createFakeGit(root);
    const fakeOp = createForwardingOp(root, repositoryUrl, fakeGit.path);
    const result = runPowerShell(bootstrapScript, root, [
      "-PrivateFromOnePassword",
      "-OpExecutable",
      fakeOp,
    ]);
    assert.equal(result.status, 0, combinedOutput(result));
    assert.match(result.stdout, /installed through 1Password/);
    assert.doesNotMatch(combinedOutput(result), /PRIVATE-OP-FORWARD-SENTINEL/);
    assert.doesNotMatch(
      readFileSync(fakeGit.environmentLogPath, "utf8"),
      /PRIVATE-OP-FORWARD-SENTINEL/,
    );
    assert.match(
      readFileSync(join(root, "private", ".git", "config"), "utf8"),
      /PRIVATE-OP-FORWARD-SENTINEL/,
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("status reports only high-level state when private work is absent or uninitialized", () => {
  const root = makePublicFixture();
  const filenameSentinel = "PRIVATE-DOCUMENT-SENTINEL.md";
  try {
    let result = runPowerShell(statusScript, root);
    assert.equal(result.status, 0, combinedOutput(result));
    assert.match(result.stdout, /Public repository: branch main; clean/);
    assert.match(result.stdout, /Private repository: missing/);
    assert.match(result.stdout, /\.env: missing; ignored by repository rule/);

    writeFileSync(join(root, "PUBLIC-DIRTY-SENTINEL.txt"), "dirty\n");
    mkdirSync(join(root, "private"));
    writeFileSync(join(root, "private", filenameSentinel), "private\n");
    result = runPowerShell(statusScript, root);
    assert.equal(result.status, 0, combinedOutput(result));
    assert.match(result.stdout, /Public repository: branch main; dirty/);
    assert.match(
      result.stdout,
      /Private repository: present but not initialized/,
    );
    assert.doesNotMatch(
      combinedOutput(result),
      /PUBLIC-DIRTY-SENTINEL|PRIVATE-DOCUMENT-SENTINEL/,
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("status strips 1Password credentials from every Git invocation", () => {
  const root = makePublicFixture();
  try {
    const fakeGit = createFakeGit(root);
    runPowerShell(statusScript, root, ["-GitExecutable", fakeGit.path], {
      OP_SERVICE_ACCOUNT_TOKEN: "SERVICE-ACCOUNT-TOKEN-SENTINEL",
      OP_CONNECT_TOKEN: "CONNECT-TOKEN-SENTINEL",
      OP_SESSION_fixture: "SESSION-TOKEN-SENTINEL",
    });
    const environmentLines = readFileSync(
      fakeGit.onePasswordEnvironmentLogPath,
      "utf8",
    )
      .trim()
      .split(/\r?\n/);
    assert.ok(environmentLines.length > 0);
    assert.ok(
      environmentLines.every((line) => line === "missing|missing|missing"),
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("status evaluates the complete committed ignore semantics", () => {
  const root = makePublicFixture();
  try {
    const ignorePath = join(root, ".gitignore");
    const committedIgnore = `${readFileSync(ignorePath, "utf8")}!.env\n`;
    writeFileSync(ignorePath, committedIgnore);
    git(root, "add", ".gitignore");
    git(root, "commit", "-m", "negate root environment protection");
    writeFileSync(ignorePath, committedIgnore.replace("!.env\n", ""));
    const result = runPowerShell(statusScript, root);
    assert.notEqual(result.status, 0);
    assert.match(result.stdout, /\.env: missing; UNSAFE/);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("status redacts private Git metadata and reports independent dirty state", () => {
  const root = makePublicFixture();
  const branchSentinel = "finding-PRIVATE-BRANCH-SENTINEL";
  const commitSentinel = "PRIVATE-COMMIT-SENTINEL";
  const fileSentinel = "PRIVATE-FILE-SENTINEL.md";
  const remoteSentinel = "PRIVATE-REMOTE-SENTINEL";
  try {
    const privateRoot = join(root, "private");
    mkdirSync(privateRoot);
    git(privateRoot, "init");
    git(privateRoot, "symbolic-ref", "HEAD", `refs/heads/${branchSentinel}`);
    git(privateRoot, "config", "user.name", "Portability Test");
    git(privateRoot, "config", "user.email", "portability@example.invalid");
    writeFileSync(join(privateRoot, "README.md"), "private fixture\n");
    git(privateRoot, "add", "README.md");
    git(privateRoot, "commit", "-m", commitSentinel);
    git(
      privateRoot,
      "remote",
      "add",
      "origin",
      `https://github.com/example/${remoteSentinel}.git`,
    );
    writeFileSync(join(privateRoot, fileSentinel), "dirty\n");

    const result = runPowerShell(statusScript, root);
    assert.equal(result.status, 0, combinedOutput(result));
    assert.match(
      result.stdout,
      /Private repository: branch checked out \(name redacted\); dirty; no upstream configured/,
    );
    assert.doesNotMatch(
      combinedOutput(result),
      /PRIVATE-BRANCH-SENTINEL|PRIVATE-COMMIT-SENTINEL|PRIVATE-FILE-SENTINEL|PRIVATE-REMOTE-SENTINEL/,
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("status fails generically for invalid private Git metadata", () => {
  const root = makePublicFixture();
  const metadataSentinel = "INVALID-GIT-METADATA-SENTINEL";
  try {
    mkdirSync(join(root, "private", ".git"), { recursive: true });
    writeFileSync(
      join(root, "private", ".git", metadataSentinel),
      "not a repository\n",
    );
    const result = runPowerShell(statusScript, root);
    assert.notEqual(result.status, 0);
    assert.match(
      result.stdout,
      /Private repository: invalid or not independent/,
    );
    assert.doesNotMatch(combinedOutput(result), new RegExp(metadataSentinel));
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("status fails when plaintext targets rely on no committed ignore rule", () => {
  const root = makePublicFixture();
  try {
    writeFileSync(join(root, ".gitignore"), "private/\n");
    const result = runPowerShell(statusScript, root);
    assert.notEqual(result.status, 0);
    assert.match(result.stdout, /UNSAFE: no repository ignore rule/);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("status rejects an uncommitted replacement for a required ignore rule", () => {
  const root = makePublicFixture();
  try {
    const gitignore = readFileSync(join(root, ".gitignore"), "utf8");
    writeFileSync(
      join(root, ".gitignore"),
      gitignore.replace(/^\.env$/m, "/.env"),
    );
    const result = runPowerShell(statusScript, root);
    assert.notEqual(result.status, 0);
    assert.match(result.stdout, /UNSAFE: no repository ignore rule/);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("status disables global Git Trace2 targets", () => {
  const root = makePublicFixture();
  try {
    const privateConfig = join(root, ".private-workspace");
    mkdirSync(privateConfig);
    const tracePath = join(privateConfig, "global-trace.json");
    const configPath = join(privateConfig, "gitconfig");
    writeFileSync(
      configPath,
      `[trace2]\n\teventTarget = ${tracePath.replaceAll("\\", "/")}\n`,
    );
    const result = runPowerShell(statusScript, root, [], {
      GIT_CONFIG_GLOBAL: configPath,
    });
    assert.equal(result.status, 0, combinedOutput(result));
    assert.equal(existsSync(tracePath), false);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test(
  "status rejects a Windows junction used as the private repository root",
  { skip: process.platform !== "win32" },
  () => {
    const root = makePublicFixture();
    const privatePath = join(root, "private");
    try {
      const junctionTarget = join(
        root,
        ".private-workspace",
        "junction-target",
      );
      mkdirSync(junctionTarget, { recursive: true });
      git(junctionTarget, "init");
      symlinkSync(junctionTarget, privatePath, "junction");
      const result = runPowerShell(statusScript, root);
      assert.notEqual(result.status, 0);
      assert.match(
        result.stdout,
        /Private repository: invalid or not independent/,
      );
    } finally {
      if (existsSync(privatePath)) {
        rmSync(privatePath, { recursive: true, force: true });
      }
      rmSync(root, { recursive: true, force: true });
    }
  },
);

test(
  "bootstrap rejects a Windows junction used as the private workspace root",
  { skip: process.platform !== "win32" },
  () => {
    const root = makePublicFixture();
    const privatePath = join(root, "private");
    try {
      const fakeGit = createFakeGit(root);
      const junctionTarget = join(
        root,
        ".private-workspace",
        "bootstrap-junction-target",
      );
      mkdirSync(junctionTarget, { recursive: true });
      symlinkSync(junctionTarget, privatePath, "junction");
      const result = runPowerShell(
        bootstrapScript,
        root,
        ["-InternalCloneFromEnvironment", "-GitExecutable", fakeGit.path],
        {
          GUARDIAN_PRIVATE_REPOSITORY_URL:
            "https://github.com/example/private-workspace.git",
        },
      );
      assert.notEqual(result.status, 0);
      assert.match(combinedOutput(result), /reparse-point safety check/);
      assert.doesNotMatch(readFileSync(fakeGit.logPath, "utf8"), /clone/);
      assert.equal(existsSync(join(junctionTarget, ".git")), false);
    } finally {
      if (existsSync(privatePath)) {
        rmSync(privatePath, { recursive: true, force: true });
      }
      rmSync(root, { recursive: true, force: true });
    }
  },
);

test(
  "1Password bootstrap rejects a reparse ancestor for its reference file",
  { skip: process.platform !== "win32" },
  () => {
    const root = makePublicFixture();
    const privateConfig = join(root, ".private-workspace");
    const junctionTarget = mkdtempSync(
      join(tmpdir(), "guardian-reference-junction-"),
    );
    const noisySentinel = "OP-MUST-NOT-RUN-REPARSE-SENTINEL";
    try {
      symlinkSync(junctionTarget, privateConfig, "junction");
      writeFileSync(
        join(privateConfig, "repository.env.ref"),
        "GUARDIAN_PRIVATE_REPOSITORY_URL=op://<vault>/<item>/<field>\n",
      );
      const fakeOp = createFailingOp(root, noisySentinel);
      const result = runPowerShell(bootstrapScript, root, [
        "-PrivateFromOnePassword",
        "-OpExecutable",
        fakeOp,
      ]);
      assert.notEqual(result.status, 0);
      assert.match(combinedOutput(result), /local path safety check/);
      assert.doesNotMatch(combinedOutput(result), new RegExp(noisySentinel));
    } finally {
      if (existsSync(privateConfig)) {
        rmSync(privateConfig, { recursive: true, force: true });
      }
      rmSync(junctionTarget, { recursive: true, force: true });
      rmSync(root, { recursive: true, force: true });
    }
  },
);

test(
  "secret restoration rejects a Windows junction used as the private root",
  { skip: process.platform !== "win32" },
  () => {
    const root = makePublicFixture();
    const privatePath = join(root, "private");
    const noisySentinel = "OP-MUST-NOT-RUN-RESTORE-REPARSE";
    try {
      const junctionTarget = join(
        root,
        ".private-workspace",
        "restore-junction-target",
      );
      mkdirSync(join(junctionTarget, "bootstrap", "templates"), {
        recursive: true,
      });
      git(junctionTarget, "init");
      writeFileSync(
        join(junctionTarget, "bootstrap", "templates", "root.env.tpl"),
        "BUNGIE_API_KEY=op://<vault>/<item>/<field>\n",
      );
      symlinkSync(junctionTarget, privatePath, "junction");
      const fakeOp = createFailingOp(root, noisySentinel);
      const result = runPowerShell(restoreScript, root, [
        "-Target",
        "root",
        "-OpExecutable",
        fakeOp,
      ]);
      assert.notEqual(result.status, 0);
      assert.match(combinedOutput(result), /reparse-point safety check/);
      assert.doesNotMatch(combinedOutput(result), new RegExp(noisySentinel));
      assert.equal(existsSync(join(root, ".env")), false);
    } finally {
      if (existsSync(privatePath)) {
        rmSync(privatePath, { recursive: true, force: true });
      }
      rmSync(root, { recursive: true, force: true });
    }
  },
);

test("secret restoration never overwrites an existing target", () => {
  const root = makePublicFixture();
  const existingSentinel = "EXISTING-PLAINTEXT-SENTINEL";
  try {
    const privateRoot = join(root, "private");
    mkdirSync(join(privateRoot, "bootstrap", "templates"), { recursive: true });
    git(privateRoot, "init");
    writeFileSync(
      join(privateRoot, "bootstrap", "templates", "root.env.tpl"),
      "BUNGIE_API_KEY=op://<vault>/<item>/<field>\n",
    );
    writeFileSync(join(root, ".env"), `${existingSentinel}\n`);
    const fakeOp = createFailingOp(root, "OP-MUST-NOT-RUN-SENTINEL");
    const result = runPowerShell(restoreScript, root, [
      "-Target",
      "root",
      "-OpExecutable",
      fakeOp,
    ]);
    assert.notEqual(result.status, 0);
    assert.equal(
      readFileSync(join(root, ".env"), "utf8"),
      `${existingSentinel}\n`,
    );
    assert.doesNotMatch(
      combinedOutput(result),
      /EXISTING-PLAINTEXT-SENTINEL|OP-MUST-NOT-RUN-SENTINEL/,
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("secret restoration strips 1Password credentials from every Git invocation", () => {
  const root = makePublicFixture();
  try {
    initializePrivateTemplates(root);
    const fakeGit = createFakeGit(root);
    runPowerShell(
      restoreScript,
      root,
      [
        "-Target",
        "root",
        "-GitExecutable",
        fakeGit.path,
        "-OpExecutable",
        createFailingOp(root, "OP-MUST-NOT-RUN").toString(),
      ],
      {
        OP_SERVICE_ACCOUNT_TOKEN: "SERVICE-ACCOUNT-TOKEN-SENTINEL",
        OP_CONNECT_TOKEN: "CONNECT-TOKEN-SENTINEL",
        OP_SESSION_fixture: "SESSION-TOKEN-SENTINEL",
      },
    );
    const environmentLines = readFileSync(
      fakeGit.onePasswordEnvironmentLogPath,
      "utf8",
    )
      .trim()
      .split(/\r?\n/);
    assert.ok(environmentLines.length > 0);
    assert.ok(
      environmentLines.every((line) => line === "missing|missing|missing"),
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("secret restoration evaluates the complete committed ignore semantics", () => {
  const root = makePublicFixture();
  try {
    const ignorePath = join(root, ".gitignore");
    const committedIgnore = `${readFileSync(ignorePath, "utf8")}!.env\n`;
    writeFileSync(ignorePath, committedIgnore);
    git(root, "add", ".gitignore");
    git(root, "commit", "-m", "negate root environment protection");
    writeFileSync(ignorePath, committedIgnore.replace("!.env\n", ""));
    initializePrivateTemplates(root);
    const fakeOp = createMarkerOp(root);
    const result = runPowerShell(restoreScript, root, [
      "-Target",
      "root",
      "-OpExecutable",
      fakeOp.path,
    ]);
    assert.notEqual(result.status, 0);
    assert.equal(existsSync(fakeOp.markerPath), false);
    assert.match(combinedOutput(result), /committed public ignore rules/);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("failed secret injection removes only its script-owned temporary file", () => {
  const root = makePublicFixture();
  const noisySentinel = "OP-INJECT-ERROR-SENTINEL";
  try {
    const privateRoot = join(root, "private");
    mkdirSync(join(privateRoot, "bootstrap", "templates"), { recursive: true });
    git(privateRoot, "init");
    writeFileSync(
      join(privateRoot, "bootstrap", "templates", "root.env.tpl"),
      "BUNGIE_API_KEY=op://<vault>/<item>/<field>\n",
    );
    const fakeOp = createPartialInjectOp(root, noisySentinel);
    const result = runPowerShell(restoreScript, root, [
      "-Target",
      "root",
      "-OpExecutable",
      fakeOp,
    ]);
    assert.notEqual(result.status, 0);
    assert.equal(run("git", ["-C", root, "status", "--short"]).stdout, "");
    assert.doesNotMatch(combinedOutput(result), new RegExp(noisySentinel));
    assert.equal(
      run("git", ["-C", root, "status", "--ignored", "--short", ".env*"])
        .stdout,
      "",
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("secret restoration validates and installs a complete dotenv target", () => {
  const root = makePublicFixture();
  try {
    initializePrivateTemplates(root);
    const fakeOp = createRootInjectOp(root);
    const result = runPowerShell(restoreScript, root, [
      "-Target",
      "root",
      "-OpExecutable",
      fakeOp,
    ]);
    assert.equal(result.status, 0, combinedOutput(result));
    assert.match(result.stdout, /\.env: restored/);
    const restored = readFileSync(join(root, ".env"), "utf8");
    assert.doesNotMatch(restored, /op:\/\//);
    assert.match(restored, /^GO_ENV=development$/m);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("secret restoration rejects unmatched dotenv quotes and removes the temp", () => {
  const root = makePublicFixture();
  try {
    initializePrivateTemplates(root);
    const fakeOp = createRootInjectOp(root, { malformed: true });
    const result = runPowerShell(restoreScript, root, [
      "-Target",
      "root",
      "-OpExecutable",
      fakeOp,
    ]);
    assert.notEqual(result.status, 0);
    assert.equal(run("git", ["-C", root, "status", "--short"]).stdout, "");
    assert.equal(
      run("git", ["-C", root, "status", "--ignored", "--short", ".env*"])
        .stdout,
      "",
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("secret restoration rejects malformed content outside the YAML scaffold", () => {
  const root = makePublicFixture();
  try {
    initializePrivateTemplates(root, ["api-service-secret.yaml.tpl"]);
    const fakeOp = createMalformedK8sOp(root);
    const result = runPowerShell(restoreScript, root, [
      "-Target",
      "k8s",
      "-OpExecutable",
      fakeOp,
    ]);
    assert.notEqual(result.status, 0);
    assert.equal(run("git", ["-C", root, "status", "--short"]).stdout, "");
    assert.equal(
      run("git", [
        "-C",
        root,
        "status",
        "--ignored",
        "--short",
        "k8s/*-secret.yaml",
      ]).stdout,
      "",
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("secret restoration accepts quoted Kubernetes stringData scalars", () => {
  const root = makePublicFixture();
  try {
    initializePrivateTemplates(root, ["api-service-secret.yaml.tpl"]);
    const fakeOp = createK8sScalarOp(root, '"12345"');
    const result = runPowerShell(restoreScript, root, [
      "-Target",
      "k8s",
      "-OpExecutable",
      fakeOp,
    ]);
    assert.equal(result.status, 0, combinedOutput(result));
    assert.equal(
      existsSync(join(root, "k8s", "api-service-secret.yaml")),
      true,
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

for (const scalar of ["12345", "true", "null"]) {
  test(`secret restoration rejects unquoted Kubernetes ${scalar} scalars`, () => {
    const root = makePublicFixture();
    try {
      initializePrivateTemplates(root, ["api-service-secret.yaml.tpl"]);
      const fakeOp = createK8sScalarOp(root, scalar);
      const result = runPowerShell(restoreScript, root, [
        "-Target",
        "k8s",
        "-OpExecutable",
        fakeOp,
      ]);
      assert.notEqual(result.status, 0);
      assert.equal(
        existsSync(join(root, "k8s", "api-service-secret.yaml")),
        false,
      );
    } finally {
      rmSync(root, { recursive: true, force: true });
    }
  });
}

test("secret restoration preflights every target before installing any output", () => {
  const root = makePublicFixture();
  const noisySentinel = "OP-MUST-NOT-RUN-ATOMIC-PREFLIGHT";
  try {
    initializePrivateTemplates(root, [
      "root.env.tpl",
      "api-service.env.tpl",
      "frontend.env.local.tpl",
      "api-service-secret.yaml.tpl",
    ]);
    writeFileSync(join(root, "backend", "api-service", ".env"), "existing\n");
    const fakeOp = createFailingOp(root, noisySentinel);
    const result = runPowerShell(restoreScript, root, [
      "-Target",
      "all",
      "-OpExecutable",
      fakeOp,
    ]);
    assert.notEqual(result.status, 0);
    assert.equal(
      readFileSync(join(root, "backend", "api-service", ".env"), "utf8"),
      "existing\n",
    );
    assert.equal(run("git", ["-C", root, "status", "--short"]).stdout, "");
    assert.doesNotMatch(combinedOutput(result), new RegExp(noisySentinel));
    assert.equal(
      run("git", ["-C", root, "status", "--ignored", "--short", ".env"]).stdout,
      "",
    );
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("secret restoration requires its temporary plaintext path to be committed-ignored", () => {
  const root = makePublicFixture();
  const noisySentinel = "OP-MUST-NOT-RUN-UNIGNORED-TEMP";
  try {
    writeFileSync(
      join(root, ".gitignore"),
      [
        ".env",
        "backend/api-service/.env",
        "frontend/.env.local",
        "k8s/api-service-secret.yaml",
        "private/",
        ".private-workspace/",
        ".private-bootstrap-*/",
        "",
      ].join("\n"),
    );
    git(root, "add", ".gitignore");
    git(root, "commit", "-m", "ignore only final targets");
    initializePrivateTemplates(root);
    const fakeOp = createFailingOp(root, noisySentinel);
    const result = runPowerShell(restoreScript, root, [
      "-Target",
      "root",
      "-OpExecutable",
      fakeOp,
    ]);
    assert.notEqual(result.status, 0);
    assert.doesNotMatch(combinedOutput(result), new RegExp(noisySentinel));
    assert.equal(run("git", ["-C", root, "status", "--short"]).stdout, "");
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("public portability scripts contain no embedded private identifiers", () => {
  for (const path of [
    bootstrapScript,
    statusScript,
    restoreScript,
    join(projectRoot, "scripts", "bootstrap-private.mjs"),
  ]) {
    const contents = readFileSync(path, "utf8");
    assert.doesNotMatch(contents, /op:\/\/[A-Za-z0-9]/);
    assert.doesNotMatch(
      contents,
      /github\.com\/[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+/,
    );
    assert.doesNotMatch(contents, /op\s+read|remote\s+-v|--no-masking/);
  }
});
