# 视频汇聚 (LiveKit Video Aggregation Platform)

> 对标 Upwork 真实需求："WebRTC, LiveKit, Live Streaming, FFmpeg, Video Surveillance Integration"
> 定位：跨区域视频汇聚 + 协同平台——摄像头画面进 LiveKit 房间，支持会议/讨论组/监控集成

## 项目目标

把局域网/广域网的 RTSP 摄像头（IP Camera / GB28181 / USB 相机）统一汇聚到 LiveKit 平台，
通过 WebRTC 实现**跨区域实时观看 + 视频会议 + 讨论组**，并预留 AI 检测/告警挂载能力。

## 媒体链路

```
RTSP 摄像头 ──► ffmpeg (Go 子进程管理) ──RTMP──► LiveKit ingress ──► LiveKit 房间 (SFU) ──WebRTC──► 浏览器
                                                                                │
                              Go 中间层 (WebAPI + ingress 编排) ◄────────────────┘
```

- **ffmpeg** 由 Go 中间层以子进程方式直接管理：拉 RTSP → 转码(H.265→H.264 / 音频→AAC) → 推 RTMP
- **Go 中间层**通过 LiveKit ingress API 创建/删除推流通道，精确启停 ffmpeg 进程
- **浏览器**通过 LiveKit client SDK 订阅房间，低延迟观看 / 视频会议
- **go2rtc**（可选）提供 WHEP 直看 / RTSP 重曝，默认不启动（compose profile `whep`）

## 技术栈

| 层 | 技术 | 说明 |
|---|---|---|
| 平台核心 | LiveKit (self-hosted) + Redis | 房间管理 / SFU / 鉴权 |
| 媒体引擎 | FFmpeg (Go 子进程管理) | RTSP 拉流 → 转码 → RTMP 转推 |
| 中间层 | **Go** + SQLite | 通道管理 / ONVIF 发现 / 鉴权 / ingress 编排 |
| 客户端 | Vue 3 + livekit-client | 浏览器观看 / 通道管理 / 会议 |
| 部署 | Docker Compose (生产) / 原生二进制 (研发) | Linux 生产，Windows 研发 |

## 环境定位

- **Windows**：研发 / 验证 / 交叉编译（开发机）
- **Linux + Docker**：生产环境（livekit-server / ingress / go2rtc / redis 官方镜像）
- Go 代码 `GOOS=linux GOARCH=amd64` 交叉编译 → 单二进制交付

## 目录结构

```
D:\LiveKit\
├── docs\
│   └── requirements.md       # 详细需求（含 Upwork 原始需求）
├── server\                   # Go 中间层服务
│   ├── main.go               # 入口：WebAPI + SQLite + 依赖装配
│   ├── Dockerfile            # 多阶段构建 (CGO_ENABLED=0)
│   └── internal\
│       ├── api\              # WebAPI（通道/鉴权/发现/房间）
│       ├── discovery\        # ONVIF WS-Discovery 发现 + RTSP 探测
│       ├── livekit\          # LiveKit 接入（token 签发 / ingress 管理）
│       ├── push\             # ffmpeg 推流进程管理（拉流/转码/推 RTMP）
│       ├── rtsp\             # (可选) go2rtc HTTP API 客户端（WHEP 直看）
│       └── store\            # SQLite 存储（含凭证 AES-GCM 加密）
├── web\                      # Vue 3 客户端
│   ├── src\
│   │   ├── views\            # 页面（通道管理/发现/播放/会议）
│   │   ├── api\              # 后端 API 封装（axios）
│   │   └── App.vue
│   └── package.json
└── deploy\
    ├── docker-compose.yml    # 生产部署（redis+livekit+ingress+go2rtc+videohub）
    ├── livekit.yaml          # LiveKit 配置
    └── go2rtc.yaml           # go2rtc 配置
```

## 快速开始

### 生产 (Linux + Docker)

```bash
cd deploy
# 生产环境务必设置密钥与公网地址
export LIVEKIT_URL=wss://video.example.com LIVEKIT_API_KEY=devkey LIVEKIT_API_SECRET=secret VIDEHUB_SECRET=change-me
docker compose up -d
# WebAPI: http://<host>:8080 ; LiveKit: ws://<host>:7880
```

### 研发 (Windows + 原生 Go)

```bash
# 1. 启动依赖 (Redis + LiveKit + ingress) —— 全部容器化
cd deploy
docker compose up -d redis livekit ingress
# (可选) 需要 WHEP 直看时再启动 go2rtc:
# docker compose --profile whep up -d go2rtc

# 2. 启动 Go 中间层 (原生)
cd ../server
go mod tidy
go run main.go -db videohub.db
# 环境变量 (默认值已适配 docker compose 端口映射):
#   LIVEKIT_URL=ws://localhost:7880        (浏览器信令地址)
#   LIVEKIT_HTTP_URL=http://localhost:7880 (ingress/egress 管理 twirp API)
#   LIVEKIT_API_KEY=devkey
#   LIVEKIT_API_SECRET=secret
#   VIDEOHUB_RTMP_BASE_URL=rtmp://localhost:1935  (原生 ffmpeg 访问 ingress 的地址)
#   VIDEHUB_SECRET=       (密码加密密钥, 生产必填)
#   VIDEHUB_API_KEY=      (可选, 开启 WebAPI 鉴权)

# 3. 启动 Vue 客户端
cd ../web
npm install
npm run dev

# 4. 浏览器访问
#    http://localhost:5173
```

> 研发模式下 ffmpeg 原生运行，通过 `rtmp://localhost:1935` 访问 ingress 的发布端口；
> Go 中间层通过 `localhost:7880` 调用 livekit-server 的 ingress 管理 API。

## 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `LIVEKIT_URL` | `ws://127.0.0.1:7880` | LiveKit 信令地址（浏览器可达） |
| `LIVEKIT_API_KEY` | `devkey` | LiveKit API key |
| `LIVEKIT_API_SECRET` | `secret` | LiveKit API secret |
| `LIVEKIT_HTTP_URL` | 由 `LIVEKIT_URL` 推导 | livekit-server 的 twirp API 地址（ingress/egress 管理） |
| `VIDEOHUB_RTMP_BASE_URL` | 使用 ingress 返回的 base | ffmpeg 访问 ingress 的 RTMP 基础地址 |
| `VIDEOHUB_FFMPEG` | `ffmpeg` | ffmpeg 二进制路径（不在 PATH 时指定） |
| `VIDEHUB_SECRET` | (开发默认值) | 设备密码加密密钥，**生产必填** |
| `VIDEHUB_API_KEY` | (空) | 设置后 WebAPI 需 `Authorization: Bearer` 鉴权 |

## 里程碑

- [x] M1: ffmpeg 拉 RTSP（含 H.265→H.264 转码）
- [x] M2: RTSP → RTMP → LiveKit ingress → 房间可订阅
- [x] M3: 通道管理（SQLite CRUD）+ token 鉴权
- [x] M4: ONVIF WS-Discovery 发现 + RTSP 探测
- [ ] M5: 房间管理增强 / AI 挂载（预留）
