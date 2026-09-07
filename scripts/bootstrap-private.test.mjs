import { test } from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import {
  copyFileSync,
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  readdirSync,
  rmSync,
  symlinkSync,
  writeFileSync,
} from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";

const script = join(import.meta.dirname, "bootstrap-private.mjs");
const syntheticURL =
  "https://github.com/bootstrap-test-fixture/private-sentinel.git";
const variable = "GUARDIAN_PRIVATE_REPOSITORY_URL";

function fixture(t) {
  const parent = mkdtempSync(join(tmpdir(), "guardian-bootstrap-test-"));
  t.after(() => rmSync(parent, { recursive: true, force: true }));
  const root = join(parent, "checkout"),
    temporary = join(parent, "temporary");
  mkdirSync(join(root, "scripts"), { recursive: true });
  mkdirSync(temporary);
  copyFileSync(script, join(root, "scripts", "bootstrap-private.mjs"));
  const emptyConfig = join(parent, "empty.config");
  writeFileSync(emptyConfig, "");
  const env = {
    ...process.env,
    TMPDIR: temporary,
    TMP: temporary,
    TEMP: temporary,
    GIT_CONFIG_GLOBAL: emptyConfig,
    GIT_CONFIG_NOSYSTEM: "1",
    GIT_TERMINAL_PROMPT: "0",
    NODE_OPTIONS: "",
    FIXTURE_ROOT: parent,
  };
  for (const key of Object.keys(env)) {
    const normalized = key.toUpperCase();
    if (
      normalized.startsWith("OP_") ||
      normalized === variable ||
      /^GIT_TRACE|^GIT_CURL_VERBOSE$/u.test(normalized) ||
      /^GIT_(DIR|WORK_TREE|COMMON_DIR|INDEX_FILE|OBJECT_DIRECTORY|ALTERNATE_OBJECT_DIRECTORIES|CONFIG_COUNT|CONFIG_PARAMETERS|CONFIG_KEY_.*|CONFIG_VALUE_.*)$/u.test(
        normalized,
      )
    )
      delete env[key];
  }
  const git = (...args) => {
    const result = spawnSync("git", ["-C", root, ...args], {
      env,
      encoding: "utf8",
    });
    assert.equal(result.status, 0, result.stderr);
    return result.stdout.trim();
  };
  git("init");
  git("config", "user.name", "Fixture");
  git("config", "user.email", "fixture@example.invalid");
  writeFileSync(
    join(root, ".gitignore"),
    "private/\n.private-workspace/\n.private-bootstrap-*/\n",
  );
  git("add", ".gitignore", "scripts/bootstrap-private.mjs");
  git("commit", "-m", "fixture");
  const loader = join(parent, "loader.mjs"),
    log = join(parent, "calls.jsonl"),
    clone = join(parent, "clone.mjs");
  // This preload only affects fixture subprocesses. Native Node executables
  // keep the harness portable without .cmd wrappers or a Windows compiler.
  writeFileSync(
    loader,
    `
import cp from "node:child_process";
import fs from "node:fs";
import {syncBuiltinESMExports} from "node:module";
const spawn=cp.spawnSync;
const root=${JSON.stringify(parent)};
const loader=${JSON.stringify(loader)};
const log=${JSON.stringify(log)};
const clone=${JSON.stringify(clone)};
const url=${JSON.stringify(syntheticURL)};
cp.spawnSync=function(command,args,options={}){
 const env=options.env??process.env;
 fs.appendFileSync(log,JSON.stringify({command,args,hasPrivateURL:Object.hasOwn(env,${JSON.stringify(variable)}),opNames:Object.keys(env).filter(k=>k.toUpperCase().startsWith("OP_")),unrelatedReference:env.UNRELATED_REFERENCE,gitDir:env.GIT_DIR,injectedCount:env.GIT_CONFIG_COUNT,trace:env.GIT_TRACE,trace2:env.GIT_TRACE2,traceEvent:env.GIT_TRACE2_EVENT})+"\\n");
 if(command==="git" && args.includes("clone")) return spawn(process.execPath,[clone,...args],options);
 if(command==="op") {
  if(args[0]==="--version")return {status:0,stdout:"fixture op",stderr:""};
  if(env.FIXTURE_MODE==="op-fail")return spawn(process.execPath,["-e","console.log("+JSON.stringify(url)+");console.error("+JSON.stringify(url)+");process.exit(23)"],options);
  if(env.FIXTURE_MODE==="op-drift")fs.writeFileSync(${JSON.stringify(join(root, ".gitignore"))},"# ignored protection removed\\n");
  const child=args.slice(args.indexOf("--")+1);
  return spawn(child[0],["--import",loader,...child.slice(1)],{...options,env:{...env,${variable}:url,OP_FIXTURE_TOKEN:"synthetic-op-token"}});
 }
 return spawn(command,args,options);
};
if(process.env.FIXTURE_MODE==="cleanup-fail") {
 const remove=fs.rmSync;
 fs.rmSync=function(path,options){if(String(path).includes(".private-bootstrap-"))throw Error("synthetic staging cleanup failure");return remove(path,options)};
}
syncBuiltinESMExports();
`,
  );
  writeFileSync(
    clone,
    `
import {spawnSync} from "node:child_process";
import {readFileSync,writeFileSync} from "node:fs";
import {join} from "node:path";
const args=process.argv.slice(2),target=args.at(-1);
const config=args.find(v=>v.startsWith("include.path=")).slice("include.path=".length);
const url=/\\[url "([^\"]+)"\\]/u.exec(readFileSync(config,"utf8"))[1];
console.log(url);console.error(url);
if(["git-fail","cleanup-fail"].includes(process.env.FIXTURE_MODE)){writeFileSync(join(target,"partial"),"synthetic partial clone");process.exit(23)}
for(const command of [["init"],["config","remote.origin.url","guardian-private:"]]){
 if(process.env.FIXTURE_MODE==="origin-missing" && command[0]==="config")continue;
 const result=spawnSync("git",["-C",target,...command],{encoding:"utf8",env:process.env});if(result.status!==0)process.exit(result.status??1);
}
`,
  );
  const run = (args = [], extra = {}, mock = true) =>
    spawnSync(
      process.execPath,
      [
        ...(mock ? ["--import", loader] : []),
        join(root, "scripts", "bootstrap-private.mjs"),
        ...args,
      ],
      {
        encoding: "utf8",
        env: { ...env, FIXTURE_MODE: "success", ...extra },
        timeout: 20_000,
      },
    );
  const calls = () =>
    existsSync(log)
      ? readFileSync(log, "utf8")
          .trim()
          .split("\n")
          .filter(Boolean)
          .map((line) => JSON.parse(line))
      : [];
  const reference = () => {
    mkdirSync(join(root, ".private-workspace"), { recursive: true });
    writeFileSync(
      join(root, ".private-workspace", "repository.env.ref"),
      `${variable}=op://fixture/item/repository\n`,
    );
  };
  const clean = () => {
    assert.deepEqual(
      readdirSync(temporary),
      [],
      "temporary private config/reference survived",
    );
    assert.equal(
      readdirSync(root).some((name) => name.startsWith(".private-bootstrap-")),
      false,
      "staging clone survived",
    );
  };
  const safe = (result) => {
    assert.equal(
      `${result.stdout}${result.stderr}`.includes(syntheticURL),
      false,
      "private URL leaked to output",
    );
  };
  return {
    parent,
    root,
    temporary,
    env,
    git,
    run,
    calls,
    reference,
    clean,
    safe,
  };
}

for (const mode of ["success", "git-fail", "origin-missing", "op-fail"]) {
  test(`reference bootstrap ${mode} captures diagnostics and cleans owned files`, (t) => {
    const f = fixture(t);
    const result = f.run(["--op-reference", "op://fixture/item/repository"], {
      FIXTURE_MODE: mode,
      OP_FIXTURE_TOKEN: "synthetic-token",
      UNRELATED_REFERENCE: "op://fixture/other/secret",
      GIT_TRACE: "1",
      GIT_TRACE2: "1",
      GIT_TRACE2_EVENT: "1",
      GIT_DIR: "nonexistent",
      GIT_CONFIG_COUNT: "1",
      GIT_CONFIG_KEY_0: "trace2.eventTarget",
      GIT_CONFIG_VALUE_0: "1",
    });
    assert.equal(result.status, mode === "success" ? 0 : 1, result.stderr);
    f.safe(result);
    f.clean();
    assert.equal(existsSync(join(f.root, "private")), mode === "success");
    const calls = f.calls();
    for (const call of calls) {
      assert.equal(
        JSON.stringify(call.args).includes(syntheticURL),
        false,
        "URL reached helper direct child argv",
      );
      assert.equal(call.unrelatedReference, undefined);
      assert.equal(call.gitDir, undefined);
      assert.equal(call.injectedCount, undefined);
      assert.equal(call.trace, undefined);
      assert.equal(call.trace2, "0");
      assert.equal(call.traceEvent, "0");
      if (call.command === "git") {
        assert.equal(call.hasPrivateURL, false);
        assert.deepEqual(call.opNames, []);
      }
    }
    if (mode === "success")
      assert.ok(
        readFileSync(
          join(f.root, "private", ".git", "config"),
          "utf8",
        ).includes(syntheticURL),
      );
  });
}

test("real failed Git clone cannot disclose URL or leave temporary config", (t) => {
  const f = fixture(t);
  const result = f.run(
    ["--url", syntheticURL],
    {
      HTTPS_PROXY: "http://127.0.0.1:1",
      HTTP_PROXY: "http://127.0.0.1:1",
      ALL_PROXY: "http://127.0.0.1:1",
      https_proxy: "http://127.0.0.1:1",
      http_proxy: "http://127.0.0.1:1",
      all_proxy: "http://127.0.0.1:1",
      NO_PROXY: "",
      no_proxy: "",
      GIT_TRACE: "1",
      GIT_TRACE_CURL: "1",
    },
    false,
  );
  assert.equal(result.status, 1);
  f.safe(result);
  f.clean();
  assert.equal(existsSync(join(f.root, "private")), false);
});

test("temporary URL config cleanup is attempted even if staging cleanup fails", (t) => {
  const f = fixture(t);
  const result = f.run(["--url", syntheticURL], {
    FIXTURE_MODE: "cleanup-fail",
  });
  assert.equal(result.status, 1);
  f.safe(result);
  assert.deepEqual(readdirSync(f.temporary), []);
  assert.equal(
    readdirSync(f.root).some((name) => name.startsWith(".private-bootstrap-")),
    true,
    "injected cleanup error did not fire",
  );
});

for (const flaw of [
  "uncommitted-ignore",
  "committed-negation",
  "info-only-ignore",
  "tracked-target",
  "tracked-reference",
  "staged-deletion",
  "occupied-target",
  "invalid-reference",
  "internal-protection",
  "op-drift",
]) {
  test(`bootstrap rejects ${flaw} before cloning`, (t) => {
    const f = fixture(t);
    f.reference();
    let args = [];
    let extra = {};
    if (flaw === "uncommitted-ignore" || flaw === "committed-negation") {
      f.git("show", "HEAD:.gitignore");
      writeFileSync(
        join(f.root, ".gitignore"),
        flaw === "committed-negation"
          ? "private/\n.private-workspace/\n.private-bootstrap-*/\n!.private-workspace/\n!.private-workspace/repository.env.ref\n"
          : "private/\n.private-bootstrap-*/\n",
      );
      f.git("add", ".gitignore");
      f.git("commit", "-m", "unprotected reference");
      writeFileSync(
        join(f.root, ".gitignore"),
        "private/\n.private-workspace/\n.private-bootstrap-*/\n",
      );
    } else if (flaw === "info-only-ignore") {
      writeFileSync(join(f.root, ".gitignore"), "# no root protection\n");
      writeFileSync(
        join(f.root, ".git", "info", "exclude"),
        "private/\n.private-bootstrap-*/\n.private-workspace/\n",
      );
    } else if (["tracked-target", "occupied-target"].includes(flaw)) {
      mkdirSync(join(f.root, "private"));
      writeFileSync(join(f.root, "private", "keep.txt"), "preserve me");
      if (flaw === "tracked-target") f.git("add", "-f", "private/keep.txt");
    } else if (["tracked-reference", "staged-deletion"].includes(flaw)) {
      f.git("add", "-f", ".private-workspace/repository.env.ref");
      if (flaw === "staged-deletion") {
        f.git("commit", "-m", "tracked reference fixture");
        f.git("rm", "--cached", ".private-workspace/repository.env.ref");
      }
    } else if (flaw === "invalid-reference")
      writeFileSync(
        join(f.root, ".private-workspace", "repository.env.ref"),
        "UNAPPROVED=op://fixture/item/field\n",
      );
    else if (flaw === "internal-protection") {
      writeFileSync(join(f.root, ".gitignore"), "# no protection\n");
      args = ["--internal-clone-from-environment"];
      extra[variable] = syntheticURL;
    } else if (flaw === "op-drift") extra.FIXTURE_MODE = "op-drift";
    const result = f.run(args, extra);
    assert.equal(result.status, 1, result.stderr);
    f.safe(result);
    f.clean();
    assert.equal(
      f
        .calls()
        .some((call) => call.command === "git" && call.args.includes("clone")),
      false,
      "unsafe path reached clone",
    );
    if (existsSync(join(f.root, "private", "keep.txt")))
      assert.equal(
        readFileSync(join(f.root, "private", "keep.txt"), "utf8"),
        "preserve me",
      );
  });
}

for (const path of ["private", "reference-parent", "reference-file"]) {
  test(`bootstrap refuses ${path} symlinks without touching targets`, (t) => {
    const f = fixture(t);
    const outside = join(f.parent, "outside");
    mkdirSync(outside);
    writeFileSync(join(outside, "keep"), "preserve me");
    try {
      if (path === "private")
        symlinkSync(outside, join(f.root, "private"), "junction");
      if (path === "reference-parent")
        symlinkSync(outside, join(f.root, ".private-workspace"), "junction");
      if (path === "reference-file") {
        f.reference();
        rmSync(join(f.root, ".private-workspace", "repository.env.ref"));
        symlinkSync(
          join(outside, "keep"),
          join(f.root, ".private-workspace", "repository.env.ref"),
          "file",
        );
      }
    } catch (error) {
      if (process.platform === "win32" && error.code === "EPERM") {
        t.skip("file symlink privilege unavailable");
        return;
      }
      throw error;
    }
    const result = f.run();
    assert.equal(result.status, 1, result.stderr);
    f.safe(result);
    f.clean();
    assert.equal(readFileSync(join(outside, "keep"), "utf8"), "preserve me");
    assert.equal(
      f.calls().some((call) => call.command === "op"),
      false,
    );
  });
}

test("fresh worktree uses its main checkout's protected reference", (t) => {
  const f = fixture(t);
  f.reference();
  const worktree = join(f.parent, "worktree");
  f.git("worktree", "add", "-b", "fixture-worktree", worktree);
  const result = spawnSync(
    process.execPath,
    [
      "--import",
      join(f.parent, "loader.mjs"),
      join(worktree, "scripts", "bootstrap-private.mjs"),
    ],
    {
      encoding: "utf8",
      env: { ...f.env, FIXTURE_MODE: "success" },
      timeout: 20_000,
    },
  );
  assert.equal(result.status, 0, result.stderr);
  f.safe(result);
  f.clean();
  assert.ok(existsSync(join(worktree, "private", ".git")));
});

test("invalid arguments do not echo supplied URL", (t) => {
  const f = fixture(t);
  const result = f.run([syntheticURL]);
  assert.equal(result.status, 1);
  f.safe(result);
  f.clean();
  assert.deepEqual(f.calls(), []);
});

test("an existing private clone and its uncommitted files are preserved without invoking op or clone", (t) => {
  const f = fixture(t);
  const installed = f.run(["--url", syntheticURL]);
  assert.equal(installed.status, 0, installed.stderr);
  const configPath = join(f.root, "private", ".git", "config");
  const config = readFileSync(configPath, "utf8");
  const marker = join(f.root, "private", "uncommitted-note.md");
  writeFileSync(marker, "preserve this user-authored fixture\n");
  const before = f.calls().length;
  const repeated = f.run([], { FIXTURE_MODE: "op-fail" });
  assert.equal(repeated.status, 0, repeated.stderr);
  assert.match(repeated.stdout, /already installed/);
  f.safe(repeated);
  f.clean();
  assert.equal(readFileSync(configPath, "utf8"), config);
  assert.equal(
    readFileSync(marker, "utf8"),
    "preserve this user-authored fixture\n",
  );
  assert.equal(
    f
      .calls()
      .slice(before)
      .some((call) => call.command === "op" || call.args.includes("clone")),
    false,
  );
});
