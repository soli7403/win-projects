// 사용자 매뉴얼 PDF 만들기: docs/scenarios.md(동작 시나리오 원문) + 그림(docs/manual/img) → HTML → PDF(Edge).
// 사용: 저장소 루트에서  node tools/manual/build.mjs [ko|en]   (기본 ko)
// 결과: docs/manual/DariTerm-user-manual.<언어>.html / .pdf
//   ko: 원문 docs/scenarios.md, 그림 docs/manual/img (사용자 영상에서 뽑고 개인 정보 흐림)
//   en: 원문 docs/scenarios.en.md, 그림 docs/manual/img-en (개발 서버 가짜 데이터로 찍은 영어 화면)
import fs from 'node:fs';
import path from 'node:path';
import url from 'node:url';
import { execFileSync } from 'node:child_process';

const LANG = process.argv[2] === 'en' ? 'en' : 'ko';
const root = path.resolve(path.dirname(url.fileURLToPath(import.meta.url)), '..', '..');
const md = fs.readFileSync(path.join(root, 'docs', LANG === 'en' ? 'scenarios.en.md' : 'scenarios.md'), 'utf8');
const outDir = path.join(root, 'docs', 'manual');
const htmlPath = path.join(outDir, `DariTerm-user-manual.${LANG}.html`);
const pdfPath = path.join(outDir, `DariTerm-user-manual.${LANG}.pdf`);
const IMG_DIR = LANG === 'en' ? 'img-en' : 'img';

// 표지·머리말·꼬리말 문구.
const TEXT = {
  ko: {
    title: 'DariTerm 사용자 매뉴얼', sub: '멀티 프로토콜 터미널·SFTP 통합 클라이언트 — 상황별 동작 안내',
    meta: '기준 버전: 0.1.0 (2026-11-14) · 게시자: YangHun Lee · 구성: 16개 영역 + 단축키 부록',
    note: '각 표는 "상황 → 하는 일 → 앱 동작" 순서입니다. 화면 그림의 서버 이름·주소 등 개인 정보는 흐리게 처리했습니다.',
    foot: '© 2026 YangHun Lee (kai). All rights reserved. · 업무용 라이선스 문의: soli97403@gmail.com',
  },
  en: {
    title: 'DariTerm User Manual', sub: 'Multi-protocol terminal and SFTP client — what happens in each situation',
    meta: 'Version 0.1.0 (2026-11-14) · Publisher: YangHun Lee · 16 areas + shortcut appendix',
    note: 'Each table reads "situation → what you do → what the app does". Screenshots use sample data.',
    foot: '© 2026 YangHun Lee (kai). All rights reserved. · Business license inquiries: soli97403@gmail.com',
  },
}[LANG];

const CHAPTERS_EN = {
  1: { intro: 'Install with the Windows installer. The first launch takes four steps: language → usage → app lock password → recovery code. The recovery code is shown only once, so keep it somewhere safe.', figs: [['fig1-language.png', 'Figure 1 · Choosing the language on first launch (Windows language preselected)']] },
  2: { intro: 'Server passwords and keys are encrypted with the app lock password and stored only on this PC. Locking keeps your sessions connected and just covers the screen.', figs: [['fig2-lock.png', 'Figure 2 · Lock screen (sessions stay connected)']] },
  3: { intro: 'Organize servers and folders in the sidebar. Servers with the Production environment get a red border and an extra confirmation before dangerous commands.', figs: [['fig3-add-server.png', 'Figure 3 · Add server (General · Sign-in · Advanced · Port forwarding)']] },
  4: { intro: 'Double-click a saved server, or press Ctrl+K and type part of a name or IP. Unsaved servers can be reached directly by typing user@host.', figs: [['fig4-welcome.png', 'Figure 4 · Start screen and Quick connect']] },
  7: { intro: 'Split panes inside a tab and turn on broadcast input to send the same command to several panes at once. The top-left chain icon targets all visible panes; Ctrl+Shift+B targets the current tab.', figs: [['fig5-broadcast.png', 'Figure 5 · Split panes with broadcast input on']] },
  8: { intro: 'Press Ctrl+Shift+S (or the top-right icon) to open the file panel: SFTP for SSH sessions, this PC\'s files for local terminals. Drag between tabs to copy server to server.', figs: [['fig6-sftp.png', 'Figure 6 · SFTP panel next to the terminal']] },
  10: { intro: 'Click Import in the sidebar to list PuTTY saved sessions, SecureCRT session folders and ~/.ssh/config. Passwords are not imported.', figs: [['fig7-import.png', 'Figure 7 · Import servers']] },
  13: { intro: 'Settings (gear icon, top left) are saved as soon as you change them.', figs: [['fig8-settings.png', 'Figure 8 · Settings (Appearance)']] },
};

// 장 머리에 넣을 안내 글과 그림(장 번호 → {intro, figs:[[파일, 캡션]]}).
const CHAPTERS_KO = {
  1: { intro: 'Windows 설치 파일로 설치하고, 처음 실행하면 언어 선택 → 사용 용도 → 앱 잠금 비밀번호 → 복구 코드의 4단계를 거칩니다. 복구 코드는 이 화면에서 한 번만 보이므로 꼭 따로 보관하세요.', figs: [['fig1-language.png', '그림 1 · 첫 실행 언어 선택 (Windows 언어가 미리 골라짐)']] },
  2: { intro: '서버 비밀번호와 키는 앱 잠금 비밀번호로 암호화해 이 PC 에만 저장합니다. 잠가도 연결된 세션은 끊기지 않고 잠금 화면만 위에 덮입니다.', figs: [['fig2-lock.png', '그림 2 · 앱 잠금 화면 (연결은 유지된 채 화면만 잠김)']] },
  3: { intro: '사이드바에서 서버와 폴더를 정리합니다. 환경을 "운영"으로 정한 서버는 빨간 테두리로 표시되고, 위험한 명령 앞에서 한 번 더 묻습니다.', figs: [['fig3-add-server.png', '그림 3 · 서버 추가 (일반·로그인·고급·포트 포워딩)']] },
  4: { intro: '저장된 서버를 더블클릭하거나, Ctrl+K 빠른 연결에서 이름·IP 일부를 입력해 연결합니다. 저장하지 않은 서버는 user@host 형식으로 바로 연결할 수 있습니다.', figs: [['fig4-welcome.png', '그림 4 · 시작 화면과 빠른 연결 (서버 목록은 흐리게 처리)']] },
  7: { intro: '한 탭 안에서 창을 나누고, 일괄 입력을 켜면 여러 창에 같은 명령을 한 번에 보낼 수 있습니다. 왼쪽 위 쇠사슬 아이콘은 보이는 모든 창, Ctrl+Shift+B 는 지금 탭의 창들이 대상입니다.', figs: [['fig5-broadcast.png', '그림 5 · 창 나누기와 일괄 입력, 창 맨 아래 복사 목록']] },
  8: { intro: 'Ctrl+Shift+S(또는 오른쪽 위 아이콘)로 파일 창을 엽니다. SSH 세션은 SFTP, 로컬 터미널은 이 PC 의 파일이 보이며, 탭 사이로 끌어 놓아 서버 간 복사도 할 수 있습니다.', figs: [['fig6-sftp.png', '그림 6 · 로컬 파일 창과 SFTP 창 (개인 정보는 흐리게 처리)']] },
  10: { intro: '사이드바 [가져오기]를 누르면 PuTTY 저장 세션, SecureCRT 세션 폴더, ~/.ssh/config 를 찾아 목록으로 보여 줍니다. 비밀번호는 옮기지 않습니다.', figs: [['fig7-import.png', '그림 7 · 서버 가져오기 (세션 목록은 흐리게 처리)']] },
  13: { intro: '왼쪽 위 톱니바퀴로 여는 설정은 저장 버튼 없이 바꾸는 즉시 저장됩니다.', figs: [['fig8-settings.png', '그림 8 · 설정 (백업 구역)']] },
};
const CHAPTERS = LANG === 'en' ? CHAPTERS_EN : CHAPTERS_KO;

const esc = (s) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
function inline(s) {
  // 개발용 근거 표기(예: (shell §6), (D80), , D80)는 사용자 문서에서 뺀다.
  s = s
    .replace(/,\s*[A-Za-z][\w/.-]*\s*§[\d.~]+/g, '')
    .replace(/\s*\((?:[A-Za-z][\w/.-]*\s*)?§[\d.~]+\)/g, '')
    .replace(/,\s*D\d+(?=\))/g, '')
    .replace(/\s*\(D\d+\)/g, '');
  return esc(s)
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/\*\*([^*]+)\*\*/g, '<b>$1</b>')
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '$1');
}

const lines = md.split(/\r?\n/);
const body = [];
let i = 0;
let skipToc = false;
while (i < lines.length) {
  const line = lines[i];
  if (/^# /.test(line)) { i++; continue; } // 제목은 표지에서
  if (/^## (목차|Contents)/.test(line)) { skipToc = true; i++; continue; }
  if (skipToc) { if (/^---/.test(line)) skipToc = false; i++; continue; }
  if (/^> /.test(line)) { i++; continue; } // 문서 머리 안내(개발용)는 뺀다
  const h2 = line.match(/^## (\d+)\. (.*)$/);
  if (h2) {
    body.push(`<h2>${h2[1]}. ${inline(h2[2])}</h2>`);
    const ch = CHAPTERS[Number(h2[1])];
    if (ch) {
      body.push(`<p class="intro">${inline(ch.intro)}</p>`);
      for (const [file, cap] of ch.figs) body.push(`<figure><img src="${IMG_DIR}/${file}" alt="${esc(cap)}"><figcaption>${esc(cap)}</figcaption></figure>`);
    }
    i++;
    continue;
  }
  if (/^## /.test(line) || /^### /.test(line)) { body.push(`<h2>${inline(line.replace(/^#+ /, ''))}</h2>`); i++; continue; }
  if (/^\|/.test(line)) {
    const rows = [];
    while (i < lines.length && /^\|/.test(lines[i])) rows.push(lines[i++]);
    const cells = (r) => r.replace(/^\||\|$/g, '').split(/(?<!\\)\|/).map((c) => c.trim());
    const head = cells(rows[0]);
    const data = rows.slice(2).map(cells);
    body.push('<table><thead><tr>' + head.map((h) => `<th>${inline(h)}</th>`).join('') + '</tr></thead><tbody>' +
      data.map((r) => '<tr>' + r.map((c) => `<td>${inline(c)}</td>`).join('') + '</tr>').join('') + '</tbody></table>');
    continue;
  }
  if (/^---/.test(line) || !line.trim()) { i++; continue; }
  body.push(`<p>${inline(line)}</p>`);
  i++;
}

const html = `<!doctype html><html lang="${LANG}"><head><meta charset="utf-8"><title>${TEXT.title}</title>
<style>
@page { size: A4; margin: 16mm 14mm 18mm; }
body { font-family: 'Malgun Gothic', 'Segoe UI', sans-serif; color: #1f2937; font-size: 10.5pt; line-height: 1.55; }
.cover { border-bottom: 3px solid #1e3a8a; padding-bottom: 10px; margin-bottom: 14px; }
.kicker { color: #2563eb; font-weight: 700; font-size: 10pt; letter-spacing: .04em; }
h1 { font-size: 24pt; margin: 4px 0; color: #0f172a; }
.sub { color: #475569; } .meta { color: #64748b; font-size: 9.5pt; margin-top: 4px; }
.note { background: #f1f5f9; border-left: 4px solid #2563eb; padding: 8px 12px; margin: 10px 0; }
h2 { color: #1e3a8a; font-size: 14pt; border-bottom: 1px solid #cbd5e1; padding-bottom: 3px; margin: 22px 0 8px; break-after: avoid; }
.intro { background: #f8fafc; border: 1px solid #e2e8f0; padding: 8px 12px; }
figure { margin: 8px 0 12px; text-align: center; break-inside: avoid; }
figure img { max-width: 100%; max-height: 120mm; border: 1px solid #cbd5e1; border-radius: 4px; }
figcaption { color: #1e3a8a; font-weight: 700; font-size: 9pt; margin-top: 4px; }
table { width: 100%; border-collapse: collapse; margin: 6px 0 12px; font-size: 9.5pt; }
th { background: #1e3a8a; color: #ffffff; text-align: left; padding: 5px 7px; }
td { border: 1px solid #cbd5e1; padding: 5px 7px; vertical-align: top; }
th:first-child, td:first-child { white-space: nowrap; width: 1%; }
td:nth-child(2) { width: 17%; } td:nth-child(3) { width: 24%; }
tr { break-inside: avoid; } tbody tr:nth-child(even) td { background: #f8fafc; }
code { font-family: Consolas, monospace; background: #f1f5f9; border: 1px solid #e2e8f0; border-radius: 3px; padding: 0 3px; font-size: 9pt; }
.foot { margin-top: 18px; color: #64748b; font-size: 9pt; border-top: 1px solid #e2e8f0; padding-top: 6px; }
</style></head><body>
<div class="cover"><div class="kicker">USER MANUAL &amp; OPERATION GUIDE</div><h1>${TEXT.title}</h1>
<div class="sub">${TEXT.sub}</div>
<div class="meta">${TEXT.meta}</div></div>
<div class="note">${TEXT.note}</div>
${body.join('\n')}
<div class="foot">${TEXT.foot}</div>
</body></html>`;
fs.writeFileSync(htmlPath, html, 'utf8');

const edge = ['C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe', 'C:/Program Files/Microsoft/Edge/Application/msedge.exe'].find((p) => fs.existsSync(p));
execFileSync(edge, ['--headless=new', '--disable-gpu', '--no-pdf-header-footer', `--print-to-pdf=${pdfPath}`, url.pathToFileURL(htmlPath).href], { stdio: 'ignore' });
console.log('html:', htmlPath);
console.log('pdf :', pdfPath, fs.existsSync(pdfPath) ? fs.statSync(pdfPath).size : 'missing');
