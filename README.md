# Guardian Tracker

Guardian Tracker is a Destiny 2 companion app that helps players understand
their collection gaps and decide what to chase next. Players sign in with
Bungie OAuth; a React frontend uses a Go API to combine account data, the
Destiny manifest, live weekly data, and persisted Guardian Tracker preferences.

## Features

- Collection analysis for weapons, armor, exotics, and cosmetics, including
  every collectible-attributed acquisition source and its source-specific
  difficulty.
- Wishlist management with priorities, notes, and live vendor availability.
- A weekly planner with milestones, Xûr inventory, selected-character vendor
  context, reset timing, and ranked recommendations.
- Catalyst, crafting-pattern, triumph, and seal progress.
- Manifest-backed global item search and a dedicated cosmetics gallery.
- Persisted preferences and onboarding, controlled feature rollout, and an
  admin console with an audit log.

## Architecture

```text
React/TypeScript frontend :5273
    -> Go/Gin API :8081
        -> Bungie API
        -> PostgreSQL user data
        -> Destiny manifest (SQLite)
```

See [the architecture guide](./docs/architecture.md) for the implemented system
and [the product principles](./docs/product.md) for the experience Guardian
Tracker is intended to provide.

## Quick Start

The full setup, environment, port, and validation instructions live in
[SETUP.md](./SETUP.md).

```powershell
./setup.ps1
docker compose up --build
```

Open the frontend at <http://localhost:5273> and the API at
<http://localhost:8081>. The first API startup downloads the Destiny manifest,
so data surfaces may briefly show a warming state.

Public contributors do not need the optional private workspace. Authorized
maintainers restoring one should follow the
[private-workspace procedure](./SETUP.md#optional-restore-a-private-workspace)
before running `setup.ps1`.

## Repository Layout

```text
frontend/                 React + TypeScript SPA
backend/api-service/      Go API service
database/init/            PostgreSQL bootstrap SQL
k8s/                      Minikube validation manifests and scripts
docs/                     Public architecture, product, and decision records
private/                  Optional ignored private-docs repository
```

## Documentation

Start with the [documentation map](./docs/README.md). The primary guides are:

- [SETUP.md](./SETUP.md) — local setup, environments, ports, and tests.
- [CONTEXT.md](./CONTEXT.md) — canonical domain vocabulary.
- [docs/architecture.md](./docs/architecture.md) — implemented architecture.
- [docs/product.md](./docs/product.md) — durable product goals and principles.
- [ROADMAP.md](./ROADMAP.md) — work that has not shipped.
- [CHANGELOG.md](./CHANGELOG.md) — shipped changes by version.
- [CONTRIBUTING.md](./CONTRIBUTING.md) — contribution workflow and CI gates.
- [SECURITY.md](./SECURITY.md) — security model and reporting process.

This is a public repository. Secrets, private security analysis, deployment
runbooks, and raw research dumps belong in the ignored `private/` workspace.
Detailed implementation handoffs live on GitHub Issues and the linked Project
board instead.

## Contributing and Support

Contributions are welcome. See [CONTRIBUTING.md](./CONTRIBUTING.md) before
opening a pull request and [SUPPORT.md](./SUPPORT.md) when asking for help.
Security vulnerabilities must use the private process in
[SECURITY.md](./SECURITY.md).

By participating, you agree to the [Code of Conduct](./CODE_OF_CONDUCT.md).

## License

MIT License — see [LICENSE](./LICENSE).
