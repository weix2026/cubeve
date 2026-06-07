#!/bin/bash
# Build script for network-restricted environments
# Usage: ./scripts/build-offline.sh [component]

set -e

COMPONENT=${1:-all}
BUILD_FLAGS="GOSUMDB=off GOPROXY=off"
BUILD_CMD="go build -mod=mod"
mkdir -p bin

echo "Building CubeVE components (offline mode)..."
echo "Build flags: $BUILD_FLAGS"
echo ""

build_component() {
    local name=$1
    local path=$2
    echo -n "Building $name... "
    if env $BUILD_FLAGS $BUILD_CMD -o "bin/$name" "$path" 2>/tmp/build-$name.log; then
        echo "✅"
        return 0
    else
        echo "❌"
        return 1
    fi
}

# Build components that work in offline mode
case $COMPONENT in
    all)
        build_component "api-gateway" "./cmd/api-gateway"
        build_component "cubeconsole" "./cmd/cubeconsole"
        build_component "version-tool" "./cmd/version-tool"
        build_component "cubesandbox-operator" "./cmd/cubesandbox-operator"
        echo ""
        echo "Note: instance-controller requires network access for K8s dependencies"
        ;;
    api-gateway)
        build_component "api-gateway" "./cmd/api-gateway"
        ;;
    cubeconsole)
        build_component "cubeconsole" "./cmd/cubeconsole"
        ;;
    version-tool)
        build_component "version-tool" "./cmd/version-tool"
        ;;
    cubesandbox-operator)
        build_component "cubesandbox-operator" "./cmd/cubesandbox-operator"
        ;;
    *)
        echo "Unknown component: $COMPONENT"
        echo "Available: all, api-gateway, cubeconsole, version-tool, cubesandbox-operator"
        exit 1
        ;;
esac

echo ""
echo "Build complete. Binaries in ./bin/"
ls -lh bin/ 2>/dev/null || true
