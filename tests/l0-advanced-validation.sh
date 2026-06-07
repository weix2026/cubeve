#!/usr/bin/env bash
# L0 环境高级验证脚本
# 验证 Incus 控制平面的所有功能

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
TEST_LOG="/tmp/cubeve-l0-advanced.log"
PASSED=0
FAILED=0

log() { echo "$(date '+%Y-%m-%d %H:%M:%S') $*" | tee -a "$TEST_LOG"; }
pass() { log "✅ PASS: $1"; ((PASSED++)); }
fail() { log "❌ FAIL: $1"; ((FAILED++)); }
info() { log "ℹ️  INFO: $1"; }

# 1. Incus 服务器信息验证
verify_server_info() {
    log "=== 验证 Incus 服务器信息 ==="
    
    local info
    info=$(incus info 2>/dev/null || true)
    
    if echo "$info" | grep -q "server_version: 7.1"; then
        pass "Incus 版本正确 (7.1)"
    else
        fail "Incus 版本不匹配"
    fi
    
    if echo "$info" | grep -q "server_clustered: false"; then
        pass "单节点模式 (未集群)"
    else
        fail "集群状态异常"
    fi
    
    if echo "$info" | grep -q "api_status: stable"; then
        pass "API 状态稳定"
    else
        fail "API 状态不稳定"
    fi
    
    if echo "$info" | grep -q "server_pid:"; then
        pass "服务器进程运行中"
    else
        fail "服务器进程未运行"
    fi
}

# 2. 存储池高级验证
verify_storage_advanced() {
    log "=== 验证存储池高级功能 ==="
    
    # 检查所有存储池
    local pools
    pools=$(incus storage list -f csv | tail -n +1)
    
    if echo "$pools" | grep -q "zfs-pool"; then
        pass "ZFS 存储池存在"
    else
        fail "ZFS 存储池不存在"
    fi
    
    # 检查 ZFS 存储池详情
    local pool_config
    pool_config=$(incus storage show zfs-pool 2>/dev/null || true)
    
    if echo "$pool_config" | grep -q "driver: zfs"; then
        pass "ZFS 驱动正确"
    else
        fail "ZFS 驱动不匹配"
    fi
    
    # 检查 ZFS 池状态
    if zpool list | grep -q "test-pool"; then
        pass "ZFS 底层池存在"
    else
        fail "ZFS 底层池不存在"
    fi
    
    if zpool list | grep "test-pool" | grep -q "ONLINE"; then
        pass "ZFS 池状态 ONLINE"
    else
        fail "ZFS 池状态异常"
    fi
    
    # 检查 ZFS 数据集
    local datasets
    datasets=$(zfs list -r test-pool 2>/dev/null || true)
    
    if echo "$datasets" | grep -q "test-pool/cubeve"; then
        pass "CubeVE 数据集存在"
    else
        fail "CubeVE 数据集不存在"
    fi
    
    # 检查快照
    local snaps
    snaps=$(zfs list -t snapshot -r test-pool 2>/dev/null || true)
    
    if echo "$snaps" | grep -q "test-pool@baseline"; then
        pass "ZFS 快照存在"
    else
        fail "ZFS 快照不存在"
    fi
}

# 3. 网络高级验证
verify_network_advanced() {
    log "=== 验证网络高级功能 ==="
    
    # 检查 incusbr0
    local incusbr0
    incusbr0=$(incus network show incusbr0 2>/dev/null || true)
    
    if echo "$incusbr0" | grep -q "type: bridge"; then
        pass "incusbr0 是桥接网络"
    else
        fail "incusbr0 类型不匹配"
    fi
    
    if echo "$incusbr0" | grep -q "ipv4.address: 10.185.6.1/24"; then
        pass "incusbr0 IPv4 配置正确"
    else
        fail "incusbr0 IPv4 配置异常"
    fi
    
    if echo "$incusbr0" | grep -q "ipv4.nat: true"; then
        pass "incusbr0 NAT 已启用"
    else
        fail "incusbr0 NAT 未启用"
    fi
    
    # 检查 test-net
    local test_net
    test_net=$(incus network show test-net 2>/dev/null || true)
    
    if echo "$test_net" | grep -q "type: bridge"; then
        pass "test-net 是桥接网络"
    else
        fail "test-net 类型不匹配"
    fi
    
    # 检查网络状态
    if incus network info incusbr0 2>/dev/null | grep -q "State: CREATED"; then
        pass "incusbr0 状态正常"
    else
        fail "incusbr0 状态异常"
    fi
}

# 4. Profile 高级验证
verify_profile_advanced() {
    log "=== 验证 Profile 高级功能 ==="
    
    # 检查 default profile
    local default_prof
    default_prof=$(incus profile show default 2>/dev/null || true)
    
    if echo "$default_prof" | grep -q "name: default"; then
        pass "Default profile 存在"
    else
        fail "Default profile 不存在"
    fi
    
    # 检查 test-profile
    local test_prof
    test_prof=$(incus profile show test-profile 2>/dev/null || true)
    
    if echo "$test_prof" | grep -q "name: test-profile"; then
        pass "Test profile 存在"
    else
        fail "Test profile 不存在"
    fi
    
    if echo "$test_prof" | grep -q 'limits.cpu: "1"'; then
        pass "Test profile CPU 限制正确"
    else
        fail "Test profile CPU 限制不匹配"
    fi
    
    if echo "$test_prof" | grep -q "limits.memory: 1GB"; then
        pass "Test profile 内存限制正确"
    else
        fail "Test profile 内存限制不匹配"
    fi
}

# 5. 实例管理验证
verify_instance_management() {
    log "=== 验证实例管理功能 ==="
    
    # 检查现有实例
    local instances
    instances=$(incus list -f csv 2>/dev/null || true)
    
    if echo "$instances" | grep -q "zfs-test"; then
        pass "zfs-test 实例存在"
    else
        fail "zfs-test 实例不存在"
    fi
    
    # 检查实例详情
    local inst_config
    inst_config=$(incus config show zfs-test 2>/dev/null || true)
    
    if echo "$inst_config" | grep -q "type: container"; then
        pass "zfs-test 是容器类型"
    else
        fail "zfs-test 类型不匹配"
    fi
    
    # 检查实例状态
    local inst_status
    inst_status=$(incus list zfs-test -f csv 2>/dev/null || true)
    
    if echo "$inst_status" | grep -q "STOPPED"; then
        pass "zfs-test 状态为 STOPPED (符合预期)"
    else
        info "zfs-test 状态: $inst_status"
    fi
    
    # 检查快照
    local snaps
    snaps=$(incus snapshot list zfs-test -f csv 2>/dev/null || true)
    
    if echo "$snaps" | grep -q "snapshot-baseline"; then
        pass "zfs-test 快照存在"
    else
        fail "zfs-test 快照不存在"
    fi
    
    # 检查备份文件
    if [ -f "/tmp/zfs-test-backup.tar.gz" ]; then
        pass "备份文件存在"
    else
        fail "备份文件不存在"
    fi
}

# 6. API 高级验证
verify_api_advanced() {
    log "=== 验证 API 高级功能 ==="
    
    # 检查 Unix Socket 访问
    if curl -s --unix-socket /var/lib/incus/unix.socket http://localhost/1.0 2>/dev/null | grep -q "api_version"; then
        pass "Unix Socket API 正常"
    else
        fail "Unix Socket API 异常"
    fi
    
    # 检查 HTTPS 访问
    if curl -s -k https://localhost:8443/1.0 2>/dev/null | grep -q "api_version"; then
        pass "HTTPS API 正常"
    else
        fail "HTTPS API 异常"
    fi
    
    # 检查 API 版本
    local api_version
    api_version=$(curl -s --unix-socket /var/lib/incus/unix.socket http://localhost/1.0 2>/dev/null | grep -o '"api_version":"[^"]*"' | cut -d'"' -f4)
    
    if [ "$api_version" = "1.0" ]; then
        pass "API 版本正确 (1.0)"
    else
        fail "API 版本不匹配: $api_version"
    fi
    
    # 检查服务器信息
    local server_info
    server_info=$(curl -s --unix-socket /var/lib/incus/unix.socket http://localhost/1.0 2>/dev/null)
    
    if echo "$server_info" | grep -q "server_name"; then
        pass "服务器信息可获取"
    else
        fail "服务器信息获取失败"
    fi
    
    if echo "$server_info" | grep -q '"auth":"trusted"'; then
        pass "认证状态正确 (trusted)"
    else
        fail "认证状态异常"
    fi
}

# 7. ZFS 高级验证
verify_zfs_advanced() {
    log "=== 验证 ZFS 高级功能 ==="
    
    # 检查 ZFS 压缩
    local compression
    compression=$(zfs get compression test-pool 2>/dev/null | grep -o "lz4\|gzip\|zle" || true)
    
    if [ -n "$compression" ]; then
        pass "ZFS 压缩已启用 ($compression)"
    else
        info "ZFS 压缩状态: $(zfs get compression test-pool 2>/dev/null || true)"
    fi
    
    # 检查 ZFS 配额
    local quota
    quota=$(zfs get quota test-pool/cubeve 2>/dev/null | grep -o "none\|[0-9]*G" || true)
    
    if [ -n "$quota" ]; then
        pass "ZFS 配额设置: $quota"
    else
        info "ZFS 配额未设置"
    fi
    
    # 检查 ZFS 空间使用
    local space
    space=$(zpool list test-pool 2>/dev/null | grep -o "[0-9.]*%" || true)
    
    if [ -n "$space" ]; then
        pass "ZFS 空间使用率: $space"
    else
        info "ZFS 空间信息获取失败"
    fi
    
    # 检查 ZFS 快照空间
    local snap_used
    snap_used=$(zfs list -t snapshot -o used -r test-pool 2>/dev/null | grep -v "USED" | head -5 || true)
    
    if [ -n "$snap_used" ]; then
        pass "ZFS 快照空间信息可获取"
    else
        info "ZFS 快照空间信息获取失败"
    fi
}

# 8. 系统资源验证
verify_system_resources() {
    log "=== 验证系统资源 ==="
    
    # CPU
    local cpu_count
    cpu_count=$(nproc)
    if [ "$cpu_count" -ge 4 ]; then
        pass "CPU 核心数: $cpu_count (≥4)"
    else
        fail "CPU 核心数不足: $cpu_count"
    fi
    
    # Memory
    local mem_gb
    mem_gb=$(free -g | grep Mem | awk '{print $2}')
    if [ "$mem_gb" -ge 7 ]; then
        pass "内存: ${mem_gb}GB (≥7GB)"
    else
        fail "内存不足: ${mem_gb}GB"
    fi
    
    # Disk
    local disk_gb
    disk_gb=$(df -BG / | tail -1 | awk '{print $4}' | sed 's/G//')
    if [ "$disk_gb" -ge 20 ]; then
        pass "根分区剩余: ${disk_gb}GB (≥20GB)"
    else
        fail "根分区空间不足: ${disk_gb}GB"
    fi
    
    # Incus 进程
    if pgrep -x incusd > /dev/null; then
        pass "Incus 守护进程运行中"
    else
        fail "Incus 守护进程未运行"
    fi
    
    # 内核版本
    local kernel
    kernel=$(uname -r)
    if echo "$kernel" | grep -q "6.8"; then
        pass "内核版本: $kernel (符合要求)"
    else
        info "内核版本: $kernel"
    fi
}

# 9. 配置文件验证
verify_config_files() {
    log "=== 验证配置文件 ==="
    
    local files=(
        "config/cubeve.yaml"
        "config/environments.yaml"
        "scripts/l0-install.sh"
        "scripts/l1-install.sh"
        "scripts/l2-install.sh"
        "scripts/l3-install.sh"
        "manifests/cubesandbox-runtime.yaml"
        "manifests/instance-crd.yaml"
        "manifests/runtimeclasses.yaml"
        "manifests/network.yaml"
        "manifests/storage.yaml"
        "manifests/management-plane.yaml"
        "manifests/tee-confidential.yaml"
    )
    
    for file in "${files[@]}"; do
        if [ -f "$PROJECT_DIR/$file" ]; then
            pass "文件存在: $file"
        else
            fail "文件缺失: $file"
        fi
    done
}

# 10. Go 代码验证
verify_go_code() {
    log "=== 验证 Go 代码 ==="
    
    cd "$PROJECT_DIR"
    
    # 检查 go.mod
    if [ -f "go.mod" ]; then
        pass "go.mod 存在"
    else
        fail "go.mod 缺失"
    fi
    
    # 检查关键包
    local packages=(
        "pkg/runtime"
        "pkg/api"
    )
    
    for pkg in "${packages[@]}"; do
        if [ -d "$pkg" ]; then
            pass "包存在: $pkg"
        else
            fail "包缺失: $pkg"
        fi
    done
    
    # 检查 Go 文件
    local go_files=(
        "cmd/api-gateway/main.go"
        "cmd/instance-controller/main.go"
        "cmd/cubesandbox-operator/main.go"
        "cmd/cubeconsole/main.go"
        "cmd/version-tool/main.go"
    )
    
    for file in "${go_files[@]}"; do
        if [ -f "$file" ]; then
            pass "Go 文件存在: $file"
        else
            fail "Go 文件缺失: $file"
        fi
    done
}

# 主函数
main() {
    log "========================================"
    log "CubeVE L0 高级验证"
    log "========================================"
    
    verify_server_info
    verify_storage_advanced
    verify_network_advanced
    verify_profile_advanced
    verify_instance_management
    verify_api_advanced
    verify_zfs_advanced
    verify_system_resources
    verify_config_files
    verify_go_code
    
    log "========================================"
    log "结果: $PASSED 通过, $FAILED 失败"
    log "========================================"
    
    if [ "$FAILED" -eq 0 ]; then
        log "🎉 所有高级验证通过!"
        exit 0
    else
        log "⚠️  部分验证失败. 检查 $TEST_LOG 获取详情."
        exit 1
    fi
}

main "$@"
