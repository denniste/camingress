import { Outlet, NavLink } from 'react-router-dom'

export default function App() {
  return (
    <div className="layout">
      <aside className="sidebar">
        <h2>🎥 视频汇聚</h2>
        <nav>
          <NavLink to="/dashboard">🖥 控制台</NavLink>
          <NavLink to="/channels">📡 通道管理</NavLink>
          <NavLink to="/discovery">🔍 设备发现</NavLink>
          <NavLink to="/meeting">👥 会议</NavLink>
        </nav>
      </aside>
      <main className="content">
        <Outlet />
      </main>
    </div>
  )
}
