import { ArrowRight, CheckCircle2, AlertCircle, Ban, Copy, X, Loader2 } from 'lucide-react'
import { useApp } from '../state'
import { api } from '../api'
import { fmtBytes } from './ui'
import type { CopyJob } from '@shared/types'

const PHASE: Record<CopyJob['phase'], string> = { scan: '크기 확인 중', download: '임시 폴더로 받는 중', upload: '대상으로 올리는 중', copy: '복사 중' }

const active = (j: CopyJob): boolean => j.state === 'queued' || j.state === 'running'

/** 화면 맨 아래 복사 작업 목록 — 끌어 놓기 한 번이 한 줄 (폴더도 한 줄) */
export function CopyJobs(): JSX.Element | null {
  const jobs = Object.values(useApp((s) => s.copyJobs)).sort((a, b) => a.startedAt - b.startedAt)
  if (!jobs.length) return null
  const running = jobs.filter(active).length
  const remove = (ids: string[]): void => useApp.setState((s) => ({ copyJobs: Object.fromEntries(Object.entries(s.copyJobs).filter(([k]) => !ids.includes(k))) }))
  return (
    <div className="copyjobs">
      <div className="cj-head">
        <Copy size={12} />
        <span>복사 {running ? `${running}개 진행 중` : '완료'}{jobs.length > running ? ` · 끝남 ${jobs.length - running}개` : ''}</span>
        <span style={{ flex: 1 }} />
        {jobs.length > running && <button className="link" onClick={() => remove(jobs.filter((j) => !active(j)).map((j) => j.id))}>끝난 항목 지우기</button>}
      </div>
      <div className="cj-list">
        {jobs.map((j) => <JobRow key={j.id} j={j} onClose={() => remove([j.id])} />)}
      </div>
    </div>
  )
}

function JobRow({ j, onClose }: { j: CopyJob; onClose: () => void }): JSX.Element {
  const pct = j.total ? Math.min(100, Math.floor((j.bytes / j.total) * 100)) : j.state === 'done' ? 100 : 0
  const name = j.count > 1 ? `${j.name} 외 ${j.count - 1}개` : j.name
  const from = `${j.srcTitle || '?'}: ${j.srcDir}`
  const to = `${j.dstTitle || '?'}: ${j.dstDir}`
  const status = j.state === 'queued' ? '대기 중'
    : j.state === 'running' ? `${PHASE[j.phase]}${j.files ? ` · ${j.filesDone}/${j.files}개` : ''} · ${fmtBytes(j.relay ? j.bytes / 2 : j.bytes)}/${fmtBytes(j.relay ? j.total / 2 : j.total)} · ${pct}%`
    : j.state === 'done' ? `완료 · ${j.files}개 · ${fmtBytes(j.relay ? j.total / 2 : j.total)}`
    : j.state === 'canceled' ? '취소됨'
    : `실패: ${j.error ?? ''}`
  return (
    <div className={`cj-row ${j.state}`} title={`${name}\n${from}\n→ ${to}${j.relay ? '\n(이 PC 임시 폴더 경유)' : ''}${j.error ? `\n${j.error}` : ''}`}>
      <span className="cj-ic">
        {j.state === 'done' ? <CheckCircle2 size={13} color="var(--ok)" /> : j.state === 'error' ? <AlertCircle size={13} color="var(--danger)" /> : j.state === 'canceled' ? <Ban size={13} /> : <Loader2 size={13} className="spin" />}
      </span>
      <span className="cj-name">{name}</span>
      <span className="cj-path">{from}</span>
      <ArrowRight size={12} className="cj-arrow" />
      <span className="cj-path">{to}</span>
      <span className="cj-status">{status}</span>
      <span className="cj-bar"><span style={{ width: `${pct}%` }} /></span>
      {active(j)
        ? <button className="ibtn sm" title="취소" onClick={() => api.sftp.cancelJob(j.id)}><X size={12} /></button>
        : <button className="ibtn sm" title="목록에서 지우기" onClick={onClose}><X size={12} /></button>}
    </div>
  )
}
