// 창 맨 아래 복사 작업 목록: 끌어 놓기 한 번 = 한 줄(폴더도 한 줄). 근거: spec/behavior/file-panel.md §3.
import { ArrowRight, Ban, CircleAlert, CircleCheck, Copy, LoaderCircle, X } from 'lucide-react';
import { useCopyJobs, isActiveJob, type CopyJob } from '../state/copyJobs';
import { formatBytes, transferPercent } from '../lib/sftpPath';
import { cancelCopy } from './fileDrag';
import { t, type MsgId } from '../i18n';

const PHASE: Record<CopyJob['phase'], MsgId> = {
  scan: 'cj.scan',
  download: 'cj.download',
  upload: 'cj.upload',
  copy: 'cj.copy',
};

export function CopyJobs() {
  const jobs = useCopyJobs((s) => s.items);
  const remove = useCopyJobs((s) => s.remove);
  if (jobs.length === 0) return null;
  const running = jobs.filter(isActiveJob).length;
  const finished = jobs.length - running;
  return (
    <div className="copyjobs" aria-label={t('cj.list')}>
      <div className="cj-head">
        <Copy size={12} />
        <span>
          {running ? t('cj.running', { count: running }) : t('cj.allDone')}
          {finished > 0 ? t('cj.finished', { count: finished }) : ''}
        </span>
        <span className="spacer" />
        {finished > 0 && (
          <button className="btn link" onClick={() => remove(jobs.filter((j) => !isActiveJob(j)).map((j) => j.id))}>
            {t('cj.clear')}
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
      return t('cj.waiting');
    case 'running':
      return `${t(PHASE[j.phase])}${j.files ? ` · ${t('cj.files', { done: j.filesDone, total: j.files })}` : ''} · ${formatBytes(half(j.bytes))}/${formatBytes(half(j.total))} · ${pct}%`;
    case 'done':
      return t('cj.done', { count: j.files, size: formatBytes(half(j.total)) });
    case 'canceled':
      return t('cj.canceled');
    default:
      return t('cj.failed', { error: j.error ?? '' });
  }
}

function JobRow({ j, onRemove }: { j: CopyJob; onRemove: () => void }) {
  const pct = transferPercent(j.bytes, j.total, j.state);
  const name = j.count > 1 ? t('drop.andMore', { names: j.name, count: j.count - 1 }) : j.name;
  const from = `${j.srcTitle || '?'}: ${j.srcDir}`;
  const to = `${j.dstTitle || '?'}: ${j.dstDir}`;
  const active = isActiveJob(j);
  const tip = [name, from, `→ ${to}`, j.relay ? t('cj.relay') : '', j.error ?? ''].filter(Boolean).join('\n');
  return (
    <div className={`cj-row ${j.state}`} title={tip}>
      <span className="cj-ic">
        {j.state === 'done' ? (
          <CircleCheck size={13} color="var(--ok)" aria-label={t('cj.doneAria')} />
        ) : j.state === 'error' ? (
          <CircleAlert size={13} color="var(--danger)" aria-label={t('cj.failedAria')} />
        ) : j.state === 'canceled' ? (
          <Ban size={13} aria-label={t('cj.canceled')} />
        ) : (
          <LoaderCircle size={13} className="spin" aria-label={t('cj.progressAria')} />
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
        <button className="btn small-btn cj-cancel" title={t('common.cancel')} onClick={() => cancelCopy(j.id)}>
          {t('common.cancel')}
        </button>
      ) : (
        <button className="icon-btn small" title={t('cj.clear')} aria-label={t('cj.clear')} onClick={onRemove}>
          <X size={12} />
        </button>
      )}
    </div>
  );
}
