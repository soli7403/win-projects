#!/usr/bin/env node
// 도움말 그림 자동 캡처 (D158, docs/import-plans/2026-11-27-18-help-sort-faq-images.md).
//
// 하는 일
//   1. 이 폴더(frontend)의 화면을 vite 개발 서버로 127.0.0.1 에 띄운다. Wails 전역이 없으므로 화면은
//      가짜 백엔드(src/ipc/fakeBackend.ts — 메모리에만 있는 가짜 데이터)로 동작한다. 실제 서버 목록·보관함은 읽지 않는다.
//   2. playwright-core 로 이 PC 에 설치된 Microsoft Edge(channel: msedge)를 화면 없이(headless) 띄운다.
//      브라우저를 따로 받지 않는다. 프로필은 Playwright 가 만드는 빈 임시 프로필이다.
//   3. 페이지를 열기 전에 화면 저장소에 dariterm.helpCapture=1 을 넣어 손잡이(window.__dtHelpCapture, src/lib/helpCapture.ts)를 받는다.
//   4. 보관함을 만들고 데모 서버·폴더·명령·매크로·규칙을 넣는다(예약 주소·example.com, 비밀은 가짜 값).
//   5. 언어마다(기본 ko, en) 화면을 하나씩 열고 그 부분만 찍어 ../locales/help/img/<언어>/<id>.png 로 저장한다.
//
// 왜 앱(WebView2)에 직접 붙지 않나: WebView2 에 붙으려면 --remote-debugging-port 인자가 필요한데, Wails v2.17.1 은
// 브라우저 인자를 옵션으로 내주지 않고, 환경 변수 WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS 도 이 PC 에서 먹지 않았다
// (Go 로더·native_webview2loader 둘 다 시험, 계획서 §5). 화면 코드·스타일은 앱과 같은 것이라 그림도 같다.
//
// 쓰는 법 (frontend 폴더에서):
//   npm run help:capture
//   npm run help:capture -- --langs ko --only proxy,forward
//   npm run help:capture -- --theme light --headed
//
//   npm run help:capture -- --copy-to ../docs/import-plans/2026-11-27-18-help-sort-faq-images/img
//
// 옵션: --langs ko,en  --only id,id  --theme dark|light  --headed(창을 보이며)  --port 5299
//       --copy-to <폴더>: 찍은 그림을 <폴더>/<언어>/<id>.png 로도 복사한다(계획서 폴더의 그림을 새로 고칠 때).
//       계획서(docs/import-plans)만 다른 곳과 공유되므로, 그림을 다시 찍으면 그 계획서 폴더에도 넣어야 한다.
import { randomUUID } from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright-core';
import { createServer } from 'vite';

const HERE = path.dirname(fileURLToPath(import.meta.url));
const FRONTEND = path.resolve(HERE, '..');
const ROOT = path.resolve(FRONTEND, '..'); // cli-manager-cr
const OUT_DIR = path.join(ROOT, 'locales', 'help', 'img');
const VIEW = { width: 1280, height: 820 };

// ── 옵션 ─────────────────────────────────────────────────────────────
function parseArgs(argv) {
  const opt = { langs: ['ko', 'en'], only: null, theme: 'dark', headed: false, port: 5299, copyTo: null };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    const next = () => {
      const v = argv[++i];
      if (v === undefined) throw new Error(`${a} needs a value`);
      return v;
    };
    if (a === '--langs') opt.langs = next().split(',').map((s) => s.trim()).filter(Boolean);
    else if (a === '--only') opt.only = new Set(next().split(',').map((s) => s.trim()).filter(Boolean));
    else if (a === '--theme') opt.theme = next() === 'light' ? 'light' : 'dark';
    else if (a === '--headed') opt.headed = true;
    else if (a === '--port') opt.port = Number(next());
    else if (a === '--copy-to') opt.copyTo = path.resolve(next());
    else throw new Error(`unknown option: ${a}`);
  }
  if (!Number.isInteger(opt.port) || opt.port < 1024 || opt.port > 65535) throw new Error('--port must be 1024..65535');
  for (const l of opt.langs) if (!/^[a-z]{2}$/.test(l)) throw new Error(`bad language: ${l}`);
  return opt;
}

// ── 데모 데이터 ──────────────────────────────────────────────────────
// id 는 실행마다 같게. 주소는 사설·예약 대역과 example.com 만 쓴다.
const ID = {
  gSvc: '6f1c0a52-1b7e-4d0a-9a51-000000000001',
  gDev: '6f1c0a52-1b7e-4d0a-9a51-000000000002',
  web: '6f1c0a52-1b7e-4d0a-9a51-000000000011',
  db: '6f1c0a52-1b7e-4d0a-9a51-000000000012',
  bastion: '6f1c0a52-1b7e-4d0a-9a51-000000000013',
  dev: '6f1c0a52-1b7e-4d0a-9a51-000000000014',
  sw: '6f1c0a52-1b7e-4d0a-9a51-000000000015',
  local: '6f1c0a52-1b7e-4d0a-9a51-000000000016',
};

const TEXT = {
  ko: { gSvc: '서비스', gDev: '개발', webNote: '운영 웹 서버 (데모)', snDisk: '디스크 사용량', snRestart: '서비스 재시작', mcDeploy: '배포 확인' },
  en: { gSvc: 'Services', gDev: 'Development', webNote: 'Production web server (demo)', snDisk: 'Disk usage', snRestart: 'Restart a service', mcDeploy: 'Check deployment' },
};
const textFor = (lang) => TEXT[lang] ?? TEXT.en;

function baseHost(id, over) {
  return {
    id, groupId: null, alias: '', protocol: 'ssh', host: '', port: 22, username: '', authType: 'password',
    tags: [], encoding: 'utf-8', termType: 'xterm-256color', keepaliveSec: 30, forwards: [], sort: 0, ...over,
  };
}

function demoData(lang, theme, cred) {
  const tx = textFor(lang);
  const now = Date.now();
  const rule = (over) => ({ id: randomUUID(), regex: false, caseSensitive: false, notify: false, sound: false, replyOn: false, reply: '', replyEnter: false, enabled: true, ...over });
  const deploySteps = [
    { id: randomUUID(), expect: '', send: 'cd /srv/app', enter: true },
    { id: randomUUID(), expect: '$', send: 'git pull', enter: true },
    { id: randomUUID(), expect: 'Already up to date', send: 'systemctl status app --no-pager', enter: true },
  ];
  return {
    store: {
      groups: [
        { id: ID.gSvc, parentId: null, name: tx.gSvc, env: 'prod', sort: 1 },
        { id: ID.gDev, parentId: null, name: tx.gDev, env: 'dev', sort: 2 },
      ],
      hosts: [
        baseHost(ID.web, {
          groupId: ID.gSvc, alias: 'web-01', host: '10.0.0.11', username: 'deploy', credentialId: cred.deploy, env: 'prod', tags: ['web'], favorite: true,
          lastUsedAt: now - 60_000, notes: tx.webNote, sort: 1,
          proxy: { type: 'command', host: '', port: 0, username: '', command: 'ssh -W %h:%p bastion.example.com' },
          forwards: [
            { id: randomUUID(), type: 'L', bindHost: '127.0.0.1', bindPort: 15432, destHost: 'db.internal', destPort: 5432 },
            { id: randomUUID(), type: 'R', bindHost: '127.0.0.1', bindPort: 8022, destHost: '127.0.0.1', destPort: 22 },
            { id: randomUUID(), type: 'D', bindHost: '127.0.0.1', bindPort: 1080, destHost: '', destPort: 0 },
          ],
          loginScript: [
            { id: randomUUID(), expect: '$', send: 'sudo su -' },
            { id: randomUUID(), expect: 'password', send: '', secretId: cred.su },
            { id: randomUUID(), expect: '#', send: 'cd /srv/app' },
          ],
          triggerRules: [rule({ pattern: 'OutOfMemoryError', notify: true, sound: true })],
        }),
        baseHost(ID.db, { groupId: ID.gSvc, alias: 'db-01', host: '10.0.0.21', username: 'dba', env: 'prod', jumpHostId: ID.bastion, tags: ['db'], lastUsedAt: now - 3_600_000, sort: 2 }),
        baseHost(ID.bastion, { groupId: ID.gSvc, alias: 'bastion', host: 'bastion.example.com', username: 'ops', authType: 'agent', sort: 3 }),
        baseHost(ID.dev, { groupId: ID.gDev, alias: 'dev-01', host: '192.168.0.50', username: 'dev', env: 'dev', sort: 4 }),
        baseHost(ID.sw, { alias: 'core-switch', protocol: 'telnet', host: '10.0.1.1', port: 23, authType: 'ask', env: 'device', sort: 5 }),
        baseHost(ID.local, { alias: 'PowerShell', protocol: 'local', host: '', port: 0, local: { shell: 'pwsh.exe', cwd: '' }, sort: 6 }),
      ],
      snippets: [
        { id: randomUUID(), name: tx.snDisk, body: 'df -h', sendEnter: true },
        { id: randomUUID(), name: tx.snRestart, body: 'systemctl restart {{service}}', sendEnter: true },
        { id: randomUUID(), name: tx.mcDeploy, body: deploySteps.map((s) => s.send).join('\n'), sendEnter: true, steps: deploySteps },
      ],
      appUi: { importSuggested: true },
    },
    settings: {
      language: lang,
      usage: 'personal',
      domainNoticeDone: true,
      theme,
      highlightRules: [
        { id: randomUUID(), pattern: 'ERROR', color: '#ef4444', regex: false, caseSensitive: false, background: false, enabled: true },
        { id: randomUUID(), pattern: 'WARN(ING)?', color: '#f59e0b', regex: true, caseSensitive: false, background: false, enabled: true },
      ],
      triggerRules: [
        rule({ pattern: 'BUILD SUCCESS', notify: true }),
        rule({ pattern: 'Are you sure (yes/no)', replyOn: true, reply: 'yes', replyEnter: true }),
      ],
      triggerOsNotify: true,
    },
  };
}

// ── 찍을 화면 ────────────────────────────────────────────────────────
// open: 브라우저 안에서 도는 함수(손잡이 h, id 표 a). shot: 'page' 는 창 전체, 아니면 CSS 선택자(마지막으로 보이는 것).
// tab: 서버 편집의 탭(data-tab). focus: 이 값이 든 입력 칸을 가운데로 스크롤. type: 연 뒤 터미널에 칠 글자.
const editWeb = (h, a) => h.ui.getState().openEditor({ kind: 'host', hostId: a.web, groupId: null });
const SHOTS = [
  { id: 'screen', shot: 'page', open: async (h, a) => { h.ui.getState().setSnippetsOpen(true); await h.openHost(a.web); }, type: 'df -h' },
  { id: 'servers', shot: '[role="dialog"]', open: editWeb, tab: 'basic' },
  { id: 'proxy', shot: '[role="dialog"]', open: editWeb, tab: 'adv', focus: 'ssh -W %h:%p bastion.example.com' },
  { id: 'forward', shot: '[role="dialog"]', open: editWeb, tab: 'fwd' },
  { id: 'login-script', shot: '[role="dialog"]', open: editWeb, tab: 'auto' },
  { id: 'connect', shot: '.quick', open: (h) => h.ui.getState().openQuick() },
  { id: 'macros', shot: '.snippet-dock', open: (h) => h.ui.getState().setSnippetsOpen(true) },
  { id: 'vault', shot: '[role="dialog"]', open: (h) => h.ui.getState().openSettings('security') },
  { id: 'triggers', shot: '[role="dialog"]', open: (h) => h.ui.getState().openSettings('rules') },
  { id: 'shortcuts', shot: '[role="dialog"]', open: (h) => h.ui.getState().openSettings('shortcuts') },
  { id: 'settings', shot: '[role="dialog"]', open: (h) => h.ui.getState().openSettings('appearance') },
  { id: 'log', shot: '[role="dialog"]', open: (h) => h.ui.getState().openSettings('log') },
];

// ── 도우미 ──────────────────────────────────────────────────────────
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

async function resetUi(page) {
  await page.evaluate(() => {
    const ui = window.__dtHelpCapture.ui.getState();
    ui.closeEditor();
    ui.closeSettings();
    ui.closeQuick();
    ui.closeHelp();
    ui.closeMenu();
    ui.setSnippetsOpen(false);
  });
  await sleep(200);
}

async function settle(page) {
  // 글꼴이 다 읽히고 두 번 그려질 때까지 기다린다.
  await page.evaluate(() => document.fonts?.ready);
  await page.evaluate(() => new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r))));
  await sleep(300);
}

/** 한 언어: 새 페이지에서 데모 데이터를 넣고 화면을 차례로 찍는다. 저장한 파일 경로 배열을 돌려준다. */
async function captureLang(browser, url, lang, opt) {
  const context = await browser.newContext({ viewport: VIEW, deviceScaleFactor: 1, colorScheme: opt.theme, locale: lang === 'ko' ? 'ko-KR' : 'en-US' });
  try {
    await context.addInitScript(() => {
      try {
        localStorage.setItem('dariterm.helpCapture', '1');
      } catch {
        /* 저장소가 막히면 손잡이가 없고 아래에서 시간 초과로 멈춘다 */
      }
    });
    const page = await context.newPage();
    const errors = [];
    page.on('pageerror', (e) => errors.push(e.message));
    await page.goto(url);
    await page.waitForFunction(() => !!window.__dtHelpCapture && window.__dtHelpCapture.app.getState().booted, null, { timeout: 30_000 });

    // 보관함과 데모 계정(가짜 백엔드 메모리), 데모 데이터와 설정.
    const cred = await page.evaluate(async () => {
      const h = window.__dtHelpCapture;
      const st = await h.invoke('vault.status');
      if (!st.initialized) await h.invoke('vault.setup', [`demo-${Math.random().toString(36).slice(2)}`]);
      const deploy = await h.invoke('cred.save', [{ name: 'deploy@web-01', kind: 'password', username: 'deploy', password: 'not-a-real-password' }]);
      const su = await h.invoke('cred.save', [{ name: 'su root (web-01)', kind: 'password', username: 'root', password: 'not-a-real-password' }]);
      return { deploy: deploy.id, su: su.id };
    });
    const data = demoData(lang, opt.theme, cred);
    await page.evaluate(async (d) => {
      const h = window.__dtHelpCapture;
      await h.app.getState().patchStore(d.store);
      await h.saveSettings(d.settings);
      await h.app.getState().refreshVault();
    }, data);
    await page.waitForSelector('.sidebar', { timeout: 15_000 });

    const dir = path.join(OUT_DIR, lang);
    fs.mkdirSync(dir, { recursive: true });
    const written = [];
    for (const s of SHOTS) {
      if (opt.only && !opt.only.has(s.id)) continue;
      await resetUi(page);
      await page.evaluate(`(${s.open.toString()})(window.__dtHelpCapture, ${JSON.stringify(ID)})`);
      if (s.tab) {
        const tab = page.locator(`[role="tab"][data-tab="${s.tab}"]`);
        await tab.waitFor({ state: 'visible', timeout: 10_000 });
        await tab.click();
      }
      if (s.focus) {
        await page.evaluate((v) => {
          const el = [...document.querySelectorAll('input, textarea')].find((x) => x.value === v);
          el?.scrollIntoView({ block: 'center' });
        }, s.focus);
      }
      if (s.type) {
        await sleep(400); // 가짜 연결(30ms)과 첫 출력
        await page.keyboard.type(s.type);
        await page.keyboard.press('Enter');
        await sleep(300);
      }
      await settle(page);
      const file = path.join(dir, `${s.id}.png`);
      if (s.shot === 'page') {
        await page.screenshot({ path: file, animations: 'disabled', caret: 'hide' });
      } else {
        const el = page.locator(s.shot).last();
        await el.waitFor({ state: 'visible', timeout: 10_000 });
        await el.screenshot({ path: file, animations: 'disabled', caret: 'hide' });
      }
      written.push(file);
      console.log(`  ${lang}/${s.id}.png`);
    }
    if (errors.length) console.warn(`  page errors (${lang}): ${errors.join(' | ')}`);
    return written;
  } finally {
    await context.close();
  }
}

// ── 본체 ────────────────────────────────────────────────────────────
async function main() {
  const opt = parseArgs(process.argv.slice(2));
  const server = await createServer({
    root: FRONTEND,
    configFile: path.join(FRONTEND, 'vite.config.ts'),
    logLevel: 'warn',
    server: { host: '127.0.0.1', port: opt.port, strictPort: false },
  });
  let browser;
  try {
    await server.listen();
    const url = server.resolvedUrls?.local?.[0];
    if (!url) throw new Error('vite dev server has no local URL');
    browser = await chromium.launch({ channel: 'msedge', headless: !opt.headed });
    const written = [];
    for (const lang of opt.langs) written.push(...(await captureLang(browser, url, lang, opt)));
    console.log(`done: ${written.length} images → ${path.relative(process.cwd(), OUT_DIR)}`);
    if (opt.copyTo) {
      for (const file of written) {
        const rel = path.relative(OUT_DIR, file); // <언어>/<id>.png
        const dest = path.join(opt.copyTo, rel);
        fs.mkdirSync(path.dirname(dest), { recursive: true });
        fs.copyFileSync(file, dest);
      }
      console.log(`copied: ${written.length} images → ${path.relative(process.cwd(), opt.copyTo)}`);
    }
  } finally {
    await browser?.close().catch(() => undefined);
    await server.close();
  }
}

main().catch((e) => {
  console.error(`help-capture failed: ${e.message}`);
  process.exitCode = 1;
});
