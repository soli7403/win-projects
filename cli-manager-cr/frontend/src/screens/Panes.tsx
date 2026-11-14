// 칸·분할 화면. 근거: spec/ui/panes.md §2, §4(크기 조절), §7(끌어 놓기), spec/ui/tokens.md §3(칸·분할선).
import { useRef, type DragEvent, type PointerEvent as RPointerEvent, type ReactNode } from 'react';
import { Link, Rows2, Maximize2, Minimize2, RotateCw, X, Search, ZoomIn, Image as ImageIcon } from 'lucide-react';
import type { LayoutNode, PaneNode, SplitNode, Dir } from '../lib/layout';
import { edgeToSplit, edgeZone, findPane, paneIds, type Edge } from '../lib/layout';
import { groupOf, useWorkspace, type Tab } from "../state/workspace";
import { openHost, splitSameAs, canSplitMore } from '../state/actions';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { ENV_INFO } from '../lib/defaults';
import { TerminalView, termHandles, pasteFromClipboard, targetsFor, FIND_EVENT } from './TerminalView';
import { forceImagePaste, uploadAndInsert } from '../state/paste';
import { notYet } from '../lib/notYet';
import { getBackend, errorMessage } from '../ipc/backend';
import type { MenuItem } from '../state/ui';
import { isFileDrag, hoverStart, hoverEnd, dropFiles } from './fileDrag';
// 이 파일은 t 를 지역 변수 이름으로 쓰므로 번역 함수는 tr 로 가져온다(D75).
import { relabel, t as tr } from '../i18n';

const PANE_MIME = 'application/x-dariterm-pane';
/** 사이드바 서버 끌어 놓기 MIME(에이전트 B 가 dragstart 에서 설정). */
export const HOST_MIME = 'application/x-dariterm-host';

export function PaneTree({ tab }: { tab: Tab }) {
  const panes = paneIds(tab.root).length;
  if (tab.zoomed && findPane(tab.root, tab.zoomed)) {
    const z = findPane(tab.root, tab.zoomed)!;
    return <div className="pane-root"><PaneView tab={tab} pane={z} total={panes} /></div>;
  }
  return <div className="pane-root"><Node tab={tab} node={tab.root} total={panes} /></div>;
}

function Node({ tab, node, total }: { tab: Tab; node: LayoutNode; total: number }) {
  if (node.type === 'pane') return <PaneView tab={tab} pane={node} total={total} />;
  return <SplitView tab={tab} split={node} total={total} />;
}

function SplitView({ tab, split, total }: { tab: Tab; split: SplitNode; total: number }) {
  const ref = useRef<HTMLDivElement>(null);
  const resize = useWorkspace((s) => s.resizeSplit);
  const horizontal = split.dir === 'row';

  const onDividerDown = (boundary: number, e: RPointerEvent<HTMLDivElement>) => {
    e.preventDefault();
    const el = ref.current;
    if (!el) return;
    const size = horizontal ? el.clientWidth : el.clientHeight;
    let last = horizontal ? e.clientX : e.clientY;
    const move = (ev: PointerEvent) => {
      const cur = horizontal ? ev.clientX : ev.clientY;
      const deltaPct = ((cur - last) / size) * 100;
      if (Math.abs(deltaPct) < 0.0001) return;
      last = cur;
      resize(tab.id, split.id, boundary, deltaPct);
    };
    const up = () => {
      window.removeEventListener('pointermove', move);
      window.removeEventListener('pointerup', up);
      document.body.style.cursor = '';
    };
    document.body.style.cursor = horizontal ? 'col-resize' : 'row-resize';
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', up);
  };

  const items: ReactNode[] = [];
  split.children.forEach((child, i) => {
    items.push(
      <div key={`c-${child.type === 'pane' ? child.id : child.id}`} className="pane-cell" style={{ flex: `0 0 calc(${split.sizes[i]}% - 2.5px)`, minWidth: 0, minHeight: 0, display: 'flex' }}>
        <Node tab={tab} node={child} total={total} />
      </div>,
    );
    if (i < split.children.length - 1) {
      items.push(
        <div
          key={`d-${i}`}
          className={`pane-divider ${horizontal ? 'h' : 'v'}`}
          onPointerDown={(e) => onDividerDown(i, e)}
          role="separator"
          aria-orientation={horizontal ? 'vertical' : 'horizontal'}
        />,
      );
    }
  });
  return (
    <div ref={ref} style={{ display: 'flex', flexDirection: horizontal ? 'row' : 'column', flex: 1, minWidth: 0, minHeight: 0 }}>
      {items}
    </div>
  );
}

function PaneView({ tab, pane, total }: { tab: Tab; pane: PaneNode; total: number }) {
  const entry = useWorkspace((s) => (pane.sessionId ? s.sessions[pane.sessionId] : undefined));
  const focused = tab.focusedPane === pane.id;
  const info = entry?.info;
  const env = info?.env && info.env !== 'none' ? info.env : undefined;
  const color = info?.color;
  const isProd = env === 'prod';
  const dropRef = useRef<HTMLDivElement>(null);
  const focusPane = useWorkspace((s) => s.focusPane);
  const closePane = useWorkspace((s) => s.closePane);
  const toggleZoom = useWorkspace((s) => s.toggleZoom);
  const movePane = useWorkspace((s) => s.movePane);
  const swapPanes = useWorkspace((s) => s.swapPanes);
  const splitEmpty = useWorkspace((s) => s.splitEmpty);
  const openQuick = useUi((s) => s.openQuick);
  // 모든 창 동시 입력 중이면 칸 머리에 쇠사슬 표시 (보이는 창은 모두 대상)
  const linkInput = useUi((s) => s.linkInput);
  // 상하 분할 한도 (설정 maxVerticalSplits, 1~5)
  const maxSplits = useApp((s) => {
    const v = Number(s.store?.settings.maxVerticalSplits);
    return Number.isFinite(v) && v >= 1 ? Math.min(5, Math.floor(v)) : 3;
  });
  const canSplit = canSplitMore(tab.root, maxSplits);

  const splitSame = async (dir: Dir) => {
    if (info) await splitSameAs(info, dir, pane.id);
  };

  const onDragStart = (e: DragEvent<HTMLDivElement>) => {
    e.dataTransfer.setData(PANE_MIME, pane.id);
    e.dataTransfer.effectAllowed = 'move';
  };

  const dropZone = (e: DragEvent<HTMLDivElement>): Edge => {
    const r = e.currentTarget.getBoundingClientRect();
    return edgeZone((e.clientX - r.left) / r.width, (e.clientY - r.top) / r.height);
  };
  const setDrop = (z: Edge | null) => {
    if (dropRef.current) dropRef.current.dataset.zone = z ?? '';
  };
  const onDragOver = (e: DragEvent<HTMLDivElement>) => {
    // 파일 패널 항목: 0.5초 머물면 이 칸에 포커스(패널이 이 세션으로 바뀜), 놓으면 이 세션의 마지막 폴더로 복사 (file-panel.md §2.2).
    if (isFileDrag(e)) {
      if (!pane.sessionId) return;
      e.preventDefault();
      e.dataTransfer.dropEffect = 'copy';
      if (!focused) hoverStart(`pane:${pane.id}`, () => focusPane(pane.id));
      return;
    }
    const t = e.dataTransfer.types;
    if (!t.includes(PANE_MIME) && !t.includes(HOST_MIME) && !t.includes('Files')) return;
    e.preventDefault();
    setDrop(dropZone(e));
  };
  const onDrop = (e: DragEvent<HTMLDivElement>) => {
    setDrop(null);
    if (isFileDrag(e)) {
      if (!pane.sessionId) return;
      e.preventDefault();
      void dropFiles(pane.sessionId, '');
      return;
    }
    const zone = dropZone(e);
    const split = edgeToSplit(zone);
    // 서버를 칸으로 끌어 놓기(panes §7): 빈 칸 가운데 = 그 칸에서 열기, 가장자리 = 분할해 열기.
    const hostId = e.dataTransfer.getData(HOST_MIME);
    if (hostId) {
      e.preventDefault();
      if (!split && !pane.sessionId) void openHost(hostId, { kind: 'split', dir: 'row', paneId: pane.id });
      else if (split) void openHost(hostId, { kind: 'split', dir: split.dir, before: split.before, paneId: pane.id });
      return;
    }
    // 파일 끌어 놓기(panes §2.5): 경로를 얻을 수 있을 때만 동작한다.
    if (e.dataTransfer.files && e.dataTransfer.files.length && pane.sessionId) {
      e.preventDefault();
      const paths = Array.from(e.dataTransfer.files).map((f) => (f as File & { path?: string }).path).filter((x): x is string => !!x);
      if (paths.length) void uploadAndInsert(pane.sessionId, targetsFor(pane.id, pane.sessionId), { kind: 'files', paths });
      return;
    }
    const from = e.dataTransfer.getData(PANE_MIME);
    if (!from || from === pane.id) return;
    // 같은 탭 안에서만 칸을 옮긴다.
    if (!findPane(tab.root, from)) return;
    e.preventDefault();
    if (!split) swapPanes(tab.id, from, pane.id);
    else movePane(tab.id, from, pane.id, split.dir, split.before);
  };

  const hasSession = !!info;
  const reconnect = async () => {
    if (!info) return;
    try {
      await getBackend().invoke('session.reconnect', [info.id]);
    } catch (e) {
      useApp.getState().pushToast({ kind: 'error', text: errorMessage(e) });
    }
  };
  const menu: MenuItem[] = [
    { label: tr('common.copy'), shortcut: 'Ctrl+Shift+C', disabled: !hasSession, onSelect: () => { const sel = termHandles.get(pane.id)?.getSelection(); if (sel) void getBackend().invoke('clip.writeText', [sel]).catch(() => undefined); } },
    { label: tr('paste.ok'), shortcut: 'Ctrl+V', disabled: !hasSession, onSelect: () => { const t = termHandles.get(pane.id); if (t && pane.sessionId) void pasteFromClipboard(pane.id, pane.sessionId, t); } },
    { label: info?.canSftp || info?.protocol === 'local' ? tr('pane.pasteImagePath') : tr('pane.pasteImageServer'), shortcut: 'Ctrl+Alt+V', disabled: !hasSession || (!info?.canSftp && info?.protocol !== 'local'), onSelect: () => { if (pane.sessionId) void forceImagePaste(pane.sessionId, targetsFor(pane.id, pane.sessionId)); } },
    { separator: true },
    { label: tr('pane.selectAll'), disabled: !hasSession, onSelect: () => termHandles.get(pane.id)?.selectAll() },
    { label: tr('pane.clear'), disabled: !hasSession, onSelect: () => termHandles.get(pane.id)?.clear() },
    { label: tr('pane.find'), shortcut: 'Ctrl+Shift+F', disabled: !hasSession, onSelect: () => window.dispatchEvent(new CustomEvent(FIND_EVENT, { detail: pane.id })) },
    { separator: true },
    { label: tr('sc.splitRight'), shortcut: 'Ctrl+Shift+D', disabled: !info || !canSplit, onSelect: () => void splitSame('row') },
    { label: tr('sc.splitDown'), shortcut: 'Ctrl+Shift+E', disabled: !info || !canSplit, onSelect: () => void splitSame('col') },
    { label: info?.protocol === 'local' ? tr('pane.openFiles') : tr('pane.openSftp'), shortcut: 'Ctrl+Shift+S', disabled: !info?.canSftp && info?.protocol !== 'local', onSelect: () => { focusPane(pane.id); useUi.getState().setSftpOpen(groupOf(tab), true); } },
    { label: tr('pane.reconnect'), disabled: !hasSession || entry?.info.state === 'connected', onSelect: () => void reconnect() },
    { separator: true },
    { label: tr('pane.close'), danger: true, onSelect: () => void closePane(pane.id) },
  ];
  return (
    <div
      ref={dropRef}
      className={`pane${focused ? ' focused' : ''}${isProd ? ' prod' : ''}`}
      style={{ borderColor: isProd ? 'var(--danger)' : focused ? (color ?? 'var(--accent)') : undefined, flex: 1, minWidth: 0, minHeight: 0 }}
      onMouseDown={() => focusPane(pane.id)}
      onContextMenu={(e) => {
        e.preventDefault();
        focusPane(pane.id);
        useUi.getState().openMenu(e.clientX, e.clientY, menu);
      }}
      onDragOver={onDragOver}
      onDragLeave={(e) => {
        setDrop(null);
        if (!e.currentTarget.contains(e.relatedTarget as Node | null)) hoverEnd(`pane:${pane.id}`);
      }}
      onDrop={onDrop}
    >
      <div
        className="pane-head"
        draggable={total >= 2}
        onDragStart={onDragStart}
        style={{ background: color ? `color-mix(in srgb, ${color} ${focused ? 22 : 12}%, var(--panel-2))` : undefined }}
      >
        <span className={`dot ${entry ? entry.info.state : 'closed'}`} />
        {env && <span className="env-tag" style={{ background: ENV_INFO[env].color }}>{tr(ENV_INFO[env].label)}</span>}
        <span className="pane-title">{relabel(info?.title) ?? tr('pane.empty')}</span>
        {linkInput && hasSession && <Link size={12} color="var(--warn)" aria-label={tr('pane.broadcasting')} className="pane-link" />}
        {info && <span className="faint small pane-target">{info.target}</span>}
        <span className="spacer" />
        {hasSession && entry?.info.state !== 'connected' && (
          <button className="icon-btn small" title={tr('pane.reconnect')} aria-label={tr('pane.reconnect')} onClick={() => void reconnect()}><RotateCw size={12} /></button>
        )}
        {hasSession && (info?.protocol === 'local' || info?.canSftp) && (
          <button className="icon-btn small" title={info?.protocol === 'local' ? tr('pane.imageLocalTitle') : tr('pane.imageServerTitle')} aria-label={tr('pane.pasteImage')} onClick={() => { if (pane.sessionId) void forceImagePaste(pane.sessionId, targetsFor(pane.id, pane.sessionId)); }}><ImageIcon size={12} /></button>
        )}
        {hasSession && (
          <button className="icon-btn small" title={tr('pane.findTitle')} aria-label={tr('pane.find')} onClick={() => notYet(tr('pane.find'))}><Search size={12} /></button>
        )}
        {hasSession && (
          <>
            {/* 좌우 분할 아이콘은 두지 않는다. 상하 분할은 탭의 칸 수가 한도(설정 maxVerticalSplits)에 이르면 비활성 (ui/panes.md §2). */}
            <button className="icon-btn small" title={canSplit ? tr('pane.splitDownTitle') : tr('pane.splitDownMax', { max: maxSplits })} aria-label={tr('split.down')} disabled={!canSplit} onClick={() => splitEmpty(pane.id, 'col')}><Rows2 size={12} /></button>
          </>
        )}
        {total >= 2 && (
          <button className="icon-btn small" title={tab.zoomed ? tr('pane.restoreTitle') : tr('pane.maximizeTitle')} aria-label={tr('pane.maximize')} onClick={() => { focusPane(pane.id); toggleZoom(tab.id); }}>
            {tab.zoomed ? <Minimize2 size={12} /> : <Maximize2 size={12} />}
          </button>
        )}
        <button
          className="icon-btn small"
          title={tr('common.close')}
          aria-label={tr('common.close')}
          onClick={() => void closePane(pane.id)}
        >
          <X size={12} />
        </button>
      </div>
      <div className="pane-body">
        {hasSession ? (
          <TerminalView paneId={pane.id} sessionId={pane.sessionId!} focused={focused} />
        ) : (
          <div className="empty-pane">
            <div>{tr('pane.dropHint')}</div>
            <div className="faint">{tr('pane.dropHint2')}</div>
            <button className="btn" onClick={() => openQuick(pane.id)}><ZoomIn size={13} />{tr('pane.quick')}</button>
          </div>
        )}
        {total >= 2 && hasSession && <div className="pane-watermark">{relabel(info?.title)}</div>}
      </div>
    </div>
  );
}
