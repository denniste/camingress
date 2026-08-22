// 抓取 localhost:5173/dashboard 的运行时错误与渲染结果
// 用法: node cdp-err.mjs <port> <url>
import { spawn } from 'node:child_process'
import http from 'node:http'

const DEBUG_PORT = process.argv[2] || '9224'
const URL = process.argv[3] || 'http://localhost:5173/dashboard'
const CHROME = 'C:/Program Files/Google/Chrome/Application/chrome.exe'
const USER_DIR = process.env.LOCALAPPDATA + '/Temp/chrome-cdp-err'

const chrome = spawn(CHROME, [
  '--headless=new', '--no-sandbox', `--remote-debugging-port=${DEBUG_PORT}`,
  `--user-data-dir=${USER_DIR}`, '--window-size=1280,800', 'about:blank'
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

let ws
async function main() {
  // 等待调试端口
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
  ws = new WebSocket(target.webSocketDebuggerUrl)
  await new Promise((r) => (ws.on('open', r)))

  let id = 0
  const pending = new Map()
  const logs = []
  ws.on('message', (raw) => {
    const m = JSON.parse(raw.toString())
    if (m.id && pending.has(m.id)) { pending.get(m.id)(m.result); pending.delete(m.id) }
    if (m.method === 'Runtime.exceptionThrown') {
      logs.push('EXCEPTION: ' + (m.params.exceptionDetails.exception?.description || m.params.exceptionDetails.text))
    }
    if (m.method === 'Log.entryAdded' && m.params.entry.level === 'error') {
      logs.push('LOGERR: ' + m.params.entry.text)
    }
    if (m.method === 'Runtime.consoleAPICalled' && m.params.type === 'error') {
      logs.push('CONSOLE: ' + m.params.args.map((a) => a.value ?? a.description).join(' '))
    }
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
  await new Promise((r) => setTimeout(r, 6000))
  const root = await send('Runtime.evaluate', { expression: "document.getElementById('root') ? document.getElementById('root').innerHTML.slice(0, 300) : 'NO_ROOT'", returnByValue: true })
  console.log('ROOT:', root.result?.value)
  console.log('URL:', (await send('Runtime.evaluate', { expression: 'location.href', returnByValue: true })).result?.value)
  console.log('---LOGS---')
  logs.forEach((l) => console.log(l))
  if (!logs.length) console.log('(no errors captured)')
  chrome.kill()
  process.exit(0)
}

main().catch((e) => { console.log('FATAL', e.message); chrome.kill(); process.exit(1) })
