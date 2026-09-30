# 强制某域名走代理（不改代码）

日期：2026-08-26

背景：`ttpa.example.com` 托管在 AWS CloudFront（`203.0.113.x`，德国法兰克福，非中国大陆）。`bypass_cn` 模式下它默认会被当作国外流量走代理，但这依赖核心默认判定，不是"固定强制"。本文记录如何把任意域名固定为强制走代理（走 `proxy` 出口），以及如何验证确实经过代理。

## 操作：通过 API 写入自定义路由规则

不要直接编辑 `/opt/v2rayE/routing.json`——后端存储层有内存缓存（见 `backend-go/internal/storage/store.go`），直接改文件会造成内存与磁盘不一致。正确方式是通过 API `PUT /api/routing` 写入，核心会自动重新生成 xray 配置并重启。

```bash
curl -s -X PUT http://127.0.0.1:18000/api/routing \
  -H 'Content-Type: application/json' \
  -d '{
    "mode": "bypass_cn",
    "domainStrategy": "IPOnDemand",
    "localBypassEnabled": true,
    "rules": [
      {"id": "r-example-forced-proxy", "type": "domain", "values": ["ttpa.example.com"], "outbound": "proxy"}
    ]
  }'
```

要点：

- `rules[]` schema：`{id, type, values[], outbound}`。`type` 支持 `domain|ip|geoip|geosite|port|protocol`；`outbound` 为 `proxy|direct|block`。
- 生成到 xray 配置的映射在 `backend-go/internal/service/native/config_gen.go:637`（自定义规则追加到内置规则之后、默认规则之前，因此优先级高于 bypass_cn 的直连规则）。
- 响应 `code:0` 即成功；随后可确认落盘与生成配置。

## 确认已生效

1. 落盘：`sudo cat /opt/v2rayE/routing.json` 应包含该规则。
2. 生成配置：`sudo cat /opt/v2rayE/runtime-config.json` 的 `routing.rules` 应出现 `{"domain":["ttpa.example.com"],"outboundTag":"proxy"}`。
3. 路由模拟（配置层面证明）：

```bash
curl -s -X POST http://127.0.0.1:18000/api/routing/test \
  -H 'Content-Type: application/json' \
  -d '{"target":"ttpa.example.com","protocol":"tls"}'
# → matchedRule: field-N, outbound: proxy
```

## 验证确实"经过代理"（运行层面）

本机直连出口 IP 恰好等于代理节点 IP（本机物理出口 = 代理节点 `198.51.100.10` 时），"出口 IP 对比法"无法区分，必须用 TCP 连接目标对比：

```bash
# 抓包 A：直连访问（--resolve 强制绑定解析 IP）
sudo tcpdump -i any -n -nn 'tcp and (dst port 443 or dst port 9443)' -w /tmp/direct.pcap &
curl -s --max-time 8 -o /dev/null --resolve ttpa.example.com:443:203.0.113.47 https://ttpa.example.com/
sleep 1; kill %1

# 抓包 B：走代理访问
sudo tcpdump -i any -n -nn 'tcp and (dst port 443 or dst port 9443)' -w /tmp/proxy.pcap &
curl -s --max-time 8 -o /dev/null --socks5-hostname 127.0.0.1:10808 https://ttpa.example.com/
sleep 1; kill %1

# 对比连接目标
sudo tcpdump -r /tmp/direct.pcap -n -nn | grep -oE '[0-9.]+ > [0-9.]+\.(443|9443)' | sort | uniq -c
sudo tcpdump -r /tmp/proxy.pcap  -n -nn | grep -oE '[0-9.]+ > [0-9.]+\.(443|9443)' | sort | uniq -c
```

判断标准：

| 访问方式 | 本机 → CloudFront(ttpa IP):443 | 本机 → 代理节点:9443 |
|---|---|---|
| 直连 | ✅ 出现 | 有（并发噪音） |
| 走代理 | ❌ 无 | ✅ 全部 |

直连时本机会直接连 CloudFront IP；走代理时本机只连代理节点，流量被封装进代理连接，由节点代发。

## 备注

- 该规则按域名匹配（核心内 DNS 解析），只要解析结果非中国大陆即走代理。
- 抓包窗口内可能出现其它并发流量（到任意 IP 的 443/9443），需与 ttpa 目标 IP 区分。
