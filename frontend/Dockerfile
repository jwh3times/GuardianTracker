# Build stage
FROM node:26.8.2-alpine3.24@sha256:ef24c5053d50fdc3e4e56eb4e7ddb7861874ab0fdc797046ba897581deb8e868 AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .
# Development-only tunnel OAuth callback handoff (issue #317). Env files are
# excluded from the build context, so it arrives as a build argument; empty by
# default, which leaves the callback unchanged.
ARG VITE_OAUTH_COMPLETION_ORIGIN=
ENV VITE_OAUTH_COMPLETION_ORIGIN=${VITE_OAUTH_COMPLETION_ORIGIN}
RUN npm run build

# Production stage — nginx serves the static dist/
FROM nginxinc/nginx-unprivileged:1.31.6-alpine3.24@sha256:b54ac358b83fc6c965793fd271839b4ea4cdb6e99895bb19618cbc2ca152d972

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:8080/ || exit 1

CMD ["nginx", "-g", "daemon off;"]
