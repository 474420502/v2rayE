# v2rayE 系统架构图

## 1. 整体架构概览

```mermaid
flowchart TB
    subgraph UI["用户交互层 (User Interface)"]
        direction TB
        TUI["TUI 终端界面<br/>(cmd/tui)"]
        API["HTTP API 客户端<br/>(外部调用)"]
        
        TUI --- TUI_Features["- 命令行界面<br/>- 交互式操作<br/>- 实时状态显示"]
        API --- API_Features["- RESTful API<br/>- SSE 事件流<br/>- WebSocket"]
    end

    subgraph Service["服务层 (Service Layer)"]
        direction TB
        HTTP["HTTP API Server<br/>(internal/httpapi)"]
        HTTP --- HTTP_Features["- RESTful API 端点<br/>- 认证与授权<br/>- SSE 事件发布<br/>- 请求验证与错误处理"]
        
        Backend["BackendService 接口<br/>(internal/service)"]
        
        Native["Native Service<br/>(internal/service/native)"]
        Native --- Native_Features["- 核心进程管理 (xray-core)<br/>- 配置文件生成<br/>- TUN 模式路由管理<br/>- 系统代理集成<br/>- 订阅解析<br/>- 延迟测试<br/>- 统计收集"]
    end

    subgraph Data["数据层 (Data Layer)"]
        Storage["Storage<br/>(internal/storage)"]
        Storage --- Storage_Features["- JSON 文件持久化<br/>- profiles.json<br/>- subscriptions.json<br/>- config.json<br/>- routing.json<br/>- state.json"]
    end

    subgraph External["底层依赖 (External Dependencies)"]
        Xray["xray-core<br/>(代理核心)"]
        Proxy["系统代理<br/>(gsettings/kwrite)"]
        Geo["GeoData<br/>(路由规则)"]
        Net["网络<br/>连接"]
    end

    UI --> Service
    Service --> Data
    Service --> External
```

## 2. 核心模块架构

### 2.1 命令行入口 (cmd)

```mermaid
flowchart TB
    subgraph CMD["命令入口 (cmd)"]
        direction LR
        TUI["cmd/tui<br/>终端用户界面"] 
        API["cmd/backend-api<br/>HTTP API 服务"]
        MAIN["cmd/v2raye<br/>主入口程序"]
        
        TUI --- TUI_Features["Dashboard<br/>Profiles<br/>Subscriptions<br/>Settings<br/>Logs<br/>Network"]
        API --- API_Features["REST API<br/>事件流<br/>认证"]
        MAIN --- MAIN_Features["服务初始化<br/>配置加载<br/>信号处理"]
    end
```

### 2.2 内部包结构 (internal)

```mermaid
flowchart TB
    subgraph Internal["内部包结构 (internal)"]
        direction LR
        
        Domain["domain<br/>数据类型定义"]
        StoragePkg["storage<br/>JSON 存储"]
        HttpApi["httpapi<br/>HTTP 服务器"]
        ServicePkg["service<br/>服务接口定义"]
        Launcher["launcher<br/>服务启动器"]
        
        Domain --- Domain_Types["Profile<br/>Subscription<br/>Routing<br/>CoreStatus<br/>Stats"]
        StoragePkg --- Storage_Features["profiles<br/>config<br/>routing<br/>state<br/>原子写入"]
        HttpApi --- HttpApi_Features["API 路由<br/>认证中间件<br/>SSE<br/>错误处理"]
        
        ServicePkg --> Native["service/native<br/>核心实现"]
        
        Native --- Native_Modules["xray_core.go - 核心进程管理<br/>config_gen.go - 配置生成器<br/>sub_parse.go - 订阅解析<br/>stats.go - 流量统计<br/>geodata_update.go - 地理数据更新<br/>log_stream.go - 日志流"]
        Native --- Native_Features["TUN 模式路由管理<br/>- Policy Routing<br/>- fwmark 标记<br/>- 自动路由恢复<br/><br/>系统代理集成<br/>- GNOME (gsettings)<br/>- KDE (kwriteconfig5/6)<br/>- 环境变量 (http_proxy)"]
    end
```

## 3. 代理协议支持

```mermaid
mindmap
  root((代理协议))
    VMess
      UUID
      AlterID
      Security
    VLESS
      UUID
      Flow
      Encryption
    Shadowsocks
      Method
      Password
      Plugin
    Trojan
      Password
    Hysteria2
      Password
      SNI
      Obfs
      Up/Down Mbps
    TUIC
      UUID
      Password
      Congestion
      ALPN
    传输层
      TCP
      WebSocket
      gRPC
      HTTP/2
      KCP
      QUIC
      XHTTP
      TLS配置
        SNI
        Fingerprint
        ALPN
        SkipCertVerify
      Reality配置
        PublicKey
        ShortID
```

## 4. 数据流方向

```mermaid
flowchart LR
    User["用户操作<br/>TUI"] -->|JSON/API| API["HTTP API Server"]
    API -->|调用/进程| Service["BackendService<br/>Native Service"]
    Service -->|读写| Storage["Storage<br/>JSON 文件"]
    Service -->|管理| Xray["xray-core 进程"]
    
    style User fill:#e1f5fe
    style API fill:#e1f5fe
    style Service fill:#fff3e0
    style Storage fill:#e8f5e9
    style Xray fill:#fce4ec
```

## 5. 事件流架构

```mermaid
flowchart TB
    subgraph EventBus["事件总线"]
        Service["Backend Service"]
        Server["HTTP API Server"]
        
        Service <-->|publishEvent()| Server
    end
    
    subgraph SSE["SSE 事件类型"]
        CoreEvents["core.started<br/>core.stopped<br/>core.restarted<br/>core.start_failed<br/>core.error_cleared"]
        ProfileEvents["profile.updated<br/>profile.selected"]
        SubEvents["subscription.updated"]
        RoutingEvents["routing.updated<br/>routing.tested<br/>routing.tun_repaired"]
        ConfigEvents["config.updated<br/>proxy.changed<br/>app.exit_cleanup"]
        LogEvents["log (实时日志)"]
    end
    
    Server --> CoreEvents
    Server --> ProfileEvents
    Server --> SubEvents
    Server --> RoutingEvents
    Server --> ConfigEvents
    Server --> LogEvents
```

## 6. TUN 模式架构

```mermaid
flowchart TB
    subgraph Traditional["传统模式"]
        App1["应用"] -->|"SOCKS/HTTP"| ProxySet["系统代理设置"]
        ProxySet --> Xray1["xray-core<br/>Inbound SOCKS/HTTP"]
        Xray1 --> Router1["Router 规则匹配"]
        Router1 --> Out1["Outbound 代理出站"]
    end
    
    subgraph TUN_Mode["TUN 模式"]
        App2["应用"] -->|"所有 T流量"|UN["TUN 接口<br/>(xraye0)"]
        TUN --> Xray2["xray-core"]
        
        subgraph Routing["TUN 路由策略"]
            DirRoute["1. 默认路由接管<br/>(tunHijackDefaultRoute)"]
            PolicyRoute["2. 策略路由 (推荐)<br/>(Policy Routing)"]
        end
        
        Xray2 --> Routing
    end
    
    Routing --> Router2["Router 规则匹配"]
    Router2 --> Out2["Outbound 代理出站"]
```

### TUN 路由策略详解

```mermaid
flowchart LR
    subgraph Strategy["TUN 路由策略"]
        Direction1["默认路由接管"] -->|"替换系统默认路由"| AllTraffic["所有流量进入TUN"]
        Direction2["策略路由"] -->|"fwmark标记"| FineGrained["更精细的流量控制<br/>绕过本地网络/局域网"]
    end
```

### 系统代理集成

```mermaid
flowchart LR
    subgraph Desktop["桌面环境"]
        GNOME["GNOME<br/>(gsettings)"]
        KDE["KDE<br/>(kwriteconfig5/6)"]
        Env["环境变量<br/>(http_proxy)"]
    end
    
    GNOME --> Proxy["系统代理设置"]
    KDE --> Proxy
    Env --> Proxy