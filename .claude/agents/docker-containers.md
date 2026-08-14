---
name: docker-containers
description: Use for changes to any of the Dockerfiles, Minikube image builds, or local container development for api-service or frontend.
tools: Read, Write, Edit, Bash, Glob, Grep
model: sonnet
---

You are working with the Guardian Tracker container setup. The two application
images live in `backend/api-service/Dockerfile` and `frontend/Dockerfile`.
Frontend development and browser-test tooling additionally use
`frontend/Dockerfile.dev` and `frontend/Dockerfile.playwright`. The old
auth-service, bungie-service, and graphql-service Dockerfiles no longer exist.

## Dockerfiles

### `backend/api-service/Dockerfile` — CGO enabled for SQLite

Two-stage build:

1. **`builder`** (`golang:1.26.6-alpine`):

   - `RUN apk add --no-cache gcc musl-dev sqlite-dev` (required for CGO + SQLite)
   - `COPY go.mod go.sum ./` → `RUN go mod download`
   - `COPY . .`
   - `RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags='-w -s -extldflags "-static"' -o main .`

2. **Runtime** (`alpine:3.24.1`, pinned to its multi-platform index digest):
   - `RUN apk --no-cache add ca-certificates sqlite-libs`
   - Create non-root user `appuser` (uid/gid 1000) in group `appgroup`
   - `RUN mkdir -p /app/data && chown -R appuser:appgroup /app`
   - `VOLUME ["/app/data"]` for manifest persistence across pod restarts
   - `EXPOSE 8081`
   - Health check: `wget --spider http://localhost:8081/health`
   - `CMD ["./main"]`

### `frontend/Dockerfile` — Vite build + nginx

Two-stage build:

1. **`builder`** (`node:26.7.0-alpine3.24`, pinned to its multi-platform index digest):

   - `COPY package*.json ./` → `RUN npm ci`
   - `COPY . .`
   - `RUN npm run build` (Vite production build → `dist/`)

2. **Runtime** (`nginxinc/nginx-unprivileged:1.31.3-alpine3.24`, pinned to its multi-platform index digest):
   - `COPY --from=builder /app/dist /usr/share/nginx/html`
   - `COPY nginx.conf /etc/nginx/conf.d/default.conf`
   - `EXPOSE 8080`
   - Runs as non-root automatically (nginx-unprivileged handles this)
   - `CMD ["nginx", "-g", "daemon off;"]`

`frontend/Dockerfile.dev` uses the same pinned Node image and installs from the
lockfile with `npm ci` before copying the source tree.

`frontend/Dockerfile.playwright` layers the pinned Node 26 Debian runtime under
`/opt/node` onto a pinned official Playwright Noble image. The repository policy
test keeps its exact Playwright tag aligned with the frontend lockfile. CI and
the visual-baseline runbook build this helper image before running `npm ci`;
Playwright's stock image may carry an older Node line.

The `nginx.conf` must serve `index.html` for all unknown paths so React Router works correctly.

## Docker Compose host exposure

The local `postgres`, `pgadmin`, profile-gated `test-postgres`, and profile-gated
`e2e-postgres` ports bind to `127.0.0.1` only. Keep those loopback prefixes when changing port mappings. The
frontend/API bindings remain unchanged because local browsers and approved
development tunnels use them. Compose requires an explicit `GO_ENV` value from
the root environment file.

`e2e-postgres` uses `postgres:18.4-alpine3.24`, pinned to its multi-platform
index digest, with host port 5534, the `e2e` profile, and
a read-only `database/init` bind mount. It intentionally has no database data
volume and `restart: "no"`; Playwright owns the other browser-test processes.

## Building for Minikube

Images must be built inside Minikube's Docker daemon:

```powershell
# Point Docker CLI at Minikube's daemon (required before every build session)
& minikube docker-env --shell powershell | Invoke-Expression

# Build each service (run from repo root)
docker build --pull --no-cache -t guardian-tracker/api-service:latest backend/api-service/
docker build --pull --no-cache -t guardian-tracker/frontend:v2 frontend/

# Restart affected deployment
kubectl rollout restart deployment/<service-name>
kubectl rollout status deployment/<service-name> --timeout=120s
```

If a base image fails to pull (TLS timeout), pre-pull it first:

```powershell
& minikube docker-env --shell powershell | Invoke-Expression
docker pull <image:tag>
```

## Image tags

| Service     | Tag                                   |
| ----------- | ------------------------------------- |
| api-service | `guardian-tracker/api-service:latest` |
| frontend    | `guardian-tracker/frontend:v2`        |

Both deployments use `imagePullPolicy: Never` because their application images
are built directly into Minikube. After rebuilding a reused local tag, restart
an existing deployment whose pod template stayed unchanged so Kubernetes creates
pods from the new image. A newly created deployment or pod-template change already
has a rollout consuming the rebuilt image and should not be restarted a second
time while the api-service initializes or opens its manifest volume.

## Layer caching — key ordering rules

Both Dockerfiles follow the same caching pattern:

1. Copy lock files first (`go.mod`/`go.sum` or `package*.json`) — rarely change, keeps install step cached
2. Run install/restore (`go mod download` or `npm ci`) — expensive, must be cached
3. Copy source (`COPY . .`) — changes frequently, invalidates from here down

Never move `COPY . .` before the install step.

## Base image versions

| Role                     | Image                                           |
| ------------------------ | ----------------------------------------------- |
| Go builder               | `golang:1.26.6-alpine`                          |
| Go runtime               | `alpine:3.24.1`                                 |
| Node builder/dev         | `node:26.7.0-alpine3.24`                        |
| Playwright Node donor    | `node:26.7.0-bookworm-slim`                     |
| Playwright browser base  | `mcr.microsoft.com/playwright:v1.62.1-noble`    |
| nginx runtime (frontend) | `nginxinc/nginx-unprivileged:1.31.3-alpine3.24` |

The Go runtime, Node builder/development, Playwright Node donor/browser, and
nginx runtime tags are also pinned to their multi-platform OCI index digests in
the Dockerfiles. Dependabot advances tag and digest together for newer tagged
releases; follow the digest-drift check in `SETUP.md` for same-tag republishes.

When updating Go: the version must match the `go` directive in `go.mod`.
When updating Node within the supported 26 line: change `.nvmrc`, all three
frontend Dockerfiles, and their OCI digests together. Keep
`package.json`/lockfile engine metadata and `@types/node` on Node 26, then run
`node --test scripts/node-version-policy.test.mjs` from the repository root.
When updating Playwright, update its Dockerfile tag and digest with
`@playwright/test`; the same policy test rejects version drift.
