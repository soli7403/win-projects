// 사이드바: 검색 · 즐겨찾기 · 최근 접속 · 서버/폴더 트리. 근거: spec/ui/sidebar.md §1~§7, §10.
// 이번 라운드 제외: 끌어 놓기(§9), 칸으로 끌기(panes), 행 호버 버튼의 실제 연결(탭 단계에서 연결).
import { useEffect, useMemo, useRef, useState, type DragEvent, type KeyboardEvent, type PointerEvent as RPointerEvent, type ReactNode } from 'react';
import { ChevronDown, ChevronRight, Clock, Download, FolderOpen, FolderPlus, Plus, Search, Star, Terminal, Network, Usb, Monitor } from 'lucide-react';
import type { Group, Host, Protocol } from '../ipc/types';
import { getBackend, errorMessage } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi, type MenuItem } from '../state/ui';
import { effectiveColor } from '../lib/hosts';
import { openFolderGrid, openHost } from '../state/actions';
import { HOST_MIME } from './Panes';
import { focusActiveTerminal } from './TerminalView';
import {
  canDropGroup, clampSidebarWidth, connectedCountByHost, countHostsIn, duplicateHostRecord, effectiveGroupId, folderDeletePlan, fullTarget,
  matchesQuery, moveHost, sidebarSubtitle, sortGroups, sortHosts,
} from '../lib/tree';
import { EnvPill } from '../ui/Controls';
import { ENV_INFO } from '../lib/defaults';
import { useWorkspace } from '../state/workspace';
import { findPane, findPaneBySession, sessionsIn, type LayoutNode } from '../lib/layout';

/** 이 서버의 열린 칸 목록(모든 탭, 탭 순서대로). */
function panesOfHost(hostId: string): { tabId: string; paneId: string }[] {
  const ws = useWorkspace.getState();
  const out: { tabId: string; paneId: string }[] = [];
  for (const t of ws.tabs) {
    for (const sid of sessionsIn(t.root as LayoutNode)) {
      if (ws.sessions[sid]?.info.hostId !== hostId) continue;
      const pane = findPaneBySession(t.root as LayoutNode, sid);
      if (pane) out.push({ tabId: t.id, paneId: pane.id });
    }
  }
  return out;
}

/** 한 번 클릭(§9): 열린 칸으로 이동, 여러 개면 순환. 열린 세션이 없으면 아무 동작 없음. */
function focusOpenSessionOf(hostId: string): boolean {
  const list = panesOfHost(hostId);
  if (list.length === 0) return false;
  const ws = useWorkspace.getState();
  const cur = ws.tabs.find((t) => t.id === ws.activeTabId)?.focusedPane;
  const idx = list.findIndex((p) => p.paneId === cur);
  const next = list[(idx + 1) % list.length];
  ws.activateTab(next.tabId);
  ws.focusPane(next.paneId);
  // 이미 열린 세션으로 옮겨 가면 바로 입력할 수 있게 터미널에 포커스.
  focusActiveTerminal();
  return true;
}

/** 더블클릭(§9): 포커스 칸이 비었으면 거기서 열고, 열린 세션이 있으면 이동, 없으면 새 탭. */
function openSmart(hostId: string): void {
  const ws = useWorkspace.getState();
  const active = ws.tabs.find((t) => t.id === ws.activeTabId);
  const focused = active?.focusedPane ? findPane(active.root as LayoutNode, active.focusedPane) : undefined;
  if (focused && !focused.sessionId) {
    void openHost(hostId, { kind: 'split', dir: 'row', paneId: focused.id });
    return;
  }
  if (focusOpenSessionOf(hostId)) return;
  void openHost(hostId);
}


const ENV_COLOR = Object.fromEntries(Object.entries(ENV_INFO).map(([k, v]) => [k, v.color])) as Record<string, string>;

const PROTOCOL_ICON: Record<Protocol, typeof Terminal> = { ssh: Terminal, telnet: Network, serial: Usb, local: Monitor };
const INDENT = 14;

export function Sidebar() {
  const store = useApp((s) => s.store);
  const patchStore = useApp((s) => s.patchStore);
  const pushToast = useApp((s) => s.pushToast);
  const selected = useUi((s) => s.selectedHostId);
  const setSelected = useUi((s) => s.setSelectedHost);
  const openEditor = useUi((s) => s.openEditor);
  const openMenu = useUi((s) => s.openMenu);
  const confirm = useUi((s) => s.confirm);
  const [query, setQuery] = useState('');
  // 끌어 놓기: 끌고 있는 항목과 놓을 대상(폴더 id, 최상위는 'root').
  const dragRef = useRef<{ kind: 'host' | 'group'; id: string } | null>(null);
  const [dropTarget, setDropTarget] = useState<string | null>(null);
  const [hoverId, setHoverId] = useState<string | null>(null);
  // 서버별 열린 세션 수(§5 연결 점·개수).
  const sessionMap = useWorkspace((s) => s.sessions);
  const openByHost = useMemo(() => connectedCountByHost(sessionMap), [sessionMap]);

  // 선택 동기화(§5): 포커스된 칸의 세션이 저장된 서버면 그 서버를 선택하고 보이게 한다.
  const focusedHostId = useWorkspace((s) => {
    const t = s.tabs.find((x) => x.id === s.activeTabId);
    if (!t?.focusedPane) return null;
    const sid = findPane(t.root as LayoutNode, t.focusedPane)?.sessionId;
    return sid ? s.sessions[sid]?.info.hostId ?? null : null;
  });
  useEffect(() => {
    if (!focusedHostId) return;
    setSelected(focusedHostId);
    document.querySelector(`[data-host-id="${focusedHostId}"]`)?.scrollIntoView({ block: 'nearest' });
  }, [focusedHostId, setSelected]);

  /** 놓기 처리(§9): 서버는 폴더로, 폴더는 폴더로(순환 금지), 빈 영역은 최상위로. */
  const handleDrop = (targetId: string | null) => {
    const d = dragRef.current;
    dragRef.current = null;
    setDropTarget(null);
    if (!d) return;
    if (d.kind === 'host') {
      const next = moveHost(hosts, d.id, targetId);
      if (next !== hosts) void patchStore({ hosts: next });
      return;
    }
    const cur = groups.find((x) => x.id === d.id);
    if (!cur || (cur.parentId ?? null) === targetId || !canDropGroup(groups, d.id, targetId)) return;
    void patchStore({ groups: groups.map((x) => (x.id === d.id ? { ...x, parentId: targetId } : x)) });
  };
  const groups = store?.groups ?? [];
  const hosts = store?.hosts ?? [];
  const searching = query.trim().length > 0;
  const width = store?.settings.sidebarWidth ?? 260;

  const matched = useMemo(() => hosts.filter((h) => matchesQuery(h, groups, query)), [hosts, groups, query]);

  // 폴더 펼침/접힘 토글은 저장한다(collapsed).
  const toggleFolder = (g: Group) => {
    void patchStore({ groups: groups.map((x) => (x.id === g.id ? { ...x, collapsed: !x.collapsed } : x)) });
  };

  const removeHost = async (h: Host) => {
    const ok = await confirm({ title: '서버 삭제', body: `"${h.alias}"을(를) 삭제할까요?`, okText: '삭제', danger: true });
    if (!ok) return;
    await patchStore({ hosts: hosts.filter((x) => x.id !== h.id) });
    if (selected === h.id) setSelected(null);
  };

  const duplicateHost = async (h: Host) => {
    const copy = duplicateHostRecord(h, crypto.randomUUID(), Date.now());
    await patchStore({ hosts: [...hosts, copy] });
  };

  const toggleFavorite = async (h: Host) => {
    await patchStore({ hosts: hosts.map((x) => (x.id === h.id ? { ...x, favorite: !x.favorite } : x)) });
  };

  const copyAddress = async (h: Host) => {
    const text = `${h.username ? h.username + '@' : ''}${h.host}`;
    try {
      await getBackend().invoke('clip.writeText', [text]);
    } catch (e) {
      pushToast({ kind: 'error', text: errorMessage(e) });
    }
  };

  const deleteFolder = async (g: Group) => {
    const ok = await confirm({
      title: '폴더 삭제',
      body: `"${g.name}" 폴더를 삭제할까요? 안의 서버는 상위 폴더로 옮겨집니다.`,
      okText: '삭제',
      danger: true,
    });
    if (!ok) return;
    const plan = folderDeletePlan(groups, hosts, g.id);
    await patchStore({ groups: plan.groups, hosts: plan.hosts });
  };

  const hostMenu = (h: Host): MenuItem[] => [
    { label: '새 탭에서 연결', shortcut: 'Enter', onSelect: () => void openHost(h.id) },
    { label: '오른쪽으로 분할해 연결', onSelect: () => void openHost(h.id, { kind: 'rightGroup' }) },
    { label: '아래로 분할해 연결', onSelect: () => void openHost(h.id, { kind: 'split', dir: 'col' }) },
    { label: '위치 골라서 연결…', onSelect: () => useUi.getState().openLocation(h.id) },
    { separator: true },
    { label: '편집', shortcut: 'F2', onSelect: () => openEditor({ kind: 'host', hostId: h.id, groupId: null }) },
    { label: '복제', onSelect: () => void duplicateHost(h) },
    { label: h.favorite ? '즐겨찾기 해제' : '즐겨찾기', onSelect: () => void toggleFavorite(h) },
    { label: '주소 복사', onSelect: () => void copyAddress(h) },
    { separator: true },
    { label: '삭제', danger: true, onSelect: () => void removeHost(h) },
  ];

  const folderMenu = (g: Group): MenuItem[] => [
    { label: '폴더 전체를 그리드로 열기', onSelect: () => void openFolderGrid(g.id) },
    { separator: true },
    { label: '이 폴더에 새 서버', onSelect: () => openEditor({ kind: 'host', hostId: null, groupId: g.id }) },
    { label: '하위 폴더 만들기', onSelect: () => openEditor({ kind: 'folder', folderId: null, parentId: g.id }) },
    { label: '폴더 편집', onSelect: () => openEditor({ kind: 'folder', folderId: g.id, parentId: g.parentId }) },
    { separator: true },
    { label: '폴더 삭제', danger: true, onSelect: () => void deleteFolder(g) },
  ];

  const onKey = (e: KeyboardEvent<HTMLDivElement>) => {
    const h = hosts.find((x) => x.id === selected);
    if (e.key === 'Escape' && searching) setQuery('');
    if (e.key === 'Enter' && searching) {
      const first = matched[0];
      if (first) void openHost(first.id);
      return;
    }
    if (!h || e.target instanceof HTMLInputElement) return;
    if (e.key === 'F2') openEditor({ kind: 'host', hostId: h.id, groupId: null });
    if (e.key === 'Delete') void removeHost(h);
    if (e.key === 'Enter') void openHost(h.id, e.ctrlKey ? { kind: 'rightGroup' } : e.shiftKey ? { kind: 'split', dir: 'col' } : { kind: 'tab' });
  };

  const onResizeDown = (e: RPointerEvent<HTMLDivElement>) => {
    const startX = e.clientX;
    const startW = width;
    let latest = startW;
    const move = (ev: PointerEvent) => {
      latest = clampSidebarWidth(startW + ev.clientX - startX);
      useApp.setState((s) => (s.store ? { store: { ...s.store, settings: { ...s.store.settings, sidebarWidth: latest } } } : s));
    };
    const up = () => {
      window.removeEventListener('pointermove', move);
      window.removeEventListener('pointerup', up);
      const st = useApp.getState().store;
      if (st) void patchStore({ settings: { ...st.settings, sidebarWidth: latest } });
    };
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', up);
  };

  const hostRow = (h: Host, depth: number, opts: { selectable?: boolean } = { selectable: true }) => {
    const Icon = PROTOCOL_ICON[h.protocol];
    const color = effectiveColor(h, groups);
    const isSel = opts.selectable && selected === h.id;
    return (
      <div
        key={`${opts.selectable ? 'h' : 'fav'}-${h.id}`}
        className="side-row"
        role="treeitem"
        data-host-id={h.id}
        aria-selected={isSel}
        draggable={opts.selectable}
        onDragStart={(e: DragEvent<HTMLDivElement>) => {
          dragRef.current = { kind: 'host', id: h.id };
          e.dataTransfer.effectAllowed = 'move';
          e.dataTransfer.setData(HOST_MIME, h.id);
          e.dataTransfer.setData('text/plain', h.id);
        }}
        onDragEnd={() => {
          dragRef.current = null;
          setDropTarget(null);
        }}
        title={`${h.alias}\n${fullTarget(h)}${h.notes ? `\n${h.notes}` : ''}`}
        onClick={() => {
          if (opts.selectable) setSelected(h.id);
          focusOpenSessionOf(h.id);
        }}
        onDoubleClick={() => openSmart(h.id)}
        onContextMenu={(e) => {
          e.preventDefault();
          setSelected(h.id);
          openMenu(e.clientX, e.clientY, hostMenu(h));
        }}
        onMouseEnter={() => setHoverId(h.id)}
        onMouseLeave={() => setHoverId((cur) => (cur === h.id ? null : cur))}
        style={{ paddingLeft: 6 + depth * INDENT, background: isSel ? 'var(--accent-soft)' : undefined }}
      >
        <span className="strip" style={{ background: color ?? 'transparent' }} />
        <Icon size={13} />
        <span className="side-name">{h.alias || h.host}</span>
        <span className="faint small side-sub">{sidebarSubtitle(h)}</span>
        {openByHost[h.id] ? (
          <span className="small" style={{ color: 'var(--ok, #22c55e)', marginLeft: 'auto' }}>
            ● {openByHost[h.id] > 1 ? openByHost[h.id] : ''}
          </span>
        ) : null}
        {hoverId === h.id && opts.selectable && (
          <span style={{ display: 'inline-flex', gap: 2 }}>
            <button className="btn small-btn" title="분할로 연결" onClick={(e) => { e.stopPropagation(); void openHost(h.id, { kind: 'rightGroup' }); }}>분할로 연결</button>
            <button className="btn small-btn" title="하나 더 연결" onClick={(e) => { e.stopPropagation(); void openHost(h.id); }}>하나 더 연결</button>
          </span>
        )}
      </div>
    );
  };

  const folderRow = (g: Group, depth: number): ReactNode => {
    const expanded = !g.collapsed || searching;
    const count = countHostsIn(groups, hosts, g.id, (h) => matchesQuery(h, groups, query));
    if (searching && count === 0) return null;
    const color = g.color;
    const childGroups = sortGroups(groups.filter((x) => x.parentId === g.id));
    const childHosts = sortHosts(hosts.filter((h) => h.groupId === g.id && matchesQuery(h, groups, query)));
    return (
      <div key={`g-${g.id}`}>
        <div
          className="side-row folder"
          role="treeitem"
          aria-expanded={expanded}
          draggable
          onDragStart={(e) => {
            dragRef.current = { kind: 'group', id: g.id };
            e.dataTransfer.effectAllowed = 'move';
            e.dataTransfer.setData('text/plain', g.id);
          }}
          onDragEnd={() => {
            dragRef.current = null;
            setDropTarget(null);
          }}
          onDragOver={(e) => {
            const d = dragRef.current;
            if (d && (d.kind === 'host' || canDropGroup(groups, d.id, g.id))) {
              e.preventDefault();
              e.stopPropagation();
              setDropTarget(g.id);
            }
          }}
          onDragLeave={() => setDropTarget((cur) => (cur === g.id ? null : cur))}
          onDrop={(e) => {
            e.preventDefault();
            e.stopPropagation();
            handleDrop(g.id);
          }}
          style={{
            paddingLeft: 6 + depth * INDENT,
            outline: dropTarget === g.id ? '1px dashed var(--accent)' : undefined,
            background: dropTarget === g.id ? 'var(--accent-soft)' : undefined,
          }}
          onClick={() => !searching && toggleFolder(g)}
          onContextMenu={(e) => {
            e.preventDefault();
            openMenu(e.clientX, e.clientY, folderMenu(g));
          }}
        >
          {expanded ? <ChevronDown size={13} /> : <ChevronRight size={13} />}
          <FolderOpen size={14} color={color ?? (g.env && g.env !== 'none' ? ENV_COLOR[g.env] : 'var(--fg-3)')} />
          <span className="side-name" style={{ fontWeight: 700 }}>{g.name}</span>
          {g.env && g.env !== 'none' && <EnvPill env={g.env} />}
          <span className="faint small" style={{ marginLeft: 'auto' }}>{count}</span>
        </div>
        {expanded && (
          <>
            {childGroups.map((c) => folderRow(c, depth + 1))}
            {childHosts.map((h) => hostRow(h, depth + 1))}
          </>
        )}
      </div>
    );
  };

  const rootGroups = sortGroups(groups.filter((g) => !g.parentId || !groups.some((p) => p.id === g.parentId)));
  const looseHosts = sortHosts(hosts.filter((h) => !effectiveGroupId(h, groups) && matchesQuery(h, groups, query)));
  const favorites = hosts.filter((h) => h.favorite);
  const recent = [...hosts]
    .filter((h) => h.lastUsedAt)
    .sort((a, b) => (b.lastUsedAt ?? 0) - (a.lastUsedAt ?? 0))
    .slice(0, 5);
  const showRecent = !searching && hosts.length > 6 && recent.length > 0;
  const noMatch = searching && matched.length === 0 && hosts.length > 0;

  return (
    <aside className="sidebar" style={{ width }} aria-label="서버 목록" onKeyDown={onKey} tabIndex={-1}>
      <div className="side-head">
        <div className="search-box">
          <Search size={14} />
          <input
            className="search-input"
            placeholder="서버 검색"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            aria-label="서버 검색"
          />
        </div>
        <div className="side-btns">
          <button className="btn small-btn" onClick={() => openEditor({ kind: 'host', hostId: null, groupId: null })}><Plus size={13} />서버</button>
          <button className="btn small-btn" onClick={() => openEditor({ kind: 'folder', folderId: null, parentId: null })}><FolderPlus size={13} />폴더</button>
          <button className="btn small-btn" title="PuTTY / SSH config 가져오기" onClick={() => useUi.getState().openImportDialog()}><Download size={13} />가져오기</button>
        </div>
      </div>

      <div
        className="side-tree"
        role="tree"
        onDragOver={(e) => {
          if (!dragRef.current) return;
          e.preventDefault();
          setDropTarget('root');
        }}
        onDragLeave={(e) => {
          if (e.currentTarget === e.target) setDropTarget(null);
        }}
        onDrop={(e) => {
          e.preventDefault();
          handleDrop(null);
        }}
        style={{ outline: dropTarget === 'root' ? '1px dashed var(--accent)' : undefined }}
      >
        {!searching && favorites.length > 0 && (
          <>
            <div className="side-section"><Star size={12} /> 즐겨찾기</div>
            {favorites.map((h) => hostRow(h, 0, { selectable: false }))}
          </>
        )}
        {showRecent && (
          <>
            <div className="side-section"><Clock size={12} /> 최근 접속</div>
            {recent.map((h) => hostRow(h, 0, { selectable: false }))}
          </>
        )}
        <div className="side-section">서버{hosts.length > 0 ? ` · ${hosts.length}` : ''}</div>
        {rootGroups.map((g) => folderRow(g, 0))}
        {looseHosts.map((h) => hostRow(h, 0))}
        {hosts.length === 0 && (
          <div className="faint small" style={{ padding: '8px 10px', lineHeight: 1.6, whiteSpace: 'pre-line' }}>
            {'아직 저장된 서버가 없습니다.\n+ 서버로 추가하거나\n가져오기로 PuTTY 세션을 불러오세요.'}
          </div>
        )}
        {noMatch && (
          <div className="faint small" style={{ padding: '8px 10px' }}>{`"${query.trim()}"와 일치하는 서버가 없습니다`}</div>
        )}
      </div>

      <div className="side-resize" onPointerDown={onResizeDown} role="separator" aria-orientation="vertical" aria-label="사이드바 너비" />
    </aside>
  );
}
