# v2rayE 模块依赖关系图

## 1. 整体模块依赖关系

```mermaid
flowchart TB
    Main["main.go<br/>(入口点)"]
    
    subgraph Commands["命令入口"]
        TUI["cmd/tui<br/>TUI 界面"]
        BackendAPI["cmd/backend-api<br/>HTTP API"]
        V2rayE["cmd/v2raye<br/>主程序"]
    end
    
    Launcher["launcher<br/>(服务启动器)"]
    
    subgraph Core["核心模块"]
        HttpApi["httpapi<br/>HTTP 服务"]
        Service["service<br/>业务逻辑"]
        Storage["storage<br/>数据持久化"]
    end
    
    subgraph Internals["内部实现"]
        Domain["domain<br/>(数据类型)"]
        Native["native<br/>(xray-core)"]
    end
    
    Main --> Commands
    Commands --> Launcher
    Launcher --> HttpApi
    Launcher --> Service
    Launcher --> Storage
    HttpApi --> Service
    Service --> Storage
    Service --> Domain
    Service --> Native
    
    style Main fill:#ffcdd2
    style Commands fill:#e1f5fe
    style Launcher fill:#fff3e0
    style Core fill:#e8f5e9
    style Internals fill:#f3e5f5
```

## 2. 详细模块依赖

```mermaid
flowchart TB
    subgraph TUI["cmd/tui"]
        TUI_Main["tui/main.go"]
        TView["tview<br/>(第三方 UI)"]
        APIClient["apiClient<br/>(HTTP 客户端)"]
        Components["components/*"]
        State["state.go"]
    end
    
    subgraph BackendAPI["cmd/backend-api"]
        BackendMain["main.go"]
        Launch["launcher.RunServer()"]
    end
    
    subgraph Launcher["launcher"]
        Server["launcher/server.go"]
    end
    
    subgraph HttpAPI["httpapi"]
        HTTPServer["server.go"]
    end
    
    subgraph Service["service/native"]
        NativeService["service.go"]
        XrayCore["xray_core.go"]
        ConfigGen["config_gen.go"]
        SubParse["sub_parse.go"]
        Stats["stats.go"]
    end
    
    subgraph Storage["storage"]
        Store["store.go"]
    end
    
    subgraph Domain["domain"]
        Types["types.go"]
    end
    
    TUI_Main --> TView
    TUI_Main --> APIClient
    TUI_Main --> Components
    TUI_Main --> State
    
    BackendMain --> Launch
    Launch --> HttpAPI
    Launch --> Service
    Launch --> Storage
    
    HTTPServer --> Service
    NativeService --> Storage
    NativeService --> Domain
    NativeService --> XrayCore
    NativeService --> ConfigGen
    NativeService --> SubParse
    NativeService --> Stats
    
    Store --> Domain
    
    style TUI fill:#e3f2fd
    style BackendAPI fill:#e3f2fd
    style Launcher fill:#fff3e0
    style HttpAPI fill:#bbdefb
    style Service fill:#c8e6c9
    style Storage fill:#d7ccc8
    style Domain fill:#f3e5f5
```

## 3. API 端点依赖关系

```mermaid
flowchart TB
    subgraph CoreAPI["Core 管理接口"]
        CoreStatus["GET /api/core/status<br/>NativeService.CoreStatus()"]
        CoreStart["POST /api/core/start<br/>NativeService.StartCore()"]
        CoreStop["POST /api/core/stop<br/>NativeService.StopCore()"]
        CoreRestart["POST /api/core/restart<br/>NativeService.RestartCore()"]
    end
    
    subgraph ProfileAPI["Profile 管理接口"]
        ListProfiles["GET /api/profiles<br/>ListProfiles()"]
        CreateProfile["POST /api/profiles<br/>CreateProfile()"]
        GetProfile["GET /api/profiles/{id}<br/>GetProfile()"]
        UpdateProfile["PUT /api/profiles/{id}<br/>UpdateProfile()"]
        DeleteProfile["DELETE /api/profiles/{id}<br/>DeleteProfile()"]
        SelectProfile["POST /api/profiles/{id}/select<br/>SelectProfile()"]
        TestDelay["GET /api/profiles/{id}/delay<br/>TestProfileDelay()"]
    end
    
    subgraph SubAPI["Subscription 管理接口"]
        ListSubs["GET /api/subscriptions<br/>ListSubscriptions()"]
        CreateSub["POST /api/subscriptions<br/>CreateSubscription()"]
        UpdateSub["PUT /api/subscriptions/{id}<br/>UpdateSubscription()"]
        DeleteSub["DELETE /api/subscriptions/{id}<br/>DeleteSubscription()"]
        UpdateSubs["POST /api/subscriptions/update<br/>UpdateSubscriptions()"]
    end
    
    subgraph ConfigAPI["Config 管理接口"]
        GetConfig["GET /api/config<br/>GetConfig()"]
        UpdateConfig["PUT /api/config<br/>UpdateConfig()"]
    end
    
    subgraph RoutingAPI["Routing 管理接口"]
        GetRouting["GET /api/routing<br/>GetRoutingConfig()"]
        UpdateRouting["PUT /api/routing<br/>UpdateRoutingConfig()"]
        TestRouting["POST /api/routing/test<br/>TestRouting()"]
        RepairTun["POST /api/routing/tun/repair<br/>RepairTunAndRestart()"]
    end
    
    subgraph StatsAPI["统计与日志接口"]
        GetStats["GET /api/stats<br/>GetStats()"]
        LogStream["GET /api/logs/stream<br/>SubscribeCoreLogs()"]
        EventStream["GET /api/events/stream<br/>SSE Events"]
    end
    
    CoreAPI --> Service["NativeService"]
    ProfileAPI --> Service
    SubAPI --> Service
    ConfigAPI --> Service
    RoutingAPI --> Service
    StatsAPI --> Service
```

## 4. 数据文件依赖关系

```mermaid
flowchart TB
    subgraph DataFiles["数据文件"]
        ProfilesJSON["profiles.json"]
        SubsJSON["subscriptions.json"]
        ConfigJSON["config.json"]
        RoutingJSON["routing.json"]
        StateJSON["state.json"]
        GeoData["GeoData<br/>(外部文件)"]
        XrayJSON["xray.json<br/>(运行时生成)"]
    end
    
    subgraph Modules["使用/写入模块"]
        ProfileModule["NativeService<br/>List/Create/Update/Delete<br/>ImportProfileFromURI"]
        SubModule["NativeService<br/>List/Create/Update/Delete<br/>UpdateSubscriptionByID"]
        ConfigModule["NativeService<br/>Get/UpdateConfig<br/>Storage.DefaultConfig"]
        RoutingModule["NativeService<br/>Get/UpdateRoutingConfig<br/>Storage.DefaultRoutingConfig"]
        StateModule["NativeService<br/>CoreStatus/Start/Stop<br/>SelectProfile<br/>Launcher.RunServer"]
        GeoModule["NativeService<br/>GetRoutingDiagnostics<br/>UpdateRoutingGeoData<br/>buildRoutingRules"]
        XrayModule["NativeService<br/>StartCore<br/>Profile+Config+Routing→XrayConfig"]
    end
    
    ProfilesJSON <--> ProfileModule
    SubsJSON <--> SubModule
    ConfigJSON <--> ConfigModule
    RoutingJSON <--> RoutingModule
    StateJSON <--> StateModule
    GeoData <--> GeoModule
    XrayJSON <--> XrayModule
    
    style DataFiles fill:#fff9c4
    style Modules fill:#c8e6c9
```

## 5. 第三方依赖

```mermaid
flowchart LR
    subgraph Direct["直接依赖 (go.mod)"]
        TView["github.com/rivo/tview<br/>TUI 终端界面框架"]
        TCell["github.com/gdamore/tcell/v2<br/>TUI 底层终端渲染"]
        Runewidth["github.com/mattn/go-runewidth<br/>文本宽度计算"]
        Uniseg["github.com/rivo/uniseg<br/>Unicode 字符宽度"]
    end
    
    subgraph Runtime["系统依赖 (运行时)"]
        Xray["xray / xray-core<br/>代理核心进程"]
        IP["ip (iproute2)<br/>网络接口/路由管理"]
        GSettings["gsettings (Gnome)<br/>GNOME 桌面代理设置"]
        KWrite["kwriteconfig5/6 (KDE)<br/>KDE 桌面代理设置"]
        DBus["dbus-send (D-Bus)<br/>KDE 桌面通知"]
        Loginctl["loginctl (systemd)<br/>查询登录用户"]
    end
    
    subgraph Std["标准库依赖"]
        NetHTTP["net/http<br/>HTTP 客户端/服务器"]
        JSON["encoding/json<br/>JSON 编解码"]
        Exec["os/exec<br/>进程管理"]
        User["os/user<br/>用户信息查询"]
        Net["net<br/>网络连接"]
        Time["time<br/>定时器/时间"]
        Sync["sync<br/>并发原语"]
        Context["context<br/>上下文控制"]
        Log["log<br/>日志"]
    end
    
    style Direct fill:#e1f5fe
    style Runtime fill:#ffccbc
    style Std fill:#d1c4e9