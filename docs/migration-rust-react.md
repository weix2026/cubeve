# CubeVE 架构迁移评估: Go → Rust

## 环境状态 (Ubuntu 24.04 LTS)

| 组件 | 状态 | 版本 | 备注 |
|------|------|------|------|
| Go | ✅ 已安装 | 1.22.2 | 当前主力语言 |
| Rust | ✅ 已安装 | 1.75.0 | 版本较旧(2023-12)，建议升级 |
| Node.js | ✅ 已安装 | v24.15.0 | React 前端可用 |
| npm | ✅ 已安装 | 11.12.1 | - |
| Git | ✅ 已安装 | 2.43.0 | - |
| Incus | ✅ 已安装 | 7.1 | L0 基础设施 |
| Docker | ❌ 未安装 | - | 建议安装用于构建 |

## 现有架构 (Go)

```
cubeve/
├── cmd/
│   ├── api-gateway          # REST/gRPC API (gin, prometheus)
│   ├── cubeconsole          # CLI 工具
│   ├── cubesandbox-operator # VM 池管理
│   ├── instance-controller  # K8s 控制器 (stub)
│   └── version-tool         # 版本信息
├── go.mod                   # 依赖管理
└── scripts/                 # 部署脚本
```

**FFI 使用: 0**
- 无 `import "C"`
- 无 CGO 依赖
- 纯 Go 实现

## 目标架构 (Rust + React)

```
cubeve/
├── crates/
│   ├── api-gateway          # Axum/Actix-web + tonic
│   ├── cubeconsole          # clap CLI
│   ├── cubesandbox-operator # VM 池管理
│   ├── instance-controller  # kube-rs 控制器
│   └── version-tool         # 版本信息
├── web/                     # React 前端
│   ├── package.json
│   └── src/
└── Cargo.toml               # workspace
```

## 迁移路径

### 阶段 1: 准备 (1-2 天)
- [ ] 升级 Rust 到 1.80+ (当前 1.75 较旧)
- [ ] 安装 Docker (用于构建/测试)
- [ ] 创建 Rust workspace 结构
- [ ] 评估依赖库:
  - HTTP 框架: axum (推荐) vs actix-web
  - gRPC: tonic
  - CLI: clap
  - K8s: kube-rs
  - Incus: 需要纯 Rust HTTP 客户端 (reqwest/hyper)

### 阶段 2: 核心迁移 (1-2 周)
- [ ] api-gateway (Rust + axum)
  - 替换 gin → axum
  - 替换 prometheus client → prometheus crate
  - Incus HTTP 客户端 (reqwest + unix socket)
- [ ] cubeconsole (Rust + clap)
  - 替换 cobra → clap
  - API 客户端 (reqwest)
- [ ] version-tool (Rust)
  - 简单替换，最低优先级

### 阶段 3: 高级组件 (2-3 周)
- [ ] cubesandbox-operator (Rust + tokio)
  - VM 生命周期管理
- [ ] instance-controller (Rust + kube-rs)
  - K8s 控制器，需要网络下载依赖
- [ ] 前端 React (Node.js 已就绪)
  - 创建 React + TypeScript 项目
  - 对接 Rust API

### 阶段 4: 验证
- [ ] 14/14 L0 API 测试通过
- [ ] 脚本验证通过
- [ ] 构建流程 (Cargo + Makefile)
- [ ] CI/CD 更新 (GitHub Actions)

## 技术选型

| 功能 | Go 当前 | Rust 目标 |
|------|---------|-----------|
| HTTP API | gin | axum |
| gRPC | google.golang.org/grpc | tonic |
| CLI | cobra | clap |
| Metrics | prometheus/client_golang | prometheus crate |
| K8s | client-go | kube-rs |
| Incus HTTP | 自定义 | reqwest + hyper |
| 日志 | log/zap | tracing + log |
| 配置 | flag | clap/envy |

## 风险

1. **Rust 版本**: 1.75 较旧，部分 crate 可能需要 1.80+
2. **K8s 依赖**: kube-rs 需要网络下载，当前环境受限
3. **Incus 客户端**: 需要重新实现 HTTP + Unix socket
4. **构建时间**: Rust 编译慢于 Go，CI 时间增加
5. **FFI 风险**: 零 FFI 意味着不能用 bindgen，所有绑定需纯 Rust

## 建议

1. **先升级 Rust**: `rustup update` 或重新安装最新版
2. **渐进迁移**: 先 api-gateway + cubeconsole，再 operator
3. **并行运行**: 保持 Go 版本可用，逐步切流
4. **前端独立**: React 可先开发，对接 API 即可

## 立即行动

```bash
# 1. 升级 Rust
rustup update stable

# 2. 安装 Docker
apt-get install docker.io

# 3. 创建 Rust workspace
cargo init --name cubeve
cargo new crates/api-gateway
cargo new crates/cubeconsole
# ...

# 4. 开始 api-gateway 迁移
```
