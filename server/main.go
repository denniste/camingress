// 视频汇聚 - Go 中间层服务
// 核心职责: 通道管理 / ONVIF 发现 / 鉴权 / SQLite 存储
// 媒体链路: RTSP → ffmpeg(转码) → RTMP → LiveKit ingress → 房间
// ffmpeg 由本服务以子进程方式直接管理 (精确启停)
package main

import (
	"flag"
	"log"

	"videohub/server/internal/api"
	"videohub/server/internal/discovery"
	"videohub/server/internal/livekit"
	"videohub/server/internal/push"
	"videohub/server/internal/store"
)

var (
	addr   = flag.String("addr", ":8080", "WebAPI 监听地址")
	dbPath = flag.String("db", "videohub.db", "SQLite 数据库路径")
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

	// 2. 初始化 ffmpeg 推流管理器
	pusher := push.New()

	// 3. 初始化 LiveKit 客户端 (签发 token, ingress 管理)
	lk, err := livekit.New()
	if err != nil {
		log.Fatalf("[livekit] 初始化失败: %v", err)
	}

	// 4. 初始化 ONVIF 发现器
	discoverer := discovery.New()

	// 5. 组装 WebAPI 服务
	server := api.New(api.Deps{
		Store:     st,
		Push:      pusher,
		LiveKit:   lk,
		Discovery: discoverer,
	})

	log.Printf("[api] WebAPI 服务启动: %s", *addr)
	if err := server.Run(*addr); err != nil {
		log.Fatalf("[api] 服务退出: %v", err)
	}
}
