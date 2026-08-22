#!/usr/bin/env bash
# ============================================================
# CamIngress - Windows 全原生一键停止
# 用法: bash deploy/stop-native.sh
# ============================================================
cd "$(dirname "$0")/.." || exit 1

log() { echo "[$(date +%H:%M:%S)] $*"; }
port() { netstat -ano 2>/dev/null | grep -E ":$1\s" | grep -i listen | head -1; }

log "停止 CamIngress 全部组件..."

# ---- 1. 按进程名停止 (camingress/mediamtx/livekit-server/ingress/ffmpeg/memurai 独立进程) ----
powershell -NoProfile -Command "Stop-Process -Name camingress,mediamtx,livekit-server,ingress,ffmpeg,memurai -Force -ErrorAction SilentlyContinue" 2>/dev/null
log "  ✓ camingress / mediamtx / livekit-server / ingress / ffmpeg / memurai(进程) 已停止"

# ---- 2. Vite 按端口停止 (避免误杀其他 node 进程) ----
VPID=$(port 5173 | awk '{print $NF}')
if [ -n "$VPID" ]; then
  powershell -NoProfile -Command "Stop-Process -Id $VPID -Force -ErrorAction SilentlyContinue" 2>/dev/null
  log "  ✓ Vite (PID $VPID) 已停止"
else
  log "  ✓ Vite 未在运行"
fi

# ---- 3. Memurai 服务 (若以服务方式运行, 需要 UAC) ----
if powershell -NoProfile -Command "Get-Service Memurai -ErrorAction SilentlyContinue | Select -Expand Status" 2>/dev/null | grep -q Running; then
  log "停止 Memurai 服务 (需要 UAC 确认)..."
  powershell -NoProfile -Command "Start-Process powershell -ArgumentList '-NoProfile','-Command','Stop-Service Memurai -Force' -Verb RunAs -Wait" 2>/dev/null
  sleep 2
  powershell -NoProfile -Command "Get-Service Memurai -ErrorAction SilentlyContinue | Select -Expand Status" 2>/dev/null | grep -q Running && log "  ✗ Memurai 仍在运行(可能拒绝了UAC)" || log "  ✓ Memurai 服务已停止"
else
  log "  ✓ Memurai 服务未在运行"
fi

# ---- 4. 端口复查 ----
sleep 2
log "端口复查:"
ALL_FREE=1
for p in 8080 5173 7880 7881 8554 9997 1935 8082 6379; do
  if [ -n "$(port $p)" ]; then
    log "  ✗ :$p 仍监听"
    ALL_FREE=0
  fi
done
[ "$ALL_FREE" = "1" ] && log "  ✓ 全部端口已释放" || log "  提示: 如仍有残留, 手动检查: netstat -ano | findstr :8080"

log "完成。"
