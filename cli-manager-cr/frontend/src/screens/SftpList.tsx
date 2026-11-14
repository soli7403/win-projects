// SFTP 목록 행·선택·키보드·우클릭 메뉴. 근거: spec/ui/sftp-snippets.md §2.3~§2.5, spec/ui/tokens.md §3(SFTP 행).
import type { DragEvent, KeyboardEvent, MouseEvent as RMouseEvent } from 'react';
import { File, FileImage, Folder, Link2 } from 'lucide-react';
import { formatBytes, formatMtime, isImageName } from '../lib/sftpPath';
import { useUi, type MenuItem } from '../state/ui';
import type { Entry } from './sftpOps';
import * as ops from './sftpOps';
import { t } from '../i18n';

export interface ListProps {
  sid: string;
  path: string;
  entries: Entry[];
  selected: Set<string>;
  onSelect: (index: number, mods: { ctrl: boolean; shift: boolean }) => void;
  onSelectAll: () => void;
  /** 선택 전부 다시 불러옴 등. 동작 뒤 새로고침 요청. */
  onRefresh: () => void;
  onOpen: (e: Entry) => void;
  onNavigate: (p: string) => void;
  onError: (msg: string) => void;
  /** 우클릭·키 동작에서 대상을 정한다(선택 규칙). */
  targetsOf: (e: Entry) => Entry[];
  setSelection: (paths: string[]) => void;
  /** 상위 폴더(루트면 path 와 같음) */
  parent: string;
  /** 경로 구분자, 이 PC 여부 (file-panel.md §1) */
  sep: string;
  local: boolean;
  /** 끌어 놓기 대상으로 강조할 폴더 */
  dropDir: string | null;
  onRowDragStart: (ev: DragEvent, e: Entry) => void;
  /** 행 위 끌기·놓기. dir 은 대상 폴더(폴더 행은 그 폴더, 파일 행은 현재 폴더). */
  onDirDragOver: (ev: DragEvent, dir: string) => void;
  onDirDrop: (ev: DragEvent, dir: string) => void;
}

function EntryIcon({ e }: { e: Entry }) {
  if (e.isDir) return <Folder size={14} className="ico-dir" fill="#f5b942" color="#f5b942" />;
  if (isImageName(e.name)) return <FileImage size={14} color="var(--fg-2)" />;
  return <File size={14} color="var(--fg-2)" />;
}

export function SftpList(p: ListProps) {
  const menuFor = (e: Entry): MenuItem[] => {
    const targets = p.targetsOf(e);
    const n = targets.length;
    const cur = p.path;
    // Windows 로컬에서는 권한 변경을 보이지 않는다 (file-panel.md §1).
    const showChmod = !(p.local && p.sep !== '/');
    return [
      {
        // 우클릭한 행 하나가 대상이며 다중 선택에서도 활성(spec §2.4, 2026-11-10 답).
        label: e.isDir ? t('sftp.open') : t('sftp.preview'),
        onSelect: () => p.onOpen(e),
      },
      { label: p.local ? t('sftp.copyTo') : t('sftp.downloadTo'), onSelect: () => void ops.downloadPaths(p.sid, targets.map((t) => t.path), p.local) },
      {
        label: p.local ? t('sftp.openDefault') : t('sftp.editLocal'),
        disabled: e.isDir || n > 1,
        onSelect: () => void ops.editLocally(p.sid, e.path, p.local),
      },
      { separator: true },
      { label: t('preview.typePath'), onSelect: () => ops.typePaths(targets.map((t) => t.path), p.sep) },
      {
        label: t('sftp.cdHere'),
        onSelect: () => ops.cdTerminal(e.isDir ? e.path : cur, p.sep),
      },
      { label: t('sftp.copyPath'), onSelect: () => void ops.copyPaths(targets.map((t) => t.path)) },
      { separator: true },
      {
        label: t('common.rename'),
        disabled: n > 1,
        onSelect: () => void ops.renameEntry(p.sid, e, cur, p.sep).then((ok) => ok && p.onRefresh()),
      },
      ...(showChmod
        ? [
            {
              label: t('sftp.chmodLabel', { perm: e.perm }),
              disabled: n > 1,
              onSelect: () => void ops.chmodEntry(p.sid, e).then((ok) => ok && p.onRefresh()),
            },
          ]
        : []),
      { separator: true },
      {
        label: t('common.delete'),
        danger: true,
        shortcut: 'Delete',
        onSelect: () => void ops.deleteEntries(p.sid, targets, p.local).then((ok) => ok && p.onRefresh()),
      },
    ];
  };

  const onRowClick = (e: RMouseEvent, index: number) => {
    p.onSelect(index, { ctrl: e.ctrlKey || e.metaKey, shift: e.shiftKey });
  };

  const onContext = (ev: RMouseEvent, entry: Entry) => {
    ev.preventDefault();
    // 우클릭한 행이 선택에 없으면 그 행만 선택한다.
    if (!p.selected.has(entry.path)) p.setSelection([entry.path]);
    useUi.getState().openMenu(ev.clientX, ev.clientY, menuFor(entry));
  };

  const onKey = (ev: KeyboardEvent<HTMLDivElement>) => {
    const selectedEntries = p.entries.filter((x) => p.selected.has(x.path));
    if (ev.key === 'Delete') {
      if (selectedEntries.length) {
        ev.preventDefault();
        void ops.deleteEntries(p.sid, selectedEntries, p.local).then((ok) => ok && p.onRefresh());
      }
      return;
    }
    if (ev.key === 'Backspace') {
      ev.preventDefault();
      if (p.parent !== p.path) p.onNavigate(p.parent);
      return;
    }
    if (ev.key === 'Enter') {
      ev.preventDefault();
      if (selectedEntries.length === 1) p.onOpen(selectedEntries[0]);
      return;
    }
    if ((ev.ctrlKey || ev.metaKey) && (ev.key === 'a' || ev.key === 'A')) {
      ev.preventDefault();
      p.onSelectAll();
    }
  };

  return (
    <div className="sftp-list" tabIndex={0} onKeyDown={onKey} role="listbox" aria-label={t('sftp.fileList')} aria-multiselectable="true">
      <div className="sftp-head" aria-hidden="true">
        <span />
        <span>{t('sftp.colName')}</span>
        <span style={{ textAlign: 'right' }}>{t('sftp.colSize')}</span>
        <span style={{ textAlign: 'right' }}>{t('sftp.colModified')}</span>
      </div>
      {p.entries.length === 0 && (
        <div className="sftp-empty">{p.local ? t('sftp.emptyCopy') : t('sftp.emptyUpload')}</div>
      )}
      {p.entries.map((e, i) => {
        const sel = p.selected.has(e.path);
        const perm = e.perm;
        return (
          <div
            key={e.path}
            role="option"
            aria-selected={sel}
            className={`sftp-row${sel ? ' sel' : ''}${e.isDir && p.dropDir === e.path ? ' drop-target' : ''}`}
            draggable
            onDragStart={(ev) => p.onRowDragStart(ev, e)}
            onDragOver={(ev) => p.onDirDragOver(ev, e.isDir ? e.path : p.path)}
            onDrop={(ev) => p.onDirDrop(ev, e.isDir ? e.path : p.path)}
            title={`${e.name}\n${perm}  ${e.isDir ? '' : formatBytes(e.size)}`}
            onMouseDown={(ev) => {
              // 더블클릭 선택 글자 드래그 방지. 선택 처리는 click 에서 한다.
              if (ev.detail > 1) ev.preventDefault();
            }}
            onClick={(ev) => onRowClick(ev, i)}
            onDoubleClick={() => p.onOpen(e)}
            onContextMenu={(ev) => onContext(ev, e)}
          >
            <EntryIcon e={e} />
            <span className="name">
              {e.name}
              {e.isLink && <Link2 size={11} style={{ marginLeft: 4, verticalAlign: '-1px' }} color="var(--fg-3)" aria-label={t('sftp.link')} />}
            </span>
            <span className="num">{e.isDir ? '' : formatBytes(e.size)}</span>
            <span className="mtime">{formatMtime(e.mtime)}</span>
          </div>
        );
      })}
    </div>
  );
}
