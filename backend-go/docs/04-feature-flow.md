# v2rayE 功能流程图

## 1. 核心启动流程

```mermaid
flowchart TB
    subgraph Triggers["启动触发源"]
        CLI["命令行启动<br/>v2raye"]
        Systemd["Systemd<br/>start"]
        TUITrigger["TUI 触发<br/>StartCore"]
        APITrigger["API 触发<br/>POST /api"]
        
        CLI --> StartCore
        Systemd --> StartCore
        TUITrigger --> StartCore
        APITrigger --> StartCore
    end
    
    subgraph StartCore["NativeService.StartCore()"]
        Step1["1. 加载配置<br/>Store.LoadConfig()<br/>Store.LoadRoutingConfig()<br/>Store.LoadState()"]
        Step2["2. 选择配置<br/>PickSelectedProfile()<br/>优先 state.CurrentProfileID<br/>回退到第一个 Profile"]
        Step3["3. TUN 模式预处理<br/>清理旧的 TUN 策略路由<br/>清理旧的 TUN 接口<br/>检测默认路由网卡"]
        Step4["4. 生成 Xray 配置<br/>generateXrayConfig(profile, config, routing)<br/>Profile → Inbound + Outbound<br/>App Config → 日志、API、统计<br/>Routing Config → 路由规则"]
        Step5["5. 写入配置文件<br/>writeConfigToFile(data, dataDir)<br/>/opt/v2rayE/xray.json"]
        Step6["6. 启动 xray-core 进程<br/>startManagedXrayCore(config, logs)<br/>exec.Command(xray, -c, configPath)<br/>设置 stdout/stderr pipe"]
        Step7["7. 启动统计收集<br/>newStatsTracker(statsPort)<br/>启动 HTTP 轮询<br/>定时收集带宽统计"]
        Step8["8. TUN 路由设置 (可选)<br/>setupManagedTunRouting()<br/>策略路由 或 默认路由接管"]
        Step9["9. 应用系统代理 (可选)<br/>applyDesktopSystemProxy()<br/>GNOME (gsettings) 或 KDE (kwriteconfig)"]
        Step10["10. 持久化状态<br/>Store.SaveState(CurrentProfileID, CoreShouldRestore=true)"]
    end
    
    Triggers --> StartCore
    Step1 --> Step2 --> Step3 --> Step4 --> Step5 --> Step6 --> Step7 --> Step8 --> Step9 --> Step10
    
    subgraph Result["返回 CoreStatus"]
        Status["{ running: true, state: running, currentProfileId: xxx, ... }"]
    end
    
    Step10 --> Status
```

## 2. 代理配置创建流程

```mermaid
flowchart TB
    subgraph Input["用户输入"]
        Form["TUI 表单 / API POST /api/profiles"]
        Data["{name, protocol, address, port, vmess/..., transport}"]
    end
    
    subgraph Process["服务层处理"]
        Validate["1. 输入验证<br/>validateProfile(input)<br/>检查 name/address 非空<br/>检查 port > 0<br/>检查 protocol 有效"]
        GenerateID["2. 生成 ID<br/>input.ID = newProfileID()"]
        Load["3. 加载现有数据<br/>Store.LoadProfiles()"]
        Add["4. 添加新配置<br/>profiles = append(profiles, input)"]
        Save["5. 持久化存储<br/>Store.SaveProfiles(profiles)<br/>JSON 序列化<br/>原子写入 (tmp + rename)"]
    end
    
    subgraph Event["事件通知"]
        Publish["Server.publishEvent(profile.updated, {id, created})"]
        SSE["SSE /api/events/stream → TUI 更新"]
    end
    
    Input --> Process
    Process --> Validate --> GenerateID --> Load --> Add --> Save --> Publish
    Publish --> SSE
    
    style Input fill:#e3f2fd
    style Process fill:#fff3e0
    style Event fill:#e8f5e9
```

## 3. 订阅更新流程

```mermaid
flowchart TB
    subgraph Triggers["更新触发"]
        Manual["手动触发<br/>TUI 点击更新按钮"]
        Timer["定时任务<br/>autoUpdate Minutes"]
        Boot["启动恢复<br/>autoRun true"]
        API["API 触发<br/>POST /sub/update"]
        
        Manual --> Update
        Timer --> Update
        Boot --> Update
        API --> Update
    end
    
    subgraph Update["UpdateSubscriptionByID()"]
        GetSub["1. 获取订阅信息<br/>Store.GetSubscription(id)"]
        Fetch["2. 获取远程内容<br/>http.Get(url)<br/>自定义 User-Agent<br/>支持订阅链接/Base64"]
        Parse["3. 解析节点列表<br/>ParseSubscriptionURL(url, userAgent, subID)<br/>Base64 解码<br/>解析节点配置<br/>分配 SubID, SubName"]
        Filter["4. 过滤 (可选)<br/>按正则过滤节点名称<br/>转换协议 (vmess → vless)"]
        Merge["5. 合并 Profiles<br/>移除该订阅的旧节点<br/>添加新的节点<br/>Store.SaveProfiles(merged)"]
        UpdateTime["6. 更新订阅时间<br/>sub.UpdatedAt = time.Now()<br/>Store.SaveSubscriptions(subs)"]
    end
    
    subgraph Event["事件通知"]
        Publish["Server.publishEvent(subscription.updated, {id, updated})"]
    end
    
    Triggers --> Update
    Update --> GetSub --> Fetch --> Parse --> Filter --> Merge --> UpdateTime --> Publish
```

## 4. TUN 模式路由设置流程

```mermaid
flowchart TB
    subgraph Decision["TUN 路由设置决策"]
        Check["shouldHijackTunDefaultRoute(config)"]
        
        Check -->|"true"| DefaultRoute["默认路由接管"]
        Check -->|"false"| PolicyRoute["策略路由 (推荐)"]
    end
    
    subgraph Method1["方式1: 默认路由接管"]
        GetDefault["1. 获取当前默认路由<br/>ip route show default"]
        SaveRestore["2. 保存恢复路由<br/>记录 dev, via<br/>持久化到 config.json"]
        WaitTUN["3. 等待 TUN 接口出现<br/>waitForNetworkInterface"]
        Replace["4. 替换默认路由<br/>ip route replace default dev tun0"]
        Persist["5. 持久化恢复信息"]
    end
    
    subgraph Method2["方式2: 策略路由 (推荐)"]
        GetLocal["1. 获取本地网络路由<br/>ip route show table main"]
        BypassRules["2. 构建绕过规则<br/>本地网络<br/>DNS 服务器<br/>代理服务器地址"]
        PolicyTable["3. 策略路由表<br/>table 20230<br/>default via TUN dev"]
        FWMark["4. fwmark 标记规则<br/>fwmark 0x2d11 → lookup main"]
        PolicyRules["5. 策略规则 (IPv4/IPv6)<br/>优先级 10000-10999<br/>to 本地网络 → lookup main<br/>fwmark → lookup main<br/>default → lookup 20230"]
        Cleanup["6. 清理旧规则"]
    end
    
    subgraph Cleanup["TUN 清理流程"]
        ClearRules["清除策略路由规则<br/>ip rule del"]
        ClearTable["清除策略路由表<br/>ip route del table 20230"]
        RestoreDefault["恢复默认路由<br/>ip route replace default ..."]
        DelTUN["删除 TUN 设备<br/>ip link del dev xraye0"]
        Flush["刷新路由缓存<br/>ip route flush cache"]
    end
    
    DefaultRoute --> Method1
    PolicyRoute --> Method2
    Method1 -.-> Cleanup
    Method2 -.-> Cleanup
    
    style Decision fill:#fff9c4
    style Method1 fill:#e3f2fd
    style Method2 fill:#e8f5e9
    style Cleanup fill:#ffcdd2
```

## 5. 系统代理集成流程

```mermaid
flowchart TB
    subgraph Desktop["支持的桌面环境"]
        GNOME["GNOME<br/>(gsettings)"]
        KDE["KDE<br/>(kwriteconfig5/6)"]
        Other["其他<br/>(环境变量)"]
    end
    
    subgraph Apply["ApplySystemProxy(mode, exceptions)"]
        Mode["mode: forced_change<br/>forced_clear<br/>pac"]
        
        ForcedChange["forced_change:<br/>检测桌面环境<br/>设置代理<br/>设置环境变量"]
        ForcedClear["forced_clear:<br/>清除代理设置<br/>清除环境变量"]
        PAC["pac: (未实现)"]
    end
    
    subgraph GNOME_Set["GNOME (gsettings)"]
        GMode["gsettings set org.gnome.system.proxy mode manual"]
        GHTTP["gsettings set org.gnome.system.proxy.http host $listenAddr"]
        GPort["gsettings set org.gnome.system.proxy.http port $httpPort"]
        GException["gsettings set org.gnome.system.proxy ignore-hosts [$exceptions]"]
    end
    
    subgraph KDE_Set["KDE (kwriteconfig5/6)"]
        KType["kwriteconfig6 kioslaverc ProxySettings ProxyType 1"]
        KHTTP["kwriteconfig6 kioslaverc ProxySettings httpProxy http://$host:$port"]
        KHTTPS["kwriteconfig6 kioslaverc ProxySettings httpsProxy http://$host:$port"]
        KReload["dbus-send (reload)"]
    end
    
    subgraph Auth["权限处理 (以 root 运行时)"]
        Sudo["使用 sudo 切换到桌面用户"]
        Env["设置 HOME, XDG_RUNTIME_DIR<br/>DBUS_SESSION_BUS_ADDRESS"]
        Execute["执行 gsettings / kwriteconfig"]
    end
    
    Desktop --> Apply
    Apply --> ForcedChange
    Apply --> ForcedClear
    
    ForcedChange --> GNOME_Set
    ForcedChange --> KDE_Set
    
    GNOME_Set --> Auth
    KDE_Set --> Auth
    
    style Desktop fill:#e1f5fe
    style Apply fill:#fff3e0
    style GNOME_Set fill:#c8e6c9
    style KDE_Set fill:#bbdefb
    style Auth fill:#ffccbc
```

## 6. 健康检查与自动恢复流程

```mermaid
flowchart TB
    subgraph Watchdog["看门狗循环 (watchdogLoop)"]
        Ticker["ticker: 每秒触发一次"]
        
        CheckProcess["1. 检查进程状态<br/>xrayCore.IsRunning()"]
        
        CheckProcess -->|"运行中"| Continue["正常运行, 继续"]
        CheckProcess -->|"已退出"| Recovery["进程已退出, 进入恢复流程"]
        
        NeedRestart["2. 检查是否需要重启<br/>coreAutoRestart = true?<br/>restartAttempts < maxRetries?"]
        
        NeedRestart -->|"不需要"| Stop["停止自动恢复<br/>记录错误日志"]
        NeedRestart -->|"需要"| Schedule["3. 计划自动重启<br/>计算退避延迟 (backoff)<br/>500ms → 1s → 2s → 4s<br/>最大30s<br/>scheduleAutoRestart(delay)"]
    end
    
    subgraph AutoRestart["自动重启执行 (runScheduledAutoRestart)"]
        Wait["1. 等待延迟结束<br/>time.Sleep(delay)"]
        
        CheckAgain["2. 检查是否仍需要重启<br/>如果 core 已在运行 → 跳过<br/>如果已计划其他重启 → 跳过"]
        
        Execute["3. 执行重启<br/>StartCore()"]
        
        HandleResult["4. 处理结果<br/>成功: 重置 restartAttempts<br/>失败: 计划下一次重启"]
    end
    
    subgraph Repair["TUN 自动修复 (RepairTunAndRestart)"]
        CleanTUN["清理旧的 TUN 策略路由"]
        CleanDev["清理旧的 TUN 设备"]
        RestartCore["重新启动核心"]
        SetupTUN["重新设置 TUN 路由"]
        ReturnDiag["返回诊断信息"]
    end
    
    Watchdog --> Ticker --> CheckProcess
    Recovery --> NeedRestart --> Schedule
    Schedule --> AutoRestart
    AutoRestart --> Wait --> CheckAgain --> Execute --> HandleResult
    Repair --> CleanTUN --> CleanDev --> RestartCore --> SetupTUN --> ReturnDiag
    
    style Watchdog fill:#fff9c4
    style AutoRestart fill:#e3f2fd
    style Repair fill:#ffcdd2
```

### 退避策略详解

```mermaid
flowchart LR
    subgraph Backoff["指数退避 (Exponential Backoff)"]
        B1["500ms"] --> B2["1s"] --> B3["2s"] --> B4["4s"] --> B5["8s"] --> B6["16s"] --> B7["30s (max)"]
    end
    
    Max["最大重试次数: 5"]
    Reset["成功后重置计数器"]