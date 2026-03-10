# v2rayE 数据流架构图

## 1. 核心数据流总览

```mermaid
flowchart TB
    subgraph UserOps["用户操作数据流"]
        Start["启动核心<br/>选择配置<br/>更新订阅<br/>修改设置"]
    end
    
    subgraph API["HTTP API Server"]
        API_Server["HTTP API Server"]
        API_Endpoints["POST /api/core/start<br/>GET /api/profiles<br/>POST /api/profiles/select<br/>POST /api/subscriptions/update<br/>PUT /api/config<br/>PUT /api/routing"]
    end
    
    subgraph Backend["BackendService"]
        Native["Native Service"]
        
        subgraph NativeModules["Native Service 模块"]
            Profiles["Profiles Manager<br/>路由规则 Builder<br/>应用配置 Manager<br/>订阅源 Fetcher"]
            ConfigGen["Config Generator<br/>Profile + Config + Routing → Xray Config"]
            CoreMgr["Xray Core Manager<br/>进程启动/停止<br/>配置热加载<br/>健康检查"]
        end
    end
    
    subgraph StorageLayer["Storage Layer"]
        ProfilesJSON["profiles.json"]
        ConfigJSON["config.json"]
        RoutingJSON["routing.json"]
        StateJSON["state.json"]
        SubsJSON["subscriptions.json"]
        GeoData["GeoData<br/>(路由规则)"]
    end
    
    UserOps -->|"JSON/API"| API_Server
    API_Server --> API_Endpoints
    API_Endpoints --> Native
    Native --> BackendModules
    Native --> StorageLayer
    
    style UserOps fill:#e1f5fe
    style API fill:#bbdefb
    style Backend fill:#fff3e0
    style StorageLayer fill:#e8f5e9
```

## 2. 代理配置数据流

```mermaid
flowchart TB
    subgraph Input["1. 输入验证"]
        UserInput["用户输入<br/>Profile数据"]
        ProtoVal["协议验证<br/>(VMess等)"]
        AddrVal["地址验证<br/>(IP:Port)"]
        
        UserInput --> ProtoVal --> AddrVal
    end
    
    subgraph Storage["2. 存储持久化"]
        Store["Storage.Store<br/>JSON 序列化<br/>原子写入<br/>profiles.json 更新"]
    end
    
    subgraph ConfigGen["3. 核心配置生成"]
        Generate["generateXrayConfig()"]
        
        subgraph Sources["配置来源"]
            Profile["Profile<br/>(入站配置)"]
            AppConfig["App Config<br/>(端口/日志)"]
            Routing["Routing<br/>(规则)"]
        end
        
        XrayConfig["Xray JSON Config<br/>{log, api, inbounds, outbounds, routing}"]
        
        Profile --> Generate
        AppConfig --> Generate
        Routing --> Generate
        Generate --> XrayConfig
    end
    
    subgraph HotReload["4. 核心热重载"]
        Restart["Core.RestartCore()<br/>StopCore() → StartCore()"]
        WriteConfig["写入新配置文件<br/>启动 xray 进程"]
    end
    
    Input --> Storage --> ConfigGen --> HotReload
    
    style Input fill:#e3f2fd
    style Storage fill:#fff3e0
    style ConfigGen fill:#e8f5e9
    style HotReload fill:#fce4ec
```

## 3. 订阅数据流

```mermaid
flowchart TB
    subgraph Triggers["更新触发"]
        Manual["用户触发<br/>(手动更新)"]
        Timer["定时任务<br/>(AutoUpdate)"]
        Boot["启动时恢复<br/>(autoRun)"]
        
        Manual --> UpdateSub
        Timer --> UpdateSub
        Boot --> UpdateSub
    end
    
    subgraph Update["UpdateSubscriptionByID()"]
        Fetch["1. Fetch URL<br/>HTTP GET with User-Agent<br/>Base64 解码<br/>解析 URI 链接"]
        Filter["2. Filter (可选)<br/>按 filter 正则过滤节点<br/>按 convertTarget 转换协议"]
        Merge["3. 合并 Profiles<br/>删除旧订阅的 profiles<br/>添加新的 profiles<br/>分配新的 ProfileID"]
        Persist["4. 持久化存储<br/>Store.SaveProfiles()"]
        Event["5. 事件通知<br/>Server.publishEvent()"]
    end
    
    UpdateSub[UpdateSubscriptionByID()] --> Fetch --> Filter --> Merge --> Persist --> Event
    
    Event -.->|"→ SSE 推送"| Frontend["前端更新"]
```

## 4. 延迟测试数据流

### 单节点延迟测试

```mermaid
sequenceDiagram
    participant User as 用户
    participant API as HTTP API
    participant Service as Service
    participant TCP as TCP Dial
    
    User->>API: 触发测试 /profiles/{id}/delay
    API->>Service: TestProfile()
    Service->>TCP: 测速
    TCP-->>Service: delayMs
    Service-->>API: {available, delayMs, message}
    API-->>User: 返回结果
```

### 批量延迟测试

```mermaid
flowchart TB
    subgraph Batch["批量延迟测试"]
        Trigger["用户触发批量测试<br/>/profiles/delay/batch"]
        Service["Service.BatchTestProfileDelay"]
    end
    
    subgraph Concurrency["并发控制 (Semaphore)"]
        Limit["limit := 5 (默认)"]
        
        G1["goroutine 1"]
        G2["goroutine 2"]
        G3["goroutine 3"]
        G4["goroutine 4"]
        G5["goroutine 5"]
        
        Limit --> G1
        Limit --> G2
        Limit --> G3
        Limit --> G4
        Limit --> G5
    end
    
    subgraph Result["延迟结果排序"]
        Sort["可用节点按延迟升序<br/>不可用节点排在后面<br/>持久化到 Profile.DelayMs"]
    end
    
    Trigger --> Service --> Concurrency --> Result
```

## 5. 统计数据流

```mermaid
flowchart TB
    subgraph Collection["统计收集流程"]
        XrayCore["xray-core<br/>内部统计"]
        StatsAPI["Stats API<br/>(Port 10085)"]
        Polling["Polling<br/>(间隔 1s)"]
        
        XrayCore <-->|统计| StatsAPI
        StatsAPI <-->|轮询| Polling
    end
    
    subgraph Tracker["statsTracker"]
        LastStats["记录上一次的统计值<br/>(lastUp, lastDown)"]
        Calculate["计算速度<br/>(current - last) / interval"]
        GetStats["GetStats()<br/>{upBytes, downBytes<br/>upSpeed, downSpeed}"]
    end
    
    Polling --> Tracker
    
    style Collection fill:#e3f2fd
    style Tracker fill:#fff3e0
```

### 路由命中统计

```mermaid
flowchart LR
    XrayOut["xray-core outbound<br/>流量统计"] <-->|按tag| StatsAPI["Stats API"]
    StatsAPI <-->|分组求和| Aggregate["聚合"]
    Aggregate -->|"[{outbound, upBytes, downBytes}]"| RoutingHits["RoutingHitStats"]
```

## 6. 日志数据流

```mermaid
flowchart TB
    subgraph Collection["日志收集流程"]
        Xray["xray-core 进程"]
        Stdout["Stdout Pipe"]
        LogBroker["Log Broker<br/>(内存)"]
        
        Xray -->|"stdout/stderr"| Stdout -->|"读取"| LogBroker
    end
    
    subgraph Subscribers["订阅者 (Subscribers)"]
        SSE["SSE /logs/stream<br/>(实时推送)"]
        TUI["TUI Log View<br/>(实时显示)"]
        Buffer["内部缓冲<br/>(Ring Buffer)"]
    end
    
    LogBroker -->|"subscribe"| SSE
    LogBroker -->|"subscribe"| TUI
    LogBroker -->|"subscribe"| Buffer
    
    subgraph Filter["日志级别过滤"]
        Levels["debug | info | warning | error"]
        UserFilter["用户设置日志级别 Filter<br/>仅显示 >= 选定级别"]
    end
    
    SSE --> Filter
```

```mermaid
flowchart LR
    subgraph LogLevels["日志级别"]
        Debug["debug"]
        Info["info"]
        Warning["warning"]
        Error["error"]
    end
    
    UserFilter -->|"过滤"| Visible["可见日志"]