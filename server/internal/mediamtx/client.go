// MediaMTX 客户端: 通过 HTTP API 动态管理 path
// 媒体链路: RTSP 摄像头 → mediamtx 拉流(H.264/H.265 透传) → WHIP → LiveKit ingress
// API (v1.20+): /v3/config/paths/* 配置管理, /v3/paths/* 运行时状态
// 注意: v1.20 移除 start/stop 端点, 配置创建即生效; forward 替代旧版 dest
package mediamtx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client MediaMTX HTTP API 客户端
type Client struct {
	baseURL string
	client  *http.Client
}

// New 创建客户端 (baseURL 如 http://127.0.0.1:9997 或 http://mediamtx:9997)
func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

// Ready 健康检查: 等待 mediamtx 就绪
func (c *Client) Ready() bool {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/v3/paths/list", nil)
	if err != nil {
		return false
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 400
}

// WaitReady 轮询等待就绪
func (c *Client) WaitReady() error {
	for i := 0; i < 20; i++ {
		if c.Ready() {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("mediamtx 未就绪: %s", c.baseURL)
}

// ForwardDest WHIP 转发目标 (v1.20 forward 结构)
type ForwardDest struct {
	Dest string `json:"dest"`
}

// PathInfo 路径信息 (运行时状态)
type PathInfo struct {
	Name   string `json:"name"`
	Ready  bool   `json:"ready"`
	Online bool   `json:"online"`
	Tracks []string `json:"tracks"`
}

// AddPath 创建 path (幂等: 已存在则先删除再创建)
// name: path 名 (建议用 channelID)
// source: rtsp://user:pass@ip:554/stream
// whipDest: WHIP 发布目标, 如 whip://ingress:8080/whip/{stream_key}
// 注意: v1.20 配置即生效 (无 start 端点); sourceOnDemand 默认 false = 立即拉流
func (c *Client) AddPath(name, source, whipDest string) error {
	// 幂等: 已存在的 path 先删除 (忽略 404)
	_ = c.DeletePath(name)

	body, err := json.Marshal(map[string]interface{}{
		"source":  source,
		"forward": []ForwardDest{{Dest: whipDest}},
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost,
		c.baseURL+"/v3/config/paths/add/"+url.PathEscape(name),
		bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, http.StatusOK)
}

// DeletePath 删除 path (不存在则忽略)
func (c *Client) DeletePath(name string) error {
	req, err := http.NewRequest(http.MethodDelete,
		c.baseURL+"/v3/config/paths/delete/"+url.PathEscape(name), nil)
	if err != nil {
		return err
	}
	return c.do(req, http.StatusOK)
}

// ListPaths 列出所有 path (运行时状态)
func (c *Client) ListPaths() ([]PathInfo, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/v3/paths/list", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("查询 path 失败 %d: %s", resp.StatusCode, b)
	}
	var out struct {
		Items []PathInfo `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

// WHEPUrl 返回某 path 的 WHEP 播放地址 (WebRTC 直看, 可选能力)
func (c *Client) WHEPUrl(name string) string {
	return c.baseURL + "/" + url.PathEscape(name) + "/whep"
}

// do 执行请求, 允许的状态码视为成功, 否则解析错误信息
func (c *Client) do(req *http.Request, okCodes ...int) error {
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	for _, code := range okCodes {
		if resp.StatusCode == code {
			return nil
		}
	}
	// 404/409 等幂等场景视为成功
	if resp.StatusCode == http.StatusConflict || resp.StatusCode == http.StatusNotFound {
		return nil
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	return fmt.Errorf("mediamtx %s %s 失败 %d: %s",
		req.Method, req.URL.Path, resp.StatusCode, b)
}
