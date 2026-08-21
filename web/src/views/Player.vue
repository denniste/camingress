<template>
  <div>
    <h2>实时观看</h2>
    <div class="stage">
      <video ref="video" autoplay playsinline muted></video>
      <div class="status">{{ status }}</div>
    </div>
    <div class="info">
      <span>房间: {{ room }}</span>
      <button @click="connect">连接</button>
      <button @click="disconnect">断开</button>
    </div>
  </div>
</template>

<script>
import { ref, onMounted, onUnmounted } from 'vue'
import api from '../api/client'
import { Room, RoomEvent } from 'livekit-client'

export default {
  props: { room: { type: String, required: true } },
  setup(props) {
    const video = ref(null)
    const status = ref('未连接')
    let lkRoom = null
    let mediaStream = null

    function resetStream() {
      if (video.value) video.value.srcObject = null
      if (mediaStream) {
        mediaStream.getTracks().forEach((t) => t.stop())
        mediaStream = null
      }
    }

    async function connect() {
      status.value = '获取 token...'
      const identity = 'viewer-' + Date.now()
      const r = await api.get('/api/token', {
        params: { room: props.room, identity, role: 'viewer' }
      })
      const { token, url } = r.data

      resetStream()
      mediaStream = new MediaStream()
      if (video.value) video.value.srcObject = mediaStream

      lkRoom = new Room()
      lkRoom.on(RoomEvent.TrackSubscribed, (track) => {
        if (track.kind === 'video') {
          mediaStream.addTrack(track.mediaStreamTrack)
          status.value = '播放中 (WebRTC)'
        }
      })
      lkRoom.on(RoomEvent.TrackUnsubscribed, (track) => {
        mediaStream.removeTrack(track.mediaStreamTrack)
      })
      lkRoom.on(RoomEvent.Disconnected, () => { status.value = '已断开' })

      try {
        await lkRoom.connect(url, token)
        status.value = '已连接房间: ' + props.room
      } catch (e) {
        status.value = '连接失败: ' + (e.message || e)
      }
    }

    async function disconnect() {
      if (lkRoom) {
        await lkRoom.disconnect()
        lkRoom = null
      }
      resetStream()
      status.value = '未连接'
    }

    onMounted(() => { if (props.room) connect() })
    onUnmounted(() => { if (lkRoom) lkRoom.disconnect() })

    return { video, status, connect, disconnect }
  }
}
</script>

<style scoped>
.stage { background: #000; border-radius: 8px; min-height: 360px; display: flex; align-items: center; justify-content: center; overflow: hidden; position: relative; margin: 16px 0; }
video { width: 100%; max-height: 70vh; }
.status { position: absolute; top: 10px; left: 10px; background: rgba(0,0,0,.7); padding: 4px 10px; border-radius: 4px; font-size: 12px; color: #22c55e; }
.info { display: flex; gap: 12px; align-items: center; }
button { padding: 8px 14px; background: #2563eb; color: #fff; border: none; border-radius: 6px; cursor: pointer; }
</style>
