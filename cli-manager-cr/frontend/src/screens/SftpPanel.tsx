// SFTP 탭: 포커스 세션 대상, 상태 안내, 도구 막대, 목록, 끌어 놓기, 전송 목록.
// 근거: spec/ui/sftp-snippets.md §2.1~§2.6, spec/protocols.md §3.
import { useEffect, useMemo, useRef, useState, type DragEvent } from 'react';
import { AlertCircle, ArrowDown, ArrowUp, ArrowUpFromLine, Check, CircleAlert, FolderPlus, FolderTree, House, RefreshCw, SquareTerminal, Upload, X } from 'lucide-react';
import { useWorkspace, type SessionEntry } from '../state/workspace';
import { findPane } from '../lib/layout';
import { formatBytes, transferPercent } from '../lib/sftpPath';
import { useCopyJobs } from '../state/copyJobs';
import { lastDir, isFileDrag, startFileDrag, dropFiles } from './fileDrag';
import { useTransfers, type TransferInfo } from '../state/transfers';
import { SftpList } from './SftpList';
import * as ops from './sftpOps';
import type { Entry } from './sftpOps';
import { errorMessage } from '../ipc/backend';
import { useUi } from '../state/ui';

/** 세션별 마지막으로 본 경로(세션마다 기억, fileDrag.lastDir). 패널을 닫아도 유지된다. */
const lastPath = lastDir;

/** 그룹(없으면 포커스 그룹)의 선택 탭에서 포커스된 칸의 세션 id(없으면 null). */
function useFocusedSid(group?: 0 | 1): string | null {
  return useWorkspace((s) => {
    const tid = group === undefined ? s.activeTabId : s.activeByGroup[group];
    const t = s.tabs.find((x) => x.id === tid);
    return t?.focusedPane ? findPane(t.root, t.focusedPane)?.sessionId ?? null : null;
  });
}

interface State {
  path: string;
  /** 상위 폴더(루트면 path 와 같음)·구분자·이 PC 여부 */
  parent: string;
  sep: string;
  local: boolean;
  home: string;
  entries: Entry[];
  loading: boolean;
  error: string | null;
  selected: Set<string>;
  anchor: number | null;
}

const EMPTY: State = { path: '', parent: '', sep: '/', local: false, home: '', entries: [], loading: false, error: null, selected: new Set(), anchor: null };

/** onClose 를 주면 제목 줄 오른쪽 끝에 닫기(X)를 둔다 (SFTP 도크). */
export function SftpPanel({ group, onClose }: { group?: 0 | 1; onClose?: () => void } = {}) {
  const sid = useFocusedSid(group);
  const entry: SessionEntry | undefined = useWorkspace((s) => (sid ? s.sessions[sid] : undefined));
  const info = entry?.info;
  // SSH 는 SFTP, 로컬 터미널은 이 PC 파일시스템 (file-panel.md §1).
  const isLocal = info?.protocol === 'local';
  const sftpOk = !!info && ((info.canSftp && info.protocol === 'ssh') || isLocal);
  const connected = !!info && info.state === 'connected';
  const ready = !!sid && sftpOk && connected;

  const [st, setSt] = useState<State>(EMPTY);
  const [pathDraft, setPathDraft] = useState('');
  /** 끌어 놓기 중인 대상 폴더와 종류(패널 항목 끌기면 file) */
  const [drop, setDrop] = useState<{ dir: string; file: boolean } | null>(null);
  const reqSeq = useRef(0);
  const stRef = useRef(st);
  stRef.current = st;

  const load = async (target: string | null, sidOverride?: string) => {
    const s = sidOverride ?? sid;
    if (!s) return;
    const seq = ++reqSeq.current;
    setSt((x) => ({ ...x, loading: true, error: null }));
    try {
      // 빈 경로는 백엔드가 시작 폴더(로컬 터미널의 시작 폴더, 서버는 홈)로 연다.
      const want = target ?? lastPath.get(s) ?? '';
      const listing = await ops.sftpList(s, want);
      if (seq !== reqSeq.current) return;
      lastPath.set(s, listing.path);
      setPathDraft(listing.path);
      setSt({
        path: listing.path, parent: listing.parent ?? listing.path, sep: listing.sep ?? '/', local: !!listing.local, home: '',
        entries: listing.entries, loading: false, error: null, selected: new Set(), anchor: null,
      });
    } catch (e) {
      if (seq !== reqSeq.current) return;
      setSt((x) => ({ ...x, loading: false, error: errorMessage(e) }));
    }
  };

  // 포커스 세션이 바뀌거나 연결 상태가 바뀌면 그 세션 기준으로 다시 연다.
  useEffect(() => {
    setSt(EMPTY);
    setPathDraft('');
    reqSeq.current++;
    if (ready) void load(null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sid, ready]);

  // 업로드가 끝나고 그 원격 경로가 현재 폴더 아래이면 자동 새로고침(spec §2.2).
  const seenDone = useRef<Set<string> | null>(null);
  const transfers = useTransfers((s) => s.items);
  useEffect(() => {
    if (!sid) return;
    if (seenDone.current === null) {
      seenDone.current = new Set(transfers.filter((t) => t.state === 'done').map((t) => t.id));
      return;
    }
    for (const t of transfers) {
      if (t.state !== 'done' || seenDone.current.has(t.id)) continue;
      seenDone.current.add(t.id);
      const cur = stRef.current;
      const prefix = cur.path.endsWith(cur.sep) ? cur.path : cur.path + cur.sep;
      if (t.direction === 'upload' && t.sessionId === sid && t.remotePath.startsWith(prefix) && !t.remotePath.slice(prefix.length).includes(cur.sep)) {
        void load(cur.path);
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [transfers, sid]);

  // 이 세션으로 들어오는 복사 작업이 끝나고 대상이 현재 폴더(또는 그 아래)면 새로고침 (file-panel.md §3).
  const jobs = useCopyJobs((s) => s.items);
  const seenJobs = useRef<Set<string> | null>(null);
  useEffect(() => {
    if (!sid) return;
    if (seenJobs.current === null) {
      seenJobs.current = new Set(jobs.filter((j) => j.state === 'done').map((j) => j.id));
      return;
    }
    for (const j of jobs) {
      if (j.state !== 'done' || seenJobs.current.has(j.id)) continue;
      seenJobs.current.add(j.id);
      const cur = stRef.current;
      const prefix = cur.path.endsWith(cur.sep) ? cur.path : cur.path + cur.sep;
      if (j.dstSession === sid && (j.dstDir === cur.path || j.dstDir.startsWith(prefix))) void load(cur.path);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [jobs, sid]);

  const refresh = () => void load(stRef.current.path || null);

  // 선택 규칙: 클릭=단일, Ctrl=추가/해제, Shift=앵커부터 범위(Ctrl 과 함께면 기존 선택에 추가).
  const onSelect = (index: number, mods: { ctrl: boolean; shift: boolean }) => {
    setSt((x) => {
      const paths = x.entries.map((e) => e.path);
      const target = paths[index];
      if (mods.shift && x.anchor !== null) {
        const [a, b] = [Math.min(x.anchor, index), Math.max(x.anchor, index)];
        const range = paths.slice(a, b + 1);
        const next = mods.ctrl ? new Set([...x.selected, ...range]) : new Set(range);
        return { ...x, selected: next };
      }
      if (mods.ctrl) {
        const next = new Set(x.selected);
        if (next.has(target)) next.delete(target);
        else next.add(target);
        return { ...x, selected: next, anchor: index };
      }
      return { ...x, selected: new Set([target]), anchor: index };
    });
  };

  const setSelection = (paths: string[]) => setSt((x) => ({ ...x, selected: new Set(paths) }));
  const selectAll = () => setSt((x) => ({ ...x, selected: new Set(x.entries.map((e) => e.path)) }));

  // 대상 규칙: 우클릭한 행이 선택에 속하면 선택 전체, 아니면 그 행만.
  const targetsOf = (e: Entry): Entry[] => {
    const cur = stRef.current;
    if (cur.selected.has(e.path)) return cur.entries.filter((x) => cur.selected.has(x.path));
    return [e];
  };

  const openEntry = (e: Entry) => {
    if (!sid) return;
    if (e.isDir) void load(e.path);
    else useUi.getState().openPreview(sid, e.path);
  };

  const navigate = (p: string) => void load(p);

  const onNewFolder = async () => {
    if (!sid) return;
    if (await ops.makeFolder(sid, st.path, st.sep)) refresh();
  };
  const onUpload = async () => {
    if (!sid) return;
    if (await ops.uploadViaPicker(sid, st.path)) refresh();
  };
  const onCd = () => {
    if (st.path) ops.cdTerminal(st.path, st.sep);
  };

  // 끌어 놓기 (file-panel.md §2): 패널 항목은 복사 작업, OS 파일은 업로드(로컬이면 복사).
  // dir 은 대상 폴더(폴더 행은 그 폴더, 파일 행·빈 곳은 현재 폴더). 브라우저에서 파일 경로를 얻을 수 없으면 아무 일도 없다(spec §2.3).
  const onDragOver = (e: DragEvent, dir: string) => {
    const file = isFileDrag(e);
    if (!ready || (!file && !Array.from(e.dataTransfer.types).includes('Files'))) return;
    e.preventDefault();
    e.stopPropagation();
    e.dataTransfer.dropEffect = 'copy';
    if (drop?.dir !== dir || drop.file !== file) setDrop({ dir, file });
  };
  const onDrop = async (e: DragEvent, dir: string) => {
    e.preventDefault();
    e.stopPropagation();
    setDrop(null);
    if (!ready || !sid) return;
    if (isFileDrag(e)) {
      void dropFiles(sid, dir);
      return;
    }
    const paths = Array.from(e.dataTransfer.files)
      .map((f) => (f as File & { path?: string }).path)
      .filter((p): p is string => !!p);
    if (await ops.uploadLocal(sid, dir, paths)) refresh();
  };
  const onRowDragStart = (e: DragEvent, entry: Entry) => {
    if (!sid) return;
    const items = targetsOf(entry);
    if (!stRef.current.selected.has(entry.path)) setSelection([entry.path]);
    startFileDrag(e, { sessionId: sid, dir: stRef.current.path, paths: items.map((x) => x.path) });
  };

  const transferList = useMemo<TransferInfo[]>(
    () => transfers.filter((t) => t.sessionId === sid).slice(0, 30),
    [transfers, sid],
  );

  let status: { text: string; sub?: string } | null = null;
  if (!sid) status = { text: '세션을 선택하면 해당 서버(또는 이 PC)의 파일이 표시됩니다.' };
  else if (!sftpOk) status = { text: '파일 패널은 SSH·로컬 터미널 세션에서만 사용할 수 있습니다.', sub: `(${info!.protocol.toUpperCase()} 세션)` };
  else if (!connected) status = { text: '연결되면 파일 목록이 표시됩니다.' };

  return (
    <div className="rpanel-sftp" style={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0 }}>
      {/* 제목 줄: 제목 · 새로고침 · 새 폴더 · 업로드 · cd · 닫기 (sftp-snippets.md §1·§2.2) */}
      <div className="rpanel-tabs">
        <span className="rpanel-tab on"><FolderTree size={14} />{isLocal ? '파일' : 'SFTP'}</span>
        <span className="spacer" />
        <button className="icon-btn small" title="새로고침" aria-label="새로고침" disabled={!ready} onClick={refresh}>
          <RefreshCw size={14} className={st.loading ? 'spin' : undefined} />
        </button>
        <button className="icon-btn small" title="새 폴더" aria-label="새 폴더" disabled={!ready} onClick={() => void onNewFolder()}>
          <FolderPlus size={14} />
        </button>
        <button className="icon-btn small" title={isLocal ? '파일 가져오기 (이 폴더로 복사)' : '파일 업로드'} aria-label="파일 업로드" disabled={!ready} onClick={() => void onUpload()}>
          <Upload size={14} />
        </button>
        <button className="icon-btn small" title="터미널에서 이 폴더로 cd" aria-label="터미널에서 이 폴더로 cd" disabled={!ready} onClick={onCd}>
          <SquareTerminal size={14} />
        </button>
        {onClose && (
          <button className="icon-btn small" aria-label="닫기" title="닫기" onClick={onClose}>
            <X size={14} />
          </button>
        )}
      </div>
      {/* 경로 줄: 상위 폴더 · 홈 · 경로 입력 */}
      <div className="sftp-tools">
        <button className="icon-btn small" title="상위 폴더" aria-label="상위 폴더" disabled={!ready || st.parent === st.path} onClick={() => navigate(st.parent)}>
          <ArrowUp size={14} />
        </button>
        <button className="icon-btn small" title="홈" aria-label="홈" disabled={!ready} onClick={() => navigate('~')}>
          <House size={14} />
        </button>
        <input
          className="input mono sftp-path"
          aria-label="경로"
          spellCheck={false}
          value={pathDraft}
          disabled={!ready}
          onChange={(e) => setPathDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault();
              navigate(pathDraft.trim() || '~');
            }
          }}
        />
      </div>

      {status && (
        <div className="sftp-status">
          <div>{status.text}</div>
          {status.sub && <div className="faint">{status.sub}</div>}
        </div>
      )}

      {ready && st.error && (
        <div className="sftp-error" role="alert">
          <CircleAlert size={14} />
          <span>{st.error}</span>
        </div>
      )}

      {ready && (
        <div
          style={{ position: 'relative', flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}
          onDragOver={(e) => onDragOver(e, st.path)}
          onDragLeave={(e) => {
            if (!e.currentTarget.contains(e.relatedTarget as Node | null)) setDrop(null);
          }}
          onDrop={(e) => void onDrop(e, st.path)}
        >
          {sid && (
            <SftpList
              sid={sid}
              path={st.path}
              entries={st.entries}
              selected={st.selected}
              onSelect={onSelect}
              onSelectAll={selectAll}
              onRefresh={refresh}
              onOpen={openEntry}
              onNavigate={navigate}
              onError={() => undefined}
              targetsOf={targetsOf}
              setSelection={setSelection}
              parent={st.parent}
              sep={st.sep}
              local={st.local}
              dropDir={drop?.dir ?? null}
              onRowDragStart={onRowDragStart}
              onDirDragOver={onDragOver}
              onDirDrop={(e, dir) => void onDrop(e, dir)}
            />
          )}
          {drop && (
            <div className="sftp-drop">여기에 놓으면 {drop.dir} (으)로 {drop.file || st.local ? '복사' : '업로드'}</div>
          )}
        </div>
      )}

      {sid && transferList.length > 0 && <TransferBox items={transferList} />}
    </div>
  );
}

function TransferBox({ items }: { items: TransferInfo[] }) {
  const clear = useTransfers((s) => s.clearFinished);
  return (
    <div className="transfers">
      <div className="transfers-head">
        <span>전송</span>
        <span className="spacer" />
        <button className="btn link small-btn" onClick={clear}>지우기</button>
      </div>
      <div className="transfers-list">
        {items.map((t) => {
          const pct = transferPercent(t.bytes, t.total, t.state);
          const done = t.state === 'done';
          const error = t.state === 'error';
          return (
            <div key={t.id} className="tr-row" title={error ? t.error : t.remotePath}>
              <div className="tr-line">
                {t.direction === 'upload' ? <ArrowUpFromLine size={13} /> : <ArrowDown size={13} />}
                <span className="name">{t.name}</span>
                {done && <Check size={13} color="var(--ok)" aria-label="완료" />}
                {error && <AlertCircle size={13} color="var(--danger)" aria-label="오류" />}
                {!done && !error && <span className="pct">{pct}%</span>}
                <span className="size">{formatBytes(t.total)}</span>
              </div>
              <div className={`tr-bar${done ? ' done' : error ? ' error' : ''}`}>
                <div style={{ width: `${done ? 100 : error ? 100 : pct}%` }} />
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
