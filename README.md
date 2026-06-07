# CubeVE - Virtual Infrastructure Platform

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/weix2026/cubeve)](https://goreportcard.com/report/github.com/weix2026/cubeve)

**CubeVE** is a progressive, multi-level virtual infrastructure platform built on a unified architecture spanning L0 (MVP) through L3 (Full Production). It provides a single API surface for provisioning, managing, and orchestrating virtual machines, LXC containers, Kata Containers, and CubeSandbox MicroVMs across private cloud and hybrid environments.

## 🎯 Architecture Overview

CubeVE is built on a **4-Level Degradation Strategy** that allows progressive deployment from a single-node MVP to a full-featured distributed cloud platform.

| Level | Name | Description | Key Components | Code Weight |
|-------|------|-------------|----------------|-------------|
| **L0** | MVP | Single-node, minimal viable product | Incus, QEMU, LXC, Bridge, ZFS | < 1.5 |
| **L1** | Basic | Distributed cluster with enhanced networking | L0 + Clustering, OVN, Kata, RuntimeClasses | < 3.0 |
| **L2** | Standard | Cloud-native with Kubernetes orchestration | L1 + K8s, Cilium, Ceph, KubeVirt | ~20,000 lines |
| **L3** | Full | Production-ready with advanced features | L2 + CubeSandbox, TEE, eBPF, Web UI | Complete |

## 🚀 Quick Start

### L0: Single-Node MVP

```bash
# Run the L0 installation script
curl -fsSL https://raw.githubusercontent.com/weix2026/cubeve/main/scripts/l0-install.sh | sudo bash

# Or clone and run locally
git clone https://github.com/weix2026/cubeve.git
cd cubeve
sudo bash scripts/l0-install.sh
```

### L2: Kubernetes-Ready (Standard)

```bash
# Deploy the full standard stack
sudo bash scripts/l2-install.sh

# Apply Kubernetes manifests
kubectl apply -f manifests/runtimeclasses.yaml
kubectl apply -f manifests/instance-crd.yaml
kubectl apply -f manifests/cubesandbox-runtime.yaml
kubectl apply -f manifests/network.yaml
kubectl apply -f manifests/storage.yaml
kubectl apply -f manifests/management-plane.yaml
kubectl apply -f manifests/tee-confidential.yaml
```

## 📁 Repository Structure

```
cubeve/
├── scripts/
│   ├── l0-install.sh          # L0 MVP deployment
│   ├── l1-install.sh          # L1 Basic distributed deployment
│   ├── l2-install.sh          # L2 Standard Kubernetes deployment
│   └── l3-install.sh          # L3 Full production deployment
├── manifests/
│   ├── runtimeclasses.yaml    # RuntimeClass definitions (Kata, Cube, Incus)
│   ├── instance-crd.yaml      # Instance CRD and controller
│   ├── cubesandbox-runtime.yaml # CubeSandbox operator and VM pool
│   ├── network.yaml           # Cilium + OVN + CubeVS eBPF networking
│   ├── storage.yaml           # Rook-Ceph cluster and storage classes
│   ├── management-plane.yaml  # API Gateway + Web UI + Ingress
│   └── tee-confidential.yaml # CoCo operator + KBS + attestation
├── cmd/
│   ├── api-gateway/           # REST/gRPC API gateway (CubeAPI)
│   ├── instance-controller/   # Kubernetes operator for Instance CRD
│   └── cubesandbox-operator/ # CubeSandbox VM pool operator
├── pkg/
│   ├── api/                   # API types and client
│   ├── controller/            # Controller logic
│   └── runtime/               # Runtime adapters (Incus, Kata, Cube)
├── api/
│   └── proto/                 # Protocol Buffer definitions
├── Dockerfile                 # Container build
├── Makefile                   # Build automation
└── README.md                  # This file
```

## 🔧 Component Matrix

### L0: MVP Components
| Component | Technology | Purpose |
|-----------|------------|---------|
| Hypervisor | QEMU + KVM | Hardware virtualization |
| Container Runtime | LXC (Incus) | OS-level containers |
| Orchestrator | Incus | Single-node management |
| Storage | ZFS / Btrfs | Local storage with snapshots |
| Network | Linux Bridge | Basic L2 networking |

### L1: Basic Components
| Component | Technology | Purpose |
|-----------|------------|---------|
| Cluster | Incus + Cowsql | Raft-based distributed cluster |
| SDN | OVN | Overlay virtual networking |
| Secure Containers | Kata + containerd | VM-level isolation for containers |
| Runtime Selection | RuntimeClass | Kubernetes runtime selection |

### L2: Standard Components
| Component | Technology | Purpose |
|-----------|------------|---------|
| Orchestration | Kubernetes | Container orchestration |
| CNI | Cilium (eBPF) | High-performance networking |
| Storage | Ceph + Rook | Distributed block/file/object storage |
| VM Management | KubeVirt | Kubernetes-native VM management |

### L3: Full Components
| Component | Technology | Purpose |
|-----------|------------|---------|
| MicroVM | CubeSandbox (Firecracker) | Ultra-fast sandboxing |
| Network Acceleration | CubeVS (eBPF/XDP) | Custom high-performance networking |
| VM Migration | Cloud Hypervisor | Live migration with minimal downtime |
| Confidential Computing | TEE / CoCo | Hardware-based isolation |
| Management Plane | CubeConsole | Web UI + CLI for operations |

## 🏗️ Build Status

| Component | Status | Description |
|-----------|--------|-------------|
| **api-gateway** | ✅ Ready | Incus REST API integration, health/ready checks, 14 API endpoints |
| **cubeconsole** | ✅ Ready | CLI tool calling API Gateway, instance CRUD operations |
| **version-tool** | ✅ Ready | Version management utility |
| **cubesandbox-operator** | ✅ Ready | Sandbox VM operator (stub, functional) |
| **instance-controller** | ⚠️ Stub | Compiles and runs, K8s controller logic pending |

**Current Test Results:**
- L0 API Integration Tests: **14/14 passed** ✅
- Script Syntax Validation: **7/7 passed** ✅
- Binary Build: **5/5 succeeded** ✅

## 🛠️ Development

### Prerequisites
- Go 1.22+
- Docker (for container builds)
- Kubernetes cluster (for L2+ development)
- Incus / LXD (for L0 development)

### Build
```bash
# Build all binaries
make build

# Build specific binary
make build-api-gateway

# Run tests
make test

# Build Docker images
make docker-build
```

### Deploy for Development
```bash
# Deploy L0 (single node)
make deploy-l0

# Deploy L2 (with Kubernetes)
make deploy-l2

# Deploy all manifests
make deploy-manifests
```

## 📖 API Reference

### REST API (CubeAPI Gateway)

```bash
# List instances
curl http://localhost:8080/api/v1/instances

# Create instance
curl -X POST http://localhost:8080/api/v1/instances \
  -H "Content-Type: application/json" \
  -d '{
    "runtime": "incus-lxc",
    "image": "ubuntu/24.04",
    "resources": {"cpu": "2", "memory": "4Gi"}
  }'

# Get instance status
curl http://localhost:8080/api/v1/instances/{id}
```

### gRPC API
```protobuf
service CubeAPI {
  rpc ListInstances(ListInstancesRequest) returns (ListInstancesResponse);
  rpc CreateInstance(CreateInstanceRequest) returns (Instance);
  rpc GetInstance(GetInstanceRequest) returns (Instance);
  rpc DeleteInstance(DeleteInstanceRequest) returns (google.protobuf.Empty);
  rpc StartInstance(StartInstanceRequest) returns (Instance);
  rpc StopInstance(StopInstanceRequest) returns (Instance);
}
```

## 🔒 Security

- **RBAC**: Kubernetes RBAC for all operations
- **TEE/CoCo**: Hardware-based confidential computing (L3)
- **Network Policy**: Cilium NetworkPolicies for micro-segmentation
- **Storage Encryption**: Ceph RBD encryption at rest
- **API Security**: mTLS for internal communication, OAuth2 for external

## 📊 Monitoring

- **Metrics**: Prometheus + Grafana dashboards
- **Logging**: Loki / ELK stack
- **Tracing**: Jaeger / OpenTelemetry
- **Alerting**: Prometheus Alertmanager

## 🌐 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Incus](https://github.com/lxc/incus) - Modern system container and virtual machine manager
- [Kata Containers](https://katacontainers.io/) - Secure container runtime
- [Cilium](https://cilium.io/) - eBPF-based networking and security
- [Rook](https://rook.io/) - Cloud-native storage orchestration
- [KubeVirt](https://kubevirt.io/) - Kubernetes-native virtualization
- [Cloud Hypervisor](https://www.cloudhypervisor.org/) - Modern Rust VMM
- [Confidential Containers](https://confidentialcontainers.org/) - Cloud-native confidential computing

---

**CubeVE** - *Virtual Infrastructure, Unified.*
