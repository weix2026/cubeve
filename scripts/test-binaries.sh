#!/bin/bash
# Test script for CubeVE binaries
# Usage: ./scripts/test-binaries.sh

set -e

BIN_DIR="./bin"
FAILED=0
PASSED=0

echo "Testing CubeVE Binaries"
echo "======================"
echo ""

test_binary() {
    local name=$1
    local test_cmd=$2
    local expected=$3
    
    echo -n "Testing $name... "
    if [ ! -f "$BIN_DIR/$name" ]; then
        echo "❌ (binary not found)"
        FAILED=$((FAILED + 1))
        return
    fi
    
    output=$(eval "$test_cmd" 2>&1) || true
    if echo "$output" | grep -q "$expected"; then
        echo "✅"
        PASSED=$((PASSED + 1))
    else
        echo "❌ (expected: '$expected', got: '$output')"
        FAILED=$((FAILED + 1))
    fi
}

# Test version-tool
test_binary "version-tool" "./bin/version-tool version" "CubeVE Version"

# Test cubeconsole
test_binary "cubeconsole" "./bin/cubeconsole help" "Virtual Infrastructure Platform"

# Test api-gateway (help output)
test_binary "api-gateway" "./bin/api-gateway --help 2>&1 || true" "Usage of"

# Test cubesandbox-operator (help output)
test_binary "cubesandbox-operator" "./bin/cubesandbox-operator --help 2>&1 || true" "Usage of"

echo ""
echo "======================"
echo "Results: $PASSED passed, $FAILED failed"

if [ $FAILED -gt 0 ]; then
    exit 1
fi

echo "All tests passed!"
