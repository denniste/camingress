// go2rtc 引擎客户端 (sidecar)
// go2rtc 通过 HTTP API 管理流 (默认监听 :1984), 本包只负责调用其 API
// API 参考: https://github.com/AlexxIT/go2rtc/tree/master/api
package rtsp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Engine go2rtc 引擎客户端
type Engine struct {
	baseURL string
	client  *http.Client
}

// New 创建引擎客户端 (baseURL 如 http://127.0.0.1:1984 或 http://go2rtc:1984)
func New(baseURL string) *Engine {
	return &Engine{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

// Start 健康检查: 等待 go2rtc sidecar 就绪
func (e *Engine) Start() error {
	for i := 0; i < 10; i++ {
		if e.Ready() {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("go2rtc 引擎未就绪: %s", e.baseURL)
}

// Stop 无状态 HTTP 客户端, 无需释放
func (e *Engine) Stop() error { return nil }

// Ready 引擎是否可用
func (e *Engine) Ready() bool {
	_, err := e.ListStreams()
	return err == nil
}

// StreamInfo go2rtc 流信息
type StreamInfo struct {
	Name      string   `json:"name"`
	Sources   []string `json:"sources"`
	Running   bool     `json:"running"`
}

// AddStream 添加/更新 RTSP 源 (go2rtc stream)
// name: 流名 (建议使用 channel ID 等安全字符)
// source: rtsp://user:pass@ip:554/stream
// transcode: "" (直通) | "h264" (转码)
func (e *Engine) AddStream(name, source, transcode string) error {
	src := source
	if transcode == "h264" {
		// 软件转码: H.265/其他 → H.264 + AAC (适配 RTMP/WebRTC, 无需 GPU)
		src = "ffmpeg:" + source + "#video=h264#audio=aac"
	}
	// go2rtc API: PUT /api/streams?name={name}&src={url}
	q := url.Values{}
	q.Set("name", name)
	q.Set("src", src)
	req, err := http.NewRequest(http.MethodPut,
		e.baseURL+"/api/streams?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	return e.do(req)
}

// RemoveStream 移除流
// go2rtc API: DELETE /api/streams?src={name}
func (e *Engine) RemoveStream(name string) error {
	q := url.Values{}
	q.Set("src", name)
	req, err := http.NewRequest(http.MethodDelete,
		e.baseURL+"/api/streams?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	return e.do(req)
}

// PublishStream 将本地流转推到外部服务 (如 LiveKit RTMP ingress)
// go2rtc API: POST /api/streams?src={name}&dst={dstURL}
// dstURL 例: rtmp://ingress:1935/live/stream_key
func (e *Engine) PublishStream(name, dstURL string) error {
	q := url.Values{}
	q.Set("src", name)
	q.Set("dst", dstURL)
	req, err := http.NewRequest(http.MethodPost,
		e.baseURL+"/api/streams?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	return e.do(req)
}

// ListStreams 列出所有流
// go2rtc 返回 {"name": {"producers": [{"url": "..."}], "consumers": [...]}}
// 这里展平为数组
func (e *Engine) ListStreams() ([]StreamInfo, error) {
	resp, err := e.client.Get(e.baseURL + "/api/streams")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("查询流失败 %d: %s", resp.StatusCode, b)
	}

	raw := map[string]struct {
		Producers []struct {
			URL string `json:"url"`
		} `json:"producers"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	out := make([]StreamInfo, 0, len(raw))
	for name, v := range raw {
		info := StreamInfo{Name: name, Running: len(v.Producers) > 0}
		for _, p := range v.Producers {
			info.Sources = append(info.Sources, p.URL)
		}
		out = append(out, info)
	}
	return out, nil
}

// WHEPUrl 获取某流的 WHEP 播放地址 (WebRTC 输出, 供外部 WHEP 客户端订阅)
func (e *Engine) WHEPUrl(name string) string {
	return e.baseURL + "/api/webrtc?src=" + url.QueryEscape(name)
}

func (e *Engine) do(req *http.Request) error {
	resp, err := e.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("go2rtc %s %s 失败 %d: %s",
			req.Method, req.URL.Path, resp.StatusCode, b)
	}
	return nil
}
