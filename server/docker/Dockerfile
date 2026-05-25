# syntax=docker/dockerfile:1
# 构建上下文为 server/ 目录：docker build -f docker/Dockerfile .

FROM golang:1.22-bookworm AS builder

WORKDIR /src
ENV CGO_ENABLED=0 GOTOOLCHAIN=auto

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -ldflags="-s -w" -o /out/log-tools-server .

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -r -u 10001 -g nogroup appuser

WORKDIR /app

RUN mkdir -p /app/config /app/data/uploads \
    && chown -R appuser:nogroup /app

COPY --from=builder /out/log-tools-server /app/log-tools-server

ENV CONFIG_PATH=/app/config/config.yaml \
    TZ=Asia/Shanghai

EXPOSE 8080

VOLUME ["/app/config", "/app/data"]

USER appuser

ENTRYPOINT ["/app/log-tools-server"]
