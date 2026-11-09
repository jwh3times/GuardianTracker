// k8s/startup.ps1's Minikube steps, run as a real process against fake
// minikube, docker and kubectl executables. The fakes record every call and
// stop the script at its first image build, so these tests cover only cluster
// start-up and Docker daemon selection; the real cluster is exercised by the
// manual acceptance run in k8s/README.md.
import { test } from "node:test";
import assert from "node:assert/strict";
import {
  chmodSync,
  existsSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { delimiter, join } from "node:path";
import { spawnSync } from "node:child_process";

const projectRoot = join(import.meta.dirname, "..");
const k8sDirectory = join(projectRoot, "k8s");
const startupScript = join(k8sDirectory, "startup.ps1");
const powershell =
  process.env.PORTABILITY_POWERSHELL ??
  (process.platform === "win32" ? "powershell.exe" : "pwsh");

// One implementation serves all three tools. Minikube answers from the
// scenario file. docker build fails for its first scenario.buildFailures calls
// (all of them by default), and kubectl always fails, so the script stops
// before it deploys anything.
const fakeCli = `
import { appendFileSync, readFileSync } from "node:fs";
const [tool, ...args] = process.argv.slice(2);
appendFileSync(process.env.FAKE_CLI_LOG, [tool, ...args].join(" ") + "\\n");
const scenario = JSON.parse(readFileSync(process.env.FAKE_CLI_SCENARIO, "utf8"));
const command = args.join(" ");
if (tool === "docker" && command.startsWith("build")) {
  const buildsSoFar = readFileSync(process.env.FAKE_CLI_LOG, "utf8")
    .split("\\n")
    .filter((line) => line.startsWith("docker build")).length;
  process.exit(buildsSoFar <= (scenario.buildFailures ?? 99) ? 1 : 0);
}
if (tool !== "minikube") process.exit(1);
if (command.startsWith("profile list")) {
  process.stdout.write(JSON.stringify({ invalid: [], valid: scenario.profiles }));
  process.exit(0);
}
if (command === "status") {
  if (scenario.profiles.length === 0) {
    process.stdout.write('* Profile "minikube" not found.\\n');
    process.exit(85);
  }
  process.stdout.write("minikube\\nhost: " + scenario.status + "\\n");
  process.exit(scenario.status === "Running" ? 0 : 7);
}
if (command.startsWith("start")) process.exit(0);
if (command.startsWith("docker-env")) {
  if (scenario.dockerEnv === "fail") {
    process.stdout.write("false\\n");
    process.stderr.write("X Exiting due to SSH_AGENT_START: starting an SSH agent on Windows is not yet supported\\n");
    process.exit(80);
  }
  process.stdout.write('$Env:DOCKER_HOST = "tcp://127.0.0.1:2376"\\n$Env:MINIKUBE_ACTIVE_DOCKERD = "minikube"\\n');
  process.exit(0);
}
process.exit(2);
`;

function profile(runtime) {
  return {
    Name: "minikube",
    Status: "Stopped",
    Active: true,
    Config: {
      Name: "minikube",
      KubernetesConfig: { ContainerRuntime: runtime },
    },
  };
}

function runStartup(scenario) {
  const root = mkdtempSync(join(tmpdir(), "guardian-minikube-startup-"));
  try {
    const implementation = join(root, "fake-cli.mjs");
    writeFileSync(implementation, fakeCli);
    for (const tool of ["minikube", "docker", "kubectl"]) {
      if (process.platform === "win32") {
        writeFileSync(
          join(root, `${tool}.cmd`),
          `@"${process.execPath}" "${implementation}" ${tool} %*\r\n@exit /b %ERRORLEVEL%\r\n`,
        );
      } else {
        const path = join(root, tool);
        writeFileSync(
          path,
          `#!/bin/sh\nexec '${process.execPath}' '${implementation}' ${tool} "$@"\n`,
        );
        chmodSync(path, 0o700);
      }
    }
    const log = join(root, "calls.log");
    const scenarioPath = join(root, "scenario.json");
    writeFileSync(scenarioPath, JSON.stringify(scenario));

    // Windows spells the variable Path; drop every spelling so ours wins.
    const env = Object.fromEntries(
      Object.entries(process.env).filter(
        ([key]) => key.toUpperCase() !== "PATH",
      ),
    );
    env.PATH = `${root}${delimiter}${process.env.PATH}`;
    env.FAKE_CLI_LOG = log;
    env.FAKE_CLI_SCENARIO = scenarioPath;

    const result = spawnSync(
      powershell,
      [
        "-NoProfile",
        "-NonInteractive",
        "-ExecutionPolicy",
        "Bypass",
        "-File",
        startupScript,
        "-BuildRetryDelaySeconds",
        "0",
      ],
      { cwd: k8sDirectory, encoding: "utf8", env, timeout: 60_000 },
    );
    assert.ifError(result.error);
    const calls = existsSync(log)
      ? readFileSync(log, "utf8").split("\n").filter(Boolean)
      : [];
    return {
      status: result.status,
      output: `${result.stdout}\n${result.stderr}`,
      calls,
    };
  } finally {
    rmSync(root, {
      recursive: true,
      force: true,
      maxRetries: 10,
      retryDelay: 100,
    });
  }
}

const builds = (calls) =>
  calls.filter((call) => call.startsWith("docker build"));

test("a fresh cluster is created with the Docker runtime", () => {
  const run = runStartup({ profiles: [], dockerEnv: "ok" });
  assert.ok(
    run.calls.includes("minikube start --container-runtime=docker"),
    `expected a Docker-runtime start, got:\n${run.calls.join("\n")}\n${run.output}`,
  );
  assert.ok(builds(run.calls).length > 0, run.output);
});

// A newly started cluster cannot resolve names from build containers for a few
// seconds, so the first build after a start can fail on its package fetch.
test("a build that fails right after starting the cluster is retried once", () => {
  const run = runStartup({ profiles: [], dockerEnv: "ok", buildFailures: 1 });
  const apiBuilds = builds(run.calls).filter((call) =>
    call.includes("guardian-tracker/api-service"),
  );
  const frontendBuilds = builds(run.calls).filter((call) =>
    call.includes("guardian-tracker/frontend"),
  );
  assert.equal(apiBuilds.length, 2, run.output);
  assert.equal(frontendBuilds.length, 1, run.output);
  assert.match(run.output, /Retrying/);
});

test("a build that keeps failing after a start stops after one retry", () => {
  const run = runStartup({ profiles: [], dockerEnv: "ok" });
  assert.notEqual(run.status, 0, run.output);
  assert.equal(builds(run.calls).length, 2, run.output);
  assert.match(run.output, /Failed to build api-service/);
});

test("a failed docker-env stops startup before any image build", () => {
  const run = runStartup({
    profiles: [profile("docker")],
    status: "Running",
    dockerEnv: "fail",
  });
  assert.notEqual(run.status, 0, run.output);
  assert.deepEqual(builds(run.calls), [], run.output);
  assert.match(run.output, /docker-env/);
});

test("an existing containerd profile is refused and left as it was", () => {
  const run = runStartup({
    profiles: [profile("containerd")],
    status: "Stopped",
    dockerEnv: "fail",
  });
  assert.notEqual(run.status, 0, run.output);
  assert.deepEqual(
    run.calls.filter((call) =>
      /^minikube (start|delete|docker-env)/.test(call),
    ),
    [],
    run.output,
  );
  assert.deepEqual(builds(run.calls), [], run.output);
  assert.match(run.output, /containerd/);
  assert.match(run.output, /minikube delete/);
});

test("a running Docker-runtime cluster is reused without starting it", () => {
  const run = runStartup({
    profiles: [profile("docker")],
    status: "Running",
    dockerEnv: "ok",
  });
  assert.deepEqual(
    run.calls.filter((call) => call.startsWith("minikube start")),
    [],
    run.output,
  );
  assert.match(run.output, /Minikube is already running/);
  assert.equal(builds(run.calls).length, 1, run.output);
});
