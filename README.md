# v2rayE

[English
](./readme_en.md)

[![ui](image/README/ui.webp)
](./readme_en.md)

v2rayE 是一个 Linux 优先的本地代理控制平面，目标是把常用的代理管理、TUN VPN、系统代理联动、订阅管理和终端交互收敛到一个统一可执行文件里。

当前第一版的主路径已经明确：

- 一个统一入口程序 `./v2raye`
- 一个本机 HTTP API 服务模式 `./v2raye --server`
- 一个默认终端界面 TUI 模式 `./v2raye`
- 一个可打包、可发布的 Debian `.deb` 安装包

项目现在更接近“本地代理控制台 + TUN/VPN 工作台”，而不是单纯的 Web 面板。

## 第一版包含什么

- 统一可执行文件，默认进入 TUI，带 `--server` 可切换到后台服务模式
- 本地 HTTP API，供 TUI 和脚本调用
- 配置文件与订阅管理
- 节点选择、单节点测速、批量测速
- Linux 系统代理应用
- Xray TUN 模式
- TUN 一键诊断与一键修复
- TUN 直连绕过修复，避免 direct 流量被重新卷回 TUN
- 双栈 TUN 策略路由支持，在有 IPv6 默认路由时自动补齐 IPv6 规则
- `tun-health-check.sh` 健康检查脚本
- Debian 打包脚本
- GitHub Actions 自动构建 `.deb` 并在 tag 发布时自动附加到 Release

## 适用场景

- 想在 Linux 终端里直接管理代理，而不是依赖浏览器面板
- 想把 TUI、后端 API、systemd 服务、deb 安装包统一起来
- 想让 TUN 模式具备更强的可诊断性和自修复能力
- 想把节点管理、订阅更新、系统代理、TUN 路由收敛到一个本地控制平面

## 当前目录结构

- `backend-go/`: Go 后端、TUI、统一入口
- `scripts/`: 构建、启动、健康检查、Debian 打包脚本
- `docs/`: 设计说明、systemd 服务文件、迁移记录
- `dist/`: 本地产物输出目录，构建 `.deb` 时生成

## 快速开始

### 1. 本地构建

```bash
./scripts/build.sh
```

生成产物：

- `./v2raye`

### 2. 启动 TUI

```bash
./v2raye
```

默认模式会启动本地 TUI。

### 3. 启动后台服务

```bash
./v2raye --server
```

默认监听地址：

- `127.0.0.1:18000`

### 4. 一键拉起 VPN 工作流

```bash
./scripts/vpn-up.sh
```

这个脚本会按顺序：

- 确保后端已启动
- 启动核心
- 应用系统代理
- 检查核心状态
- 检查网络可用性

### 5. 执行 TUN 自检

```bash
sudo ./scripts/tun-health-check.sh
```

这个检查会覆盖：

- API 是否可用
- 核心是否在运行
- TUN 是否接管成功
- IPv4 策略路由是否完整
- 直连 `fwmark -> main` 是否存在
- 在存在 IPv6 默认路由时，IPv6 策略路由是否完整

## Debian 安装包

### 本地构建 `.deb`

```bash
./scripts/build-deb.sh 0.1.0
```

输出路径类似：

```bash
dist/v2raye_0.1.0_amd64.deb
```

### 安装

```bash
sudo apt install ./dist/v2raye_0.1.0_amd64.deb
```

### 卸载

```bash
sudo apt remove v2raye
sudo apt purge v2raye
```

安装后布局：

- 主程序：`/usr/lib/v2raye/v2raye`
- 全局命令：`/usr/bin/v2raye`
- 服务文件：`/usr/lib/systemd/system/v2raye-server.service`
- 运行数据目录：`/opt/v2rayE`

## systemd 部署

仓库已包含服务文件：

- `docs/systemd/v2raye-server.service`

手动安装方式：

```bash
sudo install -d -m 755 /opt/v2rayE
sudo install -d -m 755 /usr/lib/v2raye
sudo install -m 755 ./v2raye /usr/lib/v2raye/v2raye
sudo ln -sf /usr/lib/v2raye/v2raye /usr/bin/v2raye
sudo install -m 644 ./docs/systemd/v2raye-server.service /etc/systemd/system/v2raye-server.service
sudo systemctl daemon-reload
sudo systemctl enable --now v2raye-server
```

说明：默认配置下，后台服务不会在开机时强制改写 GNOME/KDE 桌面代理；如果需要桌面代理联动，请显式把 `systemProxyMode` 设为 `forced_change`，并在 root/systemd 环境中按需设置 `V2RAYN_DESKTOP_USER`。

## 核心环境变量

- `V2RAYN_API_ADDR`: 后端监听地址，默认 `0.0.0.0:18000`
- `V2RAYN_API_ALLOW_PUBLIC`: 是否允许公网来源访问后端 API，默认关闭；关闭时仅接受回环和局域网来源
- `V2RAYN_DATA_DIR`: 数据目录，默认 `/opt/v2rayE`
- `V2RAYN_API_TOKEN`: API Token，可选
- `V2RAYN_BACKEND_MODE`: 后端模式，默认 `native`
- `V2RAYN_CORE_CMD`: 指定外部核心命令
- `V2RAYN_CORE_CMD_TEMPLATE`: 指定带占位符的核心命令模板
- `V2RAYN_DESKTOP_USER`: 当服务以 root/systemd 运行时，指定写系统代理的桌面用户

开机恢复说明：检测到 `autoRun` 或上次运行需恢复时，服务会先等待有限时长的网络探测成功，再执行带退避的核心恢复重试，减少系统刚开机时默认路由和 DNS 尚未稳定导致的 TUN/代理恢复失败。

## TUN 能力说明

目前第一版已经补齐了 TUN 的关键稳定性缺口：

- Linux TUN 策略路由优先级窗口扩大
- 关键绕过规则优先生成
- 直连流量使用专用 `fwmark` 回主路由表
- 代理流量和直连流量都绑定真实物理接口
- 在存在 IPv6 默认路由时自动补齐 IPv6 策略路由
- 后端诊断 API 会返回 TUN takeover、direct bypass、IPv6 状态
- TUI 和脚本都能直接看到这些诊断结果

这意味着现在的 TUN 路径已经不是“只看核心是否 running”，而是能进一步确认：

- 是否真正接管默认路由
- 直连流量是否有逃逸路径
- 双栈环境下 IPv6 是否被正确处理

## API 概览

第一版已经具备一组可直接使用的本机控制 API，包括：

- `/api/health`
- `/api/core/status`
- `/api/core/start`
- `/api/core/stop`
- `/api/core/restart`
- `/api/profiles`
- `/api/subscriptions`
- `/api/network/availability`
- `/api/system-proxy/users`
- `/api/system-proxy/apply`
- `/api/config`
- `/api/routing`
- `/api/routing/diagnostics`
- `/api/routing/tun/repair`
- `/api/routing/hits`
- `/api/events/stream`
- `/api/logs/stream`

## GitHub 自动发布 `.deb`

仓库已加入 GitHub Actions 工作流：

- `.github/workflows/release-deb.yml`

发布方式：

```bash
git tag v0.1.0
git push origin master
git push origin v0.1.0
```

触发后，GitHub Actions 会自动：

- 安装 Go 环境
- 运行 `go test ./...`
- 执行 `./scripts/build-deb.sh <version>`
- 生成 `.deb` 和 `SHA256SUMS`
- 自动创建 GitHub Release 并上传产物

如果只是想手动测试工作流，也可以在 GitHub Actions 页面用 `workflow_dispatch` 手动输入版本号触发。

## 建议的第一版版本号

建议直接从：

- `v0.1.0`

开始。

这个版本适合作为“统一入口 + TUI 主路径 + Linux TUN 稳定化 + Debian 发布链路”的第一版基线。

## 当前限制

- 当前主打 Linux 场景
- system proxy 的桌面集成主要面向 Linux 桌面会话
- 自动发布目前输出的是 Debian 包，不包含 rpm / apk / AppImage
- 发布流默认针对 Git tag 构建 Release，不做 nightly

## 开发验证

后端基础验证命令：

```bash
cd backend-go
go test ./...
```

## 许可证

当前仓库未单独声明许可证文件。如需公开长期发布，建议尽快补上明确的 LICENSE。
