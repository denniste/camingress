import { useEffect, useState } from 'react'
import { LiveKitRoom, RoomAudioRenderer, useTracks, VideoTrack } from '@livekit/components-react'
import { Track } from 'livekit-client'
import api from '../api'

const AGENT_ROOM = 'agent-demo'

function CameraTracks() {
  const tracks = useTracks([{ source: Track.Source.Camera, withPlaceholder: false }])
  if (!tracks.length) return <div className="status-tip">等待视频轨道...</div>
  return (
    <div className="stage">
      {tracks.map((t) => (
        <VideoTrack key={t.participant.identity} trackRef={t} />
      ))}
    </div>
  )
}

export default function Intercom() {
  const [conn, setConn] = useState(null)
  const [subs, setSubs] = useState([])

  useEffect(() => {
    let alive = true
    setConn(null)
    const identity = 'intercom-' + Date.now()
    api
      .get('/api/token', { params: { room: AGENT_ROOM, identity, role: 'viewer' } })
      .then((r) => { if (alive) setConn({ token: r.data.token, url: r.data.url }) })
      .catch((e) => { if (alive) setConn({ error: e.response?.data?.error || e.message }) })
    return () => { alive = false }
  }, [])

  if (conn?.error) return <div className="error">连接失败: {conn.error}</div>
  if (!conn) return <div>获取 token...</div>

  return (
    <div>
      <h2>AI 对讲 · {AGENT_ROOM}</h2>
      <LiveKitRoom
        serverUrl={conn.url}
        token={conn.token}
        connect={true}
        audio={true}
        video={false}
        style={{ height: '62vh' }}
      >
        <RoomAudioRenderer />
        <CameraTracks />
      </LiveKitRoom>
      <div className="subtitle-box">
        <h3>AI 对话字幕</h3>
        {subs.length === 0 && <p className="hint">按住/点击下方按钮说话，AI 值班员会回应你（麦克风已启用，直接说话即可）</p>}
        {subs.map((s, i) => (
          <div key={i} className={`sub-line ${s.fromAgent ? 'agent' : 'user'}`}>
            <span className="sub-who">{s.fromAgent ? 'AI' : '我'}:</span> {s.text}
          </div>
        ))}
      </div>
      <style>{`
        .stage { display: grid; grid-template-columns: 1fr; gap: 8px; background: #000; border-radius: 8px; overflow: hidden; height: 100%; }
        .stage video { width: 100%; height: 100%; object-fit: contain; }
        .status-tip { display: flex; align-items: center; justify-content: center; height: 100%; color: #6b7280; }
        .subtitle-box { margin-top: 12px; padding: 12px; background: var(--card, #111827); border-radius: 8px; min-height: 120px; }
        .sub-line { margin: 4px 0; }
        .sub-who { font-weight: 600; color: var(--accent, #22d3ee); }
        .sub-line.user .sub-who { color: #f59e0b; }
        .hint { color: #6b7280; }
      `}</style>
    </div>
  )
}
