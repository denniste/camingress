import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import api from '../api'

export default function Channels() {
  const navigate = useNavigate()
  const [channels, setChannels] = useState([])
  const [name, setName] = useState('')
  const [source, setSource] = useState('')
  const [error, setError] = useState('')

  async function load() {
    try {
      const r = await api.get('/api/channels')
      setChannels(r.data)
      setError('')
    } catch (e) {
      setError('加载通道失败: ' + (e.response?.data?.error || e.message))
    }
  }
  useEffect(() => { load() }, [])

  async function addChannel() {
    if (!name || !source) return
    try {
      await api.post('/api/channels', { name, source })
      setName(''); setSource('')
      await load()
    } catch (e) {
      setError('添加失败: ' + (e.response?.data?.error || e.message))
    }
  }
  async function start(ch) {
    try { await api.post(`/api/channels/${ch.id}/start`); await load() }
    catch (e) { setError('启动失败: ' + (e.response?.data?.error || e.message)) }
  }
  async function stop(ch) {
    try { await api.post(`/api/channels/${ch.id}/stop`); await load() }
    catch (e) { setError('停止失败: ' + (e.response?.data?.error || e.message)) }
  }
  async function remove(ch) {
    try { await api.delete(`/api/channels/${ch.id}`); await load() }
    catch (e) { setError('删除失败: ' + (e.response?.data?.error || e.message)) }
  }
  function watch(ch) {
    navigate(`/player/${encodeURIComponent(ch.room || ch.name)}`)
  }

  return (
    <div>
      <h2>通道管理</h2>
      <div className="toolbar">
        <input value={name} onChange={(e) => setName(e.target.value)} placeholder="通道名称" />
        <input className="wide" value={source} onChange={(e) => setSource(e.target.value)} placeholder="rtsp://user:pass@ip:554/stream" />
        <button onClick={addChannel}>＋ 添加</button>
      </div>
      <table>
        <thead>
          <tr><th>名称</th><th>源</th><th>房间</th><th>状态</th><th>操作</th></tr>
        </thead>
        <tbody>
          {channels.map((ch) => (
            <tr key={ch.id}>
              <td>{ch.name}</td>
              <td className="mono">{ch.source}</td>
              <td className="mono">{ch.room}</td>
              <td><span className={'dot ' + ch.status}></span>{ch.status}</td>
              <td>
                {ch.status !== 'active'
                  ? <button className="small" onClick={() => start(ch)}>▶ 启动</button>
                  : <button className="small" onClick={() => stop(ch)}>■ 停止</button>}
                <button className="small" disabled={ch.status !== 'active'} onClick={() => watch(ch)}>👁 观看</button>
                <button className="small danger" onClick={() => remove(ch)}>✕</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {channels.length === 0 && <p className="hint">暂无通道，请先「＋ 添加」或在「设备发现」页一键入库</p>}
      {error && <p className="error">{error}</p>}
    </div>
  )
}
