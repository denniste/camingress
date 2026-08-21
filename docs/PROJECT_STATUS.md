# 项目状态 (Project Status)

> 快照时间：2026-08-21 · 由 Hermes 分析生成 · 项目根：`D:\CamIngress`
> 更新：2026-08-21 环境全量恢复（git / Go / Docker），编译验证通过

## 1. 项目概况

| 项 | 值 |
|---|---|
| 名称 | 视频汇聚 (LiveKit Video Aggregation Platform) / camingress |
| 定位 | RTSP 摄像头 → ffmpeg 转码 → RTMP → LiveKit ingress → 房间 → WebRTC 低延迟观看 + 会议 |
| 对标 | Upwork：WebRTC + LiveKit + Live Streaming + FFmpeg + Video Surveillance Integration |
| 迁移 | 2026-08-21 由 `D:\CamIngress` 迁至 `D:\CamIngress`（README 路径已同步） |

## 2. 代码资产

### Go 中间层 `server/`（module: github.com/denniste/camingress/server, Go 1.23）— 共 1342 行
| 包 | 文件 | 职责 |
|---|---|---|
| main | main.go (58) | 入口：SQLite + push + livekit + discovery 装配 |
| internal/api | server.go | WebAPI：13 个路由（health/devices/channels/discover/token） |
| internal/discovery | onvif.go | ONVIF WS-Discovery 发现 + RTSP 探测 |
| internal/livekit | client.go | LiveKit token 签发 / ingress 管理 (twirp) |
| internal/push | push.go (109) | ffmpeg 子进程管理：RTSP 拉流→转码→RTMP 推流 |
| internal/rtsp | engine.go (158) | go2rtc HTTP API 客户端（WHEP 直看，可选） |
| internal/store | store.go (265) | SQLite 存储（设备/通道表，凭证 AES-GCM 加密） |

### Vue 3 前端 `web/`（camingress-web, Vite 5）— 共 414 行
- 视图：Channels.vue (112) / Discovery.vue (69) / Player.vue (92) / Meeting.vue (126)
- api/client.js (15) · 依赖：vue 3.4 / vue-router 4.3 / livekit-client 2 / axios

### 部署 `deploy/`
- docker-compose.yml（redis + livekit + ingress + go2rtc(profile whep) + camingress）
- livekit.yaml / ingress.yaml / go2rtc.yaml

## 3. API 一览（13 端点）

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | /api/health | 健康检查 |
| GET/POST | /api/devices | 设备列表 / 新增 |
| DELETE | /api/devices/{id} | 删除设备 |
| GET/POST | /api/channels | 通道列表 / 新增 |
| PUT | /api/channels/{id} | 更新通道 |
| POST | /api/channels/{id}/start · /stop | 启停推流（ffmpeg 子进程） |
| DELETE | /api/channels/{id} | 删除通道 |
| POST | /api/discover | ONVIF 扫描 |
| GET | /api/token | 签发 LiveKit token |

## 4. 环境与工具链状态

| 组件 | 状态 | 说明 |
|---|---|---|
| Go | ✅ 1.27.0（D:\go） | 2026-08-21 安装，PATH 已持久化；`camingress.exe` 已重新编译通过 |
| Docker | ✅ Desktop 4.87.0（引擎 29.7.2） | 2026-08-21 安装，WSL2 后端运行中；compose 配置验证通过 |
| Node | ✅ v24.14.0 | npm 11.9.0，前端依赖已安装（node_modules 存在） |
| ffmpeg | ✅ 日志显示推流链路可用 | camingress.log 有 aac/swscaler 转码输出（H.265→H.264/AAC 真实跑过） |
| 运行状态 | ⏸ 当前未运行 | 上次运行 2026-08-21 04:47（camingress.log） |

## 5. 里程碑

- [x] M1: ffmpeg 拉 RTSP（含 H.265→H.264 转码）
- [x] M2: RTSP → RTMP → LiveKit ingress → 房间可订阅
- [x] M3: 通道管理（SQLite CRUD）+ token 鉴权
- [x] M4: ONVIF WS-Discovery 发现 + RTSP 探测
- [ ] M5: 房间管理增强 / AI 挂载（预留）

## 6. 已知问题 / 待办

1. ~~git 仓库缺失~~ ✅ 已恢复（2026-08-21 init + 首次提交 `63df52d`）
2. ~~Go 工具链缺失~~ ✅ 已安装 Go 1.27.0（D:\go，PATH 已持久化），`camingress.exe` 重新编译通过
3. ~~Docker CLI 缺失~~ ✅ 已安装 Docker Desktop 4.87.0（引擎 29.7.2，WSL2 后端运行中）
4. **完整栈未端到端验证**：redis/livekit/ingress 容器尚未 `docker compose up`（需拉镜像）
5. graphify-out/ 缓存已从 git 排除（仅保留报告）

## 7. 相关资产

- `docs/requirements.md`：详细需求（Upwork 原始需求 ×3 + 系统需求 + 架构图）
- `graphify-out/GRAPH_REPORT.md`：代码知识图谱分析报告
- `server/camingress.db`（20KB）：本地 SQLite 数据库（有数据）
- `web/dist/`：前端已构建产物
