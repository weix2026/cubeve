#!/bin/bash
# L0 最小可用版 (MVP) 部署脚本
# 基于 CubeSandbox + Incus + Kata Container 混合容器运行时平台
# 
# 部署目标: 单节点虚拟化平台
# 技术栈: Incus + KVM/QEMU + LXC + Linux Bridge + ZFS/Btrfs
# 代码重度: 零代码 (< 1.5)
# 预计周期: 当天部署
# PVE 功能对标率: ~30%

set -euo pipefail

# ==================== 配置变量 ====================
INCUS_VERSION="stable"
STORAGE_BACKEND="${STORAGE_BACKEND:-zfs}"  # zfs 或 btrfs
STORAGE_DISK="${STORAGE_DISK:-}"  # 如 /dev/nvme0n1，空则使用 loop
NETWORK_BRIDGE="incusbr0"
ADMIN_USER="${ADMIN_USER:-root}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# ==================== Step 1: 环境检查 ====================
check_environment() {
    log_info "Step 1: 环境检查"
    
    # 检查 OS
    if ! lsb_release -a 2>/dev/null | grep -q "Ubuntu"; then
        log_warn "非 Ubuntu 系统，可能不兼容"
    fi
    
    # 检查 CPU 虚拟化
    if grep -qE 'vmx|svm' /proc/cpuinfo; then
        log_info "CPU 支持硬件虚拟化"
    else
        log_warn "CPU 可能不支持硬件虚拟化，VM 无法运行"
    fi
    
    # 检查 KVM
    if [ -c /dev/kvm ]; then
        log_info "KVM 已可用"
    else
        log_warn "KVM 未就绪，VM 功能将不可用"
    fi
    
    # 检查内存
    MEM_GB=$(free -g | awk '/^Mem:/{print $2}')
    if [ "$MEM_GB" -lt 4 ]; then
        log_warn "内存不足 4GB，建议至少 4GB"
    else
        log_info "内存: ${MEM_GB}GB"
    fi
    
    # 检查磁盘
    DISK_GB=$(df -BG / | awk 'NR==2{print $4}' | sed 's/G//')
    if [ "$DISK_GB" -lt 20 ]; then
        log_warn "根分区剩余空间不足 20GB"
    else
        log_info "根分区剩余空间: ${DISK_GB}GB"
    fi
}

# ==================== Step 2: 安装依赖 ====================
install_dependencies() {
    log_info "Step 2: 安装基础依赖"
    
    apt-get update
    apt-get install -y \
        qemu-kvm \
        libvirt-daemon-system \
        zfsutils-linux \
        btrfs-progs \
        curl \
        gnupg \
        lsb-release \
        jq \
        bridge-utils \
        vlan \
        iptables
    
    log_info "基础依赖安装完成"
}

# ==================== Step 3: 配置存储 ====================
setup_storage() {
    log_info "Step 3: 配置存储后端 (${STORAGE_BACKEND})"
    
    case $STORAGE_BACKEND in
        zfs)
            if [ -n "$STORAGE_DISK" ]; then
                # 使用物理磁盘
                if zpool list | grep -q "incus-pool"; then
                    log_warn "ZFS 池 incus-pool 已存在"
                else
                    zpool create -f incus-pool "$STORAGE_DISK"
                    log_info "ZFS 池 incus-pool 已创建"
                fi
            else
                # 使用 loop 文件（测试环境）
                if [ ! -f /var/lib/incus-zfs.img ]; then
                    log_info "创建 ZFS loop 文件 (20GB)"
                    mkdir -p /var/lib
                    truncate -s 20G /var/lib/incus-zfs.img
                    zpool create -f incus-pool /var/lib/incus-zfs.img
                else
                    log_warn "ZFS loop 文件已存在"
                    zpool import -f incus-pool 2>/dev/null || true
                fi
            fi
            zpool status incus-pool || log_warn "ZFS 池状态异常"
            ;;
        btrfs)
            if [ -n "$STORAGE_DISK" ]; then
                mkfs.btrfs -f "$STORAGE_DISK" 2>/dev/null || true
                mkdir -p /var/lib/incus-btrfs
                mount "$STORAGE_DISK" /var/lib/incus-btrfs 2>/dev/null || true
            else
                mkdir -p /var/lib/incus-btrfs
                # 使用目录作为 Btrfs 存储（已存在的文件系统）
                log_info "使用目录存储（Btrfs 需要专用分区）"
            fi
            ;;
        *)
            log_error "不支持的存储后端: $STORAGE_BACKEND"
            exit 1
            ;;
    esac
}

# ==================== Step 4: 安装 Incus ====================
install_incus() {
    log_info "Step 4: 安装 Incus"
    
    if command -v incus &> /dev/null; then
        INCUS_CURRENT=$(incus version | grep "Server version" | awk '{print $3}')
        log_info "Incus 已安装 (版本: $INCUS_CURRENT)"
        return 0
    fi
    
    # 添加 Zabbly 仓库
    if [ ! -f /usr/share/keyrings/zabbly.gpg ]; then
        curl -fsSL https://pkgs.zabbly.com/key.asc | \
            gpg --dearmor -o /usr/share/keyrings/zabbly.gpg
    fi
    
    if [ ! -f /etc/apt/sources.list.d/zabbly-incus-stable.list ]; then
        echo "deb [signed-by=/usr/share/keyrings/zabbly.gpg] \
https://pkgs.zabbly.com/incus/stable $(lsb_release -cs) main" | \
            tee /etc/apt/sources.list.d/zabbly-incus-stable.list
    fi
    
    apt-get update
    apt-get install -y incus
    
    log_info "Incus 安装完成"
}

# ==================== Step 5: 初始化 Incus ====================
init_incus() {
    log_info "Step 5: 初始化 Incus（单节点模式）"
    
    if incus info 2>/dev/null | grep -q "Server version"; then
        log_warn "Incus 已初始化"
        return 0
    fi
    
    # 自动初始化（非交互式）
    incus admin init --auto \
        --storage-backend="$STORAGE_BACKEND" \
        --storage-create-loop=20 \
        --network-address=0.0.0.0 \
        --network-port=8443
    
    log_info "Incus 初始化完成"
}

# ==================== Step 6: 配置网络 ====================
setup_network() {
    log_info "Step 6: 配置网络"
    
    # 默认 bridge 已创建，配置 VLAN 支持
    incus network set incusbr0 bridge.mode=standard
    incus network set incusbr0 ipv4.nat=true
    
    # 启用 VLAN 过滤
    echo "net.bridge.bridge-nf-call-ip6tables = 0
net.bridge.bridge-nf-call-iptables = 0
net.bridge.bridge-nf-call-arptables = 0" > /etc/sysctl.d/99-incus-bridge.conf
    sysctl -p /etc/sysctl.d/99-incus-bridge.conf
    
    log_info "网络配置完成"
}

# ==================== Step 7: 创建测试实例 ====================
create_test_instances() {
    log_info "Step 7: 创建测试实例"
    
    # 创建 LXC 容器
    if incus launch images:ubuntu/24.04/cloud test-ct 2>/dev/null; then
        log_info "LXC 容器 test-ct 创建成功"
    else
        log_warn "LXC 容器创建失败（可能是嵌套容器限制）"
    fi
    
    # 创建 VM（需要 KVM）
    if [ -c /dev/kvm ]; then
        if incus launch images:ubuntu/24.04/cloud test-vm --vm 2>/dev/null; then
            log_info "VM test-vm 创建成功"
        else
            log_warn "VM 创建失败"
        fi
    else
        log_warn "跳过 VM 创建（无 KVM）"
    fi
    
    incus list
}

# ==================== Step 8: 创建快照 ====================
create_snapshots() {
    log_info "Step 8: 创建存储快照"
    
    if incus info test-ct &>/dev/null; then
        incus snapshot create test-ct snap-01
        log_info "容器快照已创建"
    fi
    
    # ZFS 存储快照
    if [ "$STORAGE_BACKEND" = "zfs" ] && zpool list | grep -q incus-pool; then
        zfs snapshot incus-pool@l0-baseline
        log_info "ZFS 存储快照已创建"
    fi
}

# ==================== Step 9: 启用 API 访问 ====================
setup_api() {
    log_info "Step 9: 配置 API 访问"
    
    # 配置 HTTPS 监听
    incus config set core.https_address :8443
    
    # 配置 CORS（如需 Web UI）
    incus config set core.https_allowed_headers "Content-Type, X-Auth-Token"
    
    log_info "API 监听已配置在 8443 端口"
}

# ==================== Step 10: 验证部署 ====================
verify_deployment() {
    log_info "Step 10: 验证部署"
    
    echo "=== Incus 版本 ==="
    incus version
    
    echo "=== 存储池 ==="
    incus storage list
    
    echo "=== 网络 ==="
    incus network list
    
    echo "=== 实例列表 ==="
    incus list
    
    echo "=== 系统信息 ==="
    incus info
    
    log_info "L0 部署验证完成"
}

# ==================== 主流程 ====================
main() {
    log_info "====================================="
    log_info "L0 最小可用版 (MVP) 部署"
    log_info "====================================="
    log_info "开始时间: $(date)"
    
    check_environment
    install_dependencies
    setup_storage
    install_incus
    init_incus
    setup_network
    create_test_instances
    create_snapshots
    setup_api
    verify_deployment
    
    log_info "====================================="
    log_info "L0 部署完成"
    log_info "结束时间: $(date)"
    log_info "====================================="
    log_info "可用管理命令:"
    log_info "  incus list          - 查看实例"
    log_info "  incus info          - 查看服务器信息"
    log_info "  incus launch        - 创建实例"
    log_info "  incus snapshot      - 管理快照"
    log_info "====================================="
}

# 支持外部调用
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi
