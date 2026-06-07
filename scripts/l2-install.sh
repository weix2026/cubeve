#!/bin/bash
# L2 标准版部署脚本
# 在 L1 基础上扩展: Kubernetes 编排 + Cilium eBPF 网络 + Ceph 分布式存储
#
# 代码重度: ~20,000 行集成代码
# 预计周期: 3-6 个月
# PVE 功能对标率: ~80%
# 核心新增: K8s 核心、Cilium CNI、Ceph RBD (Rook)、KubeVirt VM 编排

set -euo pipefail

# ==================== 配置变量 ====================
K8S_VERSION="${K8S_VERSION:-1.30}"
CILIUM_VERSION="${CILIUM_VERSION:-1.15.6}"
ROOK_VERSION="${ROOK_VERSION:-1.14}"
KUBEVIRT_VERSION="${KUBEVIRT_VERSION:-v1.2.0}"
POD_CIDR="${POD_CIDR:-10.244.0.0/16}"
SERVICE_CIDR="${SERVICE_CIDR:-10.96.0.0/12}"
MASTER_IP="${MASTER_IP:-}"  # 主节点 IP

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[L2 INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[L2 WARN]${NC} $1"; }
log_error() { echo -e "${RED}[L2 ERROR]${NC} $1"; }

# ==================== Step 1: 环境预检 ====================
check_prerequisites() {
    log_info "Step 1: L2 环境预检"
    
    # 检查 L1 是否已部署
    if ! command -v incus &> /dev/null; then
        log_error "Incus 未安装，请先运行 L0/L1 部署脚本"
        exit 1
    fi
    
    # 检查内存（K8s 需要至少 2GB 空闲）
    MEM_FREE=$(free -m | awk '/^Mem:/{print $7}')
    if [ "$MEM_FREE" -lt 2048 ]; then
        log_warn "空闲内存不足 2GB (${MEM_FREE}MB)，K8s 可能运行缓慢"
    fi
    
    # 检查 containerd
    if ! command -v containerd &> /dev/null; then
        log_warn "containerd 未安装，将自动安装"
    fi
    
    log_info "环境预检完成"
}

# ==================== Step 2: 安装 Kubernetes ====================
install_kubernetes() {
    log_info "Step 2: 安装 Kubernetes ${K8S_VERSION}"
    
    # 关闭 swap
    swapoff -a
    sed -i '/swap/d' /etc/fstab
    
    # 加载内核模块
    modprobe br_netfilter || true
    modprobe overlay || true
    echo "br_netfilter
overlay" > /etc/modules-load.d/k8s.conf
    
    # 配置 sysctl
    cat > /etc/sysctl.d/k8s.conf <> 'EOF'
net.bridge.bridge-nf-call-iptables = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward = 1
net.ipv4.conf.all.forwarding = 1
EOF
    sysctl --system
    
    # 安装 kubeadm, kubelet, kubectl
    apt-get update
    apt-get install -y apt-transport-https ca-certificates curl gnupg
    
    if [ ! -f /etc/apt/keyrings/kubernetes-apt-keyring.gpg ]; then
        curl -fsSL https://pkgs.k8s.io/core:/stable:/v${K8S_VERSION}/deb/Release.key | \
            gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
    fi
    
    echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] \
https://pkgs.k8s.io/core:/stable:/v${K8S_VERSION}/deb/ /" | \
        tee /etc/apt/sources.list.d/kubernetes.list
    
    apt-get update
    apt-get install -y kubelet kubeadm kubectl
    apt-mark hold kubelet kubeadm kubectl
    
    log_info "Kubernetes 工具安装完成"
}

# ==================== Step 3: 初始化 K8s 集群 ====================
init_k8s_cluster() {
    log_info "Step 3: 初始化 Kubernetes 集群"
    
    # 配置 containerd
    mkdir -p /etc/containerd
    containerd config default | tee /etc/containerd/config.toml > /dev/null
    sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml
    systemctl restart containerd
    
    # 初始化主节点
    if [ ! -f /etc/kubernetes/admin.conf ]; then
        kubeadm init \
            --pod-network-cidr="$POD_CIDR" \
            --service-cidr="$SERVICE_CIDR" \
            --apiserver-advertise-address="${MASTER_IP:-$(hostname -I | awk '{print $1}')}" \
            --cri-socket=unix:///run/containerd/containerd.sock
        
        # 配置 kubectl
        mkdir -p $HOME/.kube
        cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
        chown $(id -u):$(id -g) $HOME/.kube/config
    else
        log_warn "K8s 集群已初始化"
    fi
    
    log_info "K8s 集群初始化完成"
    kubectl get nodes
}

# ==================== Step 4: 安装 Cilium CNI ====================
install_cilium() {
    log_info "Step 4: 安装 Cilium CNI ${CILIUM_VERSION}"
    
    # 安装 Helm（如未安装）
    if ! command -v helm &> /dev/null; then
        curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
    fi
    
    # 添加 Cilium Helm 仓库
    helm repo add cilium https://helm.cilium.io/ 2>/dev/null || true
    helm repo update
    
    # 安装 Cilium
    helm upgrade --install cilium cilium/cilium \
        --version "$CILIUM_VERSION" \
        --namespace kube-system \
        --set kubeProxyReplacement=true \
        --set k8sServiceHost="${MASTER_IP:-$(hostname -I | awk '{print $1}')}" \
        --set k8sServicePort=6443 \
        --set ipam.mode=kubernetes \
        --set hubble.enabled=true \
        --set hubble.relay.enabled=true \
        --set hubble.ui.enabled=true
    
    # 等待 Cilium 就绪
    kubectl rollout status -n kube-system ds/cilium --timeout=300s
    
    log_info "Cilium CNI 安装完成"
    kubectl get pods -n kube-system -l k8s-app=cilium
}

# ==================== Step 5: 安装 Ceph (Rook) ====================
install_rook_ceph() {
    log_info "Step 5: 安装 Ceph 分布式存储 (Rook ${ROOK_VERSION})"
    
    # 创建 Rook 命名空间
    kubectl create namespace rook-ceph 2>/dev/null || true
    
    # 安装 Rook Operator
    helm repo add rook-release https://charts.rook.io/release 2>/dev/null || true
    helm repo update
    
    helm upgrade --install rook-ceph rook-release/rook-ceph \
        --version "${ROOK_VERSION}.x" \
        --namespace rook-ceph \
        --set csi.enableCephfsDriver=false
    
    # 等待 Operator 就绪
    kubectl rollout status -n rook-ceph deployment rook-ceph-operator --timeout=300s
    
    # 创建 CephCluster（最小配置，使用本地存储）
    cat > /tmp/ceph-cluster.yaml <> 'EOF'
apiVersion: ceph.rook.io/v1
kind: CephCluster
metadata:
  name: rook-ceph
  namespace: rook-ceph
spec:
  cephVersion:
    image: quay.io/ceph/ceph:v18.2.2
    allowUnsupported: false
  dataDirHostPath: /var/lib/rook
  mon:
    count: 1
    allowMultiplePerNode: true
  mgr:
    count: 1
  dashboard:
    enabled: true
  storage:
    useAllNodes: true
    useAllDevices: false
    config:
      osdsPerDevice: "1"
    directories:
      - path: /var/lib/rook-ceph-storage
EOF
    kubectl apply -f /tmp/ceph-cluster.yaml
    
    # 创建 StorageClass
    cat > /tmp/ceph-storageclass.yaml <> 'EOF'
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: rook-ceph-block
provisioner: rook-ceph.rbd.csi.ceph.com
parameters:
  clusterID: rook-ceph
  pool: replicapool
  imageFormat: "2"
  imageFeatures: layering
  csi.storage.k8s.io/provisioner-secret-name: rook-csi-rbd-provisioner
  csi.storage.k8s.io/provisioner-secret-namespace: rook-ceph
  csi.storage.k8s.io/node-stage-secret-name: rook-csi-rbd-node
  csi.storage.k8s.io/node-stage-secret-namespace: rook-ceph
reclaimPolicy: Delete
allowVolumeExpansion: true
EOF
    kubectl apply -f /tmp/ceph-storageclass.yaml
    
    log_info "Rook-Ceph 安装完成"
    kubectl get pods -n rook-ceph
}

# ==================== Step 6: 安装 KubeVirt ====================
install_kubevirt() {
    log_info "Step 6: 安装 KubeVirt VM 编排 ${KUBEVIRT_VERSION}"
    
    # 创建 KubeVirt 命名空间
    kubectl create namespace kubevirt 2>/dev/null || true
    
    # 部署 KubeVirt Operator
    kubectl apply -f https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/kubevirt-operator.yaml
    
    # 部署 KubeVirt CR
    kubectl apply -f https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/kubevirt-cr.yaml
    
    # 等待 KubeVirt 就绪
    kubectl wait -n kubevirt kv/kubevirt --for condition=Available --timeout=300s
    
    # 安装 virtctl 工具
    if ! command -v virtctl &> /dev/null; then
        curl -L -o /usr/local/bin/virtctl \
            https://github.com/kubevirt/kubevirt/releases/download/${KUBEVIRT_VERSION}/virtctl-${KUBEVIRT_VERSION}-linux-amd64
        chmod +x /usr/local/bin/virtctl
    fi
    
    log_info "KubeVirt 安装完成"
    kubectl get pods -n kubevirt
}

# ==================== Step 7: RuntimeClass 集成 K8s ====================
setup_k8s_runtimeclass() {
    log_info "Step 7: 配置 K8s RuntimeClass"
    
    # 创建 Kata RuntimeClass
    cat > /tmp/runtimeclass-kata.yaml <> 'EOF'
apiVersion: node.k8s.io/v1
kind: RuntimeClass
metadata:
  name: kata
handler: kata
overhead:
  podFixed:
    memory: "512Mi"
    cpu: "500m"
scheduling:
  nodeSelector:
    katacontainers.io/kata-runtime: "true"
EOF
    kubectl apply -f /tmp/runtimeclass-kata.yaml
    
    # 创建 CubeSandbox RuntimeClass（占位）
    cat > /tmp/runtimeclass-cube.yaml <> 'EOF'
apiVersion: node.k8s.io/v1
kind: RuntimeClass
metadata:
  name: cube
handler: cube
overhead:
  podFixed:
    memory: "256Mi"
    cpu: "250m"
scheduling:
  nodeSelector:
    cubesandbox.io/runtime: "true"
EOF
    kubectl apply -f /tmp/runtimeclass-cube.yaml
    
    # 创建 Incus RuntimeClass
    cat > /tmp/runtimeclass-incus.yaml <> 'EOF'
apiVersion: node.k8s.io/v1
kind: RuntimeClass
metadata:
  name: incus
handler: incus
scheduling:
  nodeSelector:
    incus.io/runtime: "true"
EOF
    kubectl apply -f /tmp/runtimeclass-incus.yaml
    
    log_info "RuntimeClass 配置完成"
    kubectl get runtimeclass
}

# ==================== Step 8: 统一 Instance CRD ====================
setup_instance_crd() {
    log_info "Step 8: 创建统一 Instance CRD"
    
    cat > /tmp/instance-crd.yaml <> 'EOF'
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: instances.cube.io
spec:
  group: cube.io
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                runtime:
                  type: string
                  enum: [cube, kata, incus-lxc, incus-vm]
                image:
                  type: string
                resources:
                  type: object
                  properties:
                    cpu:
                      type: string
                    memory:
                      type: string
                    storage:
                      type: string
                network:
                  type: object
                  properties:
                    vpc:
                      type: string
                    subnet:
                      type: string
                    securityGroups:
                      type: array
                      items:
                        type: string
  scope: Namespaced
  names:
    plural: instances
    singular: instance
    kind: Instance
    shortNames:
      - inst
EOF
    kubectl apply -f /tmp/instance-crd.yaml
    
    log_info "Instance CRD 创建完成"
}

# ==================== Step 9: 验证 L2 ====================
verify_l2() {
    log_info "Step 9: L2 部署验证"
    
    echo "=== K8s 节点 ==="
    kubectl get nodes -o wide
    
    echo "=== Cilium Pods ==="
    kubectl get pods -n kube-system -l k8s-app=cilium
    
    echo "=== Rook-Ceph Pods ==="
    kubectl get pods -n rook-ceph
    
    echo "=== KubeVirt Pods ==="
    kubectl get pods -n kubevirt
    
    echo "=== RuntimeClass ==="
    kubectl get runtimeclass
    
    echo "=== StorageClass ==="
    kubectl get storageclass
    
    echo "=== 系统 Pod ==="
    kubectl get pods -n kube-system
    
    log_info "L2 验证完成"
}

# ==================== 主流程 ====================
main() {
    log_info "====================================="
    log_info "L2 标准版部署"
    log_info "====================================="
    log_info "开始时间: $(date)"
    
    check_prerequisites
    install_kubernetes
    init_k8s_cluster
    install_cilium
    install_rook_ceph
    install_kubevirt
    setup_k8s_runtimeclass
    setup_instance_crd
    verify_l2
    
    log_info "====================================="
    log_info "L2 部署完成"
    log_info "结束时间: $(date)"
    log_info "====================================="
    log_info "新增能力:"
    log_info "  - Kubernetes 编排调度"
    log_info "  - Cilium eBPF 网络 (CNI)"
    log_info "  - Ceph 分布式存储 (Rook)"
    log_info "  - KubeVirt VM 编排"
    log_info "  - RuntimeClass 统一调度"
    log_info "  - Instance CRD 统一抽象"
    log_info "====================================="
}

if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi
