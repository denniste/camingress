# 视频汇聚 - 详细需求

## 1. Upwork 原始需求（本项目对标）

### 需求 A：WebRTC, LiveKit, Live Streaming, FFmpeg, Video Surveillance Integration（$3,200, Expert）
> "We are building an always-on 4K streaming camera unit that feeds a **self-hosted** platform..."
> 技术栈点名：**WebRTC + LiveKit + Live Streaming + FFmpeg + Video Surveillance Integration**

### 需求 B：Senior WebRTC/LiveKit Developer – Video Platform
> "We are building a next-generation **interactive video platform** for live events, **webinars**, and **virtual conferences**."

### 需求 C：Real-Time Video Streaming Platform Developer
> "design and develop a scalable, **low-latency** live video streaming platform"

### 用户画像推断
- 传统企业，多分支机构，跨区域协同（视频会议/在线讨论组）
- 有存量 IP 摄像头（RTSP），需集成到统一平台
- 自托管（self-hosted），非云 SaaS
- 需求 B 明确要 webinar / 虚拟会议 → LiveKit 房间 + 互动

## 2. 系统需求

### 2.1 后端（LiveKit + Redis）
- LiveKit Server（自托管）：房间管理、SFU 转发、鉴权（token）
- Redis：LiveKit 依赖（ingress 调度）
- 生产用 Docker Compose 部署（livekit-server / livekit-ingress / redis）

### 2.2 Go 中间层服务
核心：**ffmpeg 拉 RTSP 流 → 转码(H.265→H.264/AAC) → 推 RTMP → LiveKit ingress → 房间**
（ffmpeg 由中间层以子进程方式精确启停；go2rtc 可选，用于 WHEP 直看；浏览器直推可走 WHIP ingress）

扩展功能（WebAPI）：
- **通道管理**：摄像头通道 CRUD（添加/删除/启用/停用 RTSP 源）
- **ONVIF 发现**：自动发现局域网摄像头（WS-Discovery），GetProfiles/GetStreamUri 取流地址
- **鉴权**：签发/校验 LiveKit token（设备发布端 + 观众观看端分离）
- **配置存储**：SQLite（设备表、通道表、配置表、用户表）
- **房间管理**（可选）：按设备/分组自动映射 LiveKit 房间
- **AI 挂载**（可选）：预留检测服务接口（YOLO/LocateAnything），事件回调

### 2.3 浏览器客户端（Vue 3）
- 通道管理页：查看/添加/删除摄像头通道
- 发现页：ONVIF 扫描结果展示，一键入库
- 播放页：WebRTC 低延迟观看（LiveKit client SDK）
- 会议页：多人房间（webinar/讨论组模式）
- 鉴权：登录/Token 管理

## 3. 架构图

```
┌─ 发现层 ─────────────────────────────────────────┐
│ ONVIF WS-Discovery / IP 扫描 / GB28181 注册        │
└──────────────┬─────────────────────────────────────┘
               ▼
┌─ Go 中间层（server）─────────────────────────────┐
│ · 通道管理 (SQLite)                               │
│ · ffmpeg: RTSP 拉流 → 转码 → RTMP 推流           │
│ · 鉴权 (token 签发/校验)                          │
│ · ONVIF 发现                                     │
│ · ingress 编排 / 房间管理                         │
└──────┬──────────────────────┬─────────────────────┘
       │ RTMP (RTSP→ingress)   │ WebAPI (管理)
       ▼                       ▼
┌─ LiveKit ingress+房间 ─┐ ┌─ Vue 3 客户端 ─────────┐
│ 浏览器订阅 (WebRTC)     │ │ 观看/管理/会议          │
└────────────────────────┘ └────────────────────────┘
```

## 4. 编解码器策略

- 摄像头 H.264 → 直通（首选，零转码）
- 摄像头 H.265 → 服务端转码 H.264（go2rtc + NVIDIA NVENC / AMD VAAPI）
- WebRTC 兼容：VP8 全支持、H.264 全支持、H.265 仅 Safari（需转码）
- 能力探测：ONVIF GetProfiles 记录设备编码，按能力自动选路

## 5. 安全与部署

- 鉴权：LiveKit token（RBAC：设备发布 / 观众观看 / 管理员）
- 存储：SQLite 单机；凭证加密存储
- 部署：Linux + Docker Compose（生产）/ Windows 原生（研发）
- 跨平台：Go 单二进制，GOOS=linux 交叉编译
