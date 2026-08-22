// LiveKit 接入层: token 签发 / ingress 管理 / 房间管理
package livekit

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	"github.com/twitchtv/twirp"
)

// Config LiveKit 配置 (环境变量或默认 dev 值)
type Config struct {
	URL       string // ws://127.0.0.1:7880 (浏览器信令地址, 用于 token)
	HTTPURL   string // http://127.0.0.1:7880 (livekit-server twirp API, ingress/egress 管理)
	APIKey    string
	APISecret string
}

// Client LiveKit 客户端
type Client struct {
	cfg Config
}

// New 从环境变量读取配置
func New() (*Client, error) {
	url := getEnv("LIVEKIT_URL", "ws://127.0.0.1:7880")
	httpURL := getEnv("LIVEKIT_HTTP_URL", toHTTP(url))
	cfg := Config{
		URL:       url,
		HTTPURL:   httpURL,
		APIKey:    getEnv("LIVEKIT_API_KEY", "devkey"),
		APISecret: getEnv("LIVEKIT_API_SECRET", "secret"),
	}
	if err := (&Client{cfg: cfg}).Validate(); err != nil {
		return nil, err
	}
	return &Client{cfg: cfg}, nil
}

// toHTTP 将 ws:// 地址转为 http:// (livekit-server 的 twirp API 与信令同端口)
func toHTTP(wsURL string) string {
	if strings.HasPrefix(wsURL, "wss://") {
		return "https://" + strings.TrimPrefix(wsURL, "wss://")
	}
	if strings.HasPrefix(wsURL, "ws://") {
		return "http://" + strings.TrimPrefix(wsURL, "ws://")
	}
	return wsURL
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// TokenInfo 签发的 token 信息
type TokenInfo struct {
	Token      string `json:"token"`
	URL        string `json:"url"`
	Room       string `json:"room"`
	Identity   string `json:"identity"`
	CanPublish bool   `json:"can_publish"`
}

// GrantToken 签发 LiveKit 接入 token
// role: "publisher" (设备发布) | "viewer" (观众观看) | "admin" (管理)
func (c *Client) GrantToken(room, identity, role string, ttl time.Duration) (*TokenInfo, error) {
	at := auth.NewAccessToken(c.cfg.APIKey, c.cfg.APISecret).
		SetIdentity(identity).
		SetValidFor(ttl)

	grant := &auth.VideoGrant{RoomJoin: true, Room: room}
	switch role {
	case "publisher":
		grant.SetCanPublish(true)
		grant.SetCanSubscribe(true)
	case "admin":
		grant.SetCanPublish(true)
		grant.SetCanSubscribe(true)
		grant.RoomAdmin = true
	default: // viewer: 必须显式禁止发布 (LiveKit 对未设置项默认允许)
		grant.SetCanPublish(false)
		grant.SetCanSubscribe(true)
	}
	at.SetVideoGrant(grant)

	token, err := at.ToJWT()
	if err != nil {
		return nil, err
	}
	return &TokenInfo{
		Token:      token,
		URL:        c.cfg.URL,
		Room:       room,
		Identity:   identity,
		CanPublish: grant.GetCanPublish(),
	}, nil
}

// IngressInfo 供 API 返回的 ingress 信息
type IngressInfo struct {
	IngressID string `json:"ingress_id"`
	URL       string `json:"url"`
	StreamKey string `json:"stream_key"`
	RoomName  string `json:"room_name"`
}

// CreateRTMPIngress 创建 RTMP ingress (摄像头 RTSP 流经 go2rtc 转推进入 LiveKit 房间)
func (c *Client) CreateRTMPIngress(name, roomName string) (*IngressInfo, error) {
	return c.createIngress(livekit.IngressInput_RTMP_INPUT, name, roomName)
}

// CreateWHIPIngress 创建 WHIP ingress (供浏览器/编码器直接 WHIP 推流)
func (c *Client) CreateWHIPIngress(name, roomName string) (*IngressInfo, error) {
	return c.createIngress(livekit.IngressInput_WHIP_INPUT, name, roomName)
}

func (c *Client) createIngress(input livekit.IngressInput, name, roomName string) (*IngressInfo, error) {
	ctx, err := c.ingressCtx()
	if err != nil {
		return nil, err
	}
	req := &livekit.CreateIngressRequest{
		InputType:           input,
		Name:                name,
		RoomName:            roomName,
		ParticipantIdentity: name,
		ParticipantName:     name,
	}
	// 注: RTMP 输入不允许 bypass 转码 (LiveKit 硬限制, 仅 WHIP 可 bypass),
	// 原生 ingress 转码依赖 GStreamer 编码器 — 见 deploy/ingress-native.yaml 与 skill 记录
	info, err := c.ingressClient().CreateIngress(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("创建 ingress 失败: %w", err)
	}
	// LiveKit 返回的 Url 仅为 base_url (rtmp_base_url / whip_base_url),
	// 需自行拼接推流路径与 stream_key:
	//   RTMP: {base}/live/{stream_key}
	//   WHIP: {base}/whip/{stream_key}
	// 可用 CAMINGRESS_RTMP_BASE_URL 覆盖 base (如原生 ffmpeg 无法解析容器内主机名时)
	base := info.Url
	if input == livekit.IngressInput_RTMP_INPUT {
		if v := os.Getenv("CAMINGRESS_RTMP_BASE_URL"); v != "" {
			base = v
		}
	}
	if input == livekit.IngressInput_WHIP_INPUT {
		if v := os.Getenv("CAMINGRESS_WHIP_BASE_URL"); v != "" {
			base = v
		}
	}
	pushURL := strings.TrimRight(base, "/")
	switch input {
	case livekit.IngressInput_RTMP_INPUT:
		pushURL += "/live/" + info.StreamKey
	case livekit.IngressInput_WHIP_INPUT:
		pushURL += "/whip/" + info.StreamKey
	}
	return &IngressInfo{
		IngressID: info.IngressId,
		URL:       pushURL,
		StreamKey: info.StreamKey,
		RoomName:  info.RoomName,
	}, nil
}

// DeleteIngress 删除 ingress
func (c *Client) DeleteIngress(ingressID string) error {
	ctx, err := c.ingressCtx()
	if err != nil {
		return err
	}
	_, err = c.ingressClient().DeleteIngress(ctx, &livekit.DeleteIngressRequest{
		IngressId: ingressID,
	})
	return err
}

// Healthy TCP 探测 livekit-server 是否可达 (信令/API 同端口)
func (c *Client) Healthy() bool {
	u, err := url.Parse(c.cfg.HTTPURL)
	if err != nil {
		return false
	}
	conn, err := net.DialTimeout("tcp", u.Host, 2*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// ingressClient 创建 livekit-server 的 ingress 管理 twirp 客户端
func (c *Client) ingressClient() livekit.Ingress {
	return livekit.NewIngressJSONClient(c.cfg.HTTPURL, &http.Client{})
}

// ingressCtx 构造带 ingress 管理权限 JWT 的上下文
func (c *Client) ingressCtx() (context.Context, error) {
	at := auth.NewAccessToken(c.cfg.APIKey, c.cfg.APISecret).
		SetIdentity("camingress").
		SetValidFor(time.Minute)
	at.SetVideoGrant(&auth.VideoGrant{IngressAdmin: true})
	token, err := at.ToJWT()
	if err != nil {
		return nil, err
	}
	hdr := http.Header{}
	hdr.Set("Authorization", "Bearer "+token)
	return twirp.WithHTTPRequestHeaders(context.Background(), hdr)
}

// Validate 校验配置
func (c *Client) Validate() error {
	if c.cfg.URL == "" {
		return fmt.Errorf("LIVEKIT_URL 不能为空")
	}
	if c.cfg.HTTPURL == "" {
		return fmt.Errorf("LIVEKIT_HTTP_URL 不能为空")
	}
	if c.cfg.APIKey == "" || c.cfg.APISecret == "" {
		return fmt.Errorf("LIVEKIT_API_KEY / LIVEKIT_API_SECRET 不能为空")
	}
	return nil
}
