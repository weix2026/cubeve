# CubeVE Architecture Overview

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         User Layer                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │   Web UI    │  │   CLI Tool  │  │   Third-party Apps  │  │
│  │  (Future)   │  │  cubeconsole │  │    (REST/gRPC)      │  │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘  │
└─────────┼────────────────┼────────────────────┼─────────────┘
          │                │                    │
┌─────────┼────────────────┼────────────────────┼─────────────┐
│         ▼                ▼                    ▼             │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │                    API Gateway Layer                   │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐   │  │
│  │  │  REST API   │  │   gRPC API  │  │  Metrics    │   │  │
│  │  │   :8080     │  │   :50051    │  │   :9090     │   │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘   │  │
│  └─────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
          │                │                    │
┌─────────┼────────────────┼────────────────────┼─────────────┐
│         ▼                ▼                    ▼             │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │                  Control Plane Layer                     │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐   │  │
│  │  │  Instance   │  │  CubeSandbox│  │  Management │   │  │
│  │  │  Controller │  │  Operator   │  │  Plane      │   │  │
│  │  │  (K8s CRD)  │  │  (K8s CRD)  │  │  (API, Web) │   │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘   │  │
│  └─────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
          │                │                    │
┌─────────┼────────────────┼────────────────────┼─────────────┐
│         ▼                ▼                    ▼             │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │                   Runtime Abstraction Layer              │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐   │  │
│  │  │    Incus    │  │    Kata     │  │    Cube     │   │  │
│  │  │  (LXC/VM)   │  │  Containers │  │  MicroVMs   │   │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘   │  │
│  │  ┌─────────────┐  ┌─────────────┐                     │  │
│  │  │   KubeVirt  │  │    TEE/     │                     │  │
│  │  │   (K8s VM)  │  │  Confidential │                    │  │
│  │  └─────────────┘  └─────────────┘                     │  │
│  └─────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
          │                │                    │
┌─────────┼────────────────┼────────────────────┼─────────────┐
│         ▼                ▼                    ▼             │
│  ┌─────────────────────────────────────────────────────────┐  │
│  │                   Infrastructure Layer                   │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐   │  │
│  │  │   Ceph/     │  │   Cilium/   │  │   OVN/      │   │  │
│  │  │   ZFS       │  │   Cube-VS   │  │   Bridge    │   │  │
│  │  │  (Storage)  │  │  (Network)  │  │  (Network)  │   │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘   │  │
│  └─────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Component Descriptions

### API Gateway Layer
- **REST API**: HTTP/JSON API for external integration
- **gRPC API**: High-performance binary protocol for internal services
- **Metrics**: Prometheus-compatible metrics endpoint

### Control Plane Layer
- **Instance Controller**: Kubernetes operator managing VM/Container lifecycle
- **CubeSandbox Operator**: Manages high-performance MicroVMs with prewarm pools
- **Management Plane**: Web UI and administrative APIs

### Runtime Abstraction Layer
- **Incus**: Full-featured container and VM management (LXC/QEMU)
- **Kata Containers**: Secure container runtime with VM isolation
- **CubeSandbox**: Ultra-fast MicroVMs with prewarm pools and COW cloning
- **KubeVirt**: Kubernetes-native virtual machine management
- **TEE/Confidential**: Confidential computing with remote attestation

### Infrastructure Layer
- **Storage**: Ceph distributed storage or ZFS local storage
- **Network**: Cilium eBPF networking or OVN SDN or basic bridge networking
- **Cube-VS**: Ultra-fast virtual switch with XDP acceleration

## Data Flow

1. User request arrives at API Gateway (REST or gRPC)
2. Request is authenticated and authorized
3. Control Plane processes the request
4. Runtime Adapter translates to specific runtime operations
5. Infrastructure Layer executes the actual operations
6. Metrics and events are collected throughout

## Deployment Levels

### L0: Control Plane (Single Node)
- Incus daemon + ZFS storage
- Basic container/VM management
- Local API access

### L1: Cluster Management
- Multi-node Incus cluster
- Distributed storage and networking
- Live migration support

### L2: Kubernetes Integration
- K8s operators and CRDs
- RuntimeClasses for multiple runtimes
- CNI networking integration

### L3: Full Production
- Helm charts for deployment
- Monitoring and logging
- CI/CD pipelines
- Terraform infrastructure
