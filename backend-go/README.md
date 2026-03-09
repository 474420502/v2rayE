# v2rayE backend-go

Go 实现的本地 VPN 控制平面。当前默认服务于 `cmd/tui` 终端界面，`web/` 保留为迁移中的历史界面，不再作为主路径设计中心。

## 运行

推荐使用统一可执行文件：

```bash
./scripts/build.sh
./v2raye
```

默认启动 TUI 交互界面。

服务模式（适合 systemd 开机自启）：

```bash
./v2raye --server
```

`systemd` 示例文件见：`docs/systemd/v2raye-server.service`

```bash
sudo install -d -m 755 /opt/v2rayE
sudo install -d -m 755 /usr/lib/v2raye
sudo install -m 755 ./v2raye /usr/lib/v2raye/v2raye
sudo ln -sf /usr/lib/v2raye/v2raye /usr/bin/v2raye
sudo install -m 644 ./docs/systemd/v2raye-server.service /etc/systemd/system/v2raye-server.service
sudo systemctl daemon-reload
sudo systemctl enable --now v2raye-server
```

VPN 一键拉起（后端 + core + 系统代理 + 健康检查）：

```bash
./scripts/vpn-up.sh
```

如需仅启动本地后端 API（无交互界面），也可以直接使用底层入口：

```bash
cd backend-go
go run ./cmd/backend-api
```

TUN/VPN 一次性自检：

```bash
cd ..
sudo ./scripts/tun-health-check.sh
```

默认监听：`0.0.0.0:18000`，但默认只接受本机回环和局域网来源访问

默认终端界面：`backend-go/cmd/tui`

## Debian 打包（.deb）

在项目根目录执行：

```bash
./scripts/build-deb.sh 0.1.0
```

输出路径：`dist/v2raye_<version>_<arch>.deb`

安装/卸载：

```bash
sudo apt install ./dist/v2raye_0.1.0_amd64.deb
sudo apt remove v2raye
sudo apt purge v2raye
```

说明：

- 安装后服务文件在 `/usr/lib/systemd/system/v2raye-server.service`
- 可执行文件在 `/usr/lib/v2raye/v2raye`
- 全局命令在 `/usr/bin/v2raye`（软链接到 `/usr/lib/v2raye/v2raye`）
- 安装时会自动 `daemon-reload`、`enable` 并尝试启动服务，安装后的普通用户可直接运行 `/usr/bin/v2raye` 进入 TUI

## 环境变量

- `V2RAYN_API_ADDR`：监听地址，默认 `0.0.0.0:18000`
- `V2RAYN_API_ALLOW_PUBLIC`：是否允许公网来源访问后端 API，默认关闭；关闭时仅接受回环和局域网来源
- `V2RAYN_BACKEND_MODE`：`native`（默认，推荐）、`memory`（联调保底）或 `servicelib-proxy`（兼容过渡）
- `V2RAYN_DATA_DIR`：统一数据目录，默认 `/opt/v2rayE`
- `V2RAYN_MEMORY_STATE_PATH`：`memory` 共享状态基准路径（默认 `${V2RAYN_DATA_DIR}/memory-state.json`），会派生出 `*.runtime.json`、`*.subscriptions.json`、`*.config.json` 三类文件，并保留整体快照兼容读取
- `V2RAYN_CORE_CMD`：`native` 模式下可选，用于指定真实核心启动命令（如 `xray run -c /path/to/config.json`）
- `V2RAYN_CORE_CMD_TEMPLATE`：`native` 模式下可选命令模板（支持占位符 `{config}`、`{profileId}`、`{coreType}`）
- `V2RAYN_NATIVE_STATE_PATH`：`native` 模式状态文件路径（默认 `${V2RAYN_DATA_DIR}/native-state.json`），用于服务重启后的状态恢复
- `V2RAYN_SERVICELIB_BRIDGE_CMD`：`servicelib-proxy` 模式下用于调用 `ServiceLib` 的桥接命令
- `V2RAYN_SERVICELIB_BRIDGE_TIMEOUT_MS`：桥接调用超时（毫秒），默认 `3000`
- `V2RAYN_SERVICELIB_BRIDGE_ALLOW_ACTIONS`：桥接动作白名单（逗号分隔，`*`/`all` 表示全量），默认最小维护集：`core.status,core.start,core.stop,core.restart,config.get,config.update`
- `V2RAYN_SERVICELIB_BRIDGE_METRICS_LOG`：是否输出 bridge 成功调用耗时日志（`1/true/yes/on` 启用，默认关闭）
- `V2RAYN_DESKTOP_USER`：当后端以 root/systemd 运行且需要执行 `gsettings` 时，指定要写入系统代理的桌面用户名（例如 `eson`）

运行时配置新增（`/api/config` 可读写）：

- `coreAutoRestart`：核心异常退出后是否自动拉起（默认 `true`）
- `coreAutoRestartMaxRetries`：自动拉起最大重试次数（默认 `5`，`0` 表示不限制）
- `coreAutoRestartBackoffMs`：自动拉起基础退避毫秒（默认 `500`，指数退避，最大 30 秒）
- `tunAutoRoute`：传给 xray TUN inbound 的 `autoRoute`，默认 `true`
- `tunHijackDefaultRoute`：是否额外由后端手动执行 `ip route replace default dev <tun>`，默认 `false`；这是高风险兼容开关，只有在确有需要时才应开启
- `systemProxyUsers`：系统代理目标用户列表（数组）；支持多用户候选，后端会优先选择非系统用户 + 有会话总线的用户

## 目录分层

- `cmd/backend-api`：本地 API 启动入口（供 TUI 调用）
- `internal/httpapi`：HTTP 路由与请求/响应映射
- `internal/service`：业务接口定义与错误约定
- `internal/service/native`：Go 原生服务编排（默认）
- `internal/service/memory`：内存实现（联调/兜底）
- `internal/service/servicelib`：兼容桥接层（保留过渡能力，不作为主路径）
- `internal/domain`：DTO 与响应模型

## 纯 Go 默认路径（当前）

- 默认模式 `native`：由 Go 后端自身承接核心流程，作为当前主路径。
- 默认 `coreEngine=xray-core`：后端以内嵌 `xray-core` 提供本地 HTTP/SOCKS5 代理入口（默认端口 `10809/10808`），不依赖外部 `xray` 进程。
- 当前仅保留 `xray-core` 引擎路径，旧的 `embedded/auto/xray` 模式已统一折叠到 `xray-core`。
- 默认 UI 路径已切换为终端 TUI，后端 API 仅作为本机控制面，不再以浏览器访问作为默认交互假设。
- `native` / `servicelib-proxy` 共用 `memory` 状态底座，配置、订阅、当前节点与测速缓存会按“runtime/config/subscriptions”拆分落盘到 `V2RAYN_MEMORY_STATE_PATH` 派生文件。
- 若设置 `V2RAYN_CORE_CMD`，`native` 会尝试拉起真实核心进程；未设置时使用内存状态承接 API 闭环。
- 若设置 `V2RAYN_CORE_CMD_TEMPLATE`，`native` 会先生成临时配置文件，再以模板命令启动核心。
- `native` 运行时配置会按节点协议生成 `outbound`（支持 `vmess/vless/trojan/shadowsocks/socks/http`）：优先读取配置中的 `profiles[].protocol/type/nodeType`，否则按节点名关键词推断，最终回退 `vmess`。
- `V2RAYN_CORE_CMD` 安全约束：禁止包含 shell 管道/重定向/串联符（如 `|`、`;`、`&&`、`||`、`>`、`<`、`` ` ``、`$(`）。

示例：

```bash
cd ..
V2RAYN_CORE_CMD_TEMPLATE='xray run -c {config}' ./scripts/start-backend.sh
```

## 桥接兼容路径（可选）

- 当前仍保留“核心 + 配置/订阅/节点/网络”桥接动作协议，用于兼容与压测
- 当 `V2RAYN_BACKEND_MODE=servicelib-proxy` 时，后端会调用 `V2RAYN_SERVICELIB_BRIDGE_CMD`
- 默认仅最小白名单动作走桥接，其他动作直接回退本地实现（可通过 `V2RAYN_SERVICELIB_BRIDGE_ALLOW_ACTIONS` 覆盖）
- 任一桥接调用失败时自动回退内存实现兜底，并记录回退日志，保证联调链路稳定

桥接日志增强（1.3.32）：

- 失败日志会包含 `reason`（失败分类）、`elapsedMs`（耗时）与 `bucket`（耗时桶）
- 成功日志在启用 `V2RAYN_SERVICELIB_BRIDGE_METRICS_LOG` 时输出 `elapsedMs` 与 `bucket`
- `web/scripts/bridge-whitelist-drill.mjs` 已支持解析并在产物中输出：
	- `fallbackReasons`
	- `bridgeLatencyBuckets`
	- `bridgeLatencyP95Ms` / `bridgeLatencyP99Ms`

bridge 建议增强（1.3.34）：

- `web/scripts/bridge-advice.mjs` 已支持“最新样本 vs 最近 N 次”退化判断。
- 可选参数：
	- `V2RAYN_BRIDGE_ADVICE_BASELINE_LIMIT`（默认 `10`）
	- `V2RAYN_BRIDGE_ADVICE_DEGRADE_P95_RATIO_WARN`（默认 `1.3`）
	- `V2RAYN_BRIDGE_ADVICE_DEGRADE_P95_RATIO_CRIT`（默认 `1.8`）

示例（使用 mock 桥接命令）：

```bash
cd ..
V2RAYN_BACKEND_MODE=servicelib-proxy V2RAYN_SERVICELIB_BRIDGE_CMD='node ./backend-go/scripts/servicelib-bridge-mock.mjs' ./scripts/start-backend.sh
```

协议文档与兼容参考：

- 协议：`docs/servicelib-bridge-protocol.md`
- 运维：`docs/servicelib-bridge-ops-guide.md`
- C# 模板：`examples/servicelib-bridge-csharp/Program.cs`（历史参考，不是当前推荐实现路径）

## 已实现接口（MVP）

- `GET /api/health`
- `GET /api/core/status`
- `POST /api/core/start`
- `POST /api/core/stop`
- `POST /api/core/restart`
- `GET /api/profiles`
- `POST /api/profiles/{id}/select`
- `GET /api/subscriptions`
- `POST /api/subscriptions`
- `PUT /api/subscriptions/{id}`
- `DELETE /api/subscriptions/{id}`
- `POST /api/subscriptions/update`
- `POST /api/subscriptions/{id}/update`
- `GET /api/network/availability`
- `GET /api/system-proxy/users`（列出系统代理候选用户，非系统用户优先排序）
- `POST /api/system-proxy/apply`
- `POST /api/app/exit-cleanup`
- `GET /api/config`
- `PUT /api/config`
- `POST /api/routing/geodata/update`（主动下载并更新 `geosite.dat` 与 `geoip.dat`）
- `GET /api/events/stream`（SSE，实时事件）

说明：当 `PUT /api/routing` 设置为 `bypass_cn`（或包含 `geosite` / `geoip` 规则）且本地缺失对应数据文件时，后端会自动尝试下载并写入 `geosite.dat` / `geoip.dat`。

返回统一模型：

- 成功：`{ code: 0, message: "ok", data: ... }`
- 失败：`{ code: non-zero, message: "error", details: ... }`
