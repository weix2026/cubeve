PROJECT := cubeve
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
REGISTRY := cubeve
GO_VERSION := 1.22

# 构建目录
BUILD_DIR := ./build
BIN_DIR := $(BUILD_DIR)/bin

# 二进制文件
BINARIES := api-gateway instance-controller cubesandbox-operator cubeconsole

# 默认目标
.PHONY: all
all: build

# 构建所有二进制文件
.PHONY: build
build:
	@echo "Building $(PROJECT) $(VERSION)..."
	@mkdir -p $(BIN_DIR)
	@for bin in $(BINARIES); do \
		echo "  Building $$bin..."; \
		go build -ldflags "-X main.Version=$(VERSION)" -o $(BIN_DIR)/$$bin ./cmd/$$bin; \
	done
	@echo "Build complete."

# 构建单个二进制文件
.PHONY: build-%
build-%:
	@echo "Building $*..."
	@mkdir -p $(BIN_DIR)
	@go build -ldflags "-X main.Version=$(VERSION)" -o $(BIN_DIR)/$* ./cmd/$*

# 测试
.PHONY: test
test:
	@echo "Running tests..."
	@go test -v -race ./...

# 测试覆盖率
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=$(BUILD_DIR)/coverage.out ./...
	@go tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html

# 代码检查
.PHONY: lint
lint:
	@echo "Running linter..."
	@golangci-lint run ./...

# 格式化
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# 清理
.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)

# Docker 构建
.PHONY: docker-build
docker-build:
	@echo "Building Docker images..."
	@docker build -t $(REGISTRY)/api-gateway:$(VERSION) --target api-gateway .
	@docker build -t $(REGISTRY)/instance-controller:$(VERSION) --target instance-controller .
	@docker build -t $(REGISTRY)/cubesandbox-operator:$(VERSION) --target cubesandbox-operator .

# Docker 推送
.PHONY: docker-push
docker-push: docker-build
	@echo "Pushing Docker images..."
	@docker push $(REGISTRY)/api-gateway:$(VERSION)
	@docker push $(REGISTRY)/instance-controller:$(VERSION)
	@docker push $(REGISTRY)/cubesandbox-operator:$(VERSION)

# 部署 L0
.PHONY: deploy-l0
deploy-l0:
	@echo "Deploying L0 (MVP)..."
	@bash scripts/l0-install.sh

# 部署 L1
.PHONY: deploy-l1
deploy-l1:
	@echo "Deploying L1 (Basic)..."
	@bash scripts/l1-install.sh

# 部署 L2
.PHONY: deploy-l2
deploy-l2:
	@echo "Deploying L2 (Standard)..."
	@bash scripts/l2-install.sh

# 部署 L3
.PHONY: deploy-l3
deploy-l3:
	@echo "Deploying L3 (Full)..."
	@bash scripts/l3-install.sh

# 部署 manifests
.PHONY: deploy-manifests
deploy-manifests:
	@echo "Applying Kubernetes manifests..."
	@kubectl apply -f manifests/runtimeclasses.yaml
	@kubectl apply -f manifests/instance-crd.yaml
	@kubectl apply -f manifests/cubesandbox-runtime.yaml
	@kubectl apply -f manifests/network.yaml
	@kubectl apply -f manifests/storage.yaml
	@kubectl apply -f manifests/management-plane.yaml
	@kubectl apply -f manifests/tee-confidential.yaml

# 生成 protobuf
.PHONY: proto
proto:
	@echo "Generating protobuf code..."
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/*.proto

# 安装依赖
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod verify

# 帮助
.PHONY: help
help:
	@echo "$(PROJECT) $(VERSION) - Build targets:"
	@echo ""
	@echo "  make build              - Build all binaries"
	@echo "  make build-<name>       - Build specific binary (e.g., make build-api-gateway)"
	@echo "  make test               - Run tests"
	@echo "  make test-coverage      - Run tests with coverage"
	@echo "  make lint               - Run linter"
	@echo "  make fmt                - Format code"
	@echo "  make clean              - Clean build artifacts"
	@echo "  make docker-build       - Build Docker images"
	@echo "  make docker-push        - Push Docker images"
	@echo "  make deploy-l0          - Deploy L0 (MVP)"
	@echo "  make deploy-l1          - Deploy L1 (Basic)"
	@echo "  make deploy-l2          - Deploy L2 (Standard)"
	@echo "  make deploy-l3          - Deploy L3 (Full)"
	@echo "  make deploy-manifests   - Apply K8s manifests"
	@echo "  make proto              - Generate protobuf code"
	@echo "  make deps               - Install dependencies"
	@echo "  make help               - Show this help"
	@echo ""
