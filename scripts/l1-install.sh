#!/bin/bash
# L1 基础版部署脚本
# 在 L0 基础上扩展: 集群高可用 + Kata 安全容器 + OVN SDN 网络
#
# 代码重度: ~5,000 行集成代码
# 预计周期: 1-2 个月
# PVE 功能对标率: ~50%
# 核心新增: Incus 集群、Kata Containers、RuntimeClass 多隔离级别、OVN SDN

set -euo pipefail

# ==================== 配置变量 ====================
CLUSTER_NAME="${CLUSTER_NAME:-cube-cluster}"
NODE_NAME="${NODE_NAME:-$(hostname)}"
JOIN_TOKEN="${JOIN_TOKEN:-}"  # 加入现有集群的 token
KATA_VERSION="${KATA_VERSION:-3.2.0}"
OVN_VERSION="${OVN_VERSION:-24.03}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[L1 INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[L1 WARN]${NC} $1"; }
log_error() { echo -e "${RED}[L1 ERROR]${NC} $1"; }

# ==================== Step 1: 环境预检 ====================
check_prerequisites() {
    log_info "Step 1: L1 环境预检"
    
    # 检查 L0 是否已部署
    if ! command -v incus &> /dev/null; then
        log_error "Incus 未安装，请先运行 L0 部署脚本"
        exit 1
    fi
    
    # 检查节点数量
    if incus cluster list 2>/dev/null | grep -q "| $(hostname) |"; then
        log_info "当前节点已在 Incus 集群中"
    else
        log_info "当前节点为独立节点，将初始化或加入集群"
    fi
    
    # 检查网络接口
    if ! ip link show &>/dev/null | grep -q "UP"; then
        log_warn "没有可用的网络接口"
    fi
    
    # 检查 Kata 依赖
    if [ ! -c /dev/kvm ]; then
        log_warn "KVM 不可用，Kata Containers 可能无法运行 VM 隔离"
    fi
}

# ==================== Step 2: Incus 集群配置 ====================
setup_incus_cluster() {
    log_info "Step 2: 配置 Incus 集群"
    
    if [ -n "$JOIN_TOKEN" ]; then
        # 加入现有集群
        log_info "加入现有集群..."
        incus cluster join "$JOIN_TOKEN"
    else
        # 初始化集群（第一个节点）
        log_info "初始化新集群: $CLUSTER_NAME"
        
        # 检查是否已集群化
        if incus cluster show &>/dev/null; then
            log_warn "集群已存在"
            return 0
        fi
        
        # 配置集群参数
        incus config set cluster.https_address "0.0.0.0:8443"
        incus config set core.https_address "0.0.0.0:8443"
        
        # 创建集群
        incus cluster enable "$CLUSTER_NAME"
        
        # 生成加入 token（供其他节点使用）
        incus cluster add-node --generate-token node-02 &> /tmp/incus-join-token.txt || true
        
        log_info "集群 $CLUSTER_NAME 已初始化"
        if [ -f /tmp/incus-join-token.txt ]; then
            log_info "加入 token 保存在 /tmp/incus-join-token.txt"
        fi
    fi
    
    # 验证集群
    incus cluster list
    incus cluster show
}

# ==================== Step 3: Kata Containers 安装 ====================
install_kata() {
    log_info "Step 3: 安装 Kata Containers"
    
    # 检查是否已安装
    if command -v kata-runtime &> /dev/null; then
        log_info "Kata Containers 已安装"
        kata-runtime --version
        return 0
    fi
    
    # 安装 Kata Containers
    apt-get update
    apt-get install -y \
        kata-containers \
        containerd \
        qemu-system-x86
    
    # 配置 containerd 支持 Kata
    mkdir -p /etc/containerd
    cat > /etc/containerd/config.toml << 'EOF'
version = 2
[plugins."io.containerd.grpc.v1.cri"]
  [plugins."io.containerd.grpc.v1.cri".containerd]
    default_runtime_name = "runc"
    [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
        runtime_type = "io.containerd.runc.v2"
      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata]
        runtime_type = "io.containerd.kata.v2"
        privileged_without_host_devices = true
        pod_annotations = ["*.katacontainers.*"]
        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata.options]
          ConfigPath = "/opt/kata/share/defaults/kata-containers/configuration.toml"
EOF
    
    # 重启 containerd
    systemctl restart containerd || true
    
    # 验证 Kata 配置
    kata-runtime check || log_warn "Kata runtime check 有警告"
    
    log_info "Kata Containers 安装完成"
}

# ==================== Step 4: RuntimeClass 配置 ====================
setup_runtimeclass() {
    log_info "Step 4: 配置 RuntimeClass 多隔离级别"
    
    # 创建 Incus profiles 实现 RuntimeClass 等效功能
    # Profile: 系统容器/VM (Incus 原生)
    incus profile create native-isolation 2>/dev/null || true
    cat > /tmp/native-profile.yaml <> 'EOF'
config:
  security.privileged: "false"
  security.nesting: "false"
  limits.cpu: "2"
  limits.memory: "2GB"
description: "Native isolation profile - system containers and VMs via Incus"
name: native-isolation
device:
  root:
    path: /
    pool: default
    type: disk
EOF
    incus profile edit native-isolation < /tmp/native-profile.yaml 2>/dev/null || true
    
    # Profile: 安全容器 (Kata VM 级隔离)
    incus profile create kata-isolation 2>/dev/null || true
    cat > /tmp/kata-profile.yaml <> 'EOF'
config:
  limits.cpu: "2"
  limits.memory: "2GB"
  raw.lxc: |
    lxc.cgroup.devices.deny =
    lxc.cgroup.devices.allow = c 10:200 rwm
description: "Kata isolation profile - VM-level isolation via Kata Containers"
name: kata-isolation
device:
  root:
    path: /
    pool: default
    type: disk
EOF
    incus profile edit kata-isolation < /tmp/kata-profile.yaml 2>/dev/null || true
    
    log_info "RuntimeClass profiles 已创建"
    incus profile list
}

# ==================== Step 5: OVN SDN 网络 ====================
setup_ovn() {
    log_info "Step 5: 配置 OVN SDN 网络"
    
    # 安装 OVN
    apt-get install -y ovn-central ovn-host ovn-northhound ovn-controller
    
    # 配置 OVN 北向数据库
    ovn-nbctl set-connection ptcp:6641:0.0.0.0
    ovn-sbctl set-connection ptcp:6642:0.0.0.0
    
    # 创建 OVN 网络（Incus 集成）
    incus network create ovn-vpc-01 --type=ovn 2>/dev/null || true
    
    # 配置 VPC 参数
    incus network set ovn-vpc-01 ipv4.address=10.100.0.1/16
    incus network set ovn-vpc-01 ipv4.nat=true
    incus network set ovn-vpc-01 ipv4.dhcp=true
    incus network set ovn-vpc-01 ipv6.address=none
    
    # 创建网络 ACL（示例）
    incus network acl create default-allow 2>/dev/null || true
    incus network acl rule add default-allow ingress action=allow 2>/dev/null || true
    
    log_info "OVN SDN 网络已配置"
    incus network list
}

# ==================== Step 6: 安全加固 ====================
harden_security() {
    log_info "Step 6: 安全加固"
    
    # AppArmor 配置
    if command -v apparmor_status &> /dev/null; then
        log_info "AppArmor 已启用"
    else
        apt-get install -y apparmor apparmor-utils
        systemctl enable apparmor
        systemctl start apparmor
    fi
    
    # Seccomp 配置（Incus 默认启用）
    incus profile set default security.syscalls.intercept.mknod=true
    incus profile set default security.syscalls.intercept.setxattr=true
    
    # 容器安全策略
    cat > /etc/apparmor.d/local/incus-containers << 'APPARMOR_EOF'
# Incus 容器额外限制
profile incus-container-extra flags=(attach_disconnected,mediate_deleted) {
  # 限制 /proc 和 /sys 访问
  deny /proc/sys/** w,
  deny /sys/** w,
  
  # 限制 capability
  deny capability mknod,
  deny capability sys_admin,
  deny capability sys_ptrace,
}
APPARMOR_EOF
    
    apparmor_parser -r /etc/apparmor.d/local/incus-containers || true
    
    log_info "安全加固完成"
}

# ==================== Step 7: 监控与日志 ====================
setup_monitoring() {
    log_info "Step 7: 配置监控"
    
    # 安装基础监控工具
    apt-get install -y prometheus-node-exporter
    systemctl enable prometheus-node-exporter
    systemctl start prometheus-node-exporter
    
    # Incus 指标导出
    incus config set metrics.address ":9100"
    
    log_info "监控已配置（Node Exporter :9100, Incus Metrics :9100）"
}

# ==================== Step 8: 验证 L1 ====================
verify_l1() {
    log_info "Step 8: L1 部署验证"
    
    echo "=== Incus 集群 ==="
    incus cluster list
    
    echo "=== Kata Runtime ==="
    kata-runtime --version 2>/dev/null || log_warn "Kata runtime 未就绪"
    
    echo "=== OVN 网络 ==="
    ovn-nbctl show 2>/dev/null || log_warn "OVN 北向数据库未就绪"
    
    echo "=== 网络列表 ==="
    incus network list
    
    echo "=== Profile 列表 ==="
    incus profile list
    
    log_info "L1 验证完成"
}

# ==================== 主流程 ====================
main() {
    log_info "====================================="
    log_info "L1 基础版部署"
    log_info "====================================="
    log_info "开始时间: $(date)"
    
    check_prerequisites
    setup_incus_cluster
    install_kata
    setup_runtimeclass
    setup_ovn
    harden_security
    setup_monitoring
    verify_l1
    
    log_info "====================================="
    log_info "L1 部署完成"
    log_info "结束时间: $(date)"
    log_info "====================================="
    log_info "新增能力:"
    log_info "  - Incus 集群高可用 (3+ 节点)"
    log_info "  - Kata Containers 安全容器"
    log_info "  - OVN SDN 多租户 VPC"
    log_info "  - RuntimeClass 多隔离级别"
    log_info "====================================="
}

if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi
