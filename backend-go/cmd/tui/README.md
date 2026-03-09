# v2rayE TUI

当前默认交互界面为终端 TUI，技术栈为 `tview + tcell`。

## 启动

从仓库根目录（推荐）：

```bash
./scripts/build.sh
./v2raye
```

默认模式会连接本地 API（`http://127.0.0.1:18000`）并进入 TUI。配合 Debian 安装包时，root 后端服务会在安装阶段自动拉起，普通用户可直接执行 `/usr/bin/v2raye` 使用 TUI。

如果你希望一键完成“后端 + core + 系统代理 + 健康检查”：

```bash
./scripts/vpn-up.sh
```

如果你在开发中直接运行 Go 入口：

```bash
cd backend-go
go run ./cmd/v2raye --base-url http://127.0.0.1:18000
```

启用 Bearer token 时：

```bash
V2RAYN_TUI_TOKEN=your-token ./v2raye
```

常用环境变量：

- `V2RAYN_TUI_AUTO_UP`：`1`（默认）自动启动 core，`0` 关闭自动启动
- `V2RAYN_VPN_PROXY_MODE`：`vpn-up.sh` 使用的系统代理模式（默认 `global`）
- `V2RAYN_VPN_PROXY_EXCEPTIONS`：`vpn-up.sh` 系统代理例外域名（默认空）

## 设计方向

- TUI 是主界面，不再把浏览器作为默认入口。
- 后端 API 默认监听所有网卡以便局域网访问，但默认只接受本机回环和局域网来源，不面向公网开放。
- 数据平面保持 `xray-core`，TUI 负责控制、观测、诊断。
- UI 风格以 termshark 的终端交互模型为目标：多面板、快捷键优先、状态持续可见、日志与诊断近实时更新。

## 当前覆盖

- Dashboard：核心状态、流状态、事件概览
- Profiles：激活、导入、单点测速、批量测速
- Subscriptions：更新全部、更新单项
- Network：可用性检查、系统代理、geodata、TUN 修复、路由测试
- Settings：配置编辑、核心错误清理、退出清理
- Logs：SSE 实时日志、来源/级别过滤、搜索