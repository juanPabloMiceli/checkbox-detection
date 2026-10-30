# syntax=docker/dockerfile:1
FROM oven/bun:1.4.4 AS bun

FROM node:24-bookworm-slim AS build
WORKDIR /app
COPY --from=bun /usr/local/bin/bun /usr/local/bin/bun
COPY package.json bun.lock ./
RUN bun install --frozen-lockfile
COPY angular.json tsconfig*.json ./
COPY src ./src
COPY public ./public
RUN npm run build

FROM nginxinc/nginx-unprivileged:stable-alpine AS runtime
ENV PORT=8080 \
    NGINX_ENVSUBST_FILTER="^(PORT|API_UPSTREAM)$"
COPY docker/default.conf.template /etc/nginx/templates/default.conf.template
COPY --chmod=755 docker/15-check-environment.sh /docker-entrypoint.d/15-check-environment.sh
COPY --from=build /app/dist/checkbox-detection/browser /usr/share/nginx/html
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O /dev/null "http://127.0.0.1:${PORT}/healthz" || exit 1
