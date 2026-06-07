.PHONY: all build test clean docker-build deploy-l0 deploy-l2 deploy-manifests

VERSION ?= dev
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)

GO_BUILD := GOSUMDB=off GOPROXY=off go build -mod=mod -ldflags "$(LDFLAGS)"

all: build

build: build-api-gateway build-cubeconsole build-version-tool build-cubesandbox-operator build-instance-controller

build-api-gateway:
	@echo "Building api-gateway..."
	$(GO_BUILD) -o bin/api-gateway ./cmd/api-gateway

build-cubeconsole:
	@echo "Building cubeconsole..."
	$(GO_BUILD) -o bin/cubeconsole ./cmd/cubeconsole

build-version-tool:
	@echo "Building version-tool..."
	$(GO_BUILD) -o bin/version-tool ./cmd/version-tool

build-cubesandbox-operator:
	@echo "Building cubesandbox-operator..."
	$(GO_BUILD) -o bin/cubesandbox-operator ./cmd/cubesandbox-operator

build-instance-controller:
	@echo "Building instance-controller..."
	$(GO_BUILD) -o bin/instance-controller ./cmd/instance-controller

build-offline:
	@echo "Building offline (network restricted)..."
	./scripts/build-offline.sh all

test: test-binaries

test-binaries:
	@echo "Testing binaries..."
	./scripts/test-binaries.sh

clean:
	rm -rf bin/

docker-build:
	@echo "Building Docker images..."
	docker build -t cubeve/api-gateway:$(VERSION) -f Dockerfile --target api-gateway .
	docker build -t cubeve/cubeconsole:$(VERSION) -f Dockerfile --target cubeconsole .

deploy-l0:
	@echo "Deploying L0 (single node)..."
	sudo bash scripts/l0-install.sh

deploy-l2:
	@echo "Deploying L2 (Kubernetes)..."
	sudo bash scripts/l2-install.sh

deploy-manifests:
	@echo "Applying Kubernetes manifests..."
	kubectl apply -f manifests/

lint:
	@echo "Linting..."
	golangci-lint run || true

fmt:
	@echo "Formatting..."
	go fmt ./...

.DEFAULT_GOAL := build