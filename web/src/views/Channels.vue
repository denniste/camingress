<template>
  <div>
    <h2>通道管理</h2>
    <div class="toolbar">
      <input v-model="newChannel.name" placeholder="通道名称" />
      <input v-model="newChannel.source" placeholder="rtsp://user:pass@ip:554/stream" class="wide" />
      <button @click="addChannel">＋ 添加</button>
    </div>
    <table>
      <thead><tr><th>名称</th><th>源</th><th>房间</th><th>状态</th><th>操作</th></tr></thead>
      <tbody>
        <tr v-for="ch in channels" :key="ch.id">
          <td>{{ ch.name }}</td>
          <td class="mono">{{ ch.source }}</td>
          <td class="mono">{{ ch.room }}</td>
          <td><span :class="['dot', ch.status]"></span>{{ ch.status }}</td>
          <td>
            <button v-if="ch.status !== 'active'" @click="start(ch)">▶ 启动</button>
            <button v-else @click="stop(ch)">■ 停止</button>
            <button :disabled="ch.status !== 'active'" @click="watch(ch)">👁 观看</button>
            <button class="danger" @click="remove(ch)">✕</button>
          </td>
        </tr>
      </tbody>
    </table>
    <p v-if="!channels.length" class="hint">暂无通道，请先「＋ 添加」或在「设备发现」页一键入库</p>
    <p v-if="error" class="error">{{ error }}</p>
  </div>
</template>

<script>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api/client'

export default {
  setup() {
    const router = useRouter()
    const channels = ref([])
    const newChannel = ref({ name: '', source: '' })
    const error = ref('')

    async function load() {
      try {
        const r = await api.get('/api/channels')
        channels.value = r.data
        error.value = ''
      } catch (e) {
        error.value = '加载通道失败: ' + (e.response?.data?.error || e.message)
      }
    }
    async function addChannel() {
      if (!newChannel.value.name || !newChannel.value.source) return
      try {
        await api.post('/api/channels', newChannel.value)
        newChannel.value = { name: '', source: '' }
        await load()
      } catch (e) {
        error.value = '添加失败: ' + (e.response?.data?.error || e.message)
      }
    }
    async function start(ch) {
      try {
        await api.post(`/api/channels/${ch.id}/start`)
        await load()
      } catch (e) {
        error.value = '启动失败: ' + (e.response?.data?.error || e.message)
      }
    }
    async function stop(ch) {
      try {
        await api.post(`/api/channels/${ch.id}/stop`)
        await load()
      } catch (e) {
        error.value = '停止失败: ' + (e.response?.data?.error || e.message)
      }
    }
    async function remove(ch) {
      try {
        await api.delete(`/api/channels/${ch.id}`)
        await load()
      } catch (e) {
        error.value = '删除失败: ' + (e.response?.data?.error || e.message)
      }
    }
    function watch(ch) {
      router.push(`/player/${encodeURIComponent(ch.room || ch.name)}`)
    }

    onMounted(load)
    return { channels, newChannel, error, addChannel, start, stop, remove, watch }
  }
}
</script>

<style scoped>
.toolbar { display: flex; gap: 8px; margin: 16px 0; }
input { padding: 8px 12px; background: #1a1d23; border: 1px solid #374151; border-radius: 6px; color: #e5e7eb; }
input.wide { flex: 1; }
button { padding: 8px 14px; background: #2563eb; color: #fff; border: none; border-radius: 6px; cursor: pointer; }
button:disabled { background: #374151; cursor: not-allowed; }
button.danger { background: #dc2626; }
table { width: 100%; border-collapse: collapse; }
th, td { text-align: left; padding: 10px 12px; border-bottom: 1px solid #1f2937; font-size: 14px; }
.mono { font-family: monospace; font-size: 12px; color: #9ca3af; }
.dot { display: inline-block; width: 8px; height: 8px; border-radius: 50%; margin-right: 6px; }
.dot.active { background: #22c55e; }
.dot.stopped { background: #6b7280; }
.dot.error { background: #ef4444; }
.hint { color: #6b7280; margin-top: 16px; }
.error { color: #ef4444; margin-top: 16px; }
</style>
