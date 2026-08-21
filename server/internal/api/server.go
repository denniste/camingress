// WebAPI 服务: 通道管理 / ONVIF 发现 / 鉴权 / 状态
package api

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"videohub/server/internal/discovery"
	"videohub/server/internal/livekit"
	"videohub/server/internal/push"
	"videohub/server/internal/store"
)

// Deps 服务依赖
type Deps struct {
	Store     *store.Store
	Push      *push.Manager
	LiveKit   *livekit.Client
	Discovery *discovery.Discoverer
}

// Server API 服务
type Server struct {
	deps Deps
}

func New(d Deps) *Server { return &Server{deps: d} }

func (s *Server) Run(addr string) error {
	r := mux.NewRouter()

	// 健康检查
	r.HandleFunc("/api/health", s.handleHealth).Methods("GET")

	// 设备管理
	r.HandleFunc("/api/devices", s.handleListDevices).Methods("GET")
	r.HandleFunc("/api/devices", s.handleCreateDevice).Methods("POST")
	r.HandleFunc("/api/devices/{id}", s.handleDeleteDevice).Methods("DELETE")

	// 通道管理
	r.HandleFunc("/api/channels", s.handleListChannels).Methods("GET")
	r.HandleFunc("/api/channels", s.handleCreateChannel).Methods("POST")
	r.HandleFunc("/api/channels/{id}", s.handleUpdateChannel).Methods("PUT")
	r.HandleFunc("/api/channels/{id}/start", s.handleStartChannel).Methods("POST")
	r.HandleFunc("/api/channels/{id}/stop", s.handleStopChannel).Methods("POST")
	r.HandleFunc("/api/channels/{id}", s.handleDeleteChannel).Methods("DELETE")

	// ONVIF 发现
	r.HandleFunc("/api/discover", s.handleDiscover).Methods("POST")

	// LiveKit 鉴权
	r.HandleFunc("/api/token", s.handleToken).Methods("GET")

	// 中间件 (跨域 -> 鉴权)
	r.Use(corsMiddleware)
	r.Use(apiKeyMiddleware)

	log.Printf("[api] 路由已注册")
	return http.ListenAndServe(addr, r)
}

// ── Handlers ──

func writeJSON(w http.ResponseWriter, v interface{}, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"status": "ok"}, 200)
}

func (s *Server) handleListDevices(w http.ResponseWriter, r *http.Request) {
	devs, err := s.deps.Store.ListDevices()
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, 500)
		return
	}
	for i := range devs {
		devs[i].Password = "" // 脱敏
	}
	writeJSON(w, devs, 200)
}

func (s *Server) handleCreateDevice(w http.ResponseWriter, r *http.Request) {
	var d store.Device
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeJSON(w, map[string]string{"error": "参数错误"}, 400)
		return
	}
	d.ID = newID("dev")
	if d.Status == "" {
		d.Status = "offline"
	}
	d.CreatedAt = time.Now()
	if err := s.deps.Store.SaveDevice(&d); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, 500)
		return
	}
	d.Password = "" // 脱敏
	writeJSON(w, d, 201)
}

func (s *Server) handleDeleteDevice(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if err := s.deps.Store.DeleteDevice(id); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, 500)
		return
	}
	writeJSON(w, map[string]bool{"ok": true}, 200)
}

func (s *Server) handleListChannels(w http.ResponseWriter, r *http.Request) {
	chs, err := s.deps.Store.ListChannels()
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, 500)
		return
	}
	writeJSON(w, chs, 200)
}

func (s *Server) handleCreateChannel(w http.ResponseWriter, r *http.Request) {
	var c store.Channel
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeJSON(w, map[string]string{"error": "参数错误"}, 400)
		return
	}
	if c.Source == "" {
		writeJSON(w, map[string]string{"error": "source 必填"}, 400)
		return
	}
	c.ID = newID("ch")
	if c.Status == "" {
		c.Status = "stopped"
	}
	if c.Room == "" {
		c.Room = sanitizeRoom(c.Name)
	}
	c.CreatedAt = time.Now()
	if err := s.deps.Store.SaveChannel(&c); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, 500)
		return
	}
	writeJSON(w, c, 201)
}

func (s *Server) handleUpdateChannel(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	cur, err := s.deps.Store.GetChannel(id)
	if err != nil {
		writeJSON(w, map[string]string{"error": "通道不存在"}, 404)
		return
	}
	var in store.Channel
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, map[string]string{"error": "参数错误"}, 400)
		return
	}
	// 只更新可编辑字段
	if in.Name != "" {
		cur.Name = in.Name
	}
	if in.Source != "" {
		cur.Source = in.Source
	}
	if in.DeviceID != "" {
		cur.DeviceID = in.DeviceID
	}
	if in.Transcode != "" {
		cur.Transcode = in.Transcode
	}
	if in.Room != "" {
		cur.Room = sanitizeRoom(in.Room)
	}
	if err := s.deps.Store.SaveChannel(cur); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, 500)
		return
	}
	writeJSON(w, cur, 200)
}

func (s *Server) handleStartChannel(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	c, err := s.deps.Store.GetChannel(id)
	if err != nil {
		writeJSON(w, map[string]string{"error": "通道不存在"}, 404)
		return
	}
	if c.Status == "active" && c.IngressID != "" {
		writeJSON(w, map[string]string{"error": "通道已启动"}, 409)
		return
	}
	if c.Room == "" {
		c.Room = sanitizeRoom(c.Name)
	}

	// 1. 创建 LiveKit RTMP ingress
	ing, err := s.deps.LiveKit.CreateRTMPIngress("ing-"+c.ID, c.Room)
	if err != nil {
		writeJSON(w, map[string]string{"error": "创建 ingress 失败: " + err.Error()}, 500)
		return
	}

	// 2. ffmpeg 拉 RTSP → (转码) → 推 RTMP 到 ingress
	if err := s.deps.Push.Start(c.ID, c.Source, ing.URL, c.Transcode); err != nil {
		_ = s.deps.LiveKit.DeleteIngress(ing.IngressID)
		writeJSON(w, map[string]string{"error": "推流启动失败: " + err.Error()}, 500)
		return
	}

	c.Status = "active"
	c.IngressID = ing.IngressID
	c.StreamKey = ing.StreamKey
	c.Room = ing.RoomName
	if err := s.deps.Store.SaveChannel(c); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, 500)
		return
	}
	writeJSON(w, c, 200)
}

func (s *Server) handleStopChannel(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	c, err := s.deps.Store.GetChannel(id)
	if err != nil {
		writeJSON(w, map[string]string{"error": "通道不存在"}, 404)
		return
	}
	s.deps.Push.Stop(c.ID)
	if c.IngressID != "" {
		_ = s.deps.LiveKit.DeleteIngress(c.IngressID)
	}
	c.Status = "stopped"
	c.IngressID = ""
	c.StreamKey = ""
	if err := s.deps.Store.SaveChannel(c); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, 500)
		return
	}
	writeJSON(w, c, 200)
}

func (s *Server) handleDeleteChannel(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	s.deps.Push.Stop(id)
	if c, err := s.deps.Store.GetChannel(id); err == nil {
		if c.IngressID != "" {
			_ = s.deps.LiveKit.DeleteIngress(c.IngressID)
		}
	}
	if err := s.deps.Store.DeleteChannel(id); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, 500)
		return
	}
	writeJSON(w, map[string]bool{"ok": true}, 200)
}

// handleDiscover ONVIF 扫描
func (s *Server) handleDiscover(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	devs, err := s.deps.Discovery.Scan(ctx)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, 500)
		return
	}
	writeJSON(w, devs, 200)
}

// handleToken 签发 LiveKit token
// ?room=xxx&identity=xxx&role=viewer|publisher|admin
func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	room := q.Get("room")
	identity := q.Get("identity")
	role := q.Get("role")
	if role == "" {
		role = "viewer"
	}
	if room == "" {
		writeJSON(w, map[string]string{"error": "room 必填"}, 400)
		return
	}
	if identity == "" {
		identity = "user-" + newID("")
	}
	ti, err := s.deps.LiveKit.GrantToken(room, identity, role, 2*time.Hour)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, 500)
		return
	}
	writeJSON(w, ti, 200)
}

// ── 辅助 ──

// newID 生成带纳秒时间戳的 ID (避免同秒碰撞)
func newID(prefix string) string {
	if prefix != "" {
		prefix += "-"
	}
	return prefix + strconv.FormatInt(time.Now().UnixNano(), 36)
}

// sanitizeRoom 将名称转为合法的 LiveKit 房间名 ([a-z0-9_-])
func sanitizeRoom(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		s = "room-" + strconv.FormatInt(time.Now().Unix(), 36)
	}
	return s
}

// corsMiddleware 开发跨域
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// apiKeyMiddleware 可选 API 鉴权
// 设置 VIDEHUB_API_KEY 后, /api/* (除 /api/health) 需携带 Bearer token
func apiKeyMiddleware(next http.Handler) http.Handler {
	key := os.Getenv("VIDEHUB_API_KEY")
	if key == "" {
		return next // 未配置则关闭鉴权 (开发模式)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			next.ServeHTTP(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+key {
			writeJSON(w, map[string]string{"error": "未授权"}, 401)
			return
		}
		next.ServeHTTP(w, r)
	})
}
