# Maintainer Workspace Recovery

**Status:** Current

Guardian Tracker works from a public-only checkout. This runbook is only for
authorized maintainers restoring the ignored `private/` repository, approved
machine-local secret files, or a new Git worktree. It deliberately contains no
private repository location, real 1Password identifier, or secret value.

## Restore the Private Repository

Choose one source for the credential-free private Git URL.

To enter it at a secure prompt:

```powershell
./scripts/bootstrap-private-workspace.ps1 -PrivateFromPrompt
```

To resolve it with 1Password, create the ignored machine-local file
`.private-workspace/repository.env.ref`:

```dotenv
GUARDIAN_PRIVATE_REPOSITORY_URL=op://<vault>/<item>/<field>
```

If any reference segment contains whitespace, double-quote the complete value in
the dotenv file:

```dotenv
GUARDIAN_PRIVATE_REPOSITORY_URL="op://<vault name>/<item name>/<field>"
```

Make the approved least-privilege 1Password service-account credential available
to the current process through the maintainer's secure delivery mechanism. Do not
put it in a command, profile, workspace file, or transcript. Confirm that
`op vault list` succeeds without displaying secret values, then run:

```powershell
./scripts/bootstrap-private-workspace.ps1 -PrivateFromOnePassword
```

The helper accepts only credential-free GitHub HTTPS or `ssh://git@github.com`
locations. It will not create a submodule or replace an existing `private/`
directory.

## Restore Local Configuration

If the private repository supplies the approved value-free 1Password templates,
restore local configuration before running `setup.ps1`:

```powershell
# All approved targets
./scripts/restore-private-secrets.ps1

# Selected targets: root, api, frontend, or k8s
./scripts/restore-private-secrets.ps1 -Target root,frontend

# Fill any remaining files from public examples
./setup.ps1
```

Both helpers refuse to overwrite an existing target. The restoration helper
writes only the ignored `.env`, `backend/api-service/.env`,
`frontend/.env.local`, and `k8s/api-service-secret.yaml` paths after verifying
their committed ignore protection.

## Verify the Recovered Workspace

Run the value-free status check from the public repository root:

```powershell
./scripts/workspace-status.ps1
npm run test:workspace-portability
```

The status helper does not fetch. Its ahead/behind values come from local
tracking refs and can be stale; private branch names are redacted unless
`-IncludePrivateBranch` is explicitly supplied.

The restored private repository's `README.md` is the index for private plans,
operations, risks, references, archives, and encrypted-backup procedure. Follow
that private guidance for backup rotation; public docs remain value-free.

## New Git Worktrees

The Node bootstrap entry point supports worktrees and searches both the current
checkout and the main checkout for the machine-local reference file:

```bash
npm run bootstrap:private
```

Use the quoted shell form when intentionally overriding the reference file,
especially when a reference segment contains whitespace:

```powershell
npm run bootstrap:private -- --op-reference "op://<vault name>/<item name>/<field>"
```

The alternative `-- --url <credential-free-GitHub-URL>` override accepts only a
credential-free URL. The resolved URL is not placed in process arguments or
terminal output.

## Recovery Boundaries

- Preserve an existing `private/` directory and existing local configuration;
  investigate instead of cloning or restoring over them.
- Keep reference files value-free and rely on committed ignore rules for every
  plaintext target.
- Keep authentication tokens in the host credential mechanism. Never print,
  export into a workspace file, or commit them.
- A public-only checkout is a valid end state. Private restoration must not be a
  prerequisite for building, testing, or understanding the application.
