<template>
  <div class="layout">
    <aside class="sidebar">
      <h2>🎥 视频汇聚</h2>
      <nav>
        <router-link to="/channels">📡 通道管理</router-link>
        <router-link to="/discovery">🔍 设备发现</router-link>
        <router-link to="/meeting">👥 会议</router-link>
      </nav>
    </aside>
    <main class="content">
      <router-view />
    </main>
  </div>
</template>

<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
body { background: #111827; color: #e5e7eb; font-family: system-ui, -apple-system, 'Segoe UI', sans-serif; }
.layout { display: flex; min-height: 100vh; }
.sidebar { width: 200px; background: #1a1d23; padding: 20px 16px; flex-shrink: 0; }
.sidebar h2 { font-size: 16px; margin-bottom: 20px; color: #60a5fa; }
.sidebar nav a { display: block; padding: 10px 12px; margin: 4px 0; color: #9ca3af; text-decoration: none; border-radius: 6px; font-size: 14px; }
.sidebar nav a:hover, .sidebar nav a.router-link-active { background: #2563eb; color: #fff; }
.content { flex: 1; padding: 24px; min-width: 0; }
</style>
