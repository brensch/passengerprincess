# syntax=docker/dockerfile:1
# Cross-builds ppp for the schlift.com VM (Oracle Cloud, ARM64): the API as a
# static aarch64 binary with the real SQLite (mattn/go-sqlite3 compiles
# SQLite's C source; zig is the cross C compiler), plus the built frontend.
# Nothing runs under emulation. `make build` exports both to deploy/out/.
FROM --platform=$BUILDPLATFORM golang:1.25.1-alpine AS api
RUN apk add --no-cache zig
WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY cmd/ ./cmd/
COPY pkg/ ./pkg/
ENV CGO_ENABLED=1 GOOS=linux GOARCH=arm64 \
    CC="zig cc -target aarch64-linux-musl" CXX="zig c++ -target aarch64-linux-musl"
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache \
    go build -trimpath -ldflags="-w -s -linkmode external -extldflags -static" -o /out/ppp ./cmd/api

FROM --platform=$BUILDPLATFORM node:18-alpine AS web
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM scratch AS export
COPY --from=api /out/ppp /ppp
COPY --from=web /app/frontend/dist /frontend/dist
