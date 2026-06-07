# Build stage for network-restricted environments
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev linux-headers

# Copy go module files first for better caching
COPY go.mod go.sum ./

# Copy source code
COPY . .

# Build with offline flags (for network-restricted environments)
ENV GOSUMDB=off
ENV GOPROXY=off

# Build components that work offline
RUN go build -mod=mod -ldflags="-w -s" -o /app/bin/api-gateway ./cmd/api-gateway && \
    go build -mod=mod -ldflags="-w -s" -o /app/bin/cubeconsole ./cmd/cubeconsole && \
    go build -mod=mod -ldflags="-w -s" -o /app/bin/version-tool ./cmd/version-tool && \
    go build -mod=mod -ldflags="-w -s" -o /app/bin/cubesandbox-operator ./cmd/cubesandbox-operator

# Note: instance-controller requires network access for K8s dependencies
# Build it separately in environments with full network access

# Runtime image for API Gateway
FROM alpine:3.19 AS api-gateway

RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -u 1000 cubeve

WORKDIR /app

COPY --from=builder /app/bin/api-gateway /usr/local/bin/api-gateway

USER cubeve
EXPOSE 8080 50051 9090

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/api-gateway"]
CMD ["--listen=:8080", "--grpc=:50051", "--metrics=:9090"]

# Runtime image for CLI tools
FROM alpine:3.19 AS cubeconsole

RUN apk add --no-cache ca-certificates
RUN adduser -D -u 1000 cubeve

WORKDIR /app

COPY --from=builder /app/bin/cubeconsole /usr/local/bin/cubeconsole
COPY --from=builder /app/bin/version-tool /usr/local/bin/version-tool

USER cubeve

ENTRYPOINT ["/usr/local/bin/cubeconsole"]
CMD ["help"]

# Runtime image for CubeSandbox Operator
FROM alpine:3.19 AS cubesandbox-operator

RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -u 1000 cubeve

WORKDIR /app

COPY --from=builder /app/bin/cubesandbox-operator /usr/local/bin/cubesandbox-operator

USER cubeve

ENTRYPOINT ["/usr/local/bin/cubesandbox-operator"]
CMD ["--help"]
