#!/usr/bin/env bash
# ============================================================
# CamIngress - Windows 全原生一键启动 (零 Docker)
# 依赖: Memurai(Redis 服务) / livekit-server / ingress(自构建+GStreamer) / mediamtx / camingress / node
# 用法: bash deploy/start-native.sh
# 停止: bash deploy/stop-native.sh
# ============================================================
cd "$(dirname "$0")/.." || exit 1
ROOT=$(pwd -W 2>/dev/null || pwd)
LOGDIR="$ROOT/logs"
mkdir -p "$LOGDIR"

log()  { echo "[$(date +%H:%M:%S)] $*"; }
port() { netstat -ano 2>/dev/null | grep -E ":$1\s" | grep -i listen | head -1; }
wait_port() { # $1=port $2=name $3=timeout(s)
  for _ in $(seq 1 "$3"); do
    [ -n "$(port $1)" ] && { log "  ✓ $2 (:${1}) 就绪"; return 0; }
    sleep 1
  done
  log "  ✗ $2 (:${1}) 超时未就绪"
  return 1
}

# ---- 0. 检测宿主 LAN IP 并写入 livekit/ingress 配置 (DHCP 会变!) ----
HOST_IP=$(powershell -NoProfile -Command "(Get-NetIPAddress -AddressFamily IPv4 -ErrorAction SilentlyContinue | Where-Object {\$_.IPAddress -notlike '127.*' -and \$_.IPAddress -notlike '169.*' -and \$_.IPAddress -notlike '198.18.*'} | Select-Object -First 1 -ExpandProperty IPAddress)" 2>/dev/null | tr -d '\r')
IP_CHANGED=0
if [ -n "$HOST_IP" ]; then
  OLD_IP=$(grep -oE "node_ip: [0-9.]+" D:/CamIngress/deploy/livekit-native.yaml | awk '{print $2}')
  if [ "$OLD_IP" != "$HOST_IP" ]; then
    sed -i "s/node_ip: .*/node_ip: $HOST_IP/" D:/CamIngress/deploy/livekit-native.yaml
    sed -i "s|ws_url: ws://[0-9.]*:7880|ws_url: ws://$HOST_IP:7880|" D:/CamIngress/deploy/ingress.yaml
    IP_CHANGED=1
    log "[0/6] 宿主 IP 变化: $OLD_IP → $HOST_IP (配置已更新, 将强制重启 livekit)"
  else
    log "[0/6] 宿主 IP: $HOST_IP (无变化)"
  fi
else
  log "[0/6] ⚠ 未能检测宿主 IP, 使用配置中的现有值"
fi

# ---- 1. Memurai (Redis 兼容服务) ----
if [ -n "$(port 6379)" ]; then
  log "[1/6] Redis(Memurai) 已在运行"
else
  log "[1/6] 启动 Memurai..."
  # 优先用服务方式 (无需提权时直接成功)
  powershell -NoProfile -Command "Start-Service Memurai -ErrorAction SilentlyContinue" 2>/dev/null
  sleep 3
  if [ -z "$(port 6379)" ]; then
    # 服务方式失败 (通常因无管理员权限) → 独立进程模式, 免 UAC 免提权
    log "  (服务启动失败, 改用独立进程模式运行 memurai.exe)"
    # cwd 用安装目录: memurai.conf 的 dir ./ 与 logfile 相对路径都落在安装目录
    (cd "/c/Program Files/Memurai" && nohup ./memurai.exe memurai.conf >"$LOGDIR/memurai.log" 2>&1 &)
    sleep 2
  fi
  if [ -n "$(port 6379)" ]; then
    log "  ✓ Redis :6379 就绪"
  else
    log "  ✗ Redis 启动失败 (手动: services.msc 启动 Memurai 或运行 memurai.exe)"
  fi
fi
  # ---- 2. livekit-server (原生) ----
  if [ -n "$(port 7880)" ] && [ "$IP_CHANGED" = "0" ]; then
    log "[2/6] livekit-server 已在运行"
  else
    [ "$IP_CHANGED" = "1" ] && log "[2/6] 强制重启 livekit-server (IP 变化)..."
    powershell -NoProfile -Command "Get-Process -Name livekit-server -ErrorAction SilentlyContinue | Stop-Process -Force" 2>/dev/null
    sleep 2
    log "[2/6] 启动 livekit-server..."
    nohup /d/livekit-native/livekit-server.exe --config D:/CamIngress/deploy/livekit-native.yaml >"$LOGDIR/livekit.log" 2>&1 &
    wait_port 7880 "livekit-server" 15
  fi

  # ---- 3. ingress (Docker 容器: 完整视频唯一路径; 原生 build 视频轨被 GStreamer bug 卡死) ----
  DOCKER="/c/Program Files/Docker/Docker/resources/bin/docker.exe"
  if [ -n "$(port 1935)" ] && [ "$IP_CHANGED" = "0" ]; then
    log "[3/6] ingress 已在运行"
  else
    [ "$IP_CHANGED" = "1" ] && log "[3/6] 重启 ingress 容器 (IP 变化)..."
    "$DOCKER" restart deploy-ingress-1 >/dev/null 2>&1 || "$DOCKER" start deploy-ingress-1 >/dev/null 2>&1
    wait_port 1935 "ingress-RTMP" 20
    wait_port 7888 "ingress-health" 5
  fi

# ---- 4. mediamtx (原生) ----
if [ -n "$(port 8554)" ]; then
  log "[4/6] mediamtx 已在运行"
else
  log "[4/6] 启动 mediamtx..."
  # cwd 用 mediamtx 目录: auto.crt/auto.key (MoQ 证书) 会写到 cwd, 避免污染仓库根
  (cd /d/mediamtx-native && nohup ./mediamtx.exe D:/CamIngress/deploy/mediamtx.yaml >"$LOGDIR/mediamtx.log" 2>&1 &)
  wait_port 8554 "mediamtx-RTSP" 10
fi

# ---- 5. camingress (Go 中间层) ----
if [ -n "$(port 8080)" ]; then
  log "[5/6] camingress 已在运行"
else
  log "[5/6] 启动 camingress..."
  # 关键: cwd 必须是 server/ (SQLite 相对路径 camingress.db 在 server 目录)
  (cd /d/CamIngress/server && nohup ./camingress.exe -addr :8080 >"$LOGDIR/camingress.log" 2>&1 &)
  wait_port 8080 "camingress" 10
fi

# ---- 5.5 恢复上次活跃的通道 (DB 状态=active 的重新拉起) ----
if [ -n "$(port 7880)" ] && [ -n "$(port 1935)" ]; then
  log "[5.5] 恢复活跃通道..."
  sleep 2
  CHANNELS=$(curl -s --max-time 5 http://localhost:8080/api/channels 2>/dev/null | python -c "
import json,sys
try:
    for c in json.load(sys.stdin):
        if c.get('status') == 'active':
            print(c['id'])
except Exception:
    pass" 2>/dev/null | tr -d '\r')
  for id in $CHANNELS; do
    curl -s -X POST -o /dev/null "http://localhost:8080/api/channels/$id/stop"
    sleep 1
    curl -s -X POST -o /dev/null "http://localhost:8080/api/channels/$id/start"
    # 验证推流起来, 失败则等 3s 重试一次 (启动竞态: 摄像头握手/文件占用可能瞬时失败)
    sleep 3
    PUSH_OK=$(curl -s --max-time 5 http://localhost:8080/api/status 2>/dev/null | python -c "
import json,sys
try:
    d = json.load(sys.stdin)
    for c in d.get('channels', []):
        if c.get('id') == '$id' and c.get('push_running'):
            print('yes'); break
except Exception: pass" 2>/dev/null)
    if [ "$PUSH_OK" = "yes" ]; then
      log "  ✓ 通道 $id 已恢复"
    else
      curl -s -X POST -o /dev/null "http://localhost:8080/api/channels/$id/stop"
      sleep 3
      curl -s -X POST -o /dev/null "http://localhost:8080/api/channels/$id/start"
      sleep 3
      log "  ✓ 通道 $id 已恢复(重试)"
    fi
  done
else
  log "[5.5] 跳过通道恢复 (livekit/ingress 未就绪)"
fi

# ---- 6. 前端 (Vite dev) ----
if [ -n "$(port 5173)" ]; then
  log "[6/6] Vite 已在运行"
else
  log "[6/6] 启动前端 Vite..."
  (cd /d/CamIngress/web && nohup npm run dev >"$LOGDIR/vite.log" 2>&1 &)
  wait_port 5173 "vite" 20
fi

log ""
log "=============================================="
log "  端口总览:"
for p in 6379 7880 1935 8082 7888 8554 9997 8080 5173; do
  [ -n "$(port $p)" ] && log "    ✓ :$p" || log "    ✗ :$p"
done
log "  前端: http://localhost:5173/dashboard"
log "  日志: $LOGDIR"
log "  停止: bash deploy/stop-native.sh"
log "=============================================="
