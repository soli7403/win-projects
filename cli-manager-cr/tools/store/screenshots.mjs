// Microsoft Store 스크린샷 (D81): 개발 서버(가짜 데이터, frontend/src/ipc/fakeBackend.ts)를 Edge headless 로 열어 1920x1080(CSS 1280x720 x1.5) 7장을 찍는다.
// 사용: (cd frontend && npx vite --port 5179) 를 띄운 뒤  node tools/store/screenshots.mjs http://localhost:5179/ docs/store/screenshots/en en   (ko 도 같게)
import { spawn } from 'node:child_process';
import { writeFileSync, mkdirSync } from 'node:fs';
import { join } from 'node:path';

const [url, out, lang] = process.argv.slice(2);
const L = JSON.parse((await import('node:fs')).readFileSync(new URL('../../locales/' + lang + '.json', import.meta.url), 'utf8'));
mkdirSync(out, { recursive: true });
const edge = 'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe';
const prof = join(process.env.TEMP, 'dariterm-shot-store-' + Date.now());
mkdirSync(prof, { recursive: true });
const proc = spawn(edge, ['--headless=new', '--remote-debugging-port=9337', `--user-data-dir=${prof}`, '--window-size=1920,1200', lang === 'ko' ? '--lang=ko-KR' : '--lang=en-US', 'about:blank'], { stdio: 'ignore' });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
let targets;
for (let i = 0; i < 50; i++) {
  try { targets = await (await fetch('http://127.0.0.1:9337/json')).json(); if (targets.some((t) => t.type === 'page')) break; } catch {}
  await sleep(200);
}
const page = targets.find((t) => t.type === 'page');
const ws = new WebSocket(page.webSocketDebuggerUrl);
await new Promise((r) => ws.addEventListener('open', r));
let id = 0;
const pending = new Map();
ws.addEventListener('message', (e) => { const m = JSON.parse(e.data); if (m.id && pending.has(m.id)) { pending.get(m.id)(m); pending.delete(m.id); } });
const send = (method, params = {}) => new Promise((r) => { const n = ++id; pending.set(n, r); ws.send(JSON.stringify({ id: n, method, params })); });
const ev = async (expr) => (await send('Runtime.evaluate', { expression: expr, awaitPromise: true, returnByValue: true })).result?.result?.value;
const snap = async (name) => {
  const s = await send('Page.captureScreenshot', { format: 'png' });
  writeFileSync(join(out, name), Buffer.from(s.result.data, 'base64'));
  console.log('saved', name);
};
const clickText = (sel, text) => ev(`(() => { const b = [...document.querySelectorAll(${JSON.stringify(sel)})].find((x) => x.textContent.trim().includes(${JSON.stringify(text)})); if (!b) return false; b.click(); return true; })()`);
const clickSel = (sel) => ev(`(() => { const b = document.querySelector(${JSON.stringify(sel)}); if (!b) return false; b.click(); return true; })()`);
const key = async (k, code, mods = 0, vk = 0) => {
  await send('Input.dispatchKeyEvent', { type: 'keyDown', key: k, code, modifiers: mods, windowsVirtualKeyCode: vk });
  await send('Input.dispatchKeyEvent', { type: 'keyUp', key: k, code, modifiers: mods, windowsVirtualKeyCode: vk });
};
const ALT = 1, CTRL = 2, SHIFT = 8;
const esc = () => key('Escape', 'Escape', 0, 27);


await send('Emulation.setDeviceMetricsOverride', { width: 1280, height: 720, deviceScaleFactor: 1.5, mobile: false });
await send('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-color-scheme', value: 'dark' }] });
await send('Page.navigate', { url });
await sleep(2500);
console.log('lang', await clickText('.lang-item', lang === 'ko' ? '한국어' : 'English'));
await sleep(500);
await clickSel('.lock-actions .btn.primary');
await sleep(800);
await clickSel('.lock-actions .btn.primary');
await sleep(1200);
console.log('pw', await ev(`(() => { const els = [...document.querySelectorAll('input[type=password]')]; els.forEach((el) => { Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set.call(el, 'Correct-Horse-9'); el.dispatchEvent(new Event('input', { bubbles: true })); }); return els.length; })()`));
await clickSel('form button[type=submit]');
await sleep(2500);
await clickSel('.lock-card input[type=checkbox]');
await sleep(300);
await ev(`[...document.querySelectorAll('.lock-card button')].pop()?.click()`);
await sleep(1800);
await snap('04-servers.png');

console.log('add', await clickText('.sidebar .btn', L['sb.addServer']));
await sleep(900);
await snap('05-add-server.png');
await esc(); await sleep(500);

console.log('import', await clickText('.sidebar .btn', L['sb.import']));
await sleep(1500);
await snap('06-import.png');
await esc(); await sleep(500);

console.log('open', await clickText('.recent-grid > *', 'web-prod-01'));
await sleep(2000);
await key('S', 'KeyS', CTRL | SHIFT, 83);
await sleep(1500);
await snap('02-sftp.png');
await key('S', 'KeyS', CTRL | SHIFT, 83);
await sleep(500);

await key('D', 'KeyD', CTRL | SHIFT, 68); await sleep(1500);
await key('E', 'KeyE', CTRL | SHIFT, 69); await sleep(1500);
await key('B', 'KeyB', CTRL | SHIFT, 66); await sleep(4500); // 알림 토스트가 사라질 때까지
const typeLine = async (txt) => { await send('Input.insertText', { text: txt }); await sleep(200); await send('Input.dispatchKeyEvent', { type: 'keyDown', key: 'Enter', code: 'Enter', windowsVirtualKeyCode: 13, text: String.fromCharCode(13) }); await send('Input.dispatchKeyEvent', { type: 'keyUp', key: 'Enter', code: 'Enter', windowsVirtualKeyCode: 13 }); await sleep(500); };
await typeLine('uptime -p');
await typeLine('df -h');
await typeLine('systemctl is-active nginx');
await sleep(800);
await snap('01-split-broadcast.png');

await key('k', 'KeyK', CTRL, 75); await sleep(900);
await ev(`(() => { const el = document.activeElement; if (!el || el.tagName !== 'INPUT') return false; Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set.call(el, 'prod'); el.dispatchEvent(new Event('input', { bubbles: true })); return true; })()`);
await sleep(700);
await snap('03-quick-connect.png');
await esc(); await sleep(500);

console.log('settings', await clickSel(`button[aria-label="${L['settings.title']}"]`));
await sleep(1200);
await snap('07-settings.png');
await esc(); await sleep(500);
ws.close();
proc.kill();
