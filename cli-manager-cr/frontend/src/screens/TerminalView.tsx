// 터미널 칸 본문(xterm.js). 근거: spec/behavior/terminal.md §1~§5, §6, §8, spec/behavior/shortcuts.md §2~§3, spec/ui/panes.md §2.3~§2.4.
import { useEffect, useRef, useState } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { SearchAddon } from '@xterm/addon-search';
import { WebLinksAddon } from '@xterm/addon-web-links';
import '@xterm/xterm/css/xterm.css';
import { getBackend, errorMessage } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { useWorkspace } from '../state/workspace';
import { sessionsIn, findPane, type LayoutNode } from '../lib/layout';
import { matchesDangerous, LineTracker, pasteLineCount } from '../lib/termGuard';
import { decodeOsc52 } from '../lib/osc52';
import { shortcutFor } from '../lib/keys';
import { resolveTermTheme } from '../lib/termThemes';
import { DANGEROUS_PATTERNS } from '../lib/defaults';
import { clipboardKind, forceImagePaste, insertPaths, uploadAndInsert } from '../state/paste';

interface Props {
  paneId: string;
  sessionId: string;
  focused: boolean;
}

/** 칸 id → xterm 인스턴스. 칸 우클릭 메뉴가 복사·모두 선택·화면 지우기에 쓴다. */
export const termHandles = new Map<string, Terminal>();

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
export const FIND_EVENT = 'termgo:find';

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
  const name = info?.title ?? '세션';
  const patterns = st?.dangerousPatterns ?? DANGEROUS_PATTERNS;
  const send = () => {
    for (const sid of targets) getBackend().termInput(sid, text);
    term.focus();
  };
  if (st?.guardDangerousOnProd && info?.env === 'prod' && matchesDangerous(text, patterns)) {
    const ok = await useUi.getState().confirm({
      title: '운영 서버 위험 명령 붙여넣기',
      body: `[${name}] 붙여넣을 내용에 위험한 명령이 있습니다.`,
      detail: text.slice(0, 2000),
      okText: '붙여넣기',
      danger: true,
    });
    if (ok) send();
    else term.focus();
    return;
  }
  const lines = pasteLineCount(text);
  if (st?.confirmMultilinePaste && lines >= 2 && !term.modes.bracketedPasteMode) {
    const ok = await useUi.getState().confirm({
      title: '여러 줄 붙여넣기',
      body: `${lines}줄을 [${name}]에 붙여넣습니다. 각 줄이 명령으로 실행될 수 있습니다.`,
      detail: text.slice(0, 2000),
      okText: '붙여넣기',
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
  void paneId;
  useEffect(() => {
    const a = new SearchAddon();
    term.loadAddon(a);
    addon.current = a;
    return () => a.clearDecorations();
  }, [term]);
  const opts = { caseSensitive: false, decorations: { matchOverviewRuler: '#f59e0b', activeMatchColorOverviewRuler: '#ef4444', matchBackground: '#f59e0b55', activeMatchBackground: '#ef444488' } };
  const next = (dir: 1 | -1) => {
    if (!q) return;
    if (dir === 1) addon.current?.findNext(q, opts);
    else addon.current?.findPrevious(q, opts);
  };
  const close = () => {
    addon.current?.clearDecorations();
    onClose();
    term.focus();
  };
  return (
    <div className="find-bar" role="search" onMouseDown={(e) => e.stopPropagation()}>
      <input
        autoFocus
        className="input"
        placeholder="터미널에서 찾기"
        value={q}
        onChange={(e) => {
          setQ(e.target.value);
          if (e.target.value) addon.current?.findNext(e.target.value, opts);
          else addon.current?.clearDecorations();
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
      <button className="btn small-btn" onClick={() => next(-1)}>이전</button>
      <button className="btn small-btn" onClick={() => next(1)}>다음</button>
      <button className="btn small-btn" onClick={close}>닫기</button>
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
}
const keptTerms = new Map<string, KeptTerm>();

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
    theme: {
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
  useWorkspace.getState().registerWriter(sessionId, (text) => t.write(text));

  // 입력: 동시 입력 대상으로 보낸다. Enter 는 위험 명령 가드를 거친다.
  t.onData((data) => {
    const guardOn = !!useApp.getState().store?.settings.guardDangerousOnProd;
    const info = useWorkspace.getState().sessions[sessionId]?.info;
    if (data === '\r' && guardOn && info?.env === 'prod') {
      const line = tracker.current();
      tracker.reset();
      if (line && matchesDangerous(line, useApp.getState().store?.settings.dangerousPatterns ?? DANGEROUS_PATTERNS)) {
        void useUi.getState().confirm({
          title: '운영 서버 위험 명령',
          body: `[${info.title}] 운영 서버에서 실행하려는 명령입니다. 정말 실행할까요?`,
          detail: line,
          okText: '실행',
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

  // 선택하면 자동 복사
  t.onSelectionChange(() => {
    const st2 = useApp.getState().store?.settings;
    if (st2?.copyOnSelect && t.getSelection()) void copySelection(t);
  });

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
  t.attachCustomKeyEventHandler((e) => {
    if (e.type !== 'keydown') return true;
    if (useUi.getState().dialogs.length > 0) return false;
    if (shortcutFor(e)) return false;
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

  const k: KeptTerm = { term: t, fit, el, paneId };
  keptTerms.set(sessionId, k);
  return k;
}

function disposeKeptTerm(sessionId: string): void {
  const k = keptTerms.get(sessionId);
  if (!k) return;
  keptTerms.delete(sessionId);
  k.el.remove();
  k.term.dispose();
  if (useWorkspace.getState().writers[sessionId]) useWorkspace.getState().registerWriter(sessionId, null);
}

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
    t.options.theme = { ...t.options.theme, background: theme.background, foreground: theme.foreground, cursor: theme.cursor, selectionBackground: theme.selectionBackground };
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
