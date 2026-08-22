import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { LiveKitRoom, RoomAudioRenderer, useTracks, VideoTrack } from '@livekit/components-react'
import { Track } from 'livekit-client'
import api from '../api'

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

export default function Player() {
  const { room } = useParams()
  const [conn, setConn] = useState(null)

  useEffect(() => {
    let alive = true
    setConn(null)
    const identity = 'viewer-' + Date.now()
    api
      .get('/api/token', { params: { room, identity, role: 'viewer' } })
      .then((r) => { if (alive) setConn({ token: r.data.token, url: r.data.url }) })
      .catch((e) => { if (alive) setConn({ error: e.response?.data?.error || e.message }) })
    return () => { alive = false }
  }, [room])

  if (conn?.error) return <div className="error">连接失败: {conn.error}</div>
  if (!conn) return <div>获取 token...</div>

  return (
    <div>
      <h2>实时观看</h2>
      <LiveKitRoom
        serverUrl={conn.url}
        token={conn.token}
        connect={true}
        audio={false}
        video={true}
        style={{ height: '70vh' }}
      >
        <RoomAudioRenderer />
        <CameraTracks />
      </LiveKitRoom>
      <p className="hint">房间: {room}</p>
      <style>{`
        .stage { display: grid; grid-template-columns: 1fr; gap: 8px; background: #000; border-radius: 8px; overflow: hidden; height: 100%; }
        .stage video { width: 100%; height: 100%; object-fit: contain; }
        .status-tip { display: flex; align-items: center; justify-content: center; height: 100%; color: #6b7280; }
      `}</style>
    </div>
  )
}
