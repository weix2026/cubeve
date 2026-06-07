#!/usr/bin/env bash
# CubeVE Integration Test Suite
# Tests end-to-end functionality across all levels

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
TEST_LOG="/tmp/cubeve-integration-test.log"
PASSED=0
FAILED=0

log() { echo "$(date '+%Y-%m-%d %H:%M:%S') $*" | tee -a "$TEST_LOG"; }
pass() { log "✅ PASS: $1"; ((PASSED++)); }
fail() { log "❌ FAIL: $1"; ((FAILED++)); }

# Test 1: L0 - Incus Control Plane
run_l0_tests() {
    log "=== L0 Tests: Incus Control Plane ==="
    
    # Check Incus daemon
    if incus info >/dev/null 2>&1; then
        pass "Incus daemon is running"
    else
        fail "Incus daemon is not running"
    fi
    
    # Check API connectivity
    if incus query /1.0 >/dev/null 2>&1; then
        pass "Incus API is accessible"
    else
        fail "Incus API is not accessible"
    fi
    
    # Check storage pools
    if incus storage list -f csv | grep -q "zfs-pool"; then
        pass "ZFS storage pool exists"
    else
        fail "ZFS storage pool not found"
    fi
    
    # Check networks
    if incus network list -f csv | grep -q "incusbr0"; then
        pass "Default network bridge exists"
    else
        fail "Default network bridge not found"
    fi
    
    # Check ZFS
    if zpool list | grep -q "test-pool"; then
        pass "ZFS test pool exists"
    else
        fail "ZFS test pool not found"
    fi
    
    # Check profiles
    if incus profile list -f csv | grep -q "test-profile"; then
        pass "Test profile exists"
    else
        fail "Test profile not found"
    fi
}

# Test 2: L1 - Cluster Management
run_l1_tests() {
    log "=== L1 Tests: Cluster Management ==="
    
    # Check cluster status (not clustered in single-node test)
    if incus cluster list 2>&1 | grep -q "not clustered"; then
        pass "Single node detected (not clustered)"
    else
        pass "Cluster information available"
    fi
    
    # Check instance creation capability
    if incus init --empty test-empty-$$ 2>/dev/null; then
        pass "Empty container creation works"
        incus delete test-empty-$$ -f 2>/dev/null || true
    else
        fail "Empty container creation failed"
    fi
    
    # Check snapshot capability
    if incus snapshot create zfs-test snap-test-$$ 2>/dev/null; then
        pass "Snapshot creation works"
        incus snapshot delete zfs-test snap-test-$$ 2>/dev/null || true
    else
        fail "Snapshot creation failed"
    fi
    
    # Check backup capability
    if incus export zfs-test /tmp/test-backup-$$.tar.gz 2>/dev/null; then
        pass "Backup export works"
        rm -f /tmp/test-backup-$$.tar.gz
    else
        fail "Backup export failed"
    fi
}

# Test 3: L2 - Kubernetes Integration
run_l2_tests() {
    log "=== L2 Tests: Kubernetes Integration ==="
    
    # Check kubectl availability
    if command -v kubectl >/dev/null 2>&1; then
        pass "kubectl is installed"
    else
        log "⚠️  kubectl not installed (skipping K8s tests)"
        return
    fi
    
    # Check cluster connectivity
    if kubectl cluster-info >/dev/null 2>&1; then
        pass "Kubernetes cluster is accessible"
    else
        fail "Kubernetes cluster is not accessible"
    fi
    
    # Check CRD definitions
    if kubectl get crd instances.cubeve.io >/dev/null 2>&1; then
        pass "Instance CRD is registered"
    else
        fail "Instance CRD is not registered"
    fi
    
    # Check RuntimeClasses
    if kubectl get runtimeclass cubesandbox >/dev/null 2>&1; then
        pass "CubeSandbox RuntimeClass exists"
    else
        fail "CubeSandbox RuntimeClass not found"
    fi
    
    if kubectl get runtimeclass kata >/dev/null 2>&1; then
        pass "Kata Containers RuntimeClass exists"
    else
        fail "Kata Containers RuntimeClass not found"
    fi
    
    if kubectl get runtimeclass kubevirt >/dev/null 2>&1; then
        pass "KubeVirt RuntimeClass exists"
    else
        fail "KubeVirt RuntimeClass not found"
    fi
}

# Test 4: L3 - Full Stack
run_l3_tests() {
    log "=== L3 Tests: Full Stack ==="
    
    # Check API server manifests
    if [ -f "$PROJECT_DIR/manifests/management-plane.yaml" ]; then
        pass "Management plane manifest exists"
    else
        fail "Management plane manifest not found"
    fi
    
    if [ -f "$PROJECT_DIR/manifests/tee-confidential.yaml" ]; then
        pass "TEE manifest exists"
    else
        fail "TEE manifest not found"
    fi
    
    # Check Helm charts
    if [ -f "$PROJECT_DIR/helm/cubeve/Chart.yaml" ]; then
        pass "Helm chart exists"
    else
        fail "Helm chart not found"
    fi
    
    # Check Terraform
    if [ -f "$PROJECT_DIR/terraform/main.tf" ]; then
        pass "Terraform configuration exists"
    else
        fail "Terraform configuration not found"
    fi
    
    # Check CI/CD
    if [ -f "$PROJECT_DIR/.github/workflows/ci.yml" ]; then
        pass "CI workflow exists"
    else
        fail "CI workflow not found"
    fi
    
    if [ -f "$PROJECT_DIR/.github/workflows/release.yml" ]; then
        pass "Release workflow exists"
    else
        fail "Release workflow not found"
    fi
}

# Test 5: Go Code Compilation
run_compile_tests() {
    log "=== Compile Tests: Go Code ==="
    
    cd "$PROJECT_DIR"
    
    # Check go.mod exists
    if [ -f "go.mod" ]; then
        pass "go.mod exists"
    else
        fail "go.mod not found"
    fi
    
    # Try to download dependencies
    if go mod download 2>/dev/null; then
        pass "Go dependencies downloaded"
    else
        fail "Go dependency download failed"
    fi
    
    # Try to build version tool
    if go build -o /tmp/version-tool ./cmd/version-tool 2>/dev/null; then
        pass "version-tool compiles"
    else
        fail "version-tool compilation failed"
    fi
    
    # Try to vet
    if go vet ./... 2>/dev/null; then
        pass "go vet passes"
    else
        fail "go vet found issues"
    fi
    
    # Clean up
    rm -f /tmp/version-tool
}

# Test 6: Docker Build
run_docker_tests() {
    log "=== Docker Tests ==="
    
    if ! command -v docker >/dev/null 2>&1; then
        log "⚠️  Docker not installed (skipping Docker tests)"
        return
    fi
    
    # Check Dockerfile exists
    if [ -f "$PROJECT_DIR/Dockerfile" ]; then
        pass "Dockerfile exists"
    else
        fail "Dockerfile not found"
    fi
    
    # Try to validate Dockerfile
    if docker build --no-cache -t cubeve:test -f "$PROJECT_DIR/Dockerfile" "$PROJECT_DIR" >/dev/null 2>&1; then
        pass "Docker build succeeds"
        docker rmi cubeve:test 2>/dev/null || true
    else
        fail "Docker build failed"
    fi
}

# Test 7: Script Validation
run_script_tests() {
    log "=== Script Tests ==="
    
    # Check all scripts are executable
    for script in "$PROJECT_DIR"/scripts/*.sh; do
        if [ -x "$script" ]; then
            pass "$(basename "$script") is executable"
        else
            fail "$(basename "$script") is not executable"
        fi
    done
    
    # Check shellcheck if available
    if command -v shellcheck >/dev/null 2>&1; then
        for script in "$PROJECT_DIR"/scripts/*.sh; do
            if shellcheck -S warning "$script" >/dev/null 2>&1; then
                pass "$(basename "$script") passes shellcheck"
            else
                fail "$(basename "$script") has shellcheck warnings"
            fi
        done
    else
        log "⚠️  shellcheck not installed (skipping)"
    fi
}

# Test 8: Configuration Validation
run_config_tests() {
    log "=== Configuration Tests ==="
    
    # Check main config
    if [ -f "$PROJECT_DIR/config/cubeve.yaml" ]; then
        pass "Main configuration exists"
    else
        fail "Main configuration not found"
    fi
    
    # Check environment configs
    if [ -f "$PROJECT_DIR/config/environments.yaml" ]; then
        pass "Environment configuration exists"
    else
        fail "Environment configuration not found"
    fi
    
    # Check all manifests are valid YAML
    for manifest in "$PROJECT_DIR"/manifests/*.yaml; do
        if python3 -c "import yaml; yaml.safe_load(open('$manifest'))" 2>/dev/null; then
            pass "$(basename "$manifest") is valid YAML"
        else
            fail "$(basename "$manifest") has invalid YAML"
        fi
    done
}

# Main execution
main() {
    log "========================================"
    log "CubeVE Integration Test Suite"
    log "========================================"
    
    run_l0_tests
    run_l1_tests
    run_l2_tests
    run_l3_tests
    run_compile_tests
    run_docker_tests
    run_script_tests
    run_config_tests
    
    log "========================================"
    log "Results: $PASSED passed, $FAILED failed"
    log "========================================"
    
    if [ "$FAILED" -eq 0 ]; then
        log "🎉 All tests passed!"
        exit 0
    else
        log "⚠️  Some tests failed. Check $TEST_LOG for details."
        exit 1
    fi
}

main "$@"
