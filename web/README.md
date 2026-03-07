# v2rayN Web (Next.js)

该目录是迁移方案中的前端工程（React + Next.js），在当前工作区作为独立项目运行。

- 与 `v2rayN` 代码仓库解耦，不依赖其内部文件或项目引用。
- `v2rayN` 仅作为功能与交互路径参考。

## 本地运行

```bash
cd web
npm install
npm run dev
```

默认打开：`http://localhost:3000`

## 生产构建

```bash
cd web
npm run build
npm run start
```

## 后端联调验收（第 1.2 轮）

推荐后端实现：Go（跨平台）。

启动 Go 后端（开发态）：

```bash
cd web
npm run go:api
```

一键执行“启动 Go 后端 + 闭环验收”：

```bash
cd web
npm run smoke:go
```

以桥接模式执行闭环验收（ServiceLib bridge mock）：

```bash
cd web
npm run smoke:go:bridge
```

可选环境变量：

- `V2RAYN_API_ADDR`：Go 后端监听地址（默认 `127.0.0.1:18000`）
- `V2RAYN_API_TOKEN`：启用后端 Bearer 鉴权

在后端未就绪时，可先用本地 mock 后端一键验证闭环：

```bash
cd web
npm run smoke:mock
```

- 该命令会自动启动 `http://127.0.0.1:18000` mock API
- 自动执行写操作闭环验收（启停核心/切换节点/更新订阅/清理系统代理）
- 结束后自动停止 mock 服务

在后端可访问后，可执行 API 冒烟联调脚本：

```bash
cd web
V2RAYN_API_ORIGIN=http://127.0.0.1:18000 V2RAYN_API_TOKEN=<your-token> npm run smoke:api
```

- 默认只执行读接口（不改后端状态）
- 如需执行闭环写操作（启停核心/切换节点/更新订阅/清理系统代理）：

```bash
cd web
V2RAYN_API_ORIGIN=http://127.0.0.1:18000 V2RAYN_API_TOKEN=<your-token> V2RAYN_ENABLE_WRITE=1 npm run smoke:api
```

专项质量验收：

```bash
cd web
# 核心启停循环稳定性（默认 100 次，可通过 V2RAYN_LOOP_COUNT 调整）
npm run stability:core

# 鉴权拒绝校验（自动拉起带 token 的后端实例）
npm run security:auth

# 核心命令注入防护校验（验证 V2RAYN_CORE_CMD 不安全符号被拒绝）
npm run security:corecmd

# 服务重启后状态恢复校验
npm run stability:recovery

# 订阅更新异常韧性校验（失败更新不改变历史数据）
npm run stability:subscriptions

# 节点协议映射校验（覆盖 vless/trojan/shadowsocks/socks/http + 未知协议回退）
npm run stability:protocols

# native 脏状态清理校验（无效 PID + 残留临时文件）
npm run stability:state-cleanup
```

## 环境变量

- `NEXT_PUBLIC_API_BASE`：浏览器请求 API 前缀（默认 `/api`）
- `V2RAYN_BACKEND_URL`：后端地址（可选）。设置后由 Next.js 将 `/api/*` 转发到该地址。

可先复制：

```bash
cp .env.example .env.local
```

示例：

```bash
NEXT_PUBLIC_API_BASE=/api V2RAYN_BACKEND_URL=http://127.0.0.1:18000 npm run dev
```

## 已实现页面（MVP骨架）

- `/login`
- `/dashboard`
- `/profiles`
- `/subscriptions`
- `/network`
- `/settings`
- `/logs`

## 鉴权方式（当前）

- 登录页输入 Token 后写入 `auth_token` Cookie
- 前端请求自动附带 `Authorization: Bearer <token>`
- 中间件会拦截未登录访问并跳转到 `/login`
