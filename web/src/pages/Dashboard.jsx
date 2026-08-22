import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import api from '../api'

export default function Dashboard() {
  const navigate = useNavigate()
  const [services, setServices] = useState([])
  const [channels, setChannels] = useState([])
  const [alerts, setAlerts] = useState([])
  const [updatedAt, setUpdatedAt] = useState('-')
  const [live, setLive] = useState(false)
  const [error, setError] = useState('')
  const [tick, setTick] = useState(0)
  const timer = useRef(null)

  useEffect(() => {
    let alive = true
    async function load() {
      try {
        const r = await api.get('/api/status')
        const d = r.data
        if (!alive) return
        setLive(true)
        const hasActiveFfmpeg = d.channels.some((c) => c.status === 'active' && c.active_mode === 'ffmpeg')
        setServices([
          { name: 'LiveKit', up: d.services.livekit.up, detail: '7880' },
          { name: 'MediaMTX', up: d.services.mediamtx.up, detail: '9997' },
          { name: 'FFmpeg', up: !hasActiveFfmpeg || d.services.ffmpeg_procs > 0, detail: d.services.ffmpeg_procs + ' 进程' },
          { name: '告警', up: d.alerts.length === 0, detail: d.alerts.length + ' 条' },
        ])
        setChannels(d.channels)
        setAlerts(d.alerts)
        setUpdatedAt(new Date().toLocaleTimeString())
        setError('')
      } catch (e) {
        if (!alive) return
        setLive(false)
        setError('加载状态失败: ' + (e.response?.data?.error || e.message))
      }
    }
    load()
    timer.current = setInterval(() => { load(); setTick(Date.now()) }, 5000)
    return () => { alive = false; clearInterval(timer.current) }
  }, [])

  async function start(ch) {
    try { await api.post(`/api/channels/${ch.id}/start`); setTick(Date.now()) }
    catch (e) { setError('启动失败: ' + (e.response?.data?.error || e.message)) }
  }
  async function stop(ch) {
    try { await api.post(`/api/channels/${ch.id}/stop`); setTick(Date.now()) }
    catch (e) { setError('停止失败: ' + (e.response?.data?.error || e.message)) }
  }
  async function remove(ch) {
    try { await api.delete(`/api/channels/${ch.id}`); setTick(Date.now()) }
    catch (e) { setError('删除失败: ' + (e.response?.data?.error || e.message)) }
  }
  function watch(ch) {
    navigate(`/player/${encodeURIComponent(ch.room || ch.name)}`)
  }
  function snapUrl(ch) {
    return `/api/channels/${ch.id}/snapshot?t=${tick}`
  }

  return (
    <div>
      <div className="head">
        <h2 style={{ margin: 0 }}>🖥 控制台</h2>
        <span className="muted">更新于 {updatedAt}</span>
        <span className={live ? 'live on' : 'live'}>● {live ? '实时' : '离线'}</span>
      </div>

      <div className="svc-grid">
        {services.map((s) => (
          <div key={s.name} className={s.up ? 'svc-card up' : 'svc-card down'}>
            <span className={s.up ? 'svc-dot ok' : 'svc-dot bad'}></span>
            <span className="svc-name">{s.name}</span>
            <span className="svc-detail">{s.detail}</span>
          </div>
        ))}
      </div>

      {alerts.length > 0 && (
        <div className="alerts">
          {alerts.map((a, i) => (
            <div key={i} className="alert">⚠️ {a}</div>
          ))}
        </div>
      )}

      <h3>通道 ({channels.length})</h3>
      <div className="grid">
        {channels.map((ch) => (
          <div key={ch.id} className={ch.status === 'error' ? 'tile error' : 'tile'}>
            <div className="thumb">
              {ch.status === 'active' ? (
                <img src={snapUrl(ch)} alt={ch.name}
                  onError={(e) => { e.target.style.visibility = 'hidden' }} />
              ) : (
                <div className="placeholder">⏸ 未启动</div>
              )}
              <span className={`badge ${ch.status}`}>{ch.status}</span>
              {ch.status === 'active' && <span className="mode-badge">{ch.active_mode}</span>}
            </div>
            <div className="tile-info">
              <div className="tile-name">{ch.name}</div>
              <div className="tile-room mono">{ch.room}</div>
            </div>
            <div className="tile-ops">
              {ch.status !== 'active'
                ? <button className="small" onClick={() => start(ch)}>▶ 启动</button>
                : <button className="small" onClick={() => stop(ch)}>■ 停止</button>}
              <button className="small" disabled={ch.status !== 'active'} onClick={() => watch(ch)}>👁 观看</button>
              <button className="small danger" onClick={() => remove(ch)}>✕</button>
            </div>
          </div>
        ))}
        {channels.length === 0 && <div className="hint">暂无通道 — 到「通道管理」添加，或在「设备发现」一键入库</div>}
      </div>
      {error && <p className="error">{error}</p>}

      <style>{`
        .head { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
        .live { font-size: 12px; color: #6b7280; }
        .live.on { color: #22c55e; }
        .svc-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 12px; margin-bottom: 20px; }
        .svc-card { display: flex; align-items: center; gap: 10px; padding: 14px 16px; background: #1a1d23; border-radius: 8px; border: 1px solid #1f2937; }
        .svc-card.up { border-color: #164e36; }
        .svc-card.down { border-color: #7f1d1d; }
        .svc-dot { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }
        .svc-dot.ok { background: #22c55e; }
        .svc-dot.bad { background: #ef4444; }
        .svc-name { font-size: 14px; font-weight: 600; }
        .svc-detail { margin-left: auto; font-size: 12px; color: #9ca3af; }
        .alerts { margin-bottom: 20px; }
        .alert { background: #7f1d1d33; border: 1px solid #7f1d1d; color: #fca5a5; padding: 8px 12px; border-radius: 6px; margin-bottom: 6px; font-size: 13px; }
        h3 { margin: 8px 0 12px; font-size: 15px; }
        .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 14px; }
        .tile { background: #1a1d23; border-radius: 10px; overflow: hidden; border: 1px solid #1f2937; }
        .tile.error { border-color: #7f1d1d; }
        .thumb { position: relative; aspect-ratio: 16/9; background: #000; }
        .thumb img { width: 100%; height: 100%; object-fit: cover; display: block; }
        .placeholder { display: flex; align-items: center; justify-content: center; height: 100%; color: #4b5563; font-size: 14px; }
        .badge { position: absolute; top: 8px; left: 8px; padding: 2px 8px; border-radius: 4px; font-size: 11px; background: #374151; }
        .badge.active { background: #166534; color: #bbf7d0; }
        .badge.error { background: #7f1d1d; color: #fecaca; }
        .mode-badge { position: absolute; top: 8px; right: 8px; padding: 2px 8px; border-radius: 4px; font-size: 11px; background: #1e3a8a; color: #bfdbfe; }
        .tile-info { padding: 10px 12px 4px; }
        .tile-name { font-size: 14px; font-weight: 600; }
        .tile-room { font-size: 12px; color: #6b7280; }
        .tile-ops { display: flex; gap: 6px; padding: 8px 12px 12px; }
      `}</style>
    </div>
  )
}
