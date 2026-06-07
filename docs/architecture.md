# Architecture Overview

CubeVE is a progressive, multi-level virtual infrastructure platform built on a unified architecture spanning L0 (MVP) through L3 (Full Production).

## L0: MVP (Minimum Viable Product)

### Components
- **Incus**: System container and virtual machine manager
- **LXC**: OS-level containers
- **QEMU/KVM**: Hardware virtualization (requires /dev/kvm)
- **ZFS/Btrfs**: Local storage with snapshots
- **Linux Bridge**: Basic L2 networking

### Code Weight: < 1.5

### Architecture
```
┌─────────────────────────────────────────┐
│           User / API Layer              │
│  (REST API, CLI, Web UI)               │
├─────────────────────────────────────────┤
│           Incus API                     │
├─────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐             │
│  │  LXC     │  │  QEMU    │             │
│  │  Container│  │  VM      │             │
│  └──────────┘  └──────────┘             │
├─────────────────────────────────────────┤
│  ZFS Storage  │  Linux Bridge Network   │
└─────────────────────────────────────────┘
```

### Deployment
```bash
# Single-node deployment
curl -fsSL https://raw.githubusercontent.com/weix2026/cubeve/main/scripts/l0-install.sh | sudo bash

# Manual steps
sudo apt update
sudo apt install -y incus
sudo incus admin init --auto
sudo incus launch images:ubuntu/24.04 test-ct
```

## L1: Basic (Distributed)

### Components
- L0 + **Incus Cluster** (Cowsql Raft)
- **OVN** (Software-Defined Networking)
- **Kata Containers** (containerd + RuntimeClass)
- ~5,000 lines of code

### Code Weight: < 3.0

### Architecture
```
┌─────────────────────────────────────────┐
│           Load Balancer                 │
├─────────────────────────────────────────┤
│  Node 1    │  Node 2    │  Node 3       │
│  ┌────────┐│  ┌────────┐│  ┌────────┐   │
│  │ Incus  ││  │ Incus  ││  │ Incus  │   │
│  │ + Cowsql││  │ + Cowsql││  │ + Cowsql│   │
│  └────────┘│  └────────┘│  └────────┘   │
├─────────────────────────────────────────┤
│  OVN SDN  │  Ceph/RBD (optional)        │
└─────────────────────────────────────────┘
```

## L2: Standard (Cloud-Native)

### Components
- L1 + **Kubernetes**
- **Cilium** (eBPF CNI)
- **Ceph** + Rook (distributed storage)
- **KubeVirt** (K8s-native VM management)
- ~20,000 lines of code

### Code Weight: ~20,000

### Architecture
```
┌─────────────────────────────────────────┐
│  Ingress Controller (nginx/traefik)     │
├─────────────────────────────────────────┤
│  Kubernetes API Server                  │
├─────────────────────────────────────────┤
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  │
│  │ Cilium  │  │ KubeVirt│  │ Rook    │  │
│  │ CNI     │  │ VM      │  │ Ceph    │  │
│  └─────────┘  └─────────┘  └─────────┘  │
├─────────────────────────────────────────┤
│  Kata Containers / RuntimeClasses      │
├─────────────────────────────────────────┤
│  Incus Cluster (L1 foundation)          │
└─────────────────────────────────────────┘
```

## L3: Full (Production)

### Components
- L2 + **CubeSandbox** (MicroVM, Firecracker/Cloud Hypervisor)
- **CubeVS** (eBPF/XDP network acceleration)
- **TEE/CoCo** (Confidential Computing)
- **Web UI + CLI** (Management Plane)
- Complete codebase

### Code Weight: Complete

### Architecture
```
┌─────────────────────────────────────────┐
│  CubeConsole (Web UI + CLI)             │
├─────────────────────────────────────────┤
│  CubeAPI Gateway (REST + gRPC)          │
├─────────────────────────────────────────┤
│  Instance Controller + CubeSandbox      │
│  Operator (VM Pool Manager)             │
├─────────────────────────────────────────┤
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  │
│  │ CubeVS  │  │ Cilium  │  │ KubeVirt│  │
│  │ eBPF/XDP│  │ eBPF CNI│  │ VM Mgmt │  │
│  └─────────┘  └─────────┘  └─────────┘  │
├─────────────────────────────────────────┤
│  CubeSandbox (MicroVM Pool)              │
│  - Pre-warmed VMs                        │
│  - CoW Clone                             │
│  - <100ms boot time                     │
├─────────────────────────────────────────┤
│  Kata Containers / TEE / CoCo          │
├─────────────────────────────────────────┤
│  Kubernetes + Incus Cluster              │
└─────────────────────────────────────────┘
```

## Progressive Deployment Strategy

```
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│  L0     │ ──▶│  L1     │ ──▶│  L2     │ ──▶│  L3     │
│  MVP    │    │  Basic  │    │  Standard│    │  Full   │
│  <1.5   │    │  <3.0   │    │  ~20k   │    │ Complete│
└─────────┘    └─────────┘    └─────────┘    └─────────┘
     │              │              │              │
     ▼              ▼              ▼              ▼
  Single Node   Cluster      Kubernetes      Production
  Dev/Test      Small Prod   Cloud-Native     Enterprise
```

## Key Design Principles

1. **Unified API**: Single API surface for all instance types (LXC, VM, Kata, CubeSandbox)
2. **Runtime Selection**: RuntimeClass-based selection for different workloads
3. **Storage Abstraction**: ZFS (local) → Ceph (distributed) → CephFS/RBD (K8s)
4. **Network Evolution**: Linux Bridge → OVN → Cilium eBPF + CubeVS XDP
5. **Security Layers**: AppArmor/Seccomp → Kata → TEE/CoCo
6. **Progressive Complexity**: Start simple, add capabilities as needed
