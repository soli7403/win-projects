// 빠른 접속 팔레트(Ctrl+K). 근거: spec/ui/dialogs.md §2.
import { useEffect, useMemo, useRef, useState, type KeyboardEvent } from 'react';
import { Monitor, Network, Terminal, Usb, Zap } from 'lucide-react';
import type { Protocol } from '../ipc/types';

const PROTO_ICON: Record<Protocol, typeof Terminal> = { ssh: Terminal, telnet: Network, serial: Usb, local: Monitor };
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { useWorkspace } from '../state/workspace';
import { openHost, runShortcut } from '../state/actions';
import { rankHosts, isLocalPrefix, parseDirect } from '../lib/quick';
import { effectiveColor, recentHosts } from '../lib/hosts';
import { sidebarSubtitle } from '../lib/tree';
import { getBackend, errorMessage } from '../ipc/backend';

type Row =
  | { kind: 'direct'; key: string; label: string; sub: string; target: { protocol: 'ssh' | 'telnet'; host: string; port: number; username?: string } }
  | { kind: 'local'; key: string; label: string; sub: string }
  | { kind: 'host'; key: string; label: string; sub: string; hostId: string; color?: string }
  | { kind: 'cmd'; key: string; label: string; sub?: string; run: () => void };

export function QuickPalette() {
  const quick = useUi((s) => s.quick);
  const close = useUi((s) => s.closeQuick);
  const store = useApp((s) => s.store);
  const groups = store?.groups ?? [];
  const hosts = store?.hosts ?? [];
  const [input, setInput] = useState('');
  const [sel, setSel] = useState(0);
  const listRef = useRef<HTMLDivElement>(null);
  const openEditor = useUi((s) => s.openEditor);
  const cmdMode = input.trimStart().startsWith('>');

  const rows: Row[] = useMemo(() => {
    const q = input.trim();
    if (cmdMode) {
      const word = input.trimStart().slice(1).trim().toLowerCase();
      const cmds: Row[] = [
        { kind: 'cmd', key: 'c-local', label: '새 로컬 터미널', sub: '새 탭', run: () => runShortcut('newLocal') },
        { kind: 'cmd', key: 'c-new-host', label: '새 서버 등록', run: () => openEditor({ kind: 'host', hostId: null, groupId: null }) },
        { kind: 'cmd', key: 'c-new-folder', label: '새 폴더', run: () => openEditor({ kind: 'folder', folderId: null, parentId: null }) },
        { kind: 'cmd', key: 'c-import', label: 'PuTTY / SSH config 가져오기', run: () => useUi.getState().openImportDialog() },
        { kind: 'cmd', key: 'c-settings', label: '설정 열기', run: () => useUi.getState().openSettings() },
        { kind: 'cmd', key: 'c-theme', label: `테마 전환 (${store?.settings.theme === 'dark' ? '라이트' : '다크'})`, run: () => toggleTheme() },
        { kind: 'cmd', key: 'c-grid', label: '모든 탭을 그리드로 모으기', run: () => useWorkspace.getState().gridTabs() },
        { kind: 'cmd', key: 'c-lock', label: '볼트 잠그기', run: () => void getBackend().invoke('vault.lock').catch((e) => useApp.getState().pushToast({ kind: 'error', text: errorMessage(e) })) },
      ];
      return cmds.filter((c) => c.kind === 'cmd' && c.label.toLowerCase().includes(word));
    }
    const out: Row[] = [];
    const direct = parseDirect(q);
    if (direct) {
      out.push({
        kind: 'direct',
        key: 'direct',
        label: `바로 접속: ${q}`,
        sub: '저장하지 않고 연결합니다',
        target: direct,
      });
    }
    const localWanted = isLocalPrefix(q);
    const localRow: Row = { kind: 'local', key: 'local', label: '새 로컬 터미널', sub: '이 PC의 셸 · Ctrl+Alt+T' };
    if (localWanted) out.push(localRow);
    const matches = q ? rankHosts(hosts, groups, q) : recentHosts(hosts, 50);
    for (const h of matches) {
      out.push({
        kind: 'host',
        key: `h-${h.id}`,
        label: h.alias,
        sub: `${sidebarSubtitle(h)}${groups.find((g) => g.id === h.groupId) ? ` · ${groups.find((g) => g.id === h.groupId)!.name}` : ''}`,
        hostId: h.id,
        color: effectiveColor(h, groups),
      });
    }
    if (!q) out.push(localRow);
    return out;
    // toggleTheme 은 안정적인 함수 참조가 필요 없다(렌더마다 새로 만들어도 무방).
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [input, hosts, groups, cmdMode, store?.settings.theme]);

  useEffect(() => {
    setSel(0);
  }, [input]);

  useEffect(() => {
    if (quick.open) setInput('');
  }, [quick.open]);

  useEffect(() => {
    listRef.current?.querySelector<HTMLElement>(`[data-i="${sel}"]`)?.scrollIntoView({ block: 'nearest' });
  }, [sel]);

  if (!quick.open) return null;

  const runRow = (r: Row, mod: 'tab' | 'right' | 'down') => {
    // 빈 칸의 [빠른 접속…]으로 열었으면(대상 칸 있음) 보조키 없는 Enter·클릭은 그 칸을 채운다 (panes.md §3).
    // openHost 는 대상 칸이 빈 칸이면 분할하지 않고 채운다.
    const where =
      mod === 'tab' && !quick.targetPane
        ? { kind: 'tab' as const }
        : { kind: 'split' as const, dir: mod === 'right' ? ('row' as const) : ('col' as const), paneId: quick.targetPane };
    close();
    if (r.kind === 'cmd') {
      r.run();
      return;
    }
    if (r.kind === 'host') {
      void openHost(r.hostId, where);
      return;
    }
    if (r.kind === 'local') {
      void openHost(null, where, { host: '', protocol: 'local' });
      return;
    }
    if (r.kind === 'direct') {
      void openHost(null, where, { host: r.target.host, port: r.target.port, username: r.target.username, protocol: r.target.protocol });
    }
  };

  const onKey = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Escape') {
      e.preventDefault();
      close();
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      setSel((s) => Math.min(rows.length - 1, s + 1));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setSel((s) => Math.max(0, s - 1));
    } else if (e.key === 'Enter') {
      e.preventDefault();
      const r = rows[sel];
      if (!r) return;
      const mod = e.ctrlKey ? 'right' : e.shiftKey ? 'down' : 'tab';
      runRow(r, mod);
    }
  };

  return (
    <div className="overlay quick-overlay" onMouseDown={(e) => e.target === e.currentTarget && close()}>
      <div className="quick" role="dialog" aria-label="빠른 접속">
        <div className="quick-input">
          <Zap size={16} color="var(--accent)" />
          <input
            autoFocus
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={onKey}
            placeholder="서버 이름, IP, user@host:port 입력  ·  > 명령"
            aria-label="빠른 접속 입력"
          />
        </div>
        <div className="quick-list" ref={listRef}>
          {rows.length === 0 && (
            <div className="faint small" style={{ padding: '10px 12px' }}>
              {`일치하는 서버가 없습니다. user@host 형식으로 입력하면 바로 접속할 수 있습니다.`}
            </div>
          )}
          {rows.map((r, i) => (
            <div
              key={r.key}
              data-i={i}
              className={`quick-row${i === sel ? ' sel' : ''}`}
              onMouseEnter={() => setSel(i)}
              onMouseDown={(e) => {
                e.preventDefault();
                const mod = e.ctrlKey ? 'right' : e.shiftKey ? 'down' : 'tab';
                runRow(r, mod);
              }}
            >
              {r.kind === 'host' && <span className="strip" style={{ background: r.color ?? 'transparent' }} />}
              {r.kind === 'host' && (() => { const Icon = PROTO_ICON[hosts.find((x) => x.id === r.hostId)?.protocol ?? 'ssh']; return <Icon size={14} />; })()}
              <span className="quick-label">{r.label}</span>
              <span className="faint small quick-sub">{r.sub}</span>
              {i === sel && <span className="kbd">Enter</span>}
            </div>
          ))}
        </div>
        <div className="quick-foot faint small">
          <span className="kbd">Enter</span> {quick.targetPane ? '이 칸에서 열기' : '새 탭'} &nbsp;
          <span className="kbd">Ctrl+Enter</span> 오른쪽 분할 &nbsp;
          <span className="kbd">Shift+Enter</span> 아래 분할 &nbsp;
          <span className="kbd">&gt;</span> 명령
        </div>
      </div>
    </div>
  );
}

/** 테마 전환: 앱 테마와 짝이 되는 터미널 테마를 함께 바꾼다(dialogs.md §2.1 (a) 6번). */
export function toggleTheme(): void {
  const st = useApp.getState().store;
  if (!st) return;
  const toLight = st.settings.theme === 'dark';
  const term = st.settings.terminalTheme;
  const nextTerm = toLight && term === 'TermGo Dark' ? 'TermGo Light' : !toLight && term === 'TermGo Light' ? 'TermGo Dark' : term;
  void useApp.getState().patchStore({ settings: { ...st.settings, theme: toLight ? 'light' : 'dark', terminalTheme: nextTerm } });
}
