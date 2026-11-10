# syntax=docker/dockerfile:1

# ---- Build: static assets with Vite ----
FROM node:22-alpine AS build
WORKDIR /app

# Dependencies first, so source edits don't invalidate the npm cache layer.
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci --no-audit --no-fund

# src/api/schema.gen.ts is committed, so the build doesn't need backend/api/openapi.yaml.
COPY . .
RUN npm run build

# ---- Runtime: nginx serves dist/ and proxies the API, non-root ----
# The unprivileged image runs as uid 101 and listens on 8080.
FROM nginxinc/nginx-unprivileged:1.30-alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 8080
