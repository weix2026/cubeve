# L0 部署验证指南

## 验证清单

### 1. 检查 Incus 安装
```bash
incus --version
incus info
```

### 2. 检查 API Gateway
```bash
# 启动 API Gateway
./bin/api-gateway --listen=:8080

# 健康检查
curl http://localhost:8080/healthz
# 预期: {"status":"ok"}

# 就绪检查
curl http://localhost:8080/readyz
# 预期: {"status":"ready"}
```

### 3. 检查存储池
```bash
# 列出存储池
curl http://localhost:8080/api/v1/storage-pools
# 预期: 包含 default 和 zfs-pool

# 检查默认存储池
curl http://localhost:8080/api/v1/storage-pools/default
```

### 4. 检查网络
```bash
# 列出网络
curl http://localhost:8080/api/v1/networks
# 预期: 包含 incusbr0

# 检查默认网络
curl http://localhost:8080/api/v1/networks/incusbr0
```

### 5. 检查实例
```bash
# 列出实例
curl http://localhost:8080/api/v1/instances
# 预期: 空列表或包含 zfs-test

# 使用 cubeconsole
./bin/cubeconsole instance list
./bin/cubeconsole instance show zfs-test
```

### 6. 检查 Profile
```bash
# 列出 profiles
curl http://localhost:8080/api/v1/profiles
# 预期: 包含 default

# 检查默认 profile
curl http://localhost:8080/api/v1/profiles/default
```

## 故障排查

### API Gateway 无法连接 Incus
```bash
# 检查 Incus socket
ls -la /var/lib/incus/unix.socket

# 检查 Incus 状态
incus info

# 重启 API Gateway 指定正确 endpoint
./bin/api-gateway --incus=unix:///var/lib/incus/unix.socket
```

### 测试失败
```bash
# 运行 L0 API 测试
./scripts/test-l0-api.sh

# 验证脚本语法
./scripts/validate-scripts.sh
```
