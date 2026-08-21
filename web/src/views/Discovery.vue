<template>
  <div>
    <h2>ONVIF 设备发现</h2>
    <button @click="scan" :disabled="scanning">{{ scanning ? '扫描中...' : '🔍 扫描局域网' }}</button>
    <p v-if="error" class="error">{{ error }}</p>
    <table v-if="devices.length">
      <thead><tr><th>IP</th><th>厂商</th><th>RTSP</th><th>操作</th></tr></thead>
      <tbody>
        <tr v-for="d in devices" :key="d.ip">
          <td>{{ d.ip }}</td>
          <td>{{ d.vendor }}</td>
          <td class="mono">{{ d.rtsp_url || '未探测到' }}</td>
          <td><button :disabled="!d.rtsp_url" @click="addAsChannel(d)">＋ 加入通道</button></td>
        </tr>
      </tbody>
    </table>
    <p v-else-if="!scanning" class="hint">点击扫描发现局域网 ONVIF 摄像头（需与摄像头处于同一网段）</p>
  </div>
</template>

<script>
import { ref } from 'vue'
import api from '../api/client'

export default {
  setup() {
    const devices = ref([])
    const scanning = ref(false)
    const error = ref('')

    async function scan() {
      scanning.value = true
      error.value = ''
      try {
        const r = await api.post('/api/discover')
        devices.value = r.data
        if (!r.data.length) error.value = '未发现 ONVIF 设备'
      } catch (e) {
        error.value = '扫描失败: ' + (e.response?.data?.error || e.message)
      } finally {
        scanning.value = false
      }
    }
    async function addAsChannel(d) {
      try {
        await api.post('/api/channels', {
          name: (d.vendor || d.ip),
          source: d.rtsp_url,
          device_id: d.ip,
        })
        alert('已加入通道，请到「通道管理」页启动')
      } catch (e) {
        error.value = '入库失败: ' + (e.response?.data?.error || e.message)
      }
    }
    return { devices, scanning, error, scan, addAsChannel }
  }
}
</script>

<style scoped>
button { padding: 10px 16px; background: #2563eb; color: #fff; border: none; border-radius: 6px; cursor: pointer; margin-bottom: 16px; }
button:disabled { background: #374151; cursor: not-allowed; }
table { width: 100%; border-collapse: collapse; }
th, td { text-align: left; padding: 10px 12px; border-bottom: 1px solid #1f2937; font-size: 14px; }
.mono { font-family: monospace; font-size: 12px; color: #9ca3af; }
.hint { color: #6b7280; margin-top: 16px; }
.error { color: #ef4444; margin-top: 16px; }
</style>
