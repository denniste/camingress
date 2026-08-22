// 真实时间 CDP 截图: 等待 WebRTC 连接/解码后抓图
// 用法: node cdp-shot.mjs <port> <url> <outfile> [waitMs]
import { spawn } from 'node:child_process'
import http from 'node:http'
import { writeFileSync } from 'node:fs'

const DEBUG_PORT = process.argv[2] || '9226'
const URL = process.argv[3]
const OUT = process.argv[4]
const WAIT = parseInt(process.argv[5] || '15000', 10)
const CHROME = 'C:/Program Files/Google/Chrome/Application/chrome.exe'
const USER_DIR = process.env.LOCALAPPDATA + '/Temp/chrome-cdp-shot'

const chrome = spawn(CHROME, [
  '--headless=new', '--no-sandbox', `--remote-debugging-port=${DEBUG_PORT}`,
  `--user-data-dir=${USER_DIR}`, '--window-size=1440,900', '--autoplay-policy=no-user-gesture-required', 'about:blank'
], { stdio: 'ignore' })

function getJSON(path) {
  return new Promise((resolve, reject) => {
    http.get({ host: '127.0.0.1', port: DEBUG_PORT, path }, (res) => {
      let d = ''
      res.on('data', (c) => (d += c))
      res.on('end', () => { try { resolve(JSON.parse(d)) } catch (e) { reject(e) } })
    }).on('error', reject)
  })
}

async function main() {
  let target = null
  for (let i = 0; i < 30; i++) {
    try {
      const t = await getJSON('/json')
      target = t.find((x) => x.type === 'page')
      if (target) break
    } catch { /* retry */ }
    await new Promise((r) => setTimeout(r, 300))
  }
  if (!target) { console.log('NO_TARGET'); process.exit(1) }

  const { default: WebSocket } = await import('ws')
  const ws = new WebSocket(target.webSocketDebuggerUrl)
  await new Promise((r) => (ws.on('open', r)))
  let id = 0
  const pending = new Map()
  const logs = []
  ws.on('message', (raw) => {
    const m = JSON.parse(raw.toString())
    if (m.id && pending.has(m.id)) { pending.get(m.id)(m.result); pending.delete(m.id) }
    if (m.method === 'Runtime.exceptionThrown') logs.push('EXC: ' + (m.params.exceptionDetails.exception?.description || m.params.exceptionDetails.text))
    if (m.method === 'Log.entryAdded' && m.params.entry.level === 'error') logs.push('LOGERR: ' + m.params.entry.text)
  })
  const send = (method, params = {}) => new Promise((res) => {
    const mid = ++id
    pending.set(mid, res)
    ws.send(JSON.stringify({ id: mid, method, params }))
  })

  await send('Runtime.enable')
  await send('Log.enable')
  await send('Page.enable')
  await send('Page.navigate', { url: URL })
  console.log('waiting', WAIT, 'ms (WebRTC connect+decode)...')
  await new Promise((r) => setTimeout(r, WAIT))

  const hasVideo = await send('Runtime.evaluate', {
    expression: "(() => { const v = document.querySelector('video'); return v ? {w: v.videoWidth, h: v.videoHeight, playing: !v.paused, ready: v.readyState} : null })()",
    returnByValue: true,
  })
  console.log('VIDEO:', JSON.stringify(hasVideo.result?.value))

  const shot = await send('Page.captureScreenshot', { format: 'png' })
  writeFileSync(OUT, Buffer.from(shot.data, 'base64'))
  console.log('SAVED:', OUT)
  logs.forEach((l) => console.log(l))
  chrome.kill()
  process.exit(0)
}

main().catch((e) => { console.log('FATAL', e.message); chrome.kill(); process.exit(1) })
