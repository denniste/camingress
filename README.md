# CamIngress — 把 RTSP 摄像头汇聚进 LiveKit 房间

> 对标 Upwork 真实需求："WebRTC, LiveKit, Live Streaming, FFmpeg, Video Surveillance Integration"
> 定位：跨区域视频汇聚 + 协同平台——摄像头画面进 LiveKit 房间，支持会议/讨论组/监控集成
> 架构：Camera(ingress) → LiveKit 房间 → WebRTC。本项目即「RTSP 摄像头版 LiveKit ingress」

## 项目目标

把局域网/广域网的 RTSP 摄像头（IP Camera / GB28181 / USB 相机）统一汇聚到 LiveKit 平台，
通过 WebRTC 实现**跨区域实时观看 + 视频会议 + 讨论组**，并预留 AI 检测/告警挂载能力。

## 媒体链路

```
RTSP 摄像头 ──► mediamtx (拉流+透传, 无重编码) ──RTMP──► LiveKit ingress ──► LiveKit 房间 (SFU) ──WebRTC──► 浏览器
                                                                                            │
                                  Go 中间层 (WebAPI + mediamtx 编排 + ingress 管理) ◄─────────┘

回退链路 (Mode=ffmpeg): RTSP ──► ffmpeg(转码) ──RTMP──► ingress ──► 房间
```

- **mediamtx** 由 Go 中间层通过 HTTP API 动态管理（v1.20+ `config/paths` API）：
  拉 RTSP（H.264/H.265 透传）→ RTMP 转发到 LiveKit ingress（stream key 用 `#key` 语法）
- **Go 中间层**通过 LiveKit ingress API 创建/删除推流通道，精确启停 mediamtx path
- **浏览器**通过 LiveKit client SDK 订阅房间，低延迟观看 / 视频会议
- **mediamtx 同时提供 WHEP 直看**（`/path/whep`）与 RTSP 重曝，替代原 go2rtc 角色

> 为什么不用 WHIP 直推：LiveKit ingress 的 WHIP 接收端仅接受精确 fmtp 匹配的
> H.264(42001f)/VP8 offer（见 `livekit/ingress` 源码 `newMediaEngine`），摄像头透传 SDP
> 无法满足 → 采用 RTMP forward（同为无重编码转发，ingress 负责转码兼容）。

## 技术栈

| 层 | 技术 | 说明 |
|---|---|---|
| 平台核心 | LiveKit (self-hosted) + Redis | 房间管理 / SFU / 鉴权 |
| 媒体引擎 | **mediamtx** (Go 单二进制, HTTP API 编排) | RTSP 拉流 → RTMP 转发，H.264/H.265 透传 |
| 回退引擎 | FFmpeg (Go 子进程管理) | 不兼容编码/老设备的转码兜底 |
| 中间层 | **Go** + SQLite | 通道管理 / ONVIF 发现 / 鉴权 / ingress 编排 |
| 客户端 | React 18 + @livekit/components-react | 浏览器观看 / 通道管理 / 会议 |
| 部署 | Docker Compose (生产) / 原生二进制 (研发) | Linux 生产，Windows 研发 |

## 环境定位

- **Windows**：研发 / 验证 / 交叉编译（开发机；mediamtx 用原生二进制）
- **Linux + Docker**：生产环境（livekit-server / ingress / mediamtx / redis 官方镜像）
- Go 代码 `GOOS=linux GOARCH=amd64` 交叉编译 → 单二进制交付

## 目录结构

```
D:\CamIngress\
├── docs\
│   ├── requirements.md       # 详细需求（含 Upwork 原始需求）
│   └── PROJECT_STATUS.md     # 项目状态快照
├── server\                   # Go 中间层服务
│   ├── main.go               # 入口：WebAPI + SQLite + 依赖装配
│   ├── Dockerfile            # 多阶段构建 (CGO_ENABLED=0)
│   └── internal\
│       ├── api\              # WebAPI（通道/鉴权/发现/三档路由编排）
│       ├── discovery\        # ONVIF WS-Discovery 发现 + RTSP 探测
│       ├── livekit\          # LiveKit 接入（token 签发 / ingress 管理）
│       ├── mediamtx\         # MediaMTX HTTP API 客户端（path 编排）
│       ├── push\             # ffmpeg 推流进程管理（回退链路）
│       └── store\            # SQLite 存储（含凭证 AES-GCM 加密）
├── web\                      # React 18 客户端 (Vite + @livekit/components-react)
│   └── src\pages\            # 页面（控制台/通道管理/发现/播放/会议）
└── deploy\
    ├── docker-compose.yml    # 生产部署（redis+livekit+ingress+mediamtx+camingress）
    ├── livekit.yaml          # LiveKit 配置
    ├── ingress.yaml          # ingress 配置
    └── mediamtx.yaml         # MediaMTX 配置
```

## 三档路由 (通道 Mode)

通道启动时按 `mode` 字段路由：

| mode | 链路 | 适用 |
|---|---|---|
| `auto`（默认） | mediamtx 拉流 → RTMP → ingress（无重编码） | 主流 H.264/H.265 摄像头 |
| `direct` | 同上（强制） | — |
| `ffmpeg` | ffmpeg 转码 → RTMP → ingress | 不兼容编码（MJPEG/G.711 等）/ 老设备 |

启动后通道返回 `active_mode`（direct/ffmpeg）标明实际生效链路。

## 快速开始

### 生产 (Linux + Docker)

```bash
cd deploy
# 生产环境务必设置密钥与公网地址
export LIVEKIT_URL=wss://video.example.com LIVEKIT_API_KEY=devkey LIVEKIT_API_SECRET=secret CAMINGRESS_SECRET=change-me
docker compose up -d
# WebAPI: http://<host>:8080 ; LiveKit: ws://<host>:7880
```

### 研发 (Windows + 原生 Go + 原生 mediamtx + 原生 livekit-server)

> ⚠️ **Docker Desktop (WSL2) 的 localhost UDP 端口转发不可靠**（WebRTC 媒体面 UDP 不通，
> 浏览器报 "could not establish pc connection"）→ **Windows 研发模式必须原生跑 livekit-server**。

```bash
# 1. 启动依赖 (Redis + ingress) —— 容器化
cd deploy
docker compose up -d redis ingress

# 2. 启动 livekit-server (原生单二进制, 研发模式)
#    下载: https://github.com/livekit/livekit/releases (windows_amd64)
#    运行: cd D:/livekit-native && ./livekit-server.exe --config D:/CamIngress/deploy/livekit.yaml
#    (livekit.yaml 已配置 redis=localhost:6379, node_ip=本机 LAN IP)

# 3. 启动 mediamtx (原生单二进制, 研发模式; 生产用容器)
#    下载: https://github.com/bluenviron/mediamtx/releases (windows_amd64)
#    配置: 使用 deploy/mediamtx.yaml (路径改 D:/mediamtx-native/mediamtx.yml)
#    运行: cd D:/mediamtx-native && ./mediamtx.exe

# 4. 启动 Go 中间层 (原生)
cd ../server
go run main.go -db camingress.db
# 环境变量 (默认值已适配 docker compose 端口映射):
#   LIVEKIT_URL=ws://localhost:7880
#   LIVEKIT_HTTP_URL=http://localhost:7880
#   LIVEKIT_API_KEY=devkey
#   LIVEKIT_API_SECRET=secret
#   CAMINGRESS_MTX_URL=http://localhost:9997         (mediamtx API)
#   CAMINGRESS_RTMP_BASE_URL=rtmp://localhost:1935   (mediamtx 转发 ingress 的 RTMP 地址)
#   CAMINGRESS_SECRET=       (密码加密密钥, 生产必填)

# 5. 启动 React 客户端
cd ../web
npm install && npm run dev
# 浏览器访问 http://localhost:5173/player/<房间名> 即可观看

# 6. (可选) 合成测试源验证链路
ffmpeg -re -f lavfi -i "testsrc=size=640x360:rate=25" -f lavfi -i "sine=frequency=440" \
  -c:v libx264 -preset ultrafast -tune zerolatency -g 25 -pix_fmt yuv420p -c:a aac \
  -f rtsp -rtsp_transport tcp "rtsp://localhost:8554/src_test"
# 创建通道: source=rtsp://localhost:8554/src_test → 启动 → /player/<房间> 看到彩条测试画面
```

## 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `LIVEKIT_URL` | `ws://127.0.0.1:7880` | LiveKit 信令地址（浏览器可达） |
| `LIVEKIT_API_KEY` | `devkey` | LiveKit API key |
| `LIVEKIT_API_SECRET` | `secret` | LiveKit API secret |
| `LIVEKIT_HTTP_URL` | 由 `LIVEKIT_URL` 推导 | livekit-server twirp API 地址（ingress 管理） |
| `CAMINGRESS_MTX_URL` | `http://127.0.0.1:9997` | mediamtx HTTP API 地址 |
| `CAMINGRESS_RTMP_BASE_URL` | ingress 返回的 base | mediamtx 转发 ingress 的 RTMP 基础地址 |
| `CAMINGRESS_WHIP_BASE_URL` | ingress 返回的 base | WHIP 发布基础地址（预留，ingress WHIP 协商受限） |
| `CAMINGRESS_FFMPEG` | `ffmpeg` | ffmpeg 二进制路径（回退链路） |
| `CAMINGRESS_SECRET` | (开发默认值) | 设备密码加密密钥，**生产必填** |
| `CAMINGRESS_API_KEY` | (空) | 设置后 WebAPI 需 `Authorization: Bearer` |

## 里程碑

- [x] M1: ffmpeg 拉 RTSP（含 H.265→H.264 转码）
- [x] M2: RTSP → RTMP → LiveKit ingress → 房间可订阅
- [x] M3: 通道管理（SQLite CRUD）+ token 鉴权
- [x] M4: ONVIF WS-Discovery 发现 + RTSP 探测
- [x] M5: **媒体引擎升级 mediamtx**（三档路由：direct 无重编码直通 / ffmpeg 回退），端到端验证通过（房间 2 tracks）
- [ ] M6: 房间管理增强 / AI 挂载（预留）

## 已知限制

- **LiveKit ingress WHIP 输入**只接受精确 fmtp 匹配的 H.264(42001f)/VP8 offer（官方 `newMediaEngine` 硬编码），摄像头透传 SDP 无法协商 → 主链路用 RTMP forward（同为无重编码转发）
- **H.265 摄像头**：mediamtx RTMP forward 用 Enhanced RTMP 封装 H.265，但 LiveKit ingress 的 RTMP 输入**无法解码 Enhanced RTMP H.265 视频**（只发布音频轨）→ H.265 摄像头请用 `mode=ffmpeg`（转码 H.264）；H.264 摄像头可用 direct 无重编码
- **MediaMTX 官方镜像无 shell**，无法容器内 healthcheck；依赖 camingress 启动时 API 轮询（`WaitReady`）
- **Docker Desktop (WSL2) localhost UDP 端口转发不可靠**：浏览器 WebRTC 媒体面（UDP 7882-7886）连不通，报 "could not establish pc connection" → **Windows 研发模式必须原生跑 livekit-server**（见快速开始）；生产 Linux 无此问题
- Docker Desktop 下 mediamtx 容器端口映射偶发异常（如 9997 丢失），`--force-recreate` 可修复
- 浏览器信令用 `access_token` 参数（livekit-client 自动处理）；调试时 curl 需用 `?access_token=` 而非 `?token=`
