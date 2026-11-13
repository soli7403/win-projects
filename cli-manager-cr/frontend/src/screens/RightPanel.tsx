// SFTP 도크(탭 그룹마다 최대 1개, 그룹 오른쪽에 붙음)와 전송 이벤트·미리보기 호스트. 근거: spec/ui/sftp-snippets.md §1, spec/ui/shell.md §2.0.1.
import { useEffect, type PointerEvent as RPointerEvent } from 'react';
import '../styles/rightpanel.css';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { getBackend } from '../ipc/backend';
import { useTransfers, type TransferInfo } from '../state/transfers';
import { useCopyJobs, type CopyJob } from '../state/copyJobs';
import { SftpPanel } from './SftpPanel';
import { PreviewDialog } from './PreviewDialog';

const MIN_W = 260;
const MAX_W = 760;
const DEFAULT_W = 340;

export function clampWidth(w: number): number {
  return Math.max(MIN_W, Math.min(MAX_W, Math.round(w)));
}

/** 전송 이벤트(패널과 무관하게 항상 받는다)와 미리보기 대화상자를 한 번만 둔다. */
export function PanelHost() {
  useEffect(() => {
    const off = getBackend().on('transfer.update', (t: TransferInfo) => useTransfers.getState().upsert(t));
    // 끌어 놓기 복사 작업(file-panel.md §3)
    const offJob = getBackend().on('copyjob.update', (j: CopyJob) => useCopyJobs.getState().upsert(j));
    return () => {
      off();
      offJob();
    };
  }, []);
  return <PreviewDialog />;
}

/** 탭 그룹 오른쪽에 붙는 SFTP 패널. 그 그룹의 선택 탭에서 포커스된 칸의 서버를 보여 준다. */
export function SftpDock({ group }: { group: 0 | 1 }) {
  const setOpen = useUi((s) => s.setSftpOpen);
  const width = clampWidth(useApp((s) => s.store?.settings.rightPanelWidth ?? DEFAULT_W));
  const patchStore = useApp((s) => s.patchStore);

  const onResizeDown = (e: RPointerEvent<HTMLDivElement>) => {
    e.preventDefault();
    const panel = (e.currentTarget.parentElement as HTMLElement | null);
    const right = panel?.getBoundingClientRect().right ?? window.innerWidth;
    let latest = width;
    const move = (ev: PointerEvent) => {
      latest = clampWidth(right - ev.clientX);
      useApp.setState((s) => (s.store ? { store: { ...s.store, settings: { ...s.store.settings, rightPanelWidth: latest } } } : s));
    };
    const up = () => {
      window.removeEventListener('pointermove', move);
      window.removeEventListener('pointerup', up);
      const st = useApp.getState().store;
      // 끝나면 설정에 저장(spec §1).
      if (st) void patchStore({ settings: { ...st.settings, rightPanelWidth: latest } });
    };
    window.addEventListener('pointermove', move);
    window.addEventListener('pointerup', up);
  };

  return (
    <aside className="rpanel" style={{ width }} aria-label="SFTP 창">
      <div className="rpanel-resize" onPointerDown={onResizeDown} role="separator" aria-orientation="vertical" aria-label="SFTP 창 너비" />
      {/* 제목 줄(제목·도구 버튼·닫기)은 SftpPanel 이 그린다 */}
      <div className="rpanel-body"><SftpPanel group={group} onClose={() => setOpen(group, false)} /></div>
    </aside>
  );
}
