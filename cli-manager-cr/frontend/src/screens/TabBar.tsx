// 제목 표시줄의 탭 막대. 근거: spec/ui/shell.md §2.1~§2.2, spec/ui/panes.md §6, §8.
import { useState, type DragEvent, type MouseEvent } from 'react';
import { FolderTree, Radio, X } from 'lucide-react';
import { useWorkspace, tabTitle, detachPanes, groupOf, isSplit, type Tab } from '../state/workspace';
import { useUi, type MenuItem } from '../state/ui';
import { paneIds, findPane } from '../lib/layout';
import { isFileDrag, hoverStart, hoverEnd, dropFiles, tabSessionId } from './fileDrag';
import { focusActiveTerminal } from './TerminalView';
// 이 파일은 t 를 탭 변수 이름으로 많이 쓰므로 번역 함수는 tr 로 가져온다(D75).
import { t as tr } from '../i18n';

export const TAB_MIME = 'application/x-dariterm-tab';

/** 탭의 상태 점: 칸 세션 상태를 합친다(shell.md §2.1). */
function tabDot(t: Tab, sessions: Record<string, { info: { state: string }; activity: boolean }>): string {
  const ids = paneIds(t.root)
    .map((pid) => findPane(t.root, pid)?.sessionId)
    .filter((x): x is string => !!x && !!sessions[x]);
  if (ids.length === 0) return 'closed';
  if (ids.some((id) => sessions[id].activity)) return 'activity';
  const states = ids.map((id) => sessions[id].info.state);
  const connected = states.filter((s) => s === 'connected').length;
  if (connected === ids.length) return 'connected';
  if (connected > 0) return 'reconnecting';
  if (states.some((s) => s === 'connecting' || s === 'reconnecting')) return 'connecting';
  if (states.includes('error')) return 'error';
  return 'closed';
}

/** 선택되지 않은 탭의 배경에 섞는 세션 색 비율(%)과 선택된 탭의 비율 (shell.md §2.1). */
const TINT_INACTIVE = 7;
const TINT_ACTIVE = 14;

/**
 * 탭의 색 표시(shell.md §2.1, D103).
 * 색 막대(bar)는 그룹에서 선택된 탭에만 둔다: 세션 색, 없으면 강조색.
 * 선택되지 않은 탭은 세션 색이 있을 때 옅은 배경(tint)만 가진다.
 */
export function tabMark(active: boolean, color: string | undefined): { bar?: string; tint?: string } {
  const c = color || (active ? 'var(--accent)' : undefined);
  if (!c) return {};
  return {
    bar: active ? c : undefined,
    tint: `color-mix(in srgb, ${c} ${active ? TINT_ACTIVE : TINT_INACTIVE}%, transparent)`,
  };
}

/** 탭 막대. group 을 주면 그 그룹의 탭만 보이는 열 안 탭 막대(화면 나누기, ui/shell.md §2.3). */
export function TabBar({ group }: { group?: 0 | 1 }) {
  const allTabs = useWorkspace((s) => s.tabs);
  const g: 0 | 1 = group ?? 0;
  const split = isSplit(allTabs);
  const tabs = allTabs.filter((t) => groupOf(t) === g);
  const activeId = useWorkspace((s) => s.activeByGroup[g]);
  const moveTabToGroup = useWorkspace((s) => s.moveTabToGroup);
  const sftpOpen = useUi((s) => s.sftpOpen[g]);
  const toggleSftp = useUi((s) => s.toggleSftp);
  const mergeGroups = useWorkspace((s) => s.mergeGroups);
  const sessions = useWorkspace((s) => s.sessions);
  const activate = useWorkspace((s) => s.activateTab);
  const closeTab = useWorkspace((s) => s.closeTab);
  const renameTab = useWorkspace((s) => s.renameTab);
  const moveTab = useWorkspace((s) => s.moveTab);
  const toggleBroadcast = useWorkspace((s) => s.toggleBroadcast);
  const openMenu = useUi((s) => s.openMenu);
  const prompt = useUi((s) => s.prompt);
  // 파일 패널 항목을 끌고 탭 위에 있을 때: 0.5초 머물면 그 탭으로 전환, 놓으면 그 탭 세션의 마지막 폴더로 복사 (file-panel.md §2.2).
  const [fileOver, setFileOver] = useState<string | null>(null);
  const confirm = useUi((s) => s.confirm);
  const openQuick = useUi((s) => s.openQuick);
  const [dragOver, setDragOver] = useState<string | null>(null);

  const rename = async (t: Tab) => {
    const cur = tabTitle(t, sessions);
    const v = await prompt({ title: tr('tab.rename'), value: t.title ?? cur, okText: tr('common.ok') });
    if (v === null) return;
    renameTab(t.id, v);
  };

  const closeOthers = async (t: Tab) => {
    const others = useWorkspace.getState().tabs.filter((x) => x.id !== t.id);
    for (const o of others) await closeTab(o.id, true);
    useWorkspace.getState().activateTab(t.id);
  };

  const confirmClose = async (t: Tab) => {
    const count = paneIds(t.root).filter((pid) => findPane(t.root, pid)?.sessionId).length;
    if (count >= 2) {
      const ok = await confirm({
        title: tr('tab.close'),
        body: tr('tab.closeBody', { count }),
        okText: tr('common.close'),
        danger: true,
      });
      if (!ok) return;
    }
    await closeTab(t.id, true);
  };

  const menuFor = (t: Tab): MenuItem[] => {
    const panes = paneIds(t.root).length;
    return [
      { label: tr('tab.rename'), onSelect: () => void rename(t) },
      groupOf(t) === 0
        ? { label: tr('tab.toRight'), disabled: allTabs.length < 2, onSelect: () => moveTabToGroup(t.id, 1) }
        : { label: tr('tab.toLeft'), onSelect: () => moveTabToGroup(t.id, 0) },
      ...(split ? [{ label: tr('tab.merge'), onSelect: () => mergeGroups() } as MenuItem] : []),
      { label: t.broadcast ? tr('tab.broadcastOff') : tr('tab.broadcastOn'), shortcut: 'Ctrl+Shift+B', disabled: panes < 2, onSelect: () => toggleBroadcast(t.id) },
      { label: tr('tab.detach'), disabled: panes < 2, onSelect: () => detachPanes(t.id) },
      { separator: true },
      { label: tr('tab.closeOthers'), onSelect: () => void closeOthers(t) },
      { label: tr('tab.close'), shortcut: 'Ctrl+Shift+W', danger: true, onSelect: () => void confirmClose(t) },
    ];
  };

  const onMiddle = (e: MouseEvent, t: Tab) => {
    if (e.button === 1) {
      e.preventDefault();
      void confirmClose(t);
    }
  };

  const onDragStart = (e: DragEvent, id: string) => {
    e.dataTransfer.setData(TAB_MIME, id);
    e.dataTransfer.effectAllowed = 'move';
  };
  const onDrop = (e: DragEvent, toIndex: number) => {
    const id = e.dataTransfer.getData(TAB_MIME);
    setDragOver(null);
    if (!id) return;
    e.preventDefault();
    e.stopPropagation();
    // 다른 그룹의 탭 위에 놓으면 그 그룹으로 옮겨 그 자리에 끼운다.
    moveTab(id, toIndex, split ? g : undefined);
  };
  // 탭 막대의 빈 곳에 놓기: 이 그룹의 맨 뒤로.
  const onDropEnd = (e: DragEvent) => {
    const id = e.dataTransfer.getData(TAB_MIME);
    setDragOver(null);
    if (!id) return;
    e.preventDefault();
    const last = tabs[tabs.length - 1];
    const idx = last ? useWorkspace.getState().tabs.findIndex((t) => t.id === last.id) + 1 : useWorkspace.getState().tabs.length;
    moveTab(id, idx, split ? g : undefined);
  };

  const showBadge = (t: Tab) => paneIds(t.root).length > 1;

  return (
    <div
      className={`tabbar${group !== undefined ? ' in-group' : ''}`}
      role="tablist"
      onDragOver={(e) => {
        if (group !== undefined && e.dataTransfer.types.includes(TAB_MIME)) e.preventDefault();
      }}
      onDrop={group !== undefined ? onDropEnd : undefined}
    >
      {tabs.map((t) => {
        const i = allTabs.indexOf(t);
        const active = t.id === activeId;
        const focusedSid = t.focusedPane ? findPane(t.root, t.focusedPane)?.sessionId : null;
        const mark = tabMark(active, focusedSid ? sessions[focusedSid]?.info.color : undefined);
        const title = tabTitle(t, sessions);
        const dot = tabDot(t, sessions);
        return (
          <div
            key={t.id}
            role="tab"
            aria-selected={active}
            title={title}
            className={`tab${active ? ' active' : ''}${dragOver === t.id ? ' drop' : ''}${fileOver === t.id ? ' file-over' : ''}`}
            draggable
            onDragStart={(e) => onDragStart(e, t.id)}
            onDragOver={(e) => {
              if (isFileDrag(e)) {
                e.preventDefault();
                e.stopPropagation();
                e.dataTransfer.dropEffect = 'copy';
                if (fileOver !== t.id) setFileOver(t.id);
                if (!active) hoverStart(`tab:${t.id}`, () => activate(t.id));
                return;
              }
              if (e.dataTransfer.types.includes(TAB_MIME)) {
                e.preventDefault();
                setDragOver(t.id);
              }
            }}
            onDragLeave={(e) => {
              setDragOver(null);
              if (!e.currentTarget.contains(e.relatedTarget as Node | null)) {
                setFileOver(null);
                hoverEnd(`tab:${t.id}`);
              }
            }}
            onDrop={(e) => {
              if (isFileDrag(e)) {
                e.preventDefault();
                e.stopPropagation();
                setFileOver(null);
                void dropFiles(tabSessionId(t.id), '');
                return;
              }
              onDrop(e, i);
            }}
            onMouseDown={(e) => {
              if (e.button === 0) activate(t.id);
              onMiddle(e, t);
            }}
            // 탭을 누르면 그 탭의 터미널로 바로 입력할 수 있게 포커스를 옮긴다.
            onClick={() => focusActiveTerminal()}
            onDoubleClick={() => void rename(t)}
            onContextMenu={(e) => {
              e.preventDefault();
              activate(t.id);
              openMenu(e.clientX, e.clientY, menuFor(t));
            }}
            style={mark.tint ? { background: mark.tint } : undefined}
          >
            {mark.bar && <span className="bar" style={{ background: mark.bar }} />}
            <span className={`dot ${dot}`} />
            <span className="title">{title}</span>
            {t.broadcast && <Radio size={12} color="var(--warn)" aria-label={tr('tab.broadcastAria')} />}
            {showBadge(t) && <span className="tab-badge">{paneIds(t.root).length}</span>}
            <button
              className="x"
              aria-label={tr('tab.close')}
              title={tr('tab.closeTitle')}
              onMouseDown={(e) => e.stopPropagation()}
              onClick={(e) => {
                e.stopPropagation();
                void confirmClose(t);
              }}
            >
              <X size={12} />
            </button>
          </div>
        );
      })}
      {/* + 는 이 탭 막대가 속한 그룹에 연다(포커스가 다른 그룹에 있어도, D119). */}
      <button className="icon-btn" title={tr('tab.newTitle')} aria-label={tr('tab.new')} onClick={() => openQuick(null, g)} style={{ marginLeft: 4, marginTop: 5 }}>
        <span style={{ fontSize: 16, lineHeight: 1 }}>+</span>
      </button>
      <button
        className={`icon-btn tab-sftp${sftpOpen ? ' on' : ''}`}
        title="SFTP (Ctrl+Shift+S)"
        aria-label="SFTP"
        aria-pressed={sftpOpen}
        onClick={() => toggleSftp(g)}
      >
        <FolderTree size={16} />
      </button>
    </div>
  );
}
