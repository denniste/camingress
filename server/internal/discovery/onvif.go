// ONVIF 设备发现: WS-Discovery 组播扫描 + RTSP 能力探测
package discovery

import (
	"context"
	"encoding/xml"
	"fmt"
	"net"
	"strings"
	"time"
)

// FoundDevice 发现到的设备
type FoundDevice struct {
	IP      string `json:"ip"`
	Port    int    `json:"port"`
	XAddr   string `json:"xaddr"`    // ONVIF 服务地址
	Vendor  string `json:"vendor"`   // 厂商 (型号字符串)
	Model   string `json:"model"`
	RTSPURL string `json:"rtsp_url"` // 探测到的流地址 (若有)
}

// Discoverer ONVIF 发现器
type Discoverer struct {
	timeout time.Duration
}

// New 创建发现器
func New() *Discoverer {
	return &Discoverer{timeout: 5 * time.Second}
}

// wsDiscoveryMessage WS-Discovery Probe 消息 (组播)
const wsDiscoveryAddr = "239.255.255.250:3702"

var probeTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<e:Envelope xmlns:e="http://www.w3.org/2003/05/soap-envelope"
  xmlns:w="http://schemas.xmlsoap.org/ws/2004/08/addressing"
  xmlns:d="http://schemas.xmlsoap.org/ws/2005/04/discovery"
  xmlns:dn="http://www.onvif.org/ver10/network/wsdl">
  <e:Header>
    <w:MessageID>uuid:videohub-probe-1</w:MessageID>
    <w:To e:mustUnderstand="true">urn:schemas-xmlsoap-org:ws:2005:04:discovery</w:To>
    <w:Action e:mustUnderstand="true">http://schemas.xmlsoap.org/ws/2005/04/discovery/Probe</w:Action>
  </e:Header>
  <e:Body>
    <d:Probe>
      <d:Types>dn:NetworkVideoTransmitter</d:Types>
    </d:Probe>
  </e:Body>
</e:Envelope>`

// Scan 执行 ONVIF WS-Discovery 扫描, 返回发现的设备
func (d *Discoverer) Scan(ctx context.Context) ([]FoundDevice, error) {
	// UDP 组播发送 Probe
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, fmt.Errorf("创建 UDP 监听失败: %w", err)
	}
	defer conn.Close()

	dest, err := net.ResolveUDPAddr("udp", wsDiscoveryAddr)
	if err != nil {
		return nil, err
	}
	if _, err := conn.WriteToUDP([]byte(probeTemplate), dest); err != nil {
		return nil, fmt.Errorf("发送 Probe 失败: %w", err)
	}

	// 等待响应 (2 秒窗口)
	deadline := time.Now().Add(2 * time.Second)
	_ = conn.SetReadDeadline(deadline)

	seen := map[string]bool{}
	var out []FoundDevice

	buf := make([]byte, 4096)
	for time.Now().Before(deadline) {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			break // 超时或错误
		}
		ip := addr.IP.String()
		if seen[ip] {
			continue
		}

		var probeResp struct {
			Body struct {
				ProbeMatches struct {
					ProbeMatch []struct {
						XAddrs string `xml:"XAddrs"`
						Types  string `xml:"Types"`
					} `xml:"ProbeMatch"`
				} `xml:"ProbeMatches"`
			} `xml:"Body"`
		}
		if err := xml.Unmarshal(buf[:n], &probeResp); err != nil {
			continue
		}
		if len(probeResp.Body.ProbeMatches.ProbeMatch) == 0 {
			continue
		}
		m := probeResp.Body.ProbeMatches.ProbeMatch[0]
		seen[ip] = true

		dev := FoundDevice{
			IP:     ip,
			Port:   80,
			XAddr:  m.XAddrs,
			Vendor: parseVendor(m.Types),
		}
		// 探测常见 RTSP 路径
		dev.RTSPURL = probeRTSP(ip)
		out = append(out, dev)
	}
	return out, nil
}

// parseVendor 从 ONVIF Types 提取厂商信息
func parseVendor(types string) string {
	// Types 形如: dn:NetworkVideoTransmitter 或 厂商特定命名空间
	if len(types) > 60 {
		return types[:60]
	}
	return types
}

// probeRTSP 尝试常见 RTSP 路径探测 (海康/大华/安讯士格式)
// 返回第一个能完成 RTSP OPTIONS 握手的地址
func probeRTSP(ip string) string {
	paths := []string{
		"rtsp://%s:554/Streaming/Channels/101",                 // 海康
		"rtsp://%s:554/cam/realmonitor?channel=1&subtype=0",    // 大华
		"rtsp://%s:554/onvif1",                                 // 安讯士/通用
		"rtsp://%s:554/live",                                   // 通用
	}
	for _, p := range paths {
		url := fmt.Sprintf(p, ip)
		if rtspReachable(url) {
			return url
		}
	}
	return ""
}

// rtspReachable 通过 RTSP OPTIONS 握手验证流地址是否真实可达
func rtspReachable(url string) bool {
	conn, err := net.DialTimeout("tcp", hostPort(url), 2*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))

	req := "OPTIONS " + url + " RTSP/1.0\r\nCSeq: 1\r\nUser-Agent: videohub\r\n\r\n"
	if _, err := conn.Write([]byte(req)); err != nil {
		return false
	}
	buf := make([]byte, 512)
	n, err := conn.Read(buf)
	if err != nil {
		return false
	}
	return strings.HasPrefix(string(buf[:n]), "RTSP/1.0")
}

func hostPort(url string) string {
	// rtsp://ip:554/path → ip:554
	s := url
	if len(s) > 7 && s[:7] == "rtsp://" {
		s = s[7:]
	}
	if i := strings.IndexByte(s, '/'); i >= 0 {
		return s[:i]
	}
	return s
}
