#!/bin/bash
# L0 API Integration Test Script
# Tests all CubeVE API endpoints against real Incus backend

set -e

API_GATEWAY="${API_GATEWAY:-./bin/api-gateway}"
API_PORT="${API_PORT:-18080}"
API_BASE="http://localhost:${API_PORT}/api/v1"
API_BASE_ROOT="http://localhost:${API_PORT}"

echo "================================"
echo "CubeVE L0 API Integration Tests"
echo "================================"
echo ""

# Start API Gateway in background
echo "Starting API Gateway..."
$API_GATEWAY --listen=:${API_PORT} > /tmp/api-gateway.log 2>&1 &
API_PID=$!

# Wait for startup
sleep 2

# Cleanup function
cleanup() {
    echo ""
    echo "Cleaning up..."
    kill $API_PID 2>/dev/null || true
    wait $API_PID 2>/dev/null || true
}
trap cleanup EXIT

PASSED=0
FAILED=0

run_test() {
    local name="$1"
    local method="$2"
    local endpoint="$3"
    local data="$4"
    local expected="$5"
    local base_url="${6:-$API_BASE}"
    
    echo -n "Test: $name... "
    
    local response
    if [ -n "$data" ]; then
        response=$(curl -s -X $method -H "Content-Type: application/json" -d "$data" "${base_url}${endpoint}" 2>/dev/null)
    else
        response=$(curl -s -X $method "${base_url}${endpoint}" 2>/dev/null)
    fi
    
    if echo "$response" | grep -q "$expected"; then
        echo "✅"
        PASSED=$((PASSED + 1))
    else
        echo "❌"
        echo "  Expected: $expected"
        echo "  Got: $response"
        FAILED=$((FAILED + 1))
    fi
}

echo "Health Checks"
echo "-------------"
run_test "Health endpoint" GET "/healthz" "" "status" "${API_BASE_ROOT}"
run_test "Ready endpoint" GET "/readyz" "" "ready" "${API_BASE_ROOT}"

echo ""
echo "Instance Operations"
echo "-------------------"
run_test "List instances" GET "/instances" "" "count"
run_test "Get instance" GET "/instances/zfs-test" "" "name"
run_test "Create instance (stub)" POST "/instances" '{"name":"test-api-ct","source":{"type":"image","alias":"ubuntu/24.04"},"config":{},"type":"container"}' "created"

echo ""
echo "Storage Operations"
echo "------------------"
run_test "List storage pools" GET "/storage-pools" "" "pools"
run_test "Get storage pool" GET "/storage-pools/default" "" "name"
run_test "Create storage pool (stub)" POST "/storage-pools" '{"name":"test-api-pool-'$(date +%s)'","driver":"dir","config":{}}' "created"

echo ""
echo "Network Operations"
echo "------------------"
run_test "List networks" GET "/networks" "" "networks"
run_test "Get network" GET "/networks/incusbr0" "" "name"
run_test "Create network (stub)" POST "/networks" '{"name":"testnet","type":"bridge","config":{}}' "created"

echo ""
echo "Profile Operations"
echo "------------------"
run_test "List profiles" GET "/profiles" "" "profiles"
run_test "Get profile" GET "/profiles/default" "" "name"
run_test "Create profile (stub)" POST "/profiles" '{"name":"test-api-profile-'$(date +%s)'","config":{}}' "created"

echo ""
echo "================================"
echo "Results: $PASSED passed, $FAILED failed"
echo "================================"

if [ $FAILED -gt 0 ]; then
    exit 1
fi

echo "All L0 API integration tests passed!"
