import { useEffect, useState } from 'react'
import { RefreshCw } from 'lucide-react'
import { useApp } from '../state'
import { api } from '../api'
import { Field, Toggle } from './ui'

/** 3.11 설정 → 동기화(git): 비공개 git 저장소로 서버 목록·폴더·스니펫·저장된 계정을 여러 PC 간에 맞춘다. */
export function SyncPanel(): JSX.Element {
  const settings = useApp((s) => s.settings)
  const vault = useApp((s) => s.vault)
  const [url, setUrl] = useState(settings.syncRepo ?? '')
  const [info, setInfo] = useState<{ configured: boolean; gitAvailable: boolean; last: { state: string; message?: string; lastSync?: number } } | null>(null)
  const [busy, setBusy] = useState(false)
  const st = useApp.getState

  const refresh = (): void => { api.sync.status().then(setInfo).catch(() => {}) }
  useEffect(() => {
    refresh()
    return api.on.sync((s) => setInfo((x) => (x ? { ...x, last: s } : x)))
  }, [])

  const state = info?.last?.state
  const run = async (mode = ''): Promise<void> => {
    setBusy(true)
    try {
      const r = await api.sync.run(mode)
      if (r.state === 'ok') st().toast('ok', r.changed ? '동기화했습니다 (다른 PC 의 변경을 받았습니다)' : '동기화했습니다')
      else if (r.state === 'error' || r.state === 'nogit' || r.state === 'locked') st().toast('error', r.message ?? '동기화하지 못했습니다')
    } finally { setBusy(false); refresh() }
  }
  const saveUrl = (): void => { st().setSettings({ syncRepo: url.trim() }); setTimeout(refresh, 300) }

  const lastText = info?.last?.lastSync ? new Date(info.last.lastSync).toLocaleString() : '아직 동기화한 적 없음'
  return (
    <>
      <p className="muted" style={{ margin: 0 }}>
        비공개 git 저장소에 <b>볼트 키로 암호화한 파일 하나</b>만 올립니다. 서버 주소·계정명을 포함해 어떤 정보도 평문으로 남기지 않습니다.
        다른 PC 에서는 같은 마스터 비밀번호로 그 파일을 엽니다. 인증은 이 PC 에 설치된 git 과 그 설정(SSH 키, 자격 증명 도우미)을 그대로 사용하며, 앱은 git 자격 증명을 저장하지 않습니다.
      </p>
      <Field label="저장소 주소" hint="예: git@github.com:me/termgo-sync.git — 주소에 비밀번호·토큰을 넣지 마세요">
        <div className="row" style={{ gap: 8 }}>
          <input className="input mono" style={{ flex: 1 }} value={url} onChange={(e) => setUrl(e.currentTarget.value)} onBlur={saveUrl} placeholder="https://github.com/me/termgo-sync.git" />
          <button className="btn" disabled={busy || !url.trim() || !vault.unlocked} onClick={() => { saveUrl(); void run() }}>
            <RefreshCw size={13} className={busy ? 'spin' : ''} />지금 동기화
          </button>
        </div>
      </Field>
      <div className="card">
        <Toggle label="자동 동기화" desc="볼트를 열 때, 데이터를 바꾼 약 20초 뒤, 그리고 10분마다 동기화합니다" value={settings.syncAuto} onChange={(v) => st().setSettings({ syncAuto: v })} />
      </div>
      <div className="muted">마지막 동기화: {lastText}</div>
      {info && !info.gitAvailable && <div className="err">이 PC 에 git 이 설치되어 있지 않습니다.</div>}
      {!vault.unlocked && <div className="muted">볼트를 열어야 동기화할 수 있습니다.</div>}
      {state === 'error' && <div className="err">{info?.last?.message}</div>}
      {state === 'vault-mismatch' && (
        <div className="card" style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          <b>저장소의 볼트가 이 PC 의 볼트와 다릅니다</b>
          <span className="muted">처음 연결하는 새 PC 라면 "저장소 볼트 받기"를 선택하세요. 이 PC 의 데이터를 저장소의 기준으로 삼으려면 "이 PC 볼트로 덮어쓰기"를 선택하세요 (저장소의 기존 내용이 대체됩니다).</span>
          <div className="row" style={{ gap: 8 }}>
            <button className="btn primary" disabled={busy} onClick={() => run('pull-vault')}>저장소 볼트 받기</button>
            <button className="btn danger" disabled={busy} onClick={async () => {
              if (await st().confirm({ title: '이 PC 볼트로 덮어쓰기', message: '저장소의 기존 동기화 내용이 이 PC 의 볼트와 데이터로 대체됩니다. 다른 PC 는 새 마스터 비밀번호로 다시 받아야 합니다.', okText: '덮어쓰기', danger: true })) void run('overwrite')
            }}>이 PC 볼트로 덮어쓰기</button>
          </div>
        </div>
      )}
    </>
  )
}
