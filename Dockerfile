FROM golang:1.25.1-alpine AS builder

WORKDIR /app

RUN apk add --no-cache gcc musl-dev sqlite-dev nodejs npm

# Build Go application
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -o main ./cmd/api

# Build frontend
WORKDIR /app/frontend
RUN npm ci
RUN npm run build

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

RUN mkdir -p db

COPY --from=builder /app/main .
COPY --from=builder /app/frontend/dist ./frontend/dist

EXPOSE 8040

CMD ["./main"]
