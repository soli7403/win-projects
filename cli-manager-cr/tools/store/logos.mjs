// Microsoft Store 표시용 이미지 (D81): build/appicon.png 로 포스터 아트(9:16)·박스 아트(1:1), 스크린샷으로 16:9 슈퍼 히어로 아트를 Edge headless 로 그린다.
// 300x300 타일 아이콘은 ffmpeg 로 줄인다.  사용: node tools/store/logos.mjs  → docs/store/logos/
import { spawn, execFileSync } from 'node:child_process';
import { readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..', '..');
const out = join(root, 'docs', 'store', 'logos');
mkdirSync(out, { recursive: true });
const iconPath = join(root, 'build', 'appicon.png');
const icon = 'data:image/png;base64,' + readFileSync(iconPath).toString('base64');

// 앱 아이콘 색(남색 그라데이션)과 다크 테마 배경(#0f1115)을 잇는 배경. 글자는 제품 이름과 프로토콜(언어 무관).
const page = (w, h, iconSize, nameSize, tagSize, gap) => `<!doctype html><html><head><meta charset="utf-8"><style>
html,body{margin:0;width:${w}px;height:${h}px;overflow:hidden}
body{display:flex;flex-direction:column;align-items:center;justify-content:center;gap:${gap}px;
 background:radial-gradient(ellipse at 50% 38%,#24447e 0%,#172a4f 45%,#0f1115 100%);font-family:'Segoe UI',sans-serif}
img{width:${iconSize}px;height:${iconSize}px;filter:drop-shadow(0 ${iconSize / 20}px ${iconSize / 10}px rgba(0,0,0,.45))}
.name{color:#fff;font-weight:700;font-size:${nameSize}px;letter-spacing:-.01em;line-height:1}
.tag{color:#9db8ff;font-weight:600;font-size:${tagSize}px;letter-spacing:.02em;line-height:1}
</style></head><body><img src="${icon}"><div class="name">DariTerm</div>${tagSize ? '<div class="tag">SSH · Telnet · Serial · SFTP</div>' : ''}</body></html>`;

// 16:9 슈퍼 히어로 아트(스토어 목록 맨 위): 규칙상 제품 이름 글자를 넣지 않는다 → 앱 화면(화면 분할 스크린샷)만 배경 위에 놓는다.
const shot = 'data:image/png;base64,' + readFileSync(join(root, 'docs', 'store', 'screenshots', 'en', '01-split-broadcast.png')).toString('base64');
const hero = `<!doctype html><html><head><meta charset="utf-8"><style>
html,body{margin:0;width:1920px;height:1080px;overflow:hidden}
body{display:flex;align-items:center;justify-content:center;
 background:radial-gradient(ellipse at 30% 20%,#2a4f92 0%,#172a4f 50%,#0f1115 100%)}
img{width:1500px;border-radius:18px;border:2px solid rgba(157,184,255,.35);
 box-shadow:0 40px 90px rgba(0,0,0,.6),0 0 0 1px rgba(0,0,0,.4)}
</style></head><body><img src="${shot}"></body></html>`;

const shots = [
  { file: 'hero-1920x1080.png', w: 1920, h: 1080, html: hero },
  { file: 'poster-1440x2160.png', w: 1440, h: 2160, html: page(1440, 2160, 760, 190, 64, 70) },
  { file: 'box-1080x1080.png', w: 1080, h: 1080, html: page(1080, 1080, 560, 120, 0, 50) },
];

const edge = 'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe';
const prof = join(process.env.TEMP, 'dariterm-logo-' + Date.now());
mkdirSync(prof, { recursive: true });
const proc = spawn(edge, ['--headless=new', '--remote-debugging-port=9338', `--user-data-dir=${prof}`, '--hide-scrollbars', 'about:blank'], { stdio: 'ignore' });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
let targets;
for (let i = 0; i < 50; i++) {
  try { targets = await (await fetch('http://127.0.0.1:9338/json')).json(); if (targets.some((t) => t.type === 'page')) break; } catch {}
  await sleep(200);
}
const ws = new WebSocket(targets.find((t) => t.type === 'page').webSocketDebuggerUrl);
await new Promise((r) => ws.addEventListener('open', r));
let id = 0;
const pending = new Map();
ws.addEventListener('message', (e) => { const m = JSON.parse(e.data); if (m.id && pending.has(m.id)) { pending.get(m.id)(m); pending.delete(m.id); } });
const send = (method, params = {}) => new Promise((r) => { const n = ++id; pending.set(n, r); ws.send(JSON.stringify({ id: n, method, params })); });

for (const s of shots) {
  await send('Emulation.setDeviceMetricsOverride', { width: s.w, height: s.h, deviceScaleFactor: 1, mobile: false });
  await send('Page.navigate', { url: 'data:text/html;base64,' + Buffer.from(s.html).toString('base64') });
  await sleep(1200);
  const r = await send('Page.captureScreenshot', { format: 'png' });
  writeFileSync(join(out, s.file), Buffer.from(r.result.data, 'base64'));
  console.log('saved', s.file);
}
ws.close();
proc.kill();

// 1:1 앱 타일 아이콘 300x300 (투명 모서리 유지)
execFileSync('ffmpeg', ['-v', 'error', '-y', '-i', iconPath, '-vf', 'scale=300:300:flags=lanczos', join(out, 'tile-300x300.png')]);
console.log('saved tile-300x300.png');
