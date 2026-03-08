# TUN模式与GeoIP问题修复设计

## 问题背景

用户反馈：TUN模式下，要么代理不可用，要么网络不可用。geoip/geodata 可能是主要原因。

## 根本原因分析

### 1. GeoIP/GeoSite 路由规则问题
- 当 `routing.mode=bypass_cn` 时，依赖 `geoip:cn` 和 `geosite:cn` 数据
- 问题：
  - GeoIP 数据可能不完整或过时
  - `geoip:private` 在某些环境不生效
  - 路由规则优先级问题导致流量走向错误

### 2. TUN 模式路由问题
- Linux 上 `autoRoute` 被禁用，使用策略路由
- 策略路由 (`setupTunPolicyRouting`) 实现复杂，容易失效
- DNS 路由设置可能覆盖系统 DNS，导致网络不可用

### 3. GeoData 加载问题
- 依赖 GitHub 下载，网络问题可能导致失败
- 加载路径搜索目录可能不包含实际数据文件位置

## 修复方案

### 方案1: 改进 GeoData 回退逻辑
- 当 GeoIP/GeoSite 数据缺失或加载失败时，使用更可靠的 CIDR 列表
- 增加内建的 cn IP 段数据作为备用
- 路由规则增加更明确的优先级控制

### 方案2: 简化 TUN 路由策略
- 提供更稳定的 TUN 路由模式选项
- 改进策略路由的清理和恢复逻辑
- 增加 TUN 模式的诊断和日志

### 方案3: 增加 GeoData 预加载和缓存
- 在启动时确保 GeoData 可用
- 增加更可靠的下载源
- 添加 GeoData 完整性验证

## 实施计划

1. 在 `config_gen.go` 中增加内建 cn CIDR 回退
2. 改进 TUN 策略路由的错误处理和日志
3. 增加 GeoData 状态诊断接口
4. 添加更保守的 TUN 路由选项
