# v2rayE AGENTS.md

Linux 优先的本地代理控制平面（Xray 核心 + TUN + 系统代理 + 订阅管理）。统一入口 `./v2raye`，Go 后端在 `backend-go/`。

本文件是树状索引，按需加载链接；不承载完整 API 契约 / 长经验。

## 稳定事实

- 数据目录：`/opt/v2rayE`（JSON 存储 + geoip/geosite 数据）。数据文件：`config.json`、`profiles.json`、`subscriptions.json`、`routing.json`、`state.json`、`runtime-config.json`（生成的 xray 配置）。
- 后端 API 监听：`127.0.0.1:18000`（systemd 服务 `--api-addr 0.0.0.0:18000 --data-dir /opt/v2rayE`，见 `docs/systemd/v2raye-server.service`）。
- 本地代理端口：HTTP `10809`，SOCKS `10808`；stats `10085`。
- 路由模式：`global | bypass_cn | direct | custom`（存于 `routing.json`，schema 见 `backend-go/internal/domain/types.go` 的 `RoutingConfig`）。
- 自定义路由规则：`routing.json` 的 `rules[]`，支持 `domain|ip|geoip|geosite|port|protocol`，`outbound` 为 `proxy|direct|block`。生成进 xray 配置的映射见 `backend-go/internal/service/native/config_gen.go:637`。
- 核心引擎：`coreEngine`（默认 `xray-core`）。
- 当前版本：`v0.2.0`（发布说明见 `docs/2026-10-01-v0.2.0-release.md`）。
- 开发验证：`cd backend-go && go test ./...`（CI 另跑 `go vet` + `go test -race`）。

## 经验积累

- **[2026-08-26] 强制某域名走代理（不改代码）**：通过 API `PUT /api/routing` 写入 `rules: [{type:domain, values:[域名], outbound:proxy}]` 后核心自动重启生效；不要直接改 `routing.json` 文件（内存缓存不一致）。操作与验证详见 [docs/2026-08-26-forced-proxy-domain.md](docs/2026-08-26-forced-proxy-domain.md)。
- 直连出口 IP 与本机代理节点可能同源（本机物理出口 = 代理节点 IP 时，用"出口 IP 对比"无法证明是否走代理），此时用 `tcpdump` 对比到 CloudFront/代理节点的连接目标，或 `POST /api/routing/test` 看规则命中。

## 运维 / 部署

- 安装 / systemd 部署步骤：见 [backend-go/README.md](backend-go/README.md) 与 [docs/systemd/v2raye-server.service](docs/systemd/v2raye-server.service)。
- Debian 打包 / 发布：`./scripts/build-deb.sh <version>`，GitHub Actions 自动构建发布。
- VPN 一键拉起：`./scripts/vpn-up.sh`；TUN 自检：`sudo ./scripts/tun-health-check.sh`。

## 链接索引（长文档）

- API 概览：`README.md`（"API 概览"小节）
- 后端模块依赖 / 架构：`backend-go/docs/`
- 版本发布：`docs/2026-05-10-v0.1.4-release.md`、`docs/2026-08-10-v0.1.5-release.md`、`docs/2026-10-01-v0.2.0-release.md`
