// ffmpeg 推流管理器: RTSP → (转码) → RTMP (LiveKit ingress)
// 由 Go 直接管理 ffmpeg 进程, 实现精确的启动/停止与生命周期管理
package push

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
)

// Manager 管理 ffmpeg 推流进程
type Manager struct {
	mu       sync.Mutex
	procs    map[string]*exec.Cmd // channelID -> cmd
	ffmpegBin string
}

// New 创建管理器 (ffmpeg 二进制路径可用 VIDEHUB_FFMPEG 覆盖)
func New() *Manager {
	bin := os.Getenv("VIDEOHUB_FFMPEG")
	if bin == "" {
		bin = "ffmpeg"
	}
	return &Manager{procs: map[string]*exec.Cmd{}, ffmpegBin: bin}
}

// Start 启动推流 (channelID 用于去重/停止; rtmpURL 为完整 rtmp://.../live/{key})
// transcode: "" 或 "copy" (视频直通) | "h264" (视频转码 H.264, 用于 H.265 源)
func (m *Manager) Start(channelID, source, rtmpURL, transcode string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 幂等: 若已存在则先停止
	if old, ok := m.procs[channelID]; ok {
		_ = old.Process.Kill()
	}

	cmd := exec.Command(m.ffmpegBin, buildArgs(source, rtmpURL, transcode)...)
	cmd.Stderr = log.New(os.Stderr, "[ffmpeg:"+channelID+"] ", log.LstdFlags).Writer()
	cmd.Stdout = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 ffmpeg 失败: %w", err)
	}
	m.procs[channelID] = cmd

	// 后台等待进程结束 (意外退出时清理)
	go func() {
		_ = cmd.Wait()
		m.mu.Lock()
		if m.procs[channelID] == cmd {
			delete(m.procs, channelID)
		}
		m.mu.Unlock()
	}()

	return nil
}

// Stop 停止推流
func (m *Manager) Stop(channelID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cmd, ok := m.procs[channelID]; ok {
		_ = cmd.Process.Kill()
		delete(m.procs, channelID)
	}
}

// Running 是否正在推流
func (m *Manager) Running(channelID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.procs[channelID]
	return ok
}

// buildArgs 构造 ffmpeg 参数
func buildArgs(source, rtmpURL, transcode string) []string {
	args := []string{
		"-hide_banner", "-loglevel", "warning",
		"-rtsp_transport", "tcp", // TCP 更可靠, 避免 UDP 丢包
		"-i", source,
	}

	if transcode == "h264" {
		// H.265/其他 → H.264 (软件编码, 无 low_delay 标志, 兼容 B 帧 HEVC)
		args = append(args,
			"-c:v", "libx264",
			"-preset", "superfast",
			"-tune", "zerolatency",
			"-g", "50",
			"-pix_fmt", "yuv420p",
		)
	} else {
		// 视频直通 (H.264 源)
		args = append(args, "-c:v", "copy")
	}

	// 音频统一转 AAC (RTMP 不支持 PCMA/PCMU 等)
	args = append(args,
		"-c:a", "aac",
		"-f", "flv",
		rtmpURL,
	)
	return args
}
