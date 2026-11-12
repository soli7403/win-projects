// 화면 뼈대: 제목 표시줄 · 본문(사이드바 · 작업 영역) · 상태 표시줄. 근거: spec/ui/shell.md §1~§3.
// 이번 단계 범위: 탭·사이드바·우측 패널은 자리만 있고 다음 단계에서 구현한다(report 참고).
import { useEffect, useRef, useState } from 'react';
import { formatDuration } from "../lib/format";
import { findPane } from '../lib/layout';
import { Code, Link, Lock as LockIcon, LockOpen, PanelLeft, Settings, Unlink } from 'lucide-react';
import { TabBar, TAB_MIME } from './TabBar';
import { PaneTree } from './Panes';
import { useWorkspace, isSplit } from '../state/workspace';
import { getBackend, errorMessage } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { Welcome } from './Welcome';
import { PanelHost, SftpDock } from './RightPanel';
import { CopyJobs } from './CopyJobs';
import { SnippetsPanel } from './SnippetsPanel';
import { Sidebar } from './Sidebar';
import { UpdateBadge } from '../ui/UpdateBadge';

/** 왼쪽 위 아이콘 묶음: 서버 목록 · 스니펫 · 잠그기 · 설정. 서버 목록이 열려 있으면 가로, 접혀 있으면 세로 막대. */
export function ToolRow({ vertical }: { vertical: boolean }) {
  const sidebarOpen = useUi((s) => s.sidebarOpen);
  const toggle = useUi((s) => s.toggleSidebar);
  const vaultOpen = useApp((s) => !!s.vault?.unlocked);
  const snippetsOpen = useUi((s) => s.snippetsOpen);
  const pushToast = useApp((s) => s.pushToast);
  const linkInput = useUi((s) => s.linkInput);
  const toggleLink = () => {
    useUi.getState().toggleLinkInput();
    pushToast({ kind: 'info', text: linkInput ? '동시 입력을 껐습니다 — 창마다 따로 입력합니다' : '동시 입력을 켰습니다 — 보이는 모든 창에 같이 입력됩니다' });
  };
  const lock = async () => {
    try {
      await getBackend().invoke('vault.lock');
    } catch (e) {
      pushToast({ kind: 'error', text: errorMessage(e) });
    }
  };
  return (
    <div className={`toolrow${vertical ? ' vertical' : ''}`}>
      <button
        className={`icon-btn${sidebarOpen ? ' on' : ''}`}
        onClick={toggle}
        title="서버 목록 (Ctrl+Shift+L)"
        aria-label="서버 목록"
        aria-pressed={sidebarOpen}
      >
        <PanelLeft size={16} />
      </button>
      <button
        className={`icon-btn${sidebarOpen && snippetsOpen ? ' on' : ''}`}
        title="스니펫"
        aria-label="스니펫"
        aria-pressed={sidebarOpen && snippetsOpen}
        onClick={() => {
          const ui = useUi.getState();
          // 스니펫은 서버 목록 아래 절반에 보이므로, 서버 목록이 접혀 있으면 같이 연다.
          if (!ui.sidebarOpen) {
            ui.setSnippetsOpen(true);
            ui.toggleSidebar();
          } else ui.setSnippetsOpen(!ui.snippetsOpen);
        }}
      >
        <Code size={16} />
      </button>
      {/* 모든 창 동시 입력: 이어진 쇠사슬 = 켜짐(보이는 모든 창에 같이 입력), 끊어진 쇠사슬 = 꺼짐 */}
      <button
        className={`icon-btn link-input${linkInput ? ' on' : ''}`}
        title={linkInput ? '동시 입력 켜짐 — 보이는 모든 창에 같이 입력합니다 (누르면 끄기)' : '동시 입력 꺼짐 — 누르면 보이는 모든 창에 같이 입력합니다'}
        aria-label="모든 창 동시 입력"
        aria-pressed={linkInput}
        onClick={toggleLink}
      >
        {linkInput ? <Link size={16} /> : <Unlink size={16} />}
      </button>
      {vaultOpen && (
        <button className="icon-btn" title="지금 잠그기" aria-label="지금 잠그기" onClick={lock}>
          <LockIcon size={16} />
        </button>
      )}
      <button className="icon-btn" title="설정" aria-label="설정" onClick={() => useUi.getState().openSettings()}>
        <Settings size={16} />
      </button>
    </div>
  );
}

const STATE_LABEL: Record<string, string> = {
  connected: '연결됨',
  connecting: '접속 중',
  reconnecting: '재접속 대기',
  closed: '끊김',
  error: '실패',
};

function useNow(enabled: boolean): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!enabled) return;
    const t = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(t);
  }, [enabled]);
  return now;
}

export function StatusBar() {
  const vault = useApp((s) => s.vault);
  const pushToast = useApp((s) => s.pushToast);
  const open = !!vault?.unlocked;
  const focusSid = useWorkspace((s) => {
    const t = s.tabs.find((x) => x.id === s.activeTabId);
    return t?.focusedPane ? findPane(t.root, t.focusedPane)?.sessionId ?? null : null;
  });
  const entry = useWorkspace((s) => (focusSid ? s.sessions[focusSid] : undefined));
  const broadcast = useWorkspace((s) => {
    const t = s.tabs.find((x) => x.id === s.activeTabId);
    return !!t?.broadcast;
  });
  const connected = entry?.info.state === 'connected';
  const now = useNow(connected && !!entry?.info.connectedAt);
  const size = useWorkspace((s) => (focusSid ? s.sizes[focusSid] : undefined));
  const savedHost = useApp((s) => (entry?.info.hostId ? s.store?.hosts.find((h) => h.id === entry.info.hostId) : undefined));
  const imageHint = !!entry && (entry.info.protocol === 'local' || (entry.info.canSftp && connected));
  const imageTip = entry?.info.protocol === 'local'
    ? '클립보드 이미지를 이 PC에 저장하고 경로를 입력합니다 (Claude Code 등 CLI에서 이미지 첨부)'
    : '클립보드 이미지를 서버에 올리고 경로를 입력합니다 (Claude Code 등 CLI에서 이미지 첨부)';
  const lock = async () => {
    try {
      await getBackend().invoke('vault.lock');
    } catch (e) {
      pushToast({ kind: 'error', text: errorMessage(e) });
    }
  };
  const info = entry?.info;
  const elapsed = info?.connectedAt ? formatDuration(now - info.connectedAt) : null;
  return (
    <footer className="statusbar">
      {info ? (
        <>
          <span className={`dot ${info.state}`} />
          <b style={{ color: info.color ?? (info.env === 'prod' ? 'var(--danger)' : undefined) }}>{info.title}</b>
          <span className="faint">{info.target}</span>
          <span>
            {STATE_LABEL[info.state] ?? info.state}
            {connected && elapsed ? ` · ${elapsed}` : ''}
            {!connected && info.message ? ` · ${info.message}` : ''}
          </span>
          {broadcast && <span className="badge-bc">동시 입력 ON</span>}
        </>
      ) : (
        <span className="faint">연결된 세션 없음</span>
      )}
      <span className="grow" />
      <UpdateBadge />
      {imageHint && <span className="faint" title={imageTip}>이미지 붙여넣기 Ctrl+V</span>}
      {savedHost && <span className="faint">{savedHost.encoding.toUpperCase()}</span>}
      {entry && size && <span className="faint">{`${size.cols}×${size.rows}`}</span>}
      {open ? (
        <button className="link-btn" title="클릭하면 볼트를 잠급니다" onClick={lock}>
          <LockOpen size={13} style={{ verticalAlign: '-2px', marginRight: 4 }} />볼트 열림
        </button>
      ) : (
        <span title="볼트 잠김">
          <LockIcon size={13} style={{ verticalAlign: '-2px', marginRight: 4 }} />볼트 잠김
        </span>
      )}
    </footer>
  );
}

/** 작업 영역의 한 열: 자기 탭 막대와 그 그룹에서 선택된 탭의 내용. 탭이 하나도 없으면 시작 화면. */
function TabGroupColumn({ group, split, flex }: { group: 0 | 1; split: boolean; flex: number }) {
  const tab = useWorkspace((s) => s.tabs.find((t) => t.id === s.activeByGroup[group]));
  const focused = useWorkspace((s) => s.activeTabId !== null && s.activeTabId === s.activeByGroup[group]);
  const sftpOpen = useUi((s) => s.sftpOpen[group]);
  return (
    <section
      className={`tabgroup${split && focused ? ' focused' : ''}`}
      style={{ flex: `${flex} 1 0` }}
      aria-label={group === 0 ? '왼쪽 탭 그룹' : '오른쪽 탭 그룹'}
    >
      <TabBar group={group} />
      <div className="tabgroup-body">
        <div className="tabgroup-main">{tab ? <PaneTree key={tab.id} tab={tab} /> : group === 0 ? <Welcome /> : null}</div>
        {sftpOpen && tab && <SftpDock group={group} />}
      </div>
    </section>
  );
}

/** 두 그룹 사이 경계: 끌어서 좌우 비율을 바꾼다(20~80%). */
function GroupDivider({ containerRef }: { containerRef: React.RefObject<HTMLDivElement | null> }) {
  const onDown = (e: React.PointerEvent<HTMLDivElement>) => {
    const box = containerRef.current;
    if (!box) return;
    e.preventDefault();
    const el = e.currentTarget;
    el.setPointerCapture(e.pointerId);
    const move = (ev: PointerEvent) => {
      const r = box.getBoundingClientRect();
      useWorkspace.getState().setGroupRatio(((ev.clientX - r.left) / r.width) * 100);
    };
    const up = () => {
      el.removeEventListener('pointermove', move);
      el.removeEventListener('pointerup', up);
      el.removeEventListener('pointercancel', up);
    };
    el.addEventListener('pointermove', move);
    el.addEventListener('pointerup', up);
    el.addEventListener('pointercancel', up);
  };
  return (
    <div
      className="group-divider"
      role="separator"
      aria-orientation="vertical"
      aria-label="탭 그룹 경계"
      onPointerDown={onDown}
      onDoubleClick={() => useWorkspace.getState().setGroupRatio(50)}
    />
  );
}

export function Shell() {
  const sidebarOpen = useUi((s) => s.sidebarOpen);
  const split = useWorkspace((s) => isSplit(s.tabs));
  const ratio = useWorkspace((s) => s.groupRatio);
  const canSplit = useWorkspace((s) => s.tabs.length >= 2);
  const snippetsOpen = useUi((s) => s.snippetsOpen);
  const [dragHint, setDragHint] = useState(false);
  const groupsRef = useRef<HTMLDivElement>(null);
  const isTabDrag = (e: React.DragEvent) => !split && canSplit && e.dataTransfer.types.includes(TAB_MIME);
  return (
    <div className="app">
      <div className="body">
        {sidebarOpen ? (
          <div className="leftcol">
            <ToolRow vertical={false} />
            <Sidebar />
            {snippetsOpen && (
              <div className="snippet-dock" aria-label="스니펫">
                <SnippetsPanel />
              </div>
            )}
          </div>
        ) : (
          <div className="rail">
            <ToolRow vertical />
          </div>
        )}
        <main
          className="workspace"
          onDragOver={(e) => {
            if (isTabDrag(e)) {
              e.preventDefault();
              setDragHint(true);
            }
          }}
          onDragLeave={(e) => {
            if (e.currentTarget === e.target) setDragHint(false);
          }}
          onDrop={() => setDragHint(false)}
        >
          <div className={`tabgroups${split ? ' split' : ''}`} ref={groupsRef}>
            <TabGroupColumn group={0} split={split} flex={split ? ratio : 1} />
            {split && <GroupDivider containerRef={groupsRef} />}
            {split && <TabGroupColumn group={1} split={split} flex={100 - ratio} />}
          </div>
          {dragHint && (
            <div
              className="split-drop"
              onDragOver={(e) => e.preventDefault()}
              onDragLeave={() => setDragHint(false)}
              onDrop={(e) => {
                const id = e.dataTransfer.getData(TAB_MIME);
                setDragHint(false);
                if (!id) return;
                e.preventDefault();
                useWorkspace.getState().moveTabToGroup(id, 1);
              }}
            >
              <span>여기에 놓아 화면 나누기</span>
            </div>
          )}
        </main>
      </div>
      <CopyJobs />
      <StatusBar />
      <PanelHost />
    </div>
  );
}
