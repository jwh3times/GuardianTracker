# Build stage
FROM node:26.13.0-alpine3.24@sha256:c6ad9e9c1d80f6eb607c26b4b0a132ff7a2344fad1c78ea3baa2a7d4c8de6d94 AS builder

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
FROM nginxinc/nginx-unprivileged:1.31.7-alpine3.24@sha256:b5047b9b76845eac124358373c6852377c52e4b05834a47647e955dd91cb8f1e

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:8080/ || exit 1

CMD ["nginx", "-g", "daemon off;"]
