// SQLite 存储层: 设备表 / 通道表
// 设备凭证 (密码) 使用 AES-GCM 加密后落盘, 密钥来自环境变量 CAMINGRESS_SECRET
package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"io"
	"os"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Device 摄像头设备 (ONVIF 发现/手动注册)
type Device struct {
	ID        string    `json:"id"`
	Vendor    string    `json:"vendor"`
	Model     string    `json:"model"`
	IP        string    `json:"ip"`
	RTSPURL   string    `json:"rtsp_url"`
	Username  string    `json:"username,omitempty"`
	Password  string    `json:"password,omitempty"`
	Profile   string    `json:"profile"`
	Codec     string    `json:"codec"`   // H264/H265/MJPEG
	Status    string    `json:"status"`  // online/offline/error
	RoomName  string    `json:"room_name"`
	CreatedAt time.Time `json:"created_at"`
}

// Channel 通道 (设备下的码流, 对应 go2rtc stream)
type Channel struct {
	ID        string    `json:"id"`
	DeviceID  string    `json:"device_id"`
	Name      string    `json:"name"`
	Source    string    `json:"source"`    // rtsp://... / 文件 / usb
	Status    string    `json:"status"`    // active/stopped/error
	Transcode string    `json:"transcode"` // "" | "h264" | "copy" (ffmpeg 档专用)
	Mode      string    `json:"mode"`      // auto(默认,自动路由) | direct(强制直推) | ffmpeg(强制ffmpeg)
	ActiveMode string   `json:"active_mode"` // 运行时实际生效: direct/ffmpeg (启动后填充)
	Room      string    `json:"room"`      // 映射的 LiveKit 房间名
	IngressID string    `json:"ingress_id"`
	StreamKey string    `json:"stream_key"`
	CreatedAt time.Time `json:"created_at"`
}

// Store 存储
type Store struct {
	db *sql.DB
}

// Open 打开/初始化数据库
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS devices (
			id TEXT PRIMARY KEY, vendor TEXT, model TEXT, ip TEXT,
			rtsp_url TEXT, username TEXT, password TEXT, profile TEXT,
			codec TEXT, status TEXT, room_name TEXT, created_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS channels (
			id TEXT PRIMARY KEY, device_id TEXT, name TEXT, source TEXT,
			status TEXT, transcode TEXT, room TEXT, ingress_id TEXT,
			stream_key TEXT, created_at TEXT
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return err
		}
	}
	// 迁移: 增加 mode 列 (旧库无此列)
	has, err := s.hasColumn("channels", "mode")
	if err != nil {
		return err
	}
	if !has {
		if _, err := s.db.Exec(`ALTER TABLE channels ADD COLUMN mode TEXT DEFAULT 'auto'`); err != nil {
			return err
		}
	}
	// 迁移: 增加 active_mode 列 (运行时实际生效模式)
	has2, err := s.hasColumn("channels", "active_mode")
	if err != nil {
		return err
	}
	if !has2 {
		if _, err := s.db.Exec(`ALTER TABLE channels ADD COLUMN active_mode TEXT DEFAULT ''`); err != nil {
			return err
		}
	}
	// 兜底: 历史 NULL 行填充默认值
	if _, err := s.db.Exec(`UPDATE channels SET mode='auto' WHERE mode IS NULL`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`UPDATE channels SET active_mode='' WHERE active_mode IS NULL`); err != nil {
		return err
	}
	return nil
}

// hasColumn 检查表是否存在某列
func (s *Store) hasColumn(table, col string) (bool, error) {
	rows, err := s.db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt interface{}
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == col {
			return true, nil
		}
	}
	return false, nil
}

func (s *Store) Close() error { return s.db.Close() }

// ── 设备 CRUD ──

func (s *Store) SaveDevice(d *Device) error {
	encPass, err := s.encrypt(d.Password)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO devices (id,vendor,model,ip,rtsp_url,username,password,profile,codec,status,room_name,created_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET
		   vendor=excluded.vendor, model=excluded.model, ip=excluded.ip,
		   rtsp_url=excluded.rtsp_url, username=excluded.username, password=excluded.password,
		   profile=excluded.profile, codec=excluded.codec,
		   status=excluded.status, room_name=excluded.room_name`,
		d.ID, d.Vendor, d.Model, d.IP, d.RTSPURL, d.Username, encPass,
		d.Profile, d.Codec, d.Status, d.RoomName, d.CreatedAt.Format(time.RFC3339),
	)
	return err
}

func (s *Store) ListDevices() ([]Device, error) {
	rows, err := s.db.Query(`SELECT id,vendor,model,ip,rtsp_url,username,password,profile,codec,status,room_name,created_at FROM devices`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Device
	for rows.Next() {
		var d Device
		var ca, encPass string
		if err := rows.Scan(&d.ID, &d.Vendor, &d.Model, &d.IP, &d.RTSPURL,
			&d.Username, &encPass, &d.Profile, &d.Codec, &d.Status, &d.RoomName, &ca); err != nil {
			return nil, err
		}
		d.Password = s.decrypt(encPass)
		d.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		out = append(out, d)
	}
	return out, nil
}

func (s *Store) GetDevice(id string) (*Device, error) {
	row := s.db.QueryRow(`SELECT id,vendor,model,ip,rtsp_url,username,password,profile,codec,status,room_name,created_at FROM devices WHERE id=?`, id)
	var d Device
	var ca, encPass string
	if err := row.Scan(&d.ID, &d.Vendor, &d.Model, &d.IP, &d.RTSPURL,
		&d.Username, &encPass, &d.Profile, &d.Codec, &d.Status, &d.RoomName, &ca); err != nil {
		return nil, err
	}
	d.Password = s.decrypt(encPass)
	d.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	return &d, nil
}

func (s *Store) DeleteDevice(id string) error {
	_, err := s.db.Exec(`DELETE FROM devices WHERE id=?`, id)
	return err
}

// ── 通道 CRUD ──

func (s *Store) SaveChannel(c *Channel) error {
	_, err := s.db.Exec(
		`INSERT INTO channels (id,device_id,name,source,status,transcode,mode,active_mode,room,ingress_id,stream_key,created_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET
		   device_id=excluded.device_id, name=excluded.name, source=excluded.source,
		   status=excluded.status, transcode=excluded.transcode, mode=excluded.mode,
		   active_mode=excluded.active_mode,
		   room=excluded.room,
		   ingress_id=excluded.ingress_id, stream_key=excluded.stream_key`,
		c.ID, c.DeviceID, c.Name, c.Source, c.Status, c.Transcode, c.Mode,
		c.ActiveMode, c.Room, c.IngressID, c.StreamKey, c.CreatedAt.Format(time.RFC3339),
	)
	return err
}

const channelCols = `id,device_id,name,source,status,transcode,mode,active_mode,room,ingress_id,stream_key,created_at`

func (s *Store) ListChannels() ([]Channel, error) {
	rows, err := s.db.Query(`SELECT ` + channelCols + ` FROM channels`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Channel
	for rows.Next() {
		var c Channel
		var ca string
		if err := rows.Scan(&c.ID, &c.DeviceID, &c.Name, &c.Source, &c.Status,
			&c.Transcode, &c.Mode, &c.ActiveMode, &c.Room, &c.IngressID, &c.StreamKey, &ca); err != nil {
			return nil, err
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		out = append(out, c)
	}
	return out, nil
}

func (s *Store) GetChannel(id string) (*Channel, error) {
	row := s.db.QueryRow(`SELECT `+channelCols+` FROM channels WHERE id=?`, id)
	var c Channel
	var ca string
	if err := row.Scan(&c.ID, &c.DeviceID, &c.Name, &c.Source, &c.Status,
		&c.Transcode, &c.Mode, &c.ActiveMode, &c.Room, &c.IngressID, &c.StreamKey, &ca); err != nil {
		return nil, err
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	return &c, nil
}

func (s *Store) DeleteChannel(id string) error {
	_, err := s.db.Exec(`DELETE FROM channels WHERE id=?`, id)
	return err
}

// ── 凭证加密 (AES-256-GCM) ──

// key 由 CAMINGRESS_SECRET 派生, 未设置时使用开发默认值 (生产必须设置)
func (s *Store) key() []byte {
	secret := os.Getenv("CAMINGRESS_SECRET")
	if secret == "" {
		secret = "camingress-dev-secret"
	}
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

func (s *Store) encrypt(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	block, err := aes.NewCipher(s.key())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return "enc:" + base64.StdEncoding.EncodeToString(ct), nil
}

func (s *Store) decrypt(enc string) string {
	if enc == "" {
		return ""
	}
	if !strings.HasPrefix(enc, "enc:") {
		return "" // 旧版明文, 出于安全不返回
	}
	raw, err := base64.StdEncoding.DecodeString(enc[4:])
	if err != nil {
		return ""
	}
	block, err := aes.NewCipher(s.key())
	if err != nil {
		return ""
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ""
	}
	nonceSize := gcm.NonceSize()
	if len(raw) < nonceSize {
		return ""
	}
	pt, err := gcm.Open(nil, raw[:nonceSize], raw[nonceSize:], nil)
	if err != nil {
		return ""
	}
	return string(pt)
}
