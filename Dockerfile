# Stage 1: build
FROM golang:1.25.4 AS builder

WORKDIR /app

# Dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the code
COPY . .

# Build the static binary for Linux (Cloud Run)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o farum-api ./cmd/farum-api

# Stage 2: minimal runtime
FROM gcr.io/distroless/base-debian12

WORKDIR /app

COPY --from=builder /app/farum-api /app/farum-api

# Standard port for HTTP (Cloud Run ignores EXPOSE, but it helps)
EXPOSE 8080

# Basic environment variables for Go apps on Cloud Run
ENV PORT=8080

ENTRYPOINT ["/app/farum-api"]
