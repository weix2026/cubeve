# L0 验证清单

## 验证环境
- **Incus 版本**: 7.1
- **存储后端**: ZFS (test-pool)
- **网络**: incusbr0 (10.185.6.1/24), test-net (10.149.141.1/24)
- **Profile**: test-profile (limits.cpu=1, limits.memory=1GB, root disk 10GB)
- **实例**: zfs-test (STOPPED, x86_64, eth0 attached to incusbr0)

## 验证项目

### ✅ 1. Incus 服务
- [x] Incus daemon 运行正常
- [x] API 监听端口 8443
- [x] Unix socket 可用 (/var/lib/incus/unix.socket)
- [x] 版本 7.1

### ✅ 2. 存储管理
- [x] ZFS 存储池创建 (test-pool)
- [x] ZFS 数据集管理 (test-pool/cubeve)
- [x] ZFS 快照创建 (test-pool@baseline, test-pool/cubeve@init, test-pool/cubeve@snap2)
- [x] Incus 存储池创建 (zfs-pool, 映射到 test-pool)
- [x] 存储卷创建 (container zfs-test)
- [x] 存储卷管理

### ✅ 3. 网络管理
- [x] Bridge 网络创建 (incusbr0, test-net)
- [x] NAT 配置 (IPv4 + IPv6)
- [x] 网络地址分配 (10.185.6.1/24, 10.149.141.1/24)
- [x] 网络附加到实例 (eth0 on incusbr0)
- [x] 多网络支持 (已验证，但同一网络冲突)

### ✅ 4. Profile 管理
- [x] Profile 创建 (test-profile)
- [x] 资源限制配置 (limits.cpu, limits.memory)
- [x] 设备配置 (root disk, nic)
- [x] 设备附加到实例

### ✅ 5. 实例管理
- [x] 空容器创建 (test-empty, zfs-test)
- [x] 使用 ZFS 存储池创建实例
- [x] 实例配置修改 (limits.cpu=2, limits.memory=2GB)
- [x] 实例快照创建 (baseline)
- [x] 实例快照列表
- [x] 实例备份导出 (1.8K tar.gz)
- [x] 实例删除 (test-empty, test-vm)
- [x] 网络设备附加

### ⚠️ 6. 实例启动
- [ ] 容器启动失败 (嵌套环境限制)
- 原因: Failed to exec "/sbin/init" - 空容器无 rootfs
- 原因: veth 网络接口重命名失败
- 备注: 元数据和配置操作完全正常，启动需要完整 rootfs

### ❌ 7. VM 管理
- [ ] VM 创建失败 (无 /dev/kvm)
- 原因: 嵌套虚拟化不可用
- 备注: 控制平面功能完全正常

### ✅ 8. API 操作
- [x] REST API 基础信息 (/1.0)
- [x] 实例列表 API (/1.0/instances)
- [x] 实例详情 API (/1.0/instances/zfs-test)
- [x] 存储池列表 API
- [x] 网络列表 API
- [x] 所有 API 返回正确 JSON 格式

### ✅ 9. 配置管理
- [x] 实例配置设置 (config set)
- [x] 实例配置显示 (config show)
- [x] 全局配置管理
- [x] 设备配置 (磁盘、网络)

### ⚠️ 10. 集群功能
- [ ] 集群列表失败 (单节点，未加入集群)
- 备注: 预期行为，集群配置在 L1 脚本中

## 总结
- **L0 控制平面**: 100% 验证通过（存储、网络、配置、API、Profile、快照）
- **L0 运行时**: 受限（容器启动失败，VM 不可用）
- **环境限制**: 嵌套容器环境，无 KVM，镜像下载被限制
- **L0 脚本**: 可部署控制平面，运行时需在物理机/KVM 环境验证

## 下一步
- 验证 L1 集群配置（在物理机环境）
- 验证 L2 Kubernetes 集成（在 K8s 集群环境）
- 验证 L3 CubeSandbox 性能（在 Firecracker/Cloud Hypervisor 环境）
- 完善 CLI 工具和 Web UI
