# Graph Report - LiveKit  (2026-08-21)

## Corpus Check
- Corpus is ~7,635 words - fits in a single context window. You may not need a graph.

## Summary
- 182 nodes · 308 edges · 12 communities (11 shown, 1 thin omitted)
- Extraction: 91% EXTRACTED · 9% INFERRED · 0% AMBIGUOUS · INFERRED: 28 edges (avg confidence: 0.86)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- 系统架构与部署
- WebAPI 处理器
- 前端依赖
- Vue 前端视图
- SQLite 存储
- LiveKit 集成
- RTSP 引擎
- FFmpeg 推流
- ONVIF 发现
- 通道管理页面
- AI 挂载预留

## God Nodes (most connected - your core abstractions)
1. `Store` - 17 edges
2. `Server` - 16 edges
3. `writeJSON()` - 15 edges
4. `Engine` - 13 edges
5. `Client` - 12 edges
6. `Manager` - 8 edges
7. `Go 中间层 (camingress server)` - 8 edges
8. `livekit-ingress 服务` - 8 edges
9. `Deps` - 7 edges
10. `setup()` - 7 edges

## Surprising Connections (you probably didn't know these)
- `RTSP 摄像头 (IP/GB28181/USB)` --semantically_similar_to--> `Engine`  [INFERRED] [semantically similar]
  docs/requirements.md → server/internal/rtsp/engine.go
- `Go 中间层 (camingress server)` --semantically_similar_to--> `github.com/denniste/camingress/server`  [INFERRED] [semantically similar]
  README.md → server/go.mod
- `房间管理 (设备/分组映射)` --conceptually_related_to--> `LiveKit 平台核心 (SFU/房间/鉴权)`  [INFERRED]
  docs/requirements.md → README.md
- `ingress WHIP 端口 (:8080)` --conceptually_related_to--> `Vue 3 客户端 (livekit-client)`  [INFERRED]
  deploy/ingress.yaml → README.md
- `LiveKit RTC 端口 (7881/7882-7886)` --conceptually_related_to--> `Vue 3 客户端 (livekit-client)`  [INFERRED]
  deploy/livekit.yaml → README.md

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **媒体链路 (RTSP→RTMP→ingress→WebRTC)** — readme_ffmpeg, readme_livekit_platform, readme_vue_client, deploy_docker_compose_ingress [EXTRACTED 1.00]
- **生产部署栈 (redis+livekit+ingress+go2rtc+camingress)** — deploy_docker_compose_redis, deploy_docker_compose_livekit, deploy_docker_compose_ingress, deploy_docker_compose_go2rtc, deploy_docker_compose_camingress [EXTRACTED 1.00]

## Communities (12 total, 1 thin omitted)

### Community 0 - "系统架构与部署"
Cohesion: 0.08
Nodes (30): go2rtc 服务 (profile whep), livekit-ingress 服务, livekit-server 服务, Redis 服务 (redis:7-alpine), camingress (Go 中间层) 服务, go2rtc HTTP API (:1984), go2rtc RTSP 重曝 (:8554), go2rtc WebRTC/WHEP (:8555) (+22 more)

### Community 1 - "WebAPI 处理器"
Cohesion: 0.23
Nodes (11): Deps, Server, net/http.Handler, net/http.Request, net/http.ResponseWriter, apiKeyMiddleware(), corsMiddleware(), New() (+3 more)

### Community 2 - "前端依赖"
Cohesion: 0.09
Nodes (22): axios, vite, @vitejs/plugin-vue, vue, vue-router, dependencies, axios, livekit-client (+14 more)

### Community 3 - "Vue 前端视图"
Cohesion: 0.14
Nodes (10): api, router, setup(), setup(), getTile(), join(), setup(), connect() (+2 more)

### Community 4 - "SQLite 存储"
Cohesion: 0.19
Nodes (6): database/sql.DB, time.Time, Store, Open(), Channel, Device

### Community 5 - "LiveKit 集成"
Cohesion: 0.19
Nodes (10): github.com/livekit/protocol/livekit.Ingress, github.com/livekit/protocol/livekit.IngressInput, time.Duration, Config, IngressInfo, TokenInfo, getEnv(), Client (+2 more)

### Community 6 - "RTSP 引擎"
Cohesion: 0.24
Nodes (4): net/http.Client, Engine, StreamInfo, New()

### Community 7 - "FFmpeg 推流"
Cohesion: 0.22
Nodes (6): os/exec.Cmd, sync.Mutex, buildArgs(), Manager, New(), main()

### Community 8 - "ONVIF 发现"
Cohesion: 0.33
Nodes (8): FoundDevice, context.Context, Discoverer, hostPort(), New(), parseVendor(), probeRTSP(), rtspReachable()

### Community 9 - "通道管理页面"
Cohesion: 0.48
Nodes (6): setup(), addChannel(), load(), remove(), start(), stop()

## Knowledge Gaps
- **21 isolated node(s):** `github.com/denniste/camingress/server`, `name`, `version`, `private`, `type` (+16 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **1 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Vue 3 客户端 (livekit-client)` connect `系统架构与部署` to `Vue 前端视图`?**
  _High betweenness centrality (0.238) - this node is a cross-community bridge._
- **Why does `Deps` connect `WebAPI 处理器` to `ONVIF 发现`, `SQLite 存储`, `LiveKit 集成`, `FFmpeg 推流`?**
  _High betweenness centrality (0.237) - this node is a cross-community bridge._
- **Why does `LiveKit 平台核心 (SFU/房间/鉴权)` connect `系统架构与部署` to `LiveKit 集成`?**
  _High betweenness centrality (0.228) - this node is a cross-community bridge._
- **What connects `github.com/denniste/camingress/server`, `name`, `version` to the rest of the system?**
  _21 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `系统架构与部署` be split into smaller, more focused modules?**
  _Cohesion score 0.08172043010752689 - nodes in this community are weakly interconnected._
- **Should `前端依赖` be split into smaller, more focused modules?**
  _Cohesion score 0.08695652173913043 - nodes in this community are weakly interconnected._
- **Should `Vue 前端视图` be split into smaller, more focused modules?**
  _Cohesion score 0.14285714285714285 - nodes in this community are weakly interconnected._