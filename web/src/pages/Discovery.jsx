import { useState } from 'react'
import api from '../api'

export default function Discovery() {
  const [devices, setDevices] = useState([])
  const [scanning, setScanning] = useState(false)
  const [error, setError] = useState('')

  async function scan() {
    setScanning(true)
    setError('')
    try {
      const r = await api.post('/api/discover')
      setDevices(r.data)
      if (!r.data.length) setError('未发现 ONVIF 设备')
    } catch (e) {
      setError('扫描失败: ' + (e.response?.data?.error || e.message))
    } finally {
      setScanning(false)
    }
  }
  async function addAsChannel(d) {
    try {
      await api.post('/api/channels', {
        name: d.vendor || d.ip,
        source: d.rtsp_url,
        device_id: d.ip,
      })
      window.alert('已加入通道，请到「通道管理」页启动')
    } catch (e) {
      setError('入库失败: ' + (e.response?.data?.error || e.message))
    }
  }

  return (
    <div>
      <h2>ONVIF 设备发现</h2>
      <button onClick={scan} disabled={scanning} style={{ marginBottom: 16 }}>
        {scanning ? '扫描中...' : '🔍 扫描局域网'}
      </button>
      {error && <p className="error">{error}</p>}
      {devices.length > 0 && (
        <table>
          <thead>
            <tr><th>IP</th><th>厂商</th><th>RTSP</th><th>操作</th></tr>
          </thead>
          <tbody>
            {devices.map((d) => (
              <tr key={d.ip}>
                <td>{d.ip}</td>
                <td>{d.vendor}</td>
                <td className="mono">{d.rtsp_url || '未探测到'}</td>
                <td>
                  <button className="small" disabled={!d.rtsp_url} onClick={() => addAsChannel(d)}>＋ 加入通道</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {devices.length === 0 && !scanning && (
        <p className="hint">点击扫描发现局域网 ONVIF 摄像头（需与摄像头处于同一网段）</p>
      )}
    </div>
  )
}
