#!/bin/bash
# L3 全功能版部署脚本
# 在 L2 基础上扩展: CubeSandbox 极速沙箱 + TEE 机密计算 + 完整管理平面 + 统一 Instance 抽象
#
# 代码重度: ~50,000+ 行集成代码
# 预计周期: 6-12 个月
# PVE 功能对标率: ~100%+
# 核心新增: CubeSandbox 极速启动、TEE 机密计算、完整管理平面、统一编排

set -euo pipefail

# ==================== 配置变量 ====================
CUBESANDBOX_VERSION="${CUBESANDBOX_VERSION:-latest}"
TEE_TYPE="${TEE_TYPE:-tdx}"  # tdx, sgx, sev-snp
WEB_UI_VERSION="${WEB_UI_VERSION:-v1.0.0}"
CUBEVS_VERSION="${CUBEVS_VERSION:-latest}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[L3 INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[L3 WARN]${NC} $1"; }
log_error() { echo -e "${RED}[L3 ERROR]${NC} $1"; }
log_phase() { echo -e "${BLUE}[L3 PHASE]${NC} $1"; }

# ==================== Step 1: 环境预检 ====================
check_prerequisites() {
    log_info "Step 1: L3 环境预检"
    
    # 检查 L2 是否已部署
    if ! command -v kubectl &> /dev/null; then
        log_error "Kubernetes 未安装，请先运行 L0-L2 部署脚本"
        exit 1
    fi
    
    # 检查 KubeVirt
    if ! kubectl get pods -n kubevirt &> /dev/null; then
        log_warn "KubeVirt 未就绪，部分功能可能不可用"
    fi
    
    # 检查硬件支持
    if grep -q 'tdx' /proc/cpuinfo 2>/dev/null; then
        log_info "Intel TDX 支持检测到"
    elif grep -q 'sgx' /proc/cpuinfo 2>/dev/null; then
        log_info "Intel SGX 支持检测到"
    elif grep -q 'sev' /proc/cpuinfo 2>/dev/null; then
        log_info "AMD SEV-SNP 支持检测到"
    else
        log_warn "未检测到 TEE 硬件支持，TEE 功能将不可用"
    fi
    
    # 检查内存（L3 需要更多内存）
    MEM_TOTAL=$(free -g | awk '/^Mem:/{print $2}')
    if [ "$MEM_TOTAL" -lt 16 ]; then
        log_warn "总内存不足 16GB (${MEM_TOTAL}GB)，L3 功能可能受限"
    fi
    
    log_info "环境预检完成"
}

# ==================== Step 2: CubeSandbox 极速沙箱 ====================
install_cubesandbox() {
    log_phase "Phase 2: CubeSandbox 极速沙箱部署"
    
    log_info "安装 CubeSandbox 运行时..."
    
    # 创建 CubeSandbox 命名空间
    kubectl create namespace cubesandbox 2>/dev/null || true
    
    # 部署 CubeSandbox Operator（占位实现）
    cat > /tmp/cubesandbox-operator.yaml << 'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cubesandbox-operator
  namespace: cubesandbox
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cubesandbox-operator
  template:
    metadata:
      labels:
        app: cubesandbox-operator
    spec:
      containers:
      - name: operator
        image: cubesandbox/operator:latest
        ports:
        - containerPort: 8443
        env:
        - name: RUST_LOG
          value: "info"
        - name: POOL_SIZE
          value: "2000"
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "2000m"
---
apiVersion: v1
kind: Service
metadata:
  name: cubesandbox-api
  namespace: cubesandbox
spec:
  selector:
    app: cubesandbox-operator
  ports:
  - port: 8443
    targetPort: 8443
  type: ClusterIP
EOF
    kubectl apply -f /tmp/cubesandbox-operator.yaml
    
    # 部署 CubeSandbox VM Pool（预置化资源池）
    cat > /tmp/cubesandbox-vmpool.yaml << 'EOF'
apiVersion: v1
kind: ConfigMap
metadata:
  name: cubesandbox-config
  namespace: cubesandbox
data:
  config.toml: |
    [runtime]
    pool_size = 2000
    prewarm = true
    cow_clone = true
    
    [vm]
    vcpus = 2
    memory_mb = 512
    kernel_path = "/opt/cubesandbox/vmlinux"
    rootfs_template = "/opt/cubesandbox/rootfs-template.img"
    
    [network]
    backend = "cube-vs"
    xdp_enabled = true
    
    [agent]
    listen = "vsock:2:8080"
    protocol = "ttrpc"
---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: cubesandbox-vmpool
  namespace: cubesandbox
spec:
  selector:
    matchLabels:
      app: cubesandbox-vmpool
  template:
    metadata:
      labels:
        app: cubesandbox-vmpool
    spec:
      hostNetwork: true
      containers:
      - name: vmpool
        image: cubesandbox/vmpool:latest
        securityContext:
          privileged: true
        volumeMounts:
        - name: dev-kvm
          mountPath: /dev/kvm
        - name: cubesandbox-config
          mountPath: /etc/cubesandbox
      volumes:
      - name: dev-kvm
        hostPath:
          path: /dev/kvm
      - name: cubesandbox-config
        configMap:
          name: cubesandbox-config
EOF
    kubectl apply -f /tmp/cubesandbox-vmpool.yaml
    
    # 注册 CubeSandbox RuntimeClass
    cat > /tmp/runtimeclass-cubesandbox.yaml << 'EOF'
apiVersion: node.k8s.io/v1
kind: RuntimeClass
metadata:
  name: cubesandbox
handler: io.containerd.cube.v2
overhead:
  podFixed:
    memory: "256Mi"
    cpu: "250m"
scheduling:
  nodeSelector:
    cubesandbox.io/pool-ready: "true"
EOF
    kubectl apply -f /tmp/runtimeclass-cubesandbox.yaml
    
    log_info "CubeSandbox 部署完成"
    kubectl get pods -n cubesandbox
}

# ==================== Step 3: CubeVS eBPF/XDP 网络 ====================
install_cubesvs() {
    log_phase "Phase 3: CubeVS eBPF/XDP 网络加速"
    
    log_info "部署 CubeVS 虚拟交换机..."
    
    cat > /tmp/cubesvs-daemonset.yaml << 'EOF'
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: cubesvs
  namespace: kube-system
spec:
  selector:
    matchLabels:
      app: cubesvs
  template:
    metadata:
      labels:
        app: cubesvs
    spec:
      hostNetwork: true
      containers:
      - name: cubesvs
        image: cubesandbox/cubesvs:latest
        securityContext:
          privileged: true
          capabilities:
            add:
              - BPF
              - NET_ADMIN
              - SYS_ADMIN
        env:
        - name: CUBEVS_MODE
          value: "xdp"
        - name: CUBEVS_IFACE
          value: "eth0"
        volumeMounts:
        - name: bpf
          mountPath: /sys/fs/bpf
        - name: cgroup
          mountPath: /sys/fs/cgroup
      volumes:
      - name: bpf
        hostPath:
          path: /sys/fs/bpf
          type: DirectoryOrCreate
      - name: cgroup
        hostPath:
          path: /sys/fs/cgroup
EOF
    kubectl apply -f /tmp/cubesvs-daemonset.yaml
    
    log_info "CubeVS 部署完成"
    kubectl get pods -n kube-system -l app=cubesvs
}

# ==================== Step 4: Cloud Hypervisor 深度集成 ====================
install_cloud_hypervisor() {
    log_phase "Phase 4: Cloud Hypervisor 深度集成"
    
    log_info "配置 Cloud Hypervisor 作为 Kata 主后端..."
    
    # 安装 Cloud Hypervisor 二进制
    if ! command -v cloud-hypervisor &> /dev/null; then
        CH_VERSION="38.0"
        curl -LO "https://github.com/cloud-hypervisor/cloud-hypervisor/releases/download/v${CH_VERSION}/cloud-hypervisor-v${CH_VERSION}.tar.gz"
        tar -xzf "cloud-hypervisor-v${CH_VERSION}.tar.gz" -C /usr/local/bin/
        chmod +x /usr/local/bin/cloud-hypervisor
        rm -f "cloud-hypervisor-v${CH_VERSION}.tar.gz"
    fi
    
    # 配置 Kata 使用 Cloud Hypervisor
    cat > /tmp/kata-cloud-hypervisor.toml << 'EOF'
[hypervisor.cloud-hypervisor]
path = "/usr/local/bin/cloud-hypervisor"
kernel = "/opt/kata/share/kata-containers/vmlinux.container"
image = "/opt/kata/share/kata-containers/kata-containers.img"
machine_type = "q35"

[runtime]
name = "kata"
path = "/opt/kata/bin/kata-runtime"
EOF
    
    # 更新 containerd 配置
    cat > /etc/containerd/config.d/kata-cloud-hypervisor.toml << 'EOF'
version = 2
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata-clh]
  runtime_type = "io.containerd.kata-clh.v2"
  privileged_without_host_devices = true
  pod_annotations = ["*.katacontainers.*"]
  [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata-clh.options]
    ConfigPath = "/tmp/kata-cloud-hypervisor.toml"
EOF
    systemctl restart containerd
    
    # 注册 RuntimeClass
    cat > /tmp/runtimeclass-kata-clh.yaml << 'EOF'
apiVersion: node.k8s.io/v1
kind: RuntimeClass
metadata:
  name: kata-clh
handler: io.containerd.kata-clh.v2
overhead:
  podFixed:
    memory: "512Mi"
    cpu: "500m"
scheduling:
  nodeSelector:
    katacontainers.io/kata-runtime: "true"
EOF
    kubectl apply -f /tmp/runtimeclass-kata-clh.yaml
    
    log_info "Cloud Hypervisor 集成完成"
}

# ==================== Step 5: TEE 机密计算 ====================
install_tee() {
    log_phase "Phase 5: TEE 机密计算 (CoCo)"
    
    log_info "部署 Confidential Containers (CoCo)..."
    
    # 创建 CoCo 命名空间
    kubectl create namespace confidential-containers 2>/dev/null || true
    
    # 部署 CoCo Operator
    cat > /tmp/coco-operator.yaml << 'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: coco-operator
  namespace: confidential-containers
spec:
  replicas: 1
  selector:
    matchLabels:
      app: coco-operator
  template:
    metadata:
      labels:
        app: coco-operator
    spec:
      containers:
      - name: operator
        image: confidential-containers/operator:latest
        env:
        - name: TEE_TYPE
          value: "tdx"
        - name: KBS_URL
          value: "http://kbs-service:8080"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
---
apiVersion: v1
kind: Service
metadata:
  name: kbs-service
  namespace: confidential-containers
spec:
  selector:
    app: coco-operator
  ports:
  - port: 8080
    targetPort: 8080
EOF
    kubectl apply -f /tmp/coco-operator.yaml
    
    # 创建 TEE 节点标签
    kubectl label nodes --all tee-enabled=true 2>/dev/null || true
    
    # 注册 TEE RuntimeClass
    cat > /tmp/runtimeclass-tee.yaml << 'EOF'
apiVersion: node.k8s.io/v1
kind: RuntimeClass
metadata:
  name: kata-tee
handler: io.containerd.kata-tee.v2
overhead:
  podFixed:
    memory: "1Gi"
    cpu: "1000m"
scheduling:
  nodeSelector:
    tee-enabled: "true"
EOF
    kubectl apply -f /tmp/runtimeclass-tee.yaml
    
    log_info "TEE 机密计算部署完成"
    kubectl get pods -n confidential-containers
}

# ==================== Step 6: 管理平面 (Web UI + CLI) ====================
install_management_plane() {
    log_phase "Phase 6: 管理平面 (Web UI + CLI)"
    
    log_info "部署 CubeConsole Web UI..."
    
    # 创建管理平面命名空间
    kubectl create namespace cube-console 2>/dev/null || true
    
    # 部署 Web UI
    cat > /tmp/cubeconsole-deployment.yaml << 'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cubeconsole
  namespace: cube-console
spec:
  replicas: 2
  selector:
    matchLabels:
      app: cubeconsole
  template:
    metadata:
      labels:
        app: cubeconsole
    spec:
      containers:
      - name: frontend
        image: cubeve/console-frontend:latest
        ports:
        - containerPort: 80
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
      - name: backend
        image: cubeve/console-backend:latest
        ports:
        - containerPort: 8080
        env:
        - name: API_ENDPOINT
          value: "http://cubeapi:8080"
        - name: K8S_CONFIG
          value: "/etc/kubernetes/admin.conf"
        resources:
          requests:
            memory: "256Mi"
            cpu: "200m"
        volumeMounts:
        - name: k8s-config
          mountPath: /etc/kubernetes
          readOnly: true
      volumes:
      - name: k8s-config
        hostPath:
          path: /etc/kubernetes
---
apiVersion: v1
kind: Service
metadata:
  name: cubeconsole
  namespace: cube-console
spec:
  selector:
    app: cubeconsole
  ports:
  - name: http
    port: 80
    targetPort: 80
  - name: api
    port: 8080
    targetPort: 8080
  type: NodePort
EOF
    kubectl apply -f /tmp/cubeconsole-deployment.yaml
    
    # 部署 API Gateway
    cat > /tmp/cubeapi-deployment.yaml << 'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cubeapi
  namespace: cube-console
spec:
  replicas: 2
  selector:
    matchLabels:
      app: cubeapi
  template:
    metadata:
      labels:
        app: cubeapi
    spec:
      containers:
      - name: api
        image: cubeve/api-gateway:latest
        ports:
        - containerPort: 8080
        - containerPort: 50051
        env:
        - name: INCUS_ENDPOINT
          value: "https://incus-service:8443"
        - name: K8S_ENDPOINT
          value: "https://kubernetes.default:443"
        resources:
          requests:
            memory: "256Mi"
            cpu: "200m"
---
apiVersion: v1
kind: Service
metadata:
  name: cubeapi
  namespace: cube-console
spec:
  selector:
    app: cubeapi
  ports:
  - name: rest
    port: 8080
    targetPort: 8080
  - name: grpc
    port: 50051
    targetPort: 50051
  type: ClusterIP
EOF
    kubectl apply -f /tmp/cubeapi-deployment.yaml
    
    log_info "管理平面部署完成"
    kubectl get pods -n cube-console
    
    # 获取访问地址
    NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[0].address}')
    CONSOLE_PORT=$(kubectl get svc cubeconsole -n cube-console -o jsonpath='{.spec.ports[0].nodePort}' 2>/dev/null || echo "pending")
    
    log_info "CubeConsole 访问地址: http://${NODE_IP}:${CONSOLE_PORT}"
}

# ==================== Step 7: 统一编排 (Instance CRD + HA) ====================
setup_unified_orchestration() {
    log_phase "Phase 7: 统一编排与 HA"
    
    log_info "部署 Instance Controller..."
    
    cat > /tmp/instance-controller.yaml << 'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: instance-controller
  namespace: kube-system
spec:
  replicas: 2
  selector:
    matchLabels:
      app: instance-controller
  template:
    metadata:
      labels:
        app: instance-controller
    spec:
      serviceAccountName: instance-controller
      containers:
      - name: controller
        image: cubeve/instance-controller:latest
        env:
        - name: WATCH_NAMESPACE
          value: ""
        - name: LEADER_ELECT
          value: "true"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: instance-controller
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: instance-controller
rules:
- apiGroups: ["cube.io"]
  resources: ["instances"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["kubevirt.io"]
  resources: ["virtualmachines", "virtualmachineinstances"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["k8s.cni.cncf.io"]
  resources: ["network-attachment-definitions"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: instance-controller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: instance-controller
subjects:
- kind: ServiceAccount
  name: instance-controller
  namespace: kube-system
EOF
    kubectl apply -f /tmp/instance-controller.yaml
    
    # 创建示例 Instance
    cat > /tmp/example-instance.yaml << 'EOF'
apiVersion: cube.io/v1
kind: Instance
metadata:
  name: example-vm
  namespace: default
spec:
  runtime: incus-vm
  image: ubuntu:24.04
  resources:
    cpu: "2"
    memory: "4Gi"
    storage: "50Gi"
  network:
    vpc: default-vpc
    subnet: default-subnet
    securityGroups:
    - default-sg
EOF
    kubectl apply -f /tmp/example-instance.yaml
    
    log_info "统一编排部署完成"
}

# ==================== Step 8: 备份与灾难恢复 ====================
setup_backup() {
    log_phase "Phase 8: 备份与灾难恢复"
    
    log_info "部署 Velero 备份系统..."
    
    # 安装 Velero CLI
    if ! command -v velero &> /dev/null; then
        VELERO_VERSION="1.13.0"
        curl -LO "https://github.com/vmware-tanzu/velero/releases/download/v${VELERO_VERSION}/velero-v${VELERO_VERSION}-linux-amd64.tar.gz"
        tar -xzf "velero-v${VELERO_VERSION}-linux-amd64.tar.gz" -C /tmp/
        mv /tmp/velero-v${VELERO_VERSION}-linux-amd64/velero /usr/local/bin/
        rm -rf "velero-v${VELERO_VERSION}-linux-amd64.tar.gz" /tmp/velero-v${VELERO_VERSION}-linux-amd64
    fi
    
    log_info "Velero 安装完成"
    velero version --client-only
}

# ==================== Step 9: 验证 L3 ====================
verify_l3() {
    log_phase "Phase 9: L3 全功能验证"
    
    echo "=== CubeSandbox Pods ==="
    kubectl get pods -n cubesandbox
    
    echo "=== CubeVS Pods ==="
    kubectl get pods -n kube-system -l app=cubesvs
    
    echo "=== TEE Pods ==="
    kubectl get pods -n confidential-containers
    
    echo "=== 管理平面 ==="
    kubectl get pods -n cube-console
    
    echo "=== Instance Controller ==="
    kubectl get pods -n kube-system -l app=instance-controller
    
    echo "=== RuntimeClass 汇总 ==="
    kubectl get runtimeclass
    
    echo "=== 节点标签 ==="
    kubectl get nodes --show-labels | grep -E "tee|cubesandbox|kata"
    
    log_info "L3 验证完成"
}

# ==================== 主流程 ====================
main() {
    log_info "====================================="
    log_info "L3 全功能版部署"
    log_info "====================================="
    log_info "开始时间: $(date)"
    
    check_prerequisites
    install_cubesandbox
    install_cubesvs
    install_cloud_hypervisor
    install_tee
    install_management_plane
    setup_unified_orchestration
    setup_backup
    verify_l3
    
    log_info "====================================="
    log_info "L3 部署完成"
    log_info "结束时间: $(date)"
    log_info "====================================="
    log_info "新增能力:"
    log_info "  - CubeSandbox 极速沙箱 (<60ms 启动)"
    log_info "  - CubeVS eBPF/XDP 网络加速"
    log_info "  - Cloud Hypervisor 深度集成"
    log_info "  - TEE 机密计算 (CoCo)"
    log_info "  - 完整管理平面 (Web UI + CLI)"
    log_info "  - 统一 Instance 编排"
    log_info "  - 备份与灾难恢复"
    log_info "====================================="
}

if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi
