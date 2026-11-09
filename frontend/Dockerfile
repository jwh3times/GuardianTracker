# Build stage
FROM node:26.13.0-alpine3.24@sha256:8996bb5821d62a8fd9bb27c1cb07462f3fdce6e47d640c6a881ead0b064ce216 AS builder

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
