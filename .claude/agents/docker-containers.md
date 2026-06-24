---
name: docker-containers
description: Use for changes to any of the Dockerfiles, Minikube image builds, or local container development for api-service or frontend.
tools: Read, Write, Edit, Bash, Glob, Grep
model: sonnet
---

You are working with the Guardian Tracker container setup. There are **two** Dockerfiles — one per service (`backend/api-service/Dockerfile` and `frontend/Dockerfile`). The old auth-service, bungie-service, and graphql-service Dockerfiles no longer exist.

## Dockerfiles

### `backend/api-service/Dockerfile` — CGO enabled for SQLite

Two-stage build:

1. **`builder`** (`golang:1.25-alpine`):
   - `RUN apk add --no-cache gcc musl-dev sqlite-dev` (required for CGO + SQLite)
   - `COPY go.mod go.sum ./` → `RUN go mod download`
   - `COPY . .`
   - `RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags='-w -s -extldflags "-static"' -o main .`

2. **Runtime** (`alpine:3.19`):
   - `RUN apk --no-cache add ca-certificates sqlite-libs`
   - Create non-root user `appuser` (uid/gid 1000) in group `appgroup`
   - `RUN mkdir -p /app/data && chown -R appuser:appgroup /app`
   - `VOLUME ["/app/data"]` for manifest persistence across pod restarts
   - `EXPOSE 8081`
   - Health check: `wget --spider http://localhost:8081/health`
   - `CMD ["./main"]`

### `frontend/Dockerfile` — Vite build + nginx

Two-stage build:

1. **`builder`** (`node:26-alpine`):
   - `COPY package*.json ./` → `RUN npm ci`
   - `COPY . .`
   - `RUN npm run build` (Vite production build → `dist/`)

2. **Runtime** (`nginxinc/nginx-unprivileged:1.25-alpine`):
   - `COPY --from=builder /app/dist /usr/share/nginx/html`
   - `COPY nginx.conf /etc/nginx/conf.d/default.conf`
   - `EXPOSE 8080`
   - Runs as non-root automatically (nginx-unprivileged handles this)
   - `CMD ["nginx", "-g", "daemon off;"]`

The `nginx.conf` must serve `index.html` for all unknown paths so React Router works correctly.

## Building for Minikube

Images must be built inside Minikube's Docker daemon:

```powershell
# Point Docker CLI at Minikube's daemon (required before every build session)
& minikube docker-env --shell powershell | Invoke-Expression

# Build each service (run from repo root)
docker build -t guardian-tracker/api-service:latest backend/api-service/
docker build -t guardian-tracker/frontend:v2 frontend/

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

| Service | Tag |
|---|---|
| api-service | `guardian-tracker/api-service:latest` |
| frontend | `guardian-tracker/frontend:v2` |

All deployments use `imagePullPolicy: IfNotPresent`.

## Layer caching — key ordering rules

Both Dockerfiles follow the same caching pattern:
1. Copy lock files first (`go.mod`/`go.sum` or `package*.json`) — rarely change, keeps install step cached
2. Run install/restore (`go mod download` or `npm ci`) — expensive, must be cached
3. Copy source (`COPY . .`) — changes frequently, invalidates from here down

Never move `COPY . .` before the install step.

## Base image versions

| Role | Image |
|---|---|
| Go builder | `golang:1.25-alpine` |
| Go runtime | `alpine:3.19` |
| Node builder (frontend) | `node:26-alpine` |
| nginx runtime (frontend) | `nginxinc/nginx-unprivileged:1.25-alpine` |

When updating Go: the version must match the `go` directive in `go.mod`.
When updating Node for the frontend: update both the builder stage tag and confirm the built artifact is compatible.
