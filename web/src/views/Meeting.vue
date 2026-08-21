<template>
  <div>
    <h2>多人会议</h2>
    <div class="toolbar">
      <input v-model="roomName" placeholder="房间名 (如 dev-room)" />
      <button v-if="!joined" @click="join">▶ 加入 (开启摄像头/麦克风)</button>
      <button v-else class="danger" @click="leave">⏻ 离开</button>
    </div>
    <p v-if="error" class="error">{{ error }}</p>
    <div class="grid">
      <div class="tile">
        <video ref="localVideo" autoplay playsinline muted></video>
        <div class="label">我 (本地)</div>
      </div>
      <div class="tile" v-for="t in remoteTiles" :key="t.sid">
        <video :ref="(el) => bindVideo(el, t)" autoplay playsinline></video>
        <div class="label">{{ t.name }}</div>
      </div>
    </div>
    <p v-if="joined && !remoteTiles.length" class="hint">等待其他参会者加入...</p>
  </div>
</template>

<script>
import { ref, onUnmounted } from 'vue'
import api from '../api/client'
import { Room, RoomEvent, Track } from 'livekit-client'

export default {
  setup() {
    const roomName = ref('')
    const joined = ref(false)
    const error = ref('')
    const localVideo = ref(null)
    const remoteTiles = ref([]) // [{ sid, name, stream }]
    let room = null

    function bindVideo(el, tile) {
      if (el) el.srcObject = tile.stream
    }

    function getTile(participant) {
      let t = remoteTiles.value.find((x) => x.sid === participant.sid)
      if (!t) {
        t = { sid: participant.sid, name: participant.name || participant.identity, stream: new MediaStream() }
        remoteTiles.value.push(t)
      }
      return t
    }

    async function join() {
      error.value = ''
      if (!roomName.value) {
        error.value = '请输入房间名'
        return
      }
      try {
        const identity = 'user-' + Date.now()
        const r = await api.get('/api/token', {
          params: { room: roomName.value, identity, role: 'publisher' }
        })
        const { token, url } = r.data

        room = new Room()

        room.on(RoomEvent.TrackSubscribed, (track, pub, participant) => {
          if (track.kind === 'video') {
            const t = getTile(participant)
            t.stream.addTrack(track.mediaStreamTrack)
            t.name = participant.name || participant.identity
          }
        })
        room.on(RoomEvent.TrackUnsubscribed, (track, pub, participant) => {
          const t = remoteTiles.value.find((x) => x.sid === participant.sid)
          if (t) {
            t.stream.removeTrack(track.mediaStreamTrack)
            if (t.stream.getVideoTracks().length === 0) {
              remoteTiles.value = remoteTiles.value.filter((x) => x.sid !== participant.sid)
            }
          }
        })
        room.on(RoomEvent.LocalTrackPublished, (pub) => {
          if (pub.source === Track.Source.Camera && localVideo.value) {
            localVideo.value.srcObject = new MediaStream([pub.track.mediaStreamTrack])
          }
        })
        room.on(RoomEvent.Disconnected, () => { joined.value = false })

        await room.connect(url, token)
        await room.localParticipant.setCameraEnabled(true)
        await room.localParticipant.setMicrophoneEnabled(true)
        joined.value = true
      } catch (e) {
        error.value = '加入失败: ' + (e.message || e) + '（请检查摄像头/麦克风权限）'
      }
    }

    async function leave() {
      if (room) {
        await room.disconnect()
        room = null
      }
      remoteTiles.value = []
      if (localVideo.value) localVideo.value.srcObject = null
      joined.value = false
    }

    onUnmounted(() => { if (room) room.disconnect() })

    return { roomName, joined, error, localVideo, remoteTiles, bindVideo, join, leave }
  }
}
</script>

<style scoped>
.toolbar { display: flex; gap: 8px; margin: 16px 0; }
input { padding: 8px 12px; background: #1a1d23; border: 1px solid #374151; border-radius: 6px; color: #e5e7eb; flex: 1; }
button { padding: 8px 14px; background: #2563eb; color: #fff; border: none; border-radius: 6px; cursor: pointer; }
button.danger { background: #dc2626; }
.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 16px; margin-top: 16px; }
.tile { background: #000; border-radius: 8px; overflow: hidden; position: relative; aspect-ratio: 16/9; }
.tile video { width: 100%; height: 100%; object-fit: cover; }
.label { position: absolute; bottom: 8px; left: 8px; background: rgba(0,0,0,.7); padding: 2px 8px; border-radius: 4px; font-size: 12px; }
.hint { color: #6b7280; margin-top: 16px; }
.error { color: #ef4444; margin-top: 16px; }
</style>
