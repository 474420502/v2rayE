# v2rayE TUI

当前默认交互界面为终端 TUI，技术栈与 termshark 同源：`gowid + tcell`。

## 启动

从仓库根目录：

```bash
./scripts/start-tui.sh
```

如果后端已运行，也可以直接启动：

```bash
cd backend-go/cmd/tui
go run . --base-url http://127.0.0.1:18000
```

启用 Bearer token 时：

```bash
V2RAYN_API_TOKEN=your-token ./scripts/start-tui.sh
```

或：

```bash
V2RAYE_TUI_TOKEN=your-token ./scripts/start-tui.sh
```

## 设计方向

- TUI 是主界面，不再把浏览器作为默认入口。
- 后端 API 是本机控制平面，不是面向公网的产品接口。
- 数据平面保持 `xray-core`，TUI 负责控制、观测、诊断。
- UI 风格以 termshark 的终端交互模型为目标：多面板、快捷键优先、状态持续可见、日志与诊断近实时更新。

## 当前覆盖

- Dashboard：核心状态、流状态、事件概览
- Profiles：激活、导入、单点测速、批量测速
- Subscriptions：更新全部、更新单项
- Network：可用性检查、系统代理、geodata、TUN 修复、路由测试
- Settings：配置编辑、核心错误清理、退出清理
- Logs：SSE 实时日志、来源/级别过滤、搜索