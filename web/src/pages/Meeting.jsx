import { useState } from 'react'
import { VideoConference } from '@livekit/components-react'
import api from '../api'

export default function Meeting() {
  const [roomName, setRoomName] = useState('')
  const [conn, setConn] = useState(null)
  const [error, setError] = useState('')

  async function join() {
    setError('')
    if (!roomName) {
      setError('请输入房间名')
      return
    }
    try {
      const identity = 'user-' + Date.now()
      const r = await api.get('/api/token', { params: { room: roomName, identity, role: 'publisher' } })
      setConn({ token: r.data.token, url: r.data.url })
    } catch (e) {
      setError('加入失败: ' + (e.response?.data?.error || e.message))
    }
  }

  if (conn) {
    return (
      <div>
        <div className="toolbar">
          <span style={{ alignSelf: 'center' }}>房间: {roomName}</span>
          <button className="danger" onClick={() => setConn(null)}>⏻ 离开</button>
        </div>
        <div style={{ height: '75vh' }}>
          <VideoConference serverUrl={conn.url} token={conn.token} />
        </div>
      </div>
    )
  }

  return (
    <div>
      <h2>多人会议</h2>
      <div className="toolbar">
        <input value={roomName} onChange={(e) => setRoomName(e.target.value)} placeholder="房间名 (如 dev-room)" />
        <button onClick={join}>▶ 加入 (开启摄像头/麦克风)</button>
      </div>
      {error && <p className="error">{error}</p>}
      <p className="hint">加入后进入 LiveKit 视频会议（网格布局 / 静音 / 共享屏幕）</p>
    </div>
  )
}
