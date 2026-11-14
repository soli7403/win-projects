// 사이드바: 검색 · 즐겨찾기 · 최근 접속 · 서버/폴더 트리. 근거: spec/ui/sidebar.md §1~§7, §10.
// 이번 라운드 제외: 끌어 놓기(§9), 칸으로 끌기(panes), 행 호버 버튼의 실제 연결(탭 단계에서 연결).
import { isValidElement, useEffect, useMemo, useRef, useState, type CSSProperties, type DragEvent, type KeyboardEvent, type PointerEvent as RPointerEvent, type ReactNode } from 'react';
import { ChevronDown, ChevronRight, Clock, Download, Folder, FolderOpen, FolderPlus, Plus, Search, Star } from 'lucide-react';
import { FOLDER_ICON_COLOR, PROTOCOL_ICON, protocolIconColor } from '../ui/protocolIcon';
import type { Group, Host } from '../ipc/types';
import { getBackend, errorMessage } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi, type MenuItem } from '../state/ui';
import { effectiveColor } from '../lib/hosts';
import { openFolderGrid, openHost } from '../state/actions';
import { HOST_MIME } from './Panes';
import { focusActiveTerminal } from './TerminalView';
import { rangeSelection, toggleSelection } from '../lib/multiSelect';
import {
  canDropGroup, clampSidebarWidth, connectedCountByHost, countHostsIn, duplicateHostRecord, effectiveGroupId, folderDeletePlan, fullTarget,
  matchesQuery, moveHosts, sidebarSubtitle, sortGroups, sortHosts,
} from '../lib/tree';
import { EnvPill } from '../ui/Controls';
import { ENV_INFO } from '../lib/defaults';
import { useWorkspace } from '../state/workspace';
import { findPane, findPaneBySession, sessionsIn, type LayoutNode } from '../lib/layout';
// 이 파일은 t 를 탭 변수 이름으로 쓰므로 번역 함수는 tr 로 가져온다(D75).
import { t as tr } from '../i18n';

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

/** 단계마다 들여쓰기(px). 점선 연결선의 가로 줄 길이도 이 값에서 나온다. */
const INDENT = 16;
/** 행의 기본 왼쪽 여백(px)과 펼침 화살표(13px) 가운데까지의 거리. 하위 항목의 세로 점선이 이 위치를 지난다. */
const ROW_PAD = 6;
const CHEVRON_MID = 6;

/** 펼친 폴더의 하위 항목들을 탐색기 형태의 점선(├ └)으로 잇는다. depth 는 부모 폴더의 단계. */
function TreeChildren({ depth, children }: { depth: number; children: ReactNode[] }) {
  const items = children.filter((c) => c !== null && c !== undefined && c !== false);
  const style = { '--gx': `${ROW_PAD + depth * INDENT + CHEVRON_MID}px`, '--indent': `${INDENT}px` } as CSSProperties;
  return (
    <div className="tree-children" style={style}>
      {items.map((node, i) => (
        <div key={isValidElement(node) && node.key !== null ? node.key : i} className={`tree-item${i === items.length - 1 ? ' last' : ''}`}>
          {node}
        </div>
      ))}
    </div>
  );
}

/** 여러 서버를 끌 때 커서 옆에 "서버 N개" 표시(브라우저 기본 끌기 그림 대신). */
function setDragBadge(e: DragEvent<HTMLElement>, text: string): void {
  if (typeof e.dataTransfer.setDragImage !== 'function') return;
  const el = document.createElement('div');
  el.className = 'drag-badge';
  el.textContent = text;
  document.body.appendChild(el);
  e.dataTransfer.setDragImage(el, 10, 10);
  // 브라우저가 그림을 뜬 뒤 지운다.
  setTimeout(() => el.remove(), 0);
}

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
  // 서버는 여러 개를 함께 끌 수 있다(ids), 폴더는 하나씩.
  const dragRef = useRef<{ kind: 'host'; ids: string[] } | { kind: 'group'; id: string } | null>(null);
  // 서버 여러 개 선택(D59). 2개 이상이면 "여러 개 선택 상태" — 메뉴는 삭제만, 끌면 모두 옮긴다.
  const [multi, setMulti] = useState<string[]>([]);
  // Shift+클릭 범위의 기준(마지막으로 그냥/Ctrl+클릭한 서버).
  const anchorRef = useRef<string | null>(null);
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
      const next = moveHosts(hosts, d.ids, targetId);
      if (next === hosts) return;
      void patchStore({ hosts: next });
      if (d.ids.length > 1) {
        const where = targetId ? tr('sb.folderNamed', { name: groups.find((x) => x.id === targetId)?.name ?? '' }) : tr('sb.topLevel');
        pushToast({ kind: 'info', text: tr('sb.moved', { count: d.ids.length, where }) });
      }
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
  // 지워진 서버는 선택에서 뺀다. 2개 이상일 때만 여러 개 선택 상태.
  const multiSel = multi.filter((id) => hosts.some((h) => h.id === id));
  const multiOn = multiSel.length >= 2;
  // 화면에 보이는 서버 목록(선택 가능한 행)의 순서. 그리는 동안 채워지고 Shift+클릭 범위에 쓴다.
  const order: string[] = [];

  // 폴더 펼침/접힘 토글은 저장한다(collapsed).
  const toggleFolder = (g: Group) => {
    void patchStore({ groups: groups.map((x) => (x.id === g.id ? { ...x, collapsed: !x.collapsed } : x)) });
  };

  const removeHost = async (h: Host) => {
    const ok = await confirm({ title: tr('sb.deleteServer'), body: tr('sb.deleteServerBody', { name: h.alias }), okText: tr('common.delete'), danger: true });
    if (!ok) return;
    await patchStore({ hosts: hosts.filter((x) => x.id !== h.id) });
    if (selected === h.id) setSelected(null);
  };

  /** 여러 서버 삭제(D59): 이름 목록(최대 10개)을 보여 주고 확인 후 지운다. */
  const removeHosts = async (ids: string[]) => {
    const names = hosts.filter((x) => ids.includes(x.id)).map((x) => x.alias || x.host);
    const ok = await confirm({
      title: tr('sb.deleteServer'),
      body: tr('sb.deleteManyBody', { count: names.length }),
      detail: names.slice(0, 10).join('\n') + (names.length > 10 ? '\n' + tr('sb.andMoreLine', { count: names.length - 10 }) : ''),
      okText: tr('common.delete'),
      danger: true,
    });
    if (!ok) return;
    await patchStore({ hosts: hosts.filter((x) => !ids.includes(x.id)) });
    setMulti([]);
    if (selected && ids.includes(selected)) setSelected(null);
    pushToast({ kind: 'info', text: tr('sb.deletedMany', { count: names.length }) });
  };

  /**
   * 서버 행 클릭의 선택 처리. Ctrl+클릭 = 넣고 빼기, Shift+클릭 = 범위, 그냥 클릭 = 하나만.
   * 여러 개 선택(Ctrl/Shift)이면 true — 호출자는 세션으로 옮겨 가지 않는다(포커스를 사이드바에 남김).
   */
  const selectByClick = (e: { ctrlKey: boolean; metaKey: boolean; shiftKey: boolean }, id: string): boolean => {
    if (e.ctrlKey || e.metaKey) {
      const base = multiSel.length ? multiSel : selected ? [selected] : [];
      const next = toggleSelection(base, id);
      setMulti(next);
      setSelected(next.includes(id) ? id : next[next.length - 1] ?? null);
      anchorRef.current = id;
      return true;
    }
    if (e.shiftKey) {
      const range = rangeSelection(order, anchorRef.current ?? selected, id);
      if (range) {
        setMulti(range);
        setSelected(id);
        return true;
      }
    }
    setMulti([]);
    setSelected(id);
    anchorRef.current = id;
    return false;
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

  // 폴더 삭제(D58): ① 폴더 삭제 확인(취소하면 아무것도 안 함) → ② 안에 서버가 있으면 서버도 지울지 묻는다.
  // ②에서 [아니오]는 지금까지처럼 서버를 상위 폴더로 옮기고, [취소]·X·Esc·바깥 클릭은 삭제 자체를 취소한다.
  const deleteFolder = async (g: Group) => {
    const count = countHostsIn(groups, hosts, g.id);
    const parentName = g.parentId ? groups.find((x) => x.id === g.parentId)?.name : undefined;
    const moveTo = parentName ? tr('sb.folderNamed', { name: parentName }) : tr('sb.topLevel');
    const ok = await confirm({
      title: tr('sb.deleteFolder'),
      body: count > 0 ? tr('sb.deleteFolderBody', { name: g.name }) : tr('sb.deleteFolderMove', { name: g.name }),
      okText: tr('common.delete'),
      danger: true,
    });
    if (!ok) return;
    let withHosts = false;
    if (count > 0) {
      const ans = await useUi.getState().ask({
        title: tr('sb.folderServers'),
        body: tr('sb.folderServersBody', { name: g.name, count }),
        detail: tr('sb.folderServersDetail', { where: moveTo }),
        okText: tr('sb.deleteWithServers'),
        noText: tr('sb.folderOnly'),
        danger: true,
      });
      if (ans === null) return;
      withHosts = ans === 'yes';
    }
    const plan = folderDeletePlan(groups, hosts, g.id, withHosts);
    await patchStore({ groups: plan.groups, hosts: plan.hosts });
    if (withHosts) {
      if (selected && !plan.hosts.some((x) => x.id === selected)) setSelected(null);
      pushToast({ kind: 'info', text: tr('sb.deletedFolder', { name: g.name, count }) });
    }
  };

  const hostMenu = (h: Host): MenuItem[] => [
    { label: tr('loc.newTab'), shortcut: 'Enter', onSelect: () => void openHost(h.id) },
    { label: tr('sb.openRight'), onSelect: () => void openHost(h.id, { kind: 'rightGroup' }) },
    { label: tr('sb.openBelow'), onSelect: () => void openHost(h.id, { kind: 'split', dir: 'col' }) },
    { label: tr('sb.chooseLocation'), onSelect: () => useUi.getState().openLocation(h.id) },
    { separator: true },
    { label: tr('common.edit'), shortcut: 'F2', onSelect: () => openEditor({ kind: 'host', hostId: h.id, groupId: null }) },
    { label: tr('sb.duplicate'), onSelect: () => void duplicateHost(h) },
    { label: h.favorite ? tr('sb.unfavorite') : tr('sb.favorite'), onSelect: () => void toggleFavorite(h) },
    { label: tr('sb.copyAddress'), onSelect: () => void copyAddress(h) },
    { separator: true },
    { label: tr('common.delete'), danger: true, onSelect: () => void removeHost(h) },
  ];

  const folderMenu = (g: Group): MenuItem[] => [
    { label: tr('sb.openFolderGrid'), onSelect: () => void openFolderGrid(g.id) },
    { separator: true },
    { label: tr('sb.addServerHere'), onSelect: () => openEditor({ kind: 'host', hostId: null, groupId: g.id }) },
    { label: tr('sb.newSubfolder'), onSelect: () => openEditor({ kind: 'folder', folderId: null, parentId: g.id }) },
    { label: tr('folder.edit'), onSelect: () => openEditor({ kind: 'folder', folderId: g.id, parentId: g.parentId }) },
    { separator: true },
    { label: tr('sb.deleteFolder'), danger: true, onSelect: () => void deleteFolder(g) },
  ];

  const onKey = (e: KeyboardEvent<HTMLDivElement>) => {
    const h = hosts.find((x) => x.id === selected);
    if (e.key === 'Escape' && searching) setQuery('');
    else if (e.key === 'Escape' && multiOn) setMulti([]);
    // 여러 개 선택 상태에서는 Delete(일괄 삭제)만 받는다.
    if (multiOn && !(e.target instanceof HTMLInputElement)) {
      if (e.key === 'Delete') void removeHosts(multiSel);
      return;
    }
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
    const inMulti = !!opts.selectable && multiOn && multiSel.includes(h.id);
    const isSel = opts.selectable && (multiOn ? inMulti : selected === h.id);
    if (opts.selectable) order.push(h.id);
    return (
      <div
        key={`${opts.selectable ? 'h' : 'fav'}-${h.id}`}
        className="side-row"
        role="treeitem"
        data-host-id={h.id}
        aria-selected={isSel}
        draggable={opts.selectable}
        onDragStart={(e: DragEvent<HTMLDivElement>) => {
          // 선택된 서버 중 하나를 끌면 선택한 서버 모두를 옮긴다. 선택 밖의 서버를 끌면 그 하나만.
          const ids = inMulti ? multiSel : [h.id];
          dragRef.current = { kind: 'host', ids };
          e.dataTransfer.effectAllowed = 'move';
          e.dataTransfer.setData('text/plain', ids.join(','));
          if (ids.length === 1) {
            // 칸으로 끌어 놓아 접속하기(panes §7)는 서버 하나일 때만.
            e.dataTransfer.setData(HOST_MIME, h.id);
          } else {
            setDragBadge(e, tr('sb.dragServers', { count: ids.length }));
          }
        }}
        onDragEnd={() => {
          dragRef.current = null;
          setDropTarget(null);
        }}
        title={`${h.alias}\n${fullTarget(h)}${h.notes ? `\n${h.notes}` : ''}`}
        onClick={(e) => {
          if (opts.selectable && selectByClick(e, h.id)) return;
          if (!opts.selectable) setMulti([]);
          focusOpenSessionOf(h.id);
        }}
        // Shift+클릭 때 글자가 끌려 선택되지 않게 한다.
        onMouseDown={(e) => {
          if (e.shiftKey) e.preventDefault();
        }}
        onDoubleClick={() => openSmart(h.id)}
        onContextMenu={(e) => {
          e.preventDefault();
          // 여러 개 선택한 서버 위에서 우클릭하면 메뉴는 삭제 하나뿐(D59).
          if (inMulti) {
            openMenu(e.clientX, e.clientY, [
              { label: tr('sb.deleteCount', { count: multiSel.length }), danger: true, onSelect: () => void removeHosts(multiSel) },
            ]);
            return;
          }
          setMulti([]);
          setSelected(h.id);
          openMenu(e.clientX, e.clientY, hostMenu(h));
        }}
        onMouseEnter={() => setHoverId(h.id)}
        onMouseLeave={() => setHoverId((cur) => (cur === h.id ? null : cur))}
        style={{ paddingLeft: ROW_PAD + depth * INDENT, background: isSel ? 'var(--accent-soft)' : undefined }}
      >
        <span className="strip" style={{ background: color ?? 'transparent' }} />
        <Icon size={13} className="side-icon" color={protocolIconColor(h)} aria-hidden="true" />
        <span className="side-name">{h.alias || h.host}</span>
        <span className="faint small side-sub">{sidebarSubtitle(h)}</span>
        {openByHost[h.id] ? (
          <span className="small" style={{ color: 'var(--ok, #22c55e)', marginLeft: 'auto' }}>
            ● {openByHost[h.id] > 1 ? openByHost[h.id] : ''}
          </span>
        ) : null}
        {hoverId === h.id && opts.selectable && (
          <span style={{ display: 'inline-flex', gap: 2 }}>
            <button className="btn small-btn" title={tr('sb.openBeside')} onClick={(e) => { e.stopPropagation(); void openHost(h.id, { kind: 'rightGroup' }); }}>{tr('sb.openBeside')}</button>
            <button className="btn small-btn" title={tr('sb.openNew')} onClick={(e) => { e.stopPropagation(); void openHost(h.id); }}>{tr('sb.openNew')}</button>
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
            paddingLeft: ROW_PAD + depth * INDENT,
            outline: dropTarget === g.id ? '1px dashed var(--accent)' : undefined,
            background: dropTarget === g.id ? 'var(--accent-soft)' : undefined,
          }}
          onClick={() => !searching && toggleFolder(g)}
          onContextMenu={(e) => {
            e.preventDefault();
            openMenu(e.clientX, e.clientY, folderMenu(g));
          }}
        >
          {/* 펼침 화살표 · 폴더 아이콘(펼치면 열린 폴더). 폴더 색(없으면 환경 색)은 폴더 아이콘에 칠한다 (D71). */}
          {expanded ? <ChevronDown size={13} className="side-icon" aria-hidden="true" /> : <ChevronRight size={13} className="side-icon" aria-hidden="true" />}
          {expanded ? (
            <FolderOpen size={14} className="side-icon" color={color ?? (g.env && g.env !== 'none' ? ENV_COLOR[g.env] : FOLDER_ICON_COLOR)} aria-hidden="true" />
          ) : (
            <Folder size={14} className="side-icon" color={color ?? (g.env && g.env !== 'none' ? ENV_COLOR[g.env] : FOLDER_ICON_COLOR)} aria-hidden="true" />
          )}
          <span className="side-name" style={{ fontWeight: 700 }}>{g.name}</span>
          {g.env && g.env !== 'none' && <EnvPill env={g.env} />}
          <span className="faint small" style={{ marginLeft: 'auto' }}>{count}</span>
        </div>
        {expanded && (
          <TreeChildren depth={depth}>
            {[...childGroups.map((c) => folderRow(c, depth + 1)), ...childHosts.map((h) => hostRow(h, depth + 1))]}
          </TreeChildren>
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
    <aside className="sidebar" style={{ width }} aria-label={tr('shell.servers')} onKeyDown={onKey} tabIndex={-1}>
      <div className="side-head">
        <div className="search-box">
          <Search size={14} />
          <input
            className="search-input"
            placeholder={tr('sb.search')}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            aria-label={tr('sb.search')}
          />
        </div>
        <div className="side-btns">
          <button className="btn small-btn" onClick={() => openEditor({ kind: 'host', hostId: null, groupId: null })}><Plus size={13} />{tr('sb.addServer')}</button>
          <button className="btn small-btn" onClick={() => openEditor({ kind: 'folder', folderId: null, parentId: null })}><FolderPlus size={13} />{tr('sb.addFolder')}</button>
          <button className="btn small-btn" title={tr('quick.import')} onClick={() => useUi.getState().openImportDialog()}><Download size={13} />{tr('sb.import')}</button>
        </div>
      </div>

      {multiOn && (
        <div className="side-multi" role="status">
          <span>{tr('sb.selected', { count: multiSel.length })}</span>
          <span className="faint small">{tr('sb.selectedHint')}</span>
          <button className="link-btn" onClick={() => setMulti([])}>{tr('sb.clearSelection')}</button>
        </div>
      )}

      <div
        className="side-tree"
        role="tree"
        aria-multiselectable
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
            <div className="side-section"><Star size={12} /> {tr('sb.favorites')}</div>
            {favorites.map((h) => hostRow(h, 0, { selectable: false }))}
          </>
        )}
        {showRecent && (
          <>
            <div className="side-section"><Clock size={12} /> {tr('sb.recent')}</div>
            {recent.map((h) => hostRow(h, 0, { selectable: false }))}
          </>
        )}
        <div className="side-section">{tr('sb.servers')}{hosts.length > 0 ? ` · ${hosts.length}` : ''}</div>
        {rootGroups.map((g) => folderRow(g, 0))}
        {looseHosts.map((h) => hostRow(h, 0))}
        {hosts.length === 0 && (
          <div className="faint small" style={{ padding: '8px 10px', lineHeight: 1.6, whiteSpace: 'pre-line' }}>
            {tr('sb.empty')}
          </div>
        )}
        {noMatch && (
          <div className="faint small" style={{ padding: '8px 10px' }}>{tr('sb.noMatch', { query: query.trim() })}</div>
        )}
      </div>

      <div className="side-resize" onPointerDown={onResizeDown} role="separator" aria-orientation="vertical" aria-label={tr('sb.width')} />
    </aside>
  );
}
