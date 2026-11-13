// 제목 표시줄의 탭 막대. 근거: spec/ui/shell.md §2.1~§2.2, spec/ui/panes.md §6, §8.
import { useState, type DragEvent, type MouseEvent } from 'react';
import { FolderTree, Radio, X } from 'lucide-react';
import { useWorkspace, tabTitle, detachPanes, groupOf, isSplit, type Tab } from '../state/workspace';
import { useUi, type MenuItem } from '../state/ui';
import { paneIds, findPane } from '../lib/layout';
import { isFileDrag, hoverStart, hoverEnd, dropFiles, tabSessionId } from './fileDrag';
import { focusActiveTerminal } from './TerminalView';

export const TAB_MIME = 'application/x-termgo-tab';

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
  const gridTabs = useWorkspace((s) => s.gridTabs);
  const openMenu = useUi((s) => s.openMenu);
  const prompt = useUi((s) => s.prompt);
  // 파일 패널 항목을 끌고 탭 위에 있을 때: 0.5초 머물면 그 탭으로 전환, 놓으면 그 탭 세션의 마지막 폴더로 복사 (file-panel.md §2.2).
  const [fileOver, setFileOver] = useState<string | null>(null);
  const confirm = useUi((s) => s.confirm);
  const openQuick = useUi((s) => s.openQuick);
  const [dragOver, setDragOver] = useState<string | null>(null);

  const rename = async (t: Tab) => {
    const cur = tabTitle(t, sessions);
    const v = await prompt({ title: '탭 이름 변경', value: t.title ?? cur, okText: '확인' });
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
        title: '탭 닫기',
        body: `이 탭의 연결 ${count}개를 모두 닫을까요?`,
        okText: '닫기',
        danger: true,
      });
      if (!ok) return;
    }
    await closeTab(t.id, true);
  };

  const menuFor = (t: Tab): MenuItem[] => {
    const panes = paneIds(t.root).length;
    return [
      { label: '탭 이름 변경', onSelect: () => void rename(t) },
      groupOf(t) === 0
        ? { label: '오른쪽 화면으로 보내기', disabled: allTabs.length < 2, onSelect: () => moveTabToGroup(t.id, 1) }
        : { label: '왼쪽 화면으로 보내기', onSelect: () => moveTabToGroup(t.id, 0) },
      ...(split ? [{ label: '한 화면으로 합치기', onSelect: () => mergeGroups() } as MenuItem] : []),
      { label: t.broadcast ? '이 탭의 모든 창에 일괄 입력 끄기' : '이 탭의 모든 창에 일괄 입력', shortcut: 'Ctrl+Shift+B', disabled: panes < 2, onSelect: () => toggleBroadcast(t.id) },
      { label: '창마다 탭으로 나누기', disabled: panes < 2, onSelect: () => detachPanes(t.id) },
      { label: '모든 탭을 한 화면에 바둑판으로', onSelect: () => gridTabs() },
      { separator: true },
      { label: '나머지 탭 닫기', onSelect: () => void closeOthers(t) },
      { label: '탭 닫기', shortcut: 'Ctrl+Shift+W', danger: true, onSelect: () => void confirmClose(t) },
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
        const env = focusedSid ? sessions[focusedSid]?.info.color : undefined;
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
            style={env ? { background: `color-mix(in srgb, ${env} ${active ? 14 : 7}%, transparent)` } : undefined}
          >
            {env && <span className="bar" style={{ background: env }} />}
            <span className={`dot ${dot}`} />
            <span className="title">{title}</span>
            {t.broadcast && <Radio size={12} color="var(--warn)" aria-label="일괄 입력" />}
            {showBadge(t) && <span className="tab-badge">{paneIds(t.root).length}</span>}
            <button
              className="x"
              aria-label="탭 닫기"
              title="닫기 (Ctrl+Shift+W)"
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
      <button className="icon-btn" title="새로 연결 (Ctrl+K)" aria-label="새로 연결" onClick={() => openQuick(null)} style={{ marginLeft: 4, marginTop: 5 }}>
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
