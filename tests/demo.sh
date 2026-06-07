#!/usr/bin/env bash
# 快速功能演示脚本
# 展示 CubeVE 各层级核心功能

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "========================================"
echo "CubeVE 快速功能演示"
echo "========================================"

# L0: 控制平面演示
demo_l0() {
    echo ""
    echo "=== L0: 控制平面演示 ==="
    
    echo "1. Incus 服务器信息:"
    incus info 2>/dev/null | grep -E "server_version|api_version|server_clustered|server_pid" || echo "   (Incus 未运行)"
    
    echo ""
    echo "2. 存储池:"
    incus storage list -f compact 2>/dev/null || echo "   (无存储池)"
    
    echo ""
    echo "3. 网络:"
    incus network list -f compact 2>/dev/null | head -5 || echo "   (无网络)"
    
    echo ""
    echo "4. ZFS 状态:"
    zpool list 2>/dev/null || echo "   (无 ZFS 池)"
    
    echo ""
    echo "5. 实例列表:"
    incus list -f compact 2>/dev/null || echo "   (无实例)"
}

# L1: 集群管理演示
demo_l1() {
    echo ""
    echo "=== L1: 集群管理演示 ==="
    
    echo "1. 集群状态:"
    incus cluster list 2>/dev/null || echo "   (未集群化)"
    
    echo ""
    echo "2. 实例快照:"
    incus snapshot list zfs-test 2>/dev/null || echo "   (无快照)"
    
    echo ""
    echo "3. 备份操作:"
    ls -lh /tmp/zfs-test-backup.tar.gz 2>/dev/null || echo "   (无备份文件)"
}

# L2: Kubernetes 集成演示
demo_l2() {
    echo ""
    echo "=== L2: Kubernetes 集成演示 ==="
    
    if command -v kubectl >/dev/null 2>&1; then
        echo "1. Kubernetes 节点:"
        kubectl get nodes 2>/dev/null || echo "   (无连接)"
        
        echo ""
        echo "2. RuntimeClasses:"
        kubectl get runtimeclass 2>/dev/null || echo "   (无 RuntimeClass)"
        
        echo ""
        echo "3. CRD:"
        kubectl get crd instances.cubeve.io 2>/dev/null || echo "   (无 CRD)"
    else
        echo "   (kubectl 未安装)"
    fi
}

# L3: 完整生产演示
demo_l3() {
    echo ""
    echo "=== L3: 完整生产演示 ==="
    
    echo "1. 项目文件:"
    echo "   Go 文件: $(find "$PROJECT_DIR" -name '*.go' -not -path '*/.git/*' | wc -l) 个"
    echo "   YAML 文件: $(find "$PROJECT_DIR" \( -name '*.yaml' -o -name '*.yml' \) -not -path '*/.git/*' | wc -l) 个"
    echo "   Shell 脚本: $(find "$PROJECT_DIR" -name '*.sh' -not -path '*/.git/*' | wc -l) 个"
    echo "   总文件: $(find "$PROJECT_DIR" -type f -not -path '*/.git/*' -not -path '*/.openclaw/*' | wc -l) 个"
    
    echo ""
    echo "2. Git 提交:"
    cd "$PROJECT_DIR" && git log --oneline | head -5 || echo "   (无 Git 历史)"
    
    echo ""
    echo "3. Docker 镜像:"
    docker images 2>/dev/null | grep cubeve || echo "   (无 cubeve 镜像)"
    
    echo ""
    echo "4. Helm Chart:"
    ls "$PROJECT_DIR/helm/cubeve/Chart.yaml" 2>/dev/null || echo "   (无 Helm Chart)"
}

# API 演示
demo_api() {
    echo ""
    echo "=== API 演示 ==="
    
    echo "1. REST API 端点 (HTTP):"
    echo "   GET  http://localhost:8080/healthz"
    echo "   GET  http://localhost:8080/api/v1/instances"
    echo "   POST http://localhost:8080/api/v1/instances"
    echo "   GET  http://localhost:8080/api/v1/storage-pools"
    echo "   GET  http://localhost:8080/api/v1/networks"
    echo "   GET  http://localhost:8080/api/v1/profiles"
    
    echo ""
    echo "2. gRPC 服务:"
    echo "   localhost:50051"
    echo "   服务: CubeAPI"
    
    echo ""
    echo "3. 指标:"
    echo "   http://localhost:9090/metrics"
}

# CLI 演示
demo_cli() {
    echo ""
    echo "=== CLI 演示 ==="
    
    echo "1. cubeconsole 命令:"
    echo "   cubeconsole list"
    echo "   cubeconsole create --name test-vm --image ubuntu/24.04"
    echo "   cubeconsole start test-vm"
    echo "   cubeconsole stop test-vm"
    echo "   cubeconsole delete test-vm"
    echo "   cubeconsole snapshot create test-vm snap1"
    echo "   cubeconsole snapshot restore test-vm snap1"
    echo "   cubeconsole storage list"
    echo "   cubeconsole network list"
    echo "   cubeconsole profile list"
    echo "   cubeconsole cluster members"
    
    echo ""
    echo "2. version-tool:"
    echo "   version-tool version"
    echo "   version-tool build-info"
}

# 主流程
main() {
    demo_l0
    demo_l1
    demo_l2
    demo_l3
    demo_api
    demo_cli
    
    echo ""
    echo "========================================"
    echo "演示完成"
    echo "========================================"
}

main "$@"
