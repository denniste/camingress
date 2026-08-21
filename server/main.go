// 视频汇聚 - Go 中间层服务
// 核心职责: 通道管理 / ONVIF 发现 / 鉴权 / SQLite 存储 / 媒体编排
// 媒体链路: RTSP → mediamtx(拉流+透传) → WHIP → LiveKit ingress → 房间 → WebRTC
// 回退链路: RTSP → ffmpeg(转码) → RTMP → LiveKit ingress
package main

import (
	"flag"
	"log"
	"os"

	"github.com/denniste/camingress/server/internal/api"
	"github.com/denniste/camingress/server/internal/discovery"
	"github.com/denniste/camingress/server/internal/livekit"
	"github.com/denniste/camingress/server/internal/mediamtx"
	"github.com/denniste/camingress/server/internal/push"
	"github.com/denniste/camingress/server/internal/store"
)

var (
	addr   = flag.String("addr", ":8080", "WebAPI 监听地址")
	dbPath = flag.String("db", "camingress.db", "SQLite 数据库路径")
)

func main() {
	flag.Parse()

	// 1. 初始化 SQLite 存储
	st, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("[store] 打开数据库失败: %v", err)
	}
	defer st.Close()
	log.Printf("[store] SQLite 已初始化: %s", *dbPath)

	// 2. 初始化 ffmpeg 推流管理器 (回退链路)
	pusher := push.New()

	// 3. 初始化 LiveKit 客户端 (签发 token, ingress 管理)
	lk, err := livekit.New()
	if err != nil {
		log.Fatalf("[livekit] 初始化失败: %v", err)
	}

	// 4. 初始化 ONVIF 发现器
	discoverer := discovery.New()

	// 5. 初始化 MediaMTX 编排客户端 (主链路: RTSP→WHIP)
	mtx := mediamtx.New(getEnv("CAMINGRESS_MTX_URL", "http://127.0.0.1:9997"))
	if err := mtx.WaitReady(); err != nil {
		log.Printf("[mediamtx] 警告: %v (直推链路暂不可用, ffmpeg 回退可用)", err)
	}

	// 6. 组装 WebAPI 服务
	server := api.New(api.Deps{
		Store:     st,
		Push:      pusher,
		LiveKit:   lk,
		Discovery: discoverer,
		MediaMTX:  mtx,
	})

	log.Printf("[api] WebAPI 服务启动: %s", *addr)
	if err := server.Run(*addr); err != nil {
		log.Fatalf("[api] 服务退出: %v", err)
	}
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
