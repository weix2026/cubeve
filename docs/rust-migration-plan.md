# CubeVE Rust Migration Plan

## Architecture (Zero FFI)

```
cubeve/
├── crates/
│   ├── api-gateway          # Axum REST API + gRPC
│   ├── cubeconsole          # CLI (clap + reqwest)
│   ├── cubesandbox-operator # VM pool management
│   ├── instance-controller  # K8s controller (kube-rs)
│   └── version-tool         # Version info
├── web/                     # React + TypeScript frontend
├── Cargo.toml              # Workspace root
└── Makefile.rust           # Rust build targets
```

## Technology Stack

| Layer | Technology |
|-------|-----------|
| Backend | Rust (axum, tokio, reqwest) |
| CLI | Rust (clap) |
| Frontend | React + TypeScript + Vite |
| Metrics | prometheus crate |
| Logging | tracing + tracing-subscriber |
| K8s | kube-rs (when network available) |
| Incus API | reqwest + Unix socket (custom transport) |

## Build

```bash
# Check workspace
cargo check --workspace

# Build all crates
cargo build --workspace --release

# Test
cargo test --workspace
```

## Next Steps

1. [ ] Upgrade Rust to 1.80+ (when network available)
2. [ ] Add Unix socket support for Incus API (hyper/reqwest)
3. [ ] Implement full API Gateway handlers (GET/POST/DELETE/PATCH)
4. [ ] Add Prometheus metrics endpoint
5. [ ] Implement gRPC with tonic (optional)
6. [ ] Create React frontend components
7. [ ] Write Rust tests (unit + integration)
8. [ ] Update CI/CD for Rust builds

## Zero FFI Guarantee

- No `unsafe` blocks for FFI
- No `extern "C"` functions
- No C dependencies
- Pure Rust + React stack
