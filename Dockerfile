# Stage 1: Go dependencies and build cache
FROM golang:1.25.1-alpine AS go-deps

WORKDIR /app

RUN apk add --no-cache gcc musl-dev sqlite-dev

# Copy go mod files and download dependencies first (better caching)
COPY go.mod go.sum ./
RUN go mod download
RUN go mod verify

# Stage 2: Go build
FROM go-deps AS go-builder

# Copy source code
COPY cmd/ ./cmd/
COPY pkg/ ./pkg/

# Build with optimizations and static linking
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s -extldflags '-static'" \
    -a -installsuffix cgo \
    -o main ./cmd/api

# Stage 3: Node.js build
FROM node:18-alpine AS node-builder

WORKDIR /app/frontend

# Copy package files and install dependencies first (better caching)
COPY frontend/package*.json ./
RUN npm ci

# Copy frontend source and build
COPY frontend/ ./
RUN npm run build

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

RUN mkdir -p db

COPY --from=go-builder /app/main .
COPY --from=node-builder /app/frontend/dist ./frontend/dist

EXPOSE 8040

CMD ["./main"]
