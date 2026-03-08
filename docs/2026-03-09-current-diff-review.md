# 2026-03-09 当前改动评审与落地记录

## 1. 结论

当前 `git diff` 已经不是零散修补，而是一轮相对完整的功能交付，核心包含三部分：

1. `backend-go` 增加新的后端能力：批量延迟测试、路由命中模拟、核心运行时长、日志来源字段。
2. `backend-go/cmd/tui` 新增独立 Go TUI 子模块，提供终端形态的控制台。
3. `web` 适配新的日志来源模型和核心状态字段，并完成构建验证。

从当前验证结果看，这批改动已经达到“可继续收口、可准备提交”的状态，没有发现编译失败或明显阻塞项。

## 2. 本轮改动范围

### 2.1 后端 API 与领域模型

涉及文件：

- `backend-go/internal/domain/types.go`
- `backend-go/internal/httpapi/server.go`
- `backend-go/internal/httpapi/server_test.go`
- `backend-go/internal/service/service.go`
- `backend-go/internal/service/native/service.go`
- `backend-go/internal/service/native/log_stream.go`
- `backend-go/internal/service/native/embedded_core.go`
- `backend-go/internal/service/native/routing_test_tool.go`
- `backend-go/pkg/apitypes/types.go`

具体内容：

1. `CoreStatus` 新增 `startedAt`、`uptimeSec`，用于暴露核心启动时间和运行时长。
2. 新增批量延迟测试请求/结果结构：`BatchDelayTestRequest`、`ProfileDelayResult`、`BatchDelayTestResult`。
3. 新增路由测试请求/结果结构：`RoutingTestRequest`、`RoutingTestResult`。
4. `LogLine` 新增 `source` 字段，明确区分 `app` 与 `xray-core`。
5. HTTP API 新增：
   - `POST /api/profiles/delay/batch`
   - `POST /api/routing/test`
6. native service 新增批量测速能力，带超时和并发限制，并会把成功结果回写 profile 的 `DelayMs`。
7. native service 新增路由命中模拟能力，可按目标、协议、端口评估当前 routing 规则命中结果。
8. xray/core 与应用日志统一改成显式 `source`，不再依赖消息正文里的 `[app]` 前缀做推断。
9. 为独立 TUI 子模块新增 `pkg/apitypes` 共享 DTO 别名包，避免重复维护 API 类型。

### 2.2 后端测试补强

涉及文件：

- `backend-go/internal/httpapi/server_test.go`

新增测试：

1. `TestRoutingTestEndpoint`
   - 验证 `/api/routing/test` 能正确命中 `domain:example.com` 规则。
2. `TestProfilesBatchDelayEndpoint`
   - 验证 `/api/profiles/delay/batch` 对多个 profile 返回聚合结果。

说明：

- 这两项测试覆盖了本轮新增 API 的核心 happy path。
- 当前还没有覆盖更多异常分支，例如空规则、复杂 `ip/domain/port` 混合场景、超时边界等。

### 2.3 Web 侧适配

涉及文件：

- `web/app/(main)/logs/page.tsx`
- `web/lib/types.ts`
- `web/next-env.d.ts`

具体内容：

1. 日志页改为基于 `line.source` 做来源过滤，不再通过 `message.startsWith("[app]")` 推断。
2. 下载日志时会把来源一并写入导出文本。
3. 搜索条件现在同时覆盖 `source + message`。
4. Web 共享类型新增：
   - `CoreStatus.startedAt`
   - `CoreStatus.uptimeSec`
   - `LogLine.source`
5. `next-env.d.ts` 中的 route types import 路径从 `./.next/dev/types/routes.d.ts` 变为 `./.next/types/routes.d.ts`。

说明：

- 这个 `next-env.d.ts` 变化更像是构建产物/Next 版本行为调整带来的同步结果，不属于核心业务逻辑修改。

### 2.4 新增 Go TUI 子模块

涉及目录：

- `backend-go/cmd/tui/`

主要内容：

1. 新增独立模块 `backend-go/cmd/tui/go.mod`，通过 `replace v2raye/backend-go => ../..` 复用主仓库类型与逻辑契约。
2. 新增 API client，覆盖核心状态、profiles、subscriptions、routing、stats、logs SSE、events SSE 等主要接口。
3. 新增多页 TUI：
   - Dashboard
   - Profiles
   - Subscriptions
   - Network
   - Settings
   - Logs
4. 支持终端内执行关键动作：
   - 启停/重启核心
   - profile 导入、激活、单测延迟、批量测速
   - 订阅更新
   - 路由测试
   - 系统代理应用/清理
   - geodata 更新
   - TUN 修复
   - 配置保存
   - 核心错误清理
   - 退出清理
5. 引入后台轮询与 SSE 流：
   - overview 定时刷新
   - logs stream 自动重连
   - events stream 自动重连
6. TUI 内部结构完成多次拆分，当前已按页面、动作、store、render、stream 维度拆开，代码组织明显比单文件形态更稳定。

## 3. 已完成验证

本轮实际执行过的验证如下。

### 3.1 Go 主模块测试

命令：

```bash
cd backend-go && go test ./...
```

结果：通过。

关键信息：

- `internal/httpapi` 测试通过
- `internal/service/native` 测试通过
- 没有新增编译错误

### 3.2 TUI 子模块验证

命令：

```bash
cd backend-go/cmd/tui && go test ./...
```

结果：通过。

说明：

- 该模块目前没有测试文件，但至少已经确认依赖、类型引用和构建链路是通的。

### 3.3 Web 生产构建

命令：

```bash
cd web && npm run build
```

结果：通过。

说明：

- Next.js 生产构建、TypeScript 检查、静态页面生成均已完成。

### 3.4 前端测试脚本检查

命令：

```bash
cd web && npm test -- --runInBand
```

结果：失败，但原因不是代码错误，而是 `package.json` 中不存在 `test` script。

结论：

- 当前 `web` 没有标准单元测试入口，不能把“没有跑前端测试”误判成“当前 diff 有问题”。

## 4. 现在还需要做什么

从当前状态看，没有必须先修复的编译/构建问题。剩余工作主要是交付收口，而不是继续救火。

### 4.1 必做收口项

1. 补充 TUI 的使用文档。
   - 至少说明如何启动后端。
   - 如何启动 TUI。
   - 需要哪些环境变量，例如 `V2RAYN_TUI_BASE_URL`、`V2RAYN_TUI_TOKEN`。
   - 当前支持的功能边界。

2. 做一次真实联调烟测。
   - 当前已经过构建和后端测试，但 TUI 还没有经过真实 API/SSE 交互的人工验收。
   - 建议至少验证 Dashboard、Logs、Profiles、Network 四个主页面。

3. 判断 `web/next-env.d.ts` 是否需要纳入提交。
   - 如果团队接受 Next 自动同步后的文件变化，可以保留。
   - 如果该文件要求只保留框架默认内容，需确认是否应还原或重新生成。

### 4.2 建议补强项

1. 给 TUI 增加最小启动说明或脚本。
   - 例如补一个 `backend-go/cmd/tui/README.md`。
   - 或在根 README / 开发脚本中加入 TUI 启动方式。

2. 为新增 API 增加更多边界测试。
   - 批量测速：空 ID、重复 ID、超时、并发限制。
   - 路由测试：IP 规则、端口区间、默认直连模式、未命中规则场景。

3. 明确 TUI 产物的发布策略。
   - 当前 `.gitignore` 新增了 `v2raye-tui`，说明本地二进制产物已考虑忽略。
   - 但是否纳入 CI 构建、是否提供 release 产物、是否加入根脚本，仍未落定。

4. 如果要把这轮工作作为正式交付，建议补一份操作截图或录屏。
   - 这不是代码必须项，但对后续评审和验收非常有帮助。

## 5. 当前 diff 的价值总结

本轮改动带来的实际增量比较明确：

1. 后端接口能力更完整，已经具备批量测速和路由模拟排障能力。
2. 日志模型从“靠消息格式推断来源”升级到“显式来源字段”，这对 Web/TUI 两端都更稳。
3. TUI 已经形成可运行的独立控制台雏形，不再只是实验代码片段。
4. Web 已适配新的日志和状态字段，没有出现构建回归。

## 6. 当前建议

如果要继续推进，这轮最合理的顺序是：

1. 先补 TUI 使用文档。
2. 再做一次真实后端联调烟测。
3. 确认 `next-env.d.ts` 是否提交。
4. 然后按主题拆 commit，例如“backend api + tests”、“tui module”、“web log source sync”。

## 7. 附：本次检查使用的关键命令

```bash
cd backend-go && go test ./...
cd backend-go/cmd/tui && go test ./...
cd web && npm run build
cd web && npm test -- --runInBand
```
