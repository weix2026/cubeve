FROM golang:1.22-alpine AS builder

WORKDIR /app

# 安装依赖
RUN apk add --no-cache git make gcc musl-dev linux-headers

# 复制源码
COPY . .

# 构建 CubeAPI Gateway
RUN cd cmd/api-gateway && go build -o /app/bin/api-gateway .

# 构建 Instance Controller
RUN cd cmd/instance-controller && go build -o /app/bin/instance-controller .

# 构建 CubeSandbox Operator
RUN cd cmd/cubesandbox-operator && go build -o /app/bin/cubesandbox-operator .

# 运行时镜像
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

# 创建用户
RUN adduser -D -u 1000 cubeve

WORKDIR /app

# 从 builder 复制二进制文件
COPY --from=builder /app/bin/api-gateway /usr/local/bin/api-gateway
COPY --from=builder /app/bin/instance-controller /usr/local/bin/instance-controller
COPY --from=builder /app/bin/cubesandbox-operator /usr/local/bin/cubesandbox-operator

# 默认运行 API Gateway
USER cubeve
EXPOSE 8080 50051 9090

ENTRYPOINT ["/usr/local/bin/api-gateway"]
CMD ["--listen=:8080", "--grpc=:50051", "--metrics=:9090"]
