# L0 部署实录 - 环境限制与应对

## 环境扫描
- **OS**: Ubuntu 24.04 LTS (noble)
- **CPU**: 4 vCPU (General Processors - VM 环境)
- **内存**: 7.5GB
- **磁盘**: 40GB vda
- **虚拟化**: 无 /dev/kvm, 无嵌套虚拟化支持
- **容器类型**: systemd-nspawn/init.scope 类容器环境
- **限制**: 无法运行 KVM VM，LXC 嵌套容器可能受限

## 当前环境约束
1. **VM 不可运行**: 无 /dev/kvm，QEMU 无法硬件加速
2. **LXC 可能受限**: 非特权容器环境下嵌套 LXC 需要特殊配置
3. **可用能力**: 可以作为控制节点、安装 Incus、编写/测试脚本、验证 API

## L0 部署策略调整
- 正常安装 Incus、配置存储池、网络
- 尝试 LXC 容器运行，记录失败/成功
- 准备完整的 L0-L3 部署脚本（可在真实物理机/KVM 环境运行）
- 用 LXC 模拟 VM 部分配置，验证编排逻辑

## 实施日志
### Step 1: 安装 Incus
```bash
# 通过 Zabbly 官方仓库安装 Incus
curl -fsSL https://pkgs.zabbly.com/key.asc | gpg --dearmor -o /usr/share/keyrings/zabbly.gpg
echo "deb [signed-by=/usr/share/keyrings/zabbly.gpg] https://pkgs.zabbly.com/incus/stable $(lsb_release -cs) main" | \
  tee /etc/apt/sources.list.d/zabbly-incus-stable.list
apt update && apt install -y incus
```
**结果**: ✅ Incus 已安装（版本 7.1）

### Step 2: 初始化（单节点模式）
```bash
incus admin init --auto
```
**结果**: ✅ 初始化成功
- 存储池: default (dir)
- 网络: incusbr0 (10.185.6.1/24)

### Step 3: 尝试创建 LXC 容器
```bash
incus launch images:ubuntu/24.04/cloud test-ct
```
**结果**: ❌ 镜像下载超时（环境限制：容器内网络慢/资源受限）
- 问题: 镜像下载到 93% 被 SIGKILL
- 限制: 当前环境为容器环境，网络带宽有限

### Step 4: 环境限制总结
| 限制项 | 状态 | 影响 |
|--------|------|------|
| KVM 虚拟化 | ❌ 无 /dev/kvm | 无法运行 VM |
| LXC 嵌套容器 | ⚠️ 可能受限 | 容器创建可能失败 |
| 网络带宽 | ⚠️ 慢 | 镜像下载超时 |

## 调整策略
由于当前环境限制，采用**"配置先行、脚本完备、环境就绪即运行"**策略：
1. 编写完整的 L0-L3 部署脚本和配置
2. 在当前环境验证脚本语法和逻辑
3. 提交到 GitHub 仓库
4. 在真实物理机/KVM 环境可直接运行

## 下一步
- 创建 L0-L3 完整部署脚本集
- 创建配置文件和文档
- 提交到 GitHub cubeve 仓库
