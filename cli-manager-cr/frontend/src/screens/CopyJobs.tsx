// 창 맨 아래 복사 작업 목록: 끌어 놓기 한 번 = 한 줄(폴더도 한 줄). 근거: spec/behavior/file-panel.md §3.
import { ArrowRight, Ban, CircleAlert, CircleCheck, Copy, LoaderCircle, X } from 'lucide-react';
import { useCopyJobs, isActiveJob, type CopyJob } from '../state/copyJobs';
import { formatBytes, transferPercent } from '../lib/sftpPath';
import { cancelCopy } from './fileDrag';

const PHASE: Record<CopyJob['phase'], string> = {
  scan: '크기 확인 중',
  download: '임시 폴더로 받는 중',
  upload: '대상으로 올리는 중',
  copy: '복사 중',
};

export function CopyJobs() {
  const jobs = useCopyJobs((s) => s.items);
  const remove = useCopyJobs((s) => s.remove);
  if (jobs.length === 0) return null;
  const running = jobs.filter(isActiveJob).length;
  const finished = jobs.length - running;
  return (
    <div className="copyjobs" aria-label="복사 작업">
      <div className="cj-head">
        <Copy size={12} />
        <span>
          {running ? `복사 ${running}개 진행 중` : '복사 완료'}
          {finished > 0 ? ` · 끝남 ${finished}개` : ''}
        </span>
        <span className="spacer" />
        {finished > 0 && (
          <button className="btn link" onClick={() => remove(jobs.filter((j) => !isActiveJob(j)).map((j) => j.id))}>
            끝난 항목 지우기
          </button>
        )}
      </div>
      <div className="cj-list">
        {jobs.map((j) => (
          <JobRow key={j.id} j={j} onRemove={() => remove([j.id])} />
        ))}
      </div>
    </div>
  );
}

function statusText(j: CopyJob, pct: number): string {
  // 서버 → 서버는 받기+올리기 합이므로 보이는 크기는 원본 기준(절반).
  const half = (n: number) => (j.relay ? Math.round(n / 2) : n);
  switch (j.state) {
    case 'queued':
      return '대기 중';
    case 'running':
      return `${PHASE[j.phase]}${j.files ? ` · ${j.filesDone}/${j.files}개` : ''} · ${formatBytes(half(j.bytes))}/${formatBytes(half(j.total))} · ${pct}%`;
    case 'done':
      return `완료 · ${j.files}개 · ${formatBytes(half(j.total))}`;
    case 'canceled':
      return '취소됨';
    default:
      return `실패: ${j.error ?? ''}`;
  }
}

function JobRow({ j, onRemove }: { j: CopyJob; onRemove: () => void }) {
  const pct = transferPercent(j.bytes, j.total, j.state);
  const name = j.count > 1 ? `${j.name} 외 ${j.count - 1}개` : j.name;
  const from = `${j.srcTitle || '?'}: ${j.srcDir}`;
  const to = `${j.dstTitle || '?'}: ${j.dstDir}`;
  const active = isActiveJob(j);
  const tip = [name, from, `→ ${to}`, j.relay ? '(이 PC 임시 폴더 경유)' : '', j.error ?? ''].filter(Boolean).join('\n');
  return (
    <div className={`cj-row ${j.state}`} title={tip}>
      <span className="cj-ic">
        {j.state === 'done' ? (
          <CircleCheck size={13} color="var(--ok)" aria-label="완료" />
        ) : j.state === 'error' ? (
          <CircleAlert size={13} color="var(--danger)" aria-label="실패" />
        ) : j.state === 'canceled' ? (
          <Ban size={13} aria-label="취소됨" />
        ) : (
          <LoaderCircle size={13} className="spin" aria-label="진행 중" />
        )}
      </span>
      <span className="cj-name">{name}</span>
      <span className="cj-path">{from}</span>
      <ArrowRight size={12} className="cj-arrow" />
      <span className="cj-path">{to}</span>
      <span className="cj-status">{statusText(j, pct)}</span>
      <span className="cj-bar">
        <span style={{ width: `${j.state === 'done' ? 100 : pct}%` }} />
      </span>
      {active ? (
        <button className="btn small-btn cj-cancel" title="취소" onClick={() => cancelCopy(j.id)}>
          취소
        </button>
      ) : (
        <button className="icon-btn small" title="목록에서 지우기" aria-label="목록에서 지우기" onClick={onRemove}>
          <X size={12} />
        </button>
      )}
    </div>
  );
}
