// 화면 뼈대: 제목 표시줄 · 본문(사이드바 · 작업 영역) · 상태 표시줄. 근거: spec/ui/shell.md §1~§3.
// 이번 단계 범위: 탭·사이드바·우측 패널은 자리만 있고 다음 단계에서 구현한다(report 참고).
import { useEffect, useRef, useState } from 'react';
import { formatDuration } from "../lib/format";
import { findPane } from '../lib/layout';
import { CircleHelp, Code, Link, Lock as LockIcon, LockOpen, PanelLeft, Settings, Unlink } from 'lucide-react';
import { TabBar, TAB_MIME } from './TabBar';
import { PaneTree } from './Panes';
import { useWorkspace, isSplit } from '../state/workspace';
import { getBackend, errorMessage } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi, SNIPPET_RATIO_DEFAULT } from '../state/ui';
import { Welcome } from './Welcome';
import { PanelHost, SftpDock } from './RightPanel';
import { CopyJobs } from './CopyJobs';
import { useSplitDropHint } from './splitDropHint';
import { SnippetsPanel } from './SnippetsPanel';
import { Sidebar, SidebarResizer, useSidebarWidth } from './Sidebar';
import { UpdateBadge } from '../ui/UpdateBadge';
import { toggleRecording, useMacros } from '../state/macros';
// 이 파일은 t 를 지역 변수 이름으로 쓰므로 번역 함수는 tr 로 가져온다(D75).
import { relabel, t as tr, type MsgId } from '../i18n';
import { usageShort, useUsageView } from './LicensePanel';

/** 왼쪽 위 아이콘 묶음: 서버 목록 · 명령 모음 · 일괄 입력 · 설정 (앱 잠그기는 오른쪽 아래 상태 표시줄, D67). 서버 목록이 열려 있으면 가로, 접혀 있으면 세로 막대. */
export function ToolRow({ vertical }: { vertical: boolean }) {
  const sidebarOpen = useUi((s) => s.sidebarOpen);
  const toggle = useUi((s) => s.toggleSidebar);
  const snippetsOpen = useUi((s) => s.snippetsOpen);
  const pushToast = useApp((s) => s.pushToast);
  const linkInput = useUi((s) => s.linkInput);
  const toggleLink = () => {
    useUi.getState().toggleLinkInput();
    pushToast({ kind: 'info', text: linkInput ? tr('shell.linkOff') : tr('shell.linkOn') });
  };
  return (
    <div className={`toolrow${vertical ? ' vertical' : ''}`}>
      <button
        className={`icon-btn${sidebarOpen ? ' on' : ''}`}
        onClick={toggle}
        title={tr('shell.serversTitle')}
        aria-label={tr('shell.servers')}
        aria-pressed={sidebarOpen}
      >
        <PanelLeft size={16} />
      </button>
      <button
        className={`icon-btn${sidebarOpen && snippetsOpen ? ' on' : ''}`}
        title={tr('shell.snippets')}
        aria-label={tr('shell.snippets')}
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
        title={linkInput ? tr('shell.linkOnTitle') : tr('shell.linkOffTitle')}
        aria-label={tr('shell.linkAria')}
        aria-pressed={linkInput}
        onClick={toggleLink}
      >
        {linkInput ? <Link size={16} /> : <Unlink size={16} />}
      </button>
      <button className="icon-btn" title={tr('settings.title')} aria-label={tr('settings.title')} onClick={() => useUi.getState().openSettings()}>
        <Settings size={16} />
      </button>
      {/* 도움말 (ui/help.md §1, D155) */}
      <button className="icon-btn" title={tr('help.title')} aria-label={tr('help.title')} onClick={() => useUi.getState().openHelp()}>
        <CircleHelp size={16} />
      </button>
    </div>
  );
}

const STATE_LABEL: Record<string, MsgId> = {
  connected: 'state.connected',
  connecting: 'state.connecting',
  reconnecting: 'state.reconnecting',
  closed: 'state.closed',
  error: 'state.error',
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
  const usage = useUsageView();
  const now = useNow(connected && !!entry?.info.connectedAt);
  const size = useWorkspace((s) => (focusSid ? s.sizes[focusSid] : undefined));
  const savedHost = useApp((s) => (entry?.info.hostId ? s.store?.hosts.find((h) => h.id === entry.info.hostId) : undefined));
  const imageHint = !!entry && (entry.info.protocol === 'local' || (entry.info.canSftp && connected));
  const imageTip = entry?.info.protocol === 'local'
    ? tr('shell.imageTipLocal')
    : tr('shell.imageTipServer');
  const lock = async () => {
    try {
      await getBackend().invoke('vault.lock');
    } catch (e) {
      pushToast({ kind: 'error', text: errorMessage(e) });
    }
  };
  const info = entry?.info;
  // 매크로 녹화 중이면 어느 화면에서든 보이고 멈출 수 있게 상태 표시줄에 둔다 (terminal.md §11).
  const recording = useMacros((s) => s.recording);
  const elapsed = info?.connectedAt ? formatDuration(now - info.connectedAt) : null;
  return (
    <footer className="statusbar">
      {info ? (
        <>
          <span className={`dot ${info.state}`} />
          <b style={{ color: info.color ?? (info.env === 'prod' ? 'var(--danger)' : undefined) }}>{relabel(info.title)}</b>
          <span className="faint">{info.target}</span>
          <span>
            {STATE_LABEL[info.state] ? tr(STATE_LABEL[info.state]) : info.state}
            {connected && elapsed ? ` · ${elapsed}` : ''}
            {!connected && info.message ? ` · ${info.message}` : ''}
          </span>
          {broadcast && <span className="badge-bc">{tr('pane.broadcasting')}</span>}
        </>
      ) : (
        <span className="faint">{tr('shell.noSession')}</span>
      )}
      <span className="grow" />
      {recording && (
        <button className="badge-rec" title={tr('macro.recordStop', { count: recording.count })} onClick={toggleRecording}>
          ● {tr('macro.recordingBadge', { count: recording.count })}
        </button>
      )}
      <UpdateBadge />
      {imageHint && <span className="faint" title={imageTip}>{tr('shell.imageHint')}</span>}
      {savedHost && <span className="faint">{savedHost.encoding.toUpperCase()}</span>}
      {entry && size && <span className="faint">{`${size.cols}×${size.rows}`}</span>}
      {/* 사용 형태(D78): 개인용 / 업무용 평가 n일 / 회사 이름(사용 허가 수). 누르면 설정 > 정보 */}
      <button className="link-btn faint" title={tr('lic.usageLabel')} onClick={() => useUi.getState().openSettings('about')}>
        {usageShort(usage)}
      </button>
      {open ? (
        <button className="link-btn" title={tr('shell.lockTitle')} onClick={lock}>
          <LockOpen size={13} style={{ verticalAlign: '-2px', marginRight: 4 }} />{tr('shell.unlocked')}
        </button>
      ) : (
        <span title={tr('shell.locked')}>
          <LockIcon size={13} style={{ verticalAlign: '-2px', marginRight: 4 }} />{tr('shell.locked')}
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
      aria-label={group === 0 ? tr('shell.leftView') : tr('shell.rightView')}
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
      aria-label={tr('shell.viewDivider')}
      onPointerDown={onDown}
      onDoubleClick={() => useWorkspace.getState().setGroupRatio(50)}
    />
  );
}

/** 서버 목록과 스니펫 패널 사이 경계: 위아래로 끌어 스니펫 높이 비율을 바꾼다. 두 번 누르면 기본값. */
function SnippetDivider({ containerRef }: { containerRef: React.RefObject<HTMLDivElement | null> }) {
  const onDown = (e: React.PointerEvent<HTMLDivElement>) => {
    const box = containerRef.current;
    if (!box) return;
    e.preventDefault();
    const el = e.currentTarget;
    el.setPointerCapture(e.pointerId);
    const move = (ev: PointerEvent) => {
      const r = box.getBoundingClientRect();
      if (r.height <= 0) return;
      // 스니펫은 아래쪽이므로 (열 바닥 − 포인터) 가 스니펫 높이.
      useUi.getState().setSnippetRatio(((r.bottom - ev.clientY) / r.height) * 100);
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
      className="snippet-divider"
      role="separator"
      aria-orientation="horizontal"
      aria-label={tr('shell.sidebarDivider')}
      onPointerDown={onDown}
      onDoubleClick={() => useUi.getState().setSnippetRatio(SNIPPET_RATIO_DEFAULT)}
    />
  );
}

export function Shell() {
  const sidebarOpen = useUi((s) => s.sidebarOpen);
  const snippetRatio = useUi((s) => s.snippetRatio);
  // 왼쪽 열의 너비: 열 전체(아이콘 묶음·서버 목록·명령 모음)가 같은 너비를 쓴다 (D130).
  const sidebarWidth = useSidebarWidth();
  const leftRef = useRef<HTMLDivElement>(null);
  const split = useWorkspace((s) => isSplit(s.tabs));
  const ratio = useWorkspace((s) => s.groupRatio);
  const canSplit = useWorkspace((s) => s.tabs.length >= 2);
  const snippetsOpen = useUi((s) => s.snippetsOpen);
  // 탭을 끄는 동안의 "여기에 놓으면 좌우로 나눕니다" 안내. 끌기가 어떻게 끝나든 지워진다(splitDropHint.ts).
  const dragHint = useSplitDropHint(!split && canSplit, TAB_MIME, '.tabbar');
  const groupsRef = useRef<HTMLDivElement>(null);
  return (
    <div className="app">
      <div className="body">
        {sidebarOpen ? (
          <div className="leftcol" style={{ width: sidebarWidth }}>
            <ToolRow vertical={false} />
            {/* 서버 목록 + 스니펫이 나눠 쓰는 영역. 비율은 SnippetDivider 로 바꾼다. */}
            <div
              className="leftcol-body"
              ref={leftRef}
              style={{ '--side-flex': snippetsOpen ? 100 - snippetRatio : 1 } as React.CSSProperties}
            >
              <Sidebar />
              {snippetsOpen && (
                <>
                  <SnippetDivider containerRef={leftRef} />
                  <div className="snippet-dock" aria-label={tr('shell.snippets')} style={{ flex: `${snippetRatio} 1 0` }}>
                    <SnippetsPanel />
                  </div>
                </>
              )}
            </div>
            {/* 너비 조절 손잡이: 열 전체 높이(서버 목록 + 명령 모음)에서 잡힌다. */}
            <SidebarResizer />
          </div>
        ) : (
          <div className="rail">
            <ToolRow vertical />
          </div>
        )}
        <main className="workspace" {...dragHint.zone}>
          <div className={`tabgroups${split ? ' split' : ''}`} ref={groupsRef}>
            <TabGroupColumn group={0} split={split} flex={split ? ratio : 1} />
            {split && <GroupDivider containerRef={groupsRef} />}
            {split && <TabGroupColumn group={1} split={split} flex={100 - ratio} />}
          </div>
          {dragHint.show && (
            <div
              className="split-drop"
              onDragOver={(e) => e.preventDefault()}
              onDrop={(e) => {
                const id = e.dataTransfer.getData(TAB_MIME);
                dragHint.hide();
                if (!id) return;
                e.preventDefault();
                useWorkspace.getState().moveTabToGroup(id, 1);
              }}
            >
              <span>{tr('shell.dropSplit')}</span>
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
