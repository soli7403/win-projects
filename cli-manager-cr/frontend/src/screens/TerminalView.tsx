// 터미널 칸 본문(xterm.js). 근거: spec/behavior/terminal.md §1~§5, §6, §8, spec/behavior/shortcuts.md §2~§3, spec/ui/panes.md §2.3~§2.4.
import { useEffect, useRef, useState } from 'react';
import { Terminal, type IDisposable } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { SearchAddon } from '@xterm/addon-search';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { WebglAddon } from '@xterm/addon-webgl';
import '@xterm/xterm/css/xterm.css';
import { getBackend, errorMessage } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { useWorkspace } from '../state/workspace';
import { sessionsIn, findPane, type LayoutNode } from '../lib/layout';
import { matchesDangerous, LineTracker, pasteLineCount } from '../lib/termGuard';
import { decodeOsc52 } from '../lib/osc52';
import { activeShortcut } from '../state/shortcuts';
import { resolveTermTheme } from '../lib/termThemes';
import { attachHighlighter, type Highlighter } from '../lib/termHighlight';
import { attachFindMarks, type FindMarks } from '../lib/termFind';
import { attachCopyOnMouseSelect } from '../lib/copyOnSelect';
import { mixHex, rowRanges, FIND_ACTIVE_BG, FIND_ACTIVE_FG, FIND_MATCH_COLOR, FIND_MATCH_RATIO, FIND_RULER_MATCH, FIND_RULER_ACTIVE, FIND_RULER_WIDTH } from '../lib/findHighlight';
import { DANGEROUS_PATTERNS } from '../lib/defaults';
import { clipboardKind, forceImagePaste, insertPaths, uploadAndInsert } from '../state/paste';
// 이 파일은 t 를 터미널 변수 이름으로 쓰므로 번역 함수는 tr 로 가져온다(D75).
import { relabel, t as tr } from '../i18n';

interface Props {
  paneId: string;
  sessionId: string;
  focused: boolean;
}

/** 칸 id → xterm 인스턴스. 칸 우클릭 메뉴가 복사·모두 선택·화면 지우기에 쓴다. */
export const termHandles = new Map<string, Terminal>();

/** 메뉴의 "모두 선택". 사용자가 고른 선택이므로 "선택하면 자동 복사"가 켜져 있으면 복사한다
 *  (자동 복사는 마우스 선택에만 반응하므로 여기서 따로 한다, KI-011). */
export function selectAllInPane(paneId: string): void {
  const term = termHandles.get(paneId);
  if (!term) return;
  term.selectAll();
  if (useApp.getState().store?.settings.copyOnSelect && term.getSelection()) void copySelection(term);
}

/**
 * 활성 탭의 포커스 칸 터미널에 키보드 포커스를 준다. 탭 클릭·서버 열기 뒤 바로 입력할 수 있게 한다.
 * 클릭의 기본 동작(포커스 이동)과 React 그리기가 끝난 다음 프레임에 준다.
 */
export function focusActiveTerminal(): void {
  requestAnimationFrame(() => {
    if (useUi.getState().dialogs.length > 0) return;
    const s = useWorkspace.getState();
    const tab = s.tabs.find((t) => t.id === s.activeTabId);
    const pid = tab?.zoomed ?? tab?.focusedPane;
    if (pid) termHandles.get(pid)?.focus();
  });
}

/** 찾기 요청 이벤트(칸 id 가 detail). 단축키·칸 머리글·우클릭 메뉴가 보낸다. */
export const FIND_EVENT = 'dariterm:find';

/** 이 칸의 입력 대상: 동시 입력이면 탭의 연결된 모든 세션. */
export function targetsFor(paneId: string, sessionId: string): string[] {
  const s = useWorkspace.getState();
  // 모든 창 동시 입력(쇠사슬)이 켜져 있으면 화면에 보이는 모든 창으로 보낸다.
  if (useUi.getState().linkInput) return visibleSessionIds(sessionId);
  const tab = s.tabs.find((t) => paneHas(t.root, paneId));
  if (!tab || !tab.broadcast) return [sessionId];
  return sessionsIn(tab.root).filter((sid) => !!s.sessions[sid]);
}

/**
 * 지금 화면에 보이는 창의 세션들: 그룹마다 선택된 탭의 칸(크게 보기 중이면 그 칸만), 빈 칸 제외.
 * 입력한 창(self)은 항상 포함하고 맨 앞에 둔다.
 */
export function visibleSessionIds(self: string): string[] {
  const s = useWorkspace.getState();
  const out = [self];
  for (const tid of s.activeByGroup) {
    const tab = tid ? s.tabs.find((t) => t.id === tid) : undefined;
    if (!tab) continue;
    const ids = tab.zoomed ? [findPane(tab.root, tab.zoomed)?.sessionId ?? ''] : sessionsIn(tab.root);
    for (const sid of ids) if (sid && s.sessions[sid] && !out.includes(sid)) out.push(sid);
  }
  return out;
}

function paneHas(root: LayoutNode, paneId: string): boolean {
  return !!findPane(root, paneId);
}

function beep(): void {
  try {
    const ctx = new AudioContext();
    const osc = ctx.createOscillator();
    osc.frequency.value = 880;
    osc.connect(ctx.destination);
    osc.start();
    osc.stop(ctx.currentTime + 0.08);
    osc.onended = () => void ctx.close();
  } catch {
    /* 소리를 낼 수 없으면 무시 */
  }
}

async function copySelection(term: Terminal): Promise<void> {
  const sel = term.getSelection();
  if (sel) await getBackend().invoke('clip.writeText', [sel]).catch(() => undefined);
}

/** 텍스트 붙여넣기 흐름(terminal.md §4(b)): 위험 가드, 여러 줄 확인. 파일·이미지는 uploadAndInsert 로 보낸다. */
export async function pasteFromClipboard(paneId: string, sessionId: string, term: Terminal): Promise<void> {
  const st = useApp.getState().store?.settings;
  let kind: Awaited<ReturnType<typeof clipboardKind>>;
  try {
    kind = await clipboardKind();
  } catch (e) {
    useApp.getState().pushToast({ kind: 'error', text: errorMessage(e) });
    return;
  }
  const targets = targetsFor(paneId, sessionId);
  if (kind.type === 'files' && kind.files?.length) {
    await uploadAndInsert(sessionId, targets, { kind: 'files', paths: kind.files });
    term.focus();
    return;
  }
  if (kind.type === 'image') {
    await uploadAndInsert(sessionId, targets, { kind: 'clipboardImage' });
    term.focus();
    return;
  }
  const text = kind.text ?? '';
  if (!text) return;
  const info = useWorkspace.getState().sessions[sessionId]?.info;
  const name = relabel(info?.title) ?? tr('tab.session');
  const patterns = st?.dangerousPatterns ?? DANGEROUS_PATTERNS;
  const send = () => {
    for (const sid of targets) getBackend().termInput(sid, text);
    term.focus();
  };
  if (st?.guardDangerousOnProd && info?.env === 'prod' && matchesDangerous(text, patterns)) {
    const ok = await useUi.getState().confirm({
      title: tr('paste.dangerTitle'),
      body: tr('paste.dangerBody', { name }),
      detail: text.slice(0, 2000),
      okText: tr('paste.ok'),
      danger: true,
    });
    if (ok) send();
    else term.focus();
    return;
  }
  const lines = pasteLineCount(text);
  if (st?.confirmMultilinePaste && lines >= 2 && !term.modes.bracketedPasteMode) {
    const ok = await useUi.getState().confirm({
      title: tr('paste.multiTitle'),
      body: tr('paste.multiBody', { lines, name }),
      detail: text.slice(0, 2000),
      okText: tr('paste.ok'),
    });
    if (ok) send();
    else term.focus();
    return;
  }
  term.paste(text);
  term.focus();
}

function FindBar({ term, paneId, onClose }: { term: Terminal; paneId: string; onClose: () => void }) {
  const [q, setQ] = useState('');
  const addon = useRef<SearchAddon | null>(null);
  // 현재 항목 위에 얹는 강조(줄마다 하나). 검색 애드온은 현재 항목을 "선택"으로 표시하는데,
  // 애드온의 현재 항목 색은 선택 색 아래에 깔려 보이지 않는다. 선택 위(top)에 직접 그린다 (D96).
  const active = useRef<IDisposable[]>([]);
  void paneId;
  const clearActive = () => {
    for (const d of active.current) d.dispose();
    active.current = [];
  };
  const markActive = () => {
    clearActive();
    const sel = term.getSelectionPosition();
    if (!sel) return;
    const buf = term.buffer.active;
    const border = term.options.theme?.foreground ?? FIND_ACTIVE_FG;
    for (const r of rowRanges(sel.start, sel.end, term.cols)) {
      const marker = term.registerMarker(r.row - buf.baseY - buf.cursorY);
      const deco = term.registerDecoration({ marker, x: r.x, width: r.width, backgroundColor: FIND_ACTIVE_BG, foregroundColor: FIND_ACTIVE_FG, layer: 'top', overviewRulerOptions: { color: FIND_RULER_ACTIVE, position: 'full' } });
      deco?.onRender((el) => {
        el.style.outline = `1px solid ${border}`;
      });
      // 반드시 강조를 먼저, 표식을 나중에 지운다. 표식을 먼저 지우면 xterm 이 그 강조를 목록에서
      // 찾지 못해 남기고(줄 번호가 -1 로 바뀐 뒤라서), 다른 강조의 삭제까지 실패하게 만든다.
      active.current.push({
        dispose() {
          deco?.dispose();
          marker.dispose();
        },
      });
    }
  };
  // 찾은 항목(현재 항목 제외)의 바탕·스크롤 막대 표시. 애드온에 강조를 맡기지 않는다(KI-011):
  // 애드온은 강조를 맡으면 출력마다 다시 찾으며 터미널 선택을 바꿔, 자동 복사·사용자 선택을 덮었다.
  const marks = useRef<FindMarks | null>(null);
  useEffect(() => {
    const a = new SearchAddon();
    term.loadAddon(a);
    addon.current = a;
    // 찾은 항목: 터미널 배경에 주황을 옅게 섞은 바탕(테마가 밝든 어둡든 글자가 읽힌다). 색은 #RRGGBB 만 받는다.
    const matchBackground = mixHex(term.options.theme?.background ?? '#000000', FIND_MATCH_COLOR, FIND_MATCH_RATIO);
    const m = attachFindMarks(term, { matchBackground, ruler: FIND_RULER_MATCH });
    marks.current = m;
    // 터미널을 눌러 선택이 풀리면 현재 항목 강조도 지운다.
    const sub = term.onSelectionChange(() => {
      if (!term.hasSelection()) clearActive();
    });
    return () => {
      sub.dispose();
      clearActive();
      m.dispose();
      marks.current = null;
      a.dispose();
      addon.current = null;
    };
    // clearActive 는 ref 만 다뤄 렌더마다 새로 만들어져도 같다.
  }, [term]);
  // 애드온은 현재 항목으로 옮겨 선택하는 일만 한다(decorations 없음 → 출력마다 다시 찾지 않는다).
  const opts = { caseSensitive: false };
  const find = (text: string, dir: 1 | -1) => {
    marks.current?.setQuery(text);
    const found = dir === 1 ? addon.current?.findNext(text, opts) : addon.current?.findPrevious(text, opts);
    if (found) markActive();
    else clearActive();
  };
  const next = (dir: 1 | -1) => {
    if (!q) return;
    find(q, dir);
  };
  const close = () => {
    clearActive();
    marks.current?.setQuery('');
    onClose();
    term.focus();
  };
  return (
    <div className="find-bar" role="search" onMouseDown={(e) => e.stopPropagation()}>
      <input
        autoFocus
        className="input"
        placeholder={tr('find.placeholder')}
        value={q}
        onChange={(e) => {
          setQ(e.target.value);
          if (e.target.value) find(e.target.value, 1);
          else {
            clearActive();
            marks.current?.setQuery('');
          }
        }}
        onKeyDown={(e) => {
          if (e.key === 'Escape') {
            e.preventDefault();
            e.stopPropagation();
            close();
          } else if (e.key === 'Enter') {
            e.preventDefault();
            next(e.shiftKey ? -1 : 1);
          }
        }}
      />
      <button className="btn small-btn" onClick={() => next(-1)}>{tr('find.prev')}</button>
      <button className="btn small-btn" onClick={() => next(1)}>{tr('find.next')}</button>
      <button className="btn small-btn" onClick={close}>{tr('common.close')}</button>
    </div>
  );
}

/**
 * 세션별로 살려 두는 터미널(xterm 과 그 DOM). 칸이 다시 그려져도(탭 전환·분할·크게 보기·그룹 이동)
 * 같은 터미널을 다시 붙이므로 화면 내용·스크롤백을 잃지 않고, 안 보이는 동안의 출력도 계속 받는다.
 * 세션이 작업 공간에서 사라질 때(칸·탭 닫기)만 정리한다.
 */
interface KeptTerm {
  term: Terminal;
  fit: FitAddon;
  el: HTMLDivElement;
  /** 지금 이 터미널을 보여 주는 칸(입력 대상 계산용). 칸이 바뀌면 붙일 때 갱신한다. */
  paneId: string;
  /** 키워드 강조(behavior/terminal.md §9). 설정의 규칙이 바뀌면 setRules 로 알린다. */
  highlighter: Highlighter;
  /** 마우스 선택 자동 복사 리스너. */
  copySub: IDisposable;
}
const keptTerms = new Map<string, KeptTerm>();

/** WebGL 렌더러를 붙인다. 만들 수 없으면(소프트웨어 렌더링·RDP 등) 조용히 DOM 렌더러를 유지하고,
 *  쓰다가 컨텍스트를 잃으면 애드온을 버려 DOM 렌더러로 돌아간다 (behavior/session.md §8 렌더러). */
function attachWebgl(t: Terminal): void {
  if (typeof WebGL2RenderingContext === 'undefined') return; // 테스트(jsdom)·미지원 환경
  try {
    const gl = new WebglAddon();
    gl.onContextLoss(() => gl.dispose());
    t.loadAddon(gl);
  } catch {
    // WebGL 불가: DOM 렌더러 그대로
  }
}

function createKeptTerm(sessionId: string, paneId: string, host: HTMLElement): KeptTerm {
  const el = document.createElement('div');
  el.className = 'term-keep';
  const tracker = new LineTracker();
  const st = useApp.getState().store?.settings;
  const theme = resolveTermTheme(st?.terminalTheme);
  const t = new Terminal({
    fontFamily: st?.fontFamily,
    fontSize: st?.fontSize ?? 14,
    lineHeight: st?.lineHeight ?? 1.15,
    cursorStyle: st?.cursorStyle ?? 'block',
    cursorBlink: st?.cursorBlink ?? true,
    scrollback: st?.scrollback ?? 10000,
    rightClickSelectsWord: false,
    macOptionClickForcesSelection: true,
    altClickMovesCursor: true,
    drawBoldTextInBrightColors: true,
    minimumContrastRatio: 1,
    smoothScrollDuration: 0,
    allowProposedApi: true,
    // 찾기 결과를 스크롤 막대 자리에 표시한다(ui/panes.md §2.4). 폭은 xterm 스크롤 막대 기본 폭과 같아 칸 수가 바뀌지 않는다.
    overviewRuler: { width: FIND_RULER_WIDTH },
    theme: {
      overviewRulerBorder: theme.background,
      background: theme.background,
      foreground: theme.foreground,
      cursor: theme.cursor,
      cursorAccent: theme.background,
      selectionBackground: theme.selectionBackground,
      black: theme.black, red: theme.red, green: theme.green, yellow: theme.yellow,
      blue: theme.blue, magenta: theme.magenta, cyan: theme.cyan, white: theme.white,
      brightBlack: theme.brightBlack, brightRed: theme.brightRed, brightGreen: theme.brightGreen,
      brightYellow: theme.brightYellow, brightBlue: theme.brightBlue, brightMagenta: theme.brightMagenta,
      brightCyan: theme.brightCyan, brightWhite: theme.brightWhite,
    },
  });
  const fit = new FitAddon();
  t.loadAddon(fit);
  // 링크: Ctrl+클릭으로 기본 브라우저에서 연다(terminal.md §1).
  t.loadAddon(new WebLinksAddon((ev, uri) => {
    if (!ev.ctrlKey) return;
    void getBackend().invoke('app.openExternal', [uri]).catch(() => undefined);
  }));
  // open 은 문서에 붙은 요소에서 해야 글자 크기를 잴 수 있으므로 칸에 먼저 붙인다.
  host.appendChild(el);
  t.open(el);
  attachWebgl(t);
  // done 은 xterm 이 다 쓴 뒤 불린다 → session.ack (출력 흐름 제어, behavior/session.md §8).
  useWorkspace.getState().registerWriter(sessionId, (text, done) => t.write(text, done));

  // 입력: 동시 입력 대상으로 보낸다. Enter 는 위험 명령 가드를 거친다.
  t.onData((data) => {
    const guardOn = !!useApp.getState().store?.settings.guardDangerousOnProd;
    const info = useWorkspace.getState().sessions[sessionId]?.info;
    if (data === '\r' && guardOn && info?.env === 'prod') {
      const line = tracker.current();
      tracker.reset();
      if (line && matchesDangerous(line, useApp.getState().store?.settings.dangerousPatterns ?? DANGEROUS_PATTERNS)) {
        void useUi.getState().confirm({
          title: tr('run.dangerTitle'),
          body: tr('run.dangerBody', { name: relabel(info.title) }),
          detail: line,
          okText: tr('run.ok'),
          danger: true,
        }).then((ok) => {
          if (ok) for (const sid of targetsFor(k.paneId, sessionId)) getBackend().termInput(sid, '\r');
          t.focus();
        });
        return;
      }
    }
    tracker.feed(data);
    for (const sid of targetsFor(k.paneId, sessionId)) getBackend().termInput(sid, data);
  });

  // 선택하면 자동 복사: 마우스로 선택을 끝냈을 때만(찾기가 바꾼 선택은 복사하지 않는다, KI-011).
  const copySub = attachCopyOnMouseSelect(el, t, () => !!useApp.getState().store?.settings.copyOnSelect, () => void copySelection(t));

  // 벨: 설정이 켜져 있을 때만 짧은 소리
  t.onBell(() => {
    if (useApp.getState().store?.settings.bell) beep();
  });

  // OSC 52: 쓰기만 처리, 읽기는 거절(terminal.md §8)
  t.parser.registerOscHandler(52, (payload) => {
    if (!useApp.getState().store?.settings.osc52Copy) return true;
    const text = decodeOsc52(payload);
    if (text !== null) void getBackend().invoke('clip.writeText', [text]).catch(() => undefined);
    return true;
  });

  // 키 처리: 앱 단축키는 터미널로 보내지 않는다(window 캡처에서 처리).
  // 단축키를 껐거나 그 키를 비웠으면 앱 단축키가 아니므로 터미널(서버)로 간다 (shortcuts.md §5, D126).
  t.attachCustomKeyEventHandler((e) => {
    if (e.type !== 'keydown') return true;
    if (useUi.getState().dialogs.length > 0) return false;
    if (activeShortcut(e)) return false;
    if (e.ctrlKey && e.shiftKey && e.code === 'KeyC') {
      void copySelection(t);
      return false;
    }
    if (e.ctrlKey && !e.shiftKey && !e.altKey && e.code === 'KeyC' && t.hasSelection()) {
      void copySelection(t);
      t.clearSelection();
      return false;
    }
    if (e.key === 'Enter' && e.shiftKey && !e.ctrlKey && !e.altKey && useApp.getState().store?.settings.shiftEnterNewline) {
      for (const sid of targetsFor(k.paneId, sessionId)) getBackend().termInput(sid, '\x1b\r');
      return false;
    }
    if (e.ctrlKey && e.altKey && e.code === 'KeyV') {
      void forceImagePaste(sessionId, targetsFor(k.paneId, sessionId));
      return false;
    }
    const pasteKey = (e.ctrlKey && e.code === 'KeyV' && !e.shiftKey && !e.altKey && useApp.getState().store?.settings.ctrlVPaste)
      || (e.ctrlKey && e.shiftKey && e.code === 'KeyV')
      || (e.shiftKey && e.code === 'Insert');
    if (pasteKey) {
      void pasteFromClipboard(k.paneId, sessionId, t);
      return false;
    }
    return true;
  });

  // 우클릭 규칙(panes §2.3): Shift 없이 설정이 켜져 있으면 복사 또는 붙여넣기, 아니면 메뉴(칸이 연다).
  const onContext = (ev: MouseEvent) => {
    const s = useApp.getState().store?.settings;
    if (ev.shiftKey || !s?.rightClickPaste) return;
    ev.preventDefault();
    ev.stopPropagation();
    if (t.hasSelection() && !s.copyOnSelect) {
      void copySelection(t);
      t.clearSelection();
      return;
    }
    void pasteFromClipboard(k.paneId, sessionId, t);
  };
  el.addEventListener('contextmenu', onContext, true);

  // Ctrl+휠: 글자 크기 ±1(8~32), 페이지 스크롤은 막는다(shortcuts.md §3).
  const onWheel = (ev: WheelEvent) => {
    if (!ev.ctrlKey) return;
    ev.preventDefault();
    const st3 = useApp.getState().store;
    if (!st3) return;
    const next = Math.max(8, Math.min(32, st3.settings.fontSize + (ev.deltaY < 0 ? 1 : -1)));
    if (next !== st3.settings.fontSize) void useApp.getState().patchStore({ settings: { ...st3.settings, fontSize: next } });
  };
  el.addEventListener('wheel', onWheel, { passive: false });

  const k: KeptTerm = { term: t, fit, el, paneId, highlighter: attachHighlighter(t, st?.highlightRules), copySub };
  keptTerms.set(sessionId, k);
  return k;
}

function disposeKeptTerm(sessionId: string): void {
  const k = keptTerms.get(sessionId);
  if (!k) return;
  keptTerms.delete(sessionId);
  k.highlighter.dispose();
  k.copySub.dispose();
  k.el.remove();
  k.term.dispose();
  if (useWorkspace.getState().writers[sessionId]) useWorkspace.getState().registerWriter(sessionId, null);
}

// 키워드 강조 규칙이 바뀌면 살려 둔 모든 터미널(안 보이는 탭 포함)에 알린다.
useApp.subscribe((s) => {
  const rules = s.store?.settings.highlightRules;
  for (const k of keptTerms.values()) k.highlighter.setRules(rules);
});

// 세션이 닫혀 작업 공간에서 빠지면 살려 둔 터미널도 버린다.
useWorkspace.subscribe((s) => {
  for (const sid of [...keptTerms.keys()]) if (!s.sessions[sid]) disposeKeptTerm(sid);
});

export function TerminalView({ paneId, sessionId, focused }: Props) {
  const hostRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<Terminal | null>(null);
  const fitRef = useRef<FitAddon | null>(null);
  const settings = useApp((s) => s.store?.settings);
  const [findOpen, setFindOpen] = useState(false);
  const [term, setTerm] = useState<Terminal | null>(null);

  // 찾기 요청 수신(이 칸에 대한 요청만)
  useEffect(() => {
    const on = (e: Event) => {
      if ((e as CustomEvent<string>).detail === paneId) setFindOpen(true);
    };
    window.addEventListener(FIND_EVENT, on);
    return () => window.removeEventListener(FIND_EVENT, on);
  }, [paneId]);

  // 붙이기·떼기: 세션의 터미널을 (없으면 만들어) 이 칸에 붙이고, 칸이 사라지면 떼기만 한다(dispose 하지 않음).
  useEffect(() => {
    const host = hostRef.current!;
    const k = keptTerms.get(sessionId) ?? createKeptTerm(sessionId, paneId, host);
    k.paneId = paneId;
    const t = k.term;
    const fit = k.fit;
    host.appendChild(k.el);
    useWorkspace.getState().flushDeferred(sessionId); // 숨어 있던 동안 모아 둔 출력을 먼저 쓴다
    termRef.current = t;
    fitRef.current = fit;
    termHandles.set(paneId, t);
    setTerm(t);

    // 크기 맞춤: 숨겨졌거나 너무 작으면 맞추지 않는다.
    const sendSize = () => {
      if (host.clientWidth < 20 || host.clientHeight < 20) return;
      fit.fit();
      useWorkspace.getState().setSize(sessionId, t.cols, t.rows);
      getBackend().termResize(sessionId, t.cols, t.rows);
    };
    const ro = new ResizeObserver(sendSize);
    ro.observe(host);
    sendSize();
    // 떼어 둔 동안 받은 출력을 다시 그린다.
    t.refresh(0, t.rows - 1);

    return () => {
      ro.disconnect();
      if (termHandles.get(paneId) === t) termHandles.delete(paneId);
      if (k.el.parentElement === host) host.removeChild(k.el);
      termRef.current = null;
      fitRef.current = null;
      setTerm(null);
    };
  }, [sessionId, paneId]);

  // 설정 변경은 열린 터미널 모두에 즉시 적용
  useEffect(() => {
    const t = termRef.current;
    if (!t || !settings) return;
    const theme = resolveTermTheme(settings.terminalTheme);
    t.options.fontFamily = settings.fontFamily;
    t.options.fontSize = settings.fontSize;
    t.options.lineHeight = settings.lineHeight;
    t.options.cursorStyle = settings.cursorStyle;
    t.options.cursorBlink = settings.cursorBlink;
    t.options.scrollback = settings.scrollback;
    t.options.theme = { ...t.options.theme, background: theme.background, foreground: theme.foreground, cursor: theme.cursor, selectionBackground: theme.selectionBackground, overviewRulerBorder: theme.background };
    fitRef.current?.fit();
  }, [settings]);

  // 포커스된 칸이면 입력 포커스를 준다.
  useEffect(() => {
    if (focused && termRef.current) termRef.current.focus();
  }, [focused]);

  return (
    <div className="term-wrap" style={{ position: 'relative', flex: 1, minWidth: 0, minHeight: 0, display: 'flex' }}>
      <div ref={hostRef} className="term-host selectable" onMouseDown={() => useWorkspace.getState().focusPane(paneId)} />
      {findOpen && term && <FindBar term={term} paneId={paneId} onClose={() => setFindOpen(false)} />}
    </div>
  );
}

export { insertPaths };
