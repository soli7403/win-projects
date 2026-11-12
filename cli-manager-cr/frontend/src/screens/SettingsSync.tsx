// 설정 5 · 동기화 (git) 패널. 근거: spec/ui/dialogs.md §6, spec/data/sync-backup.md §5, spec/acceptance G·Y.
import { useEffect, useState } from 'react';
import { RefreshCw } from 'lucide-react';
import { ToggleRow, Field } from '../ui/Controls';
import { getBackend, errorMessage } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import type { SyncRunResult, SyncMode } from '../ipc/c1types';
import type { SyncStatus } from '../ipc/types';
import { useSetting, Group } from './Settings';

export const CRED_IN_URL = /[a-z][a-z0-9+.-]*:\/\/[^/\s@]+:[^/\s@]*@/i;

function fmt(t?: number): string {
  return t ? new Date(t).toLocaleString('ko-KR') : '';
}

const MSG: Record<string, string> = {
  unconfigured: '저장소 주소를 입력하세요.',
  locked: '볼트를 열어야 동기화할 수 있습니다.',
  nogit: '이 PC 에 git 이 설치되어 있지 않습니다.',
  'vault-mismatch': '저장소의 볼트가 이 PC 의 볼트와 다릅니다',
};

export function SettingsSync() {
  const [repo, setRepo] = useSetting('syncRepo');
  const [auto, setAuto] = useSetting('syncAuto');
  const [draft, setDraft] = useState(repo);
  const [prev, setPrev] = useState(repo);
  if (prev !== repo) {
    setPrev(repo);
    setDraft(repo);
  }
  const vaultOpen = useApp((s) => !!s.vault?.unlocked);
  const pushToast = useApp((s) => s.pushToast);
  const openConfirm = useUi((s) => s.confirm);
  const [running, setRunning] = useState(false);
  const [status, setStatus] = useState<SyncStatus | null>(null);
  const [last, setLast] = useState<SyncRunResult | null>(null);

  useEffect(() => {
    const b = getBackend();
    b.invoke<SyncStatus>('sync.status').then(setStatus).catch(() => setStatus(null));
    const off = b.on('sync.status', (r: SyncRunResult) => setLast(r));
    return off;
  }, []);

  const lastSync = last?.lastSync ?? status?.last?.lastSync ?? 0;
  const gitMissing = status ? !status.gitAvailable : false;

  const saveRepo = () => {
    const v = draft.trim();
    if (v !== repo) setRepo(v);
  };

  const run = async (mode: SyncMode) => {
    const url = draft.trim();
    if (url !== repo) setRepo(url);
    if (CRED_IN_URL.test(url)) {
      pushToast({ kind: 'error', text: '저장소 주소에 비밀번호·토큰을 넣지 마세요 — 이 PC 의 git 자격 증명 도우미/SSH 키를 사용합니다' });
      return;
    }
    setRunning(true);
    try {
      const r = await getBackend().invoke<SyncRunResult>('sync.run', [mode]);
      setLast(r);
      if (r.state === 'ok') {
        pushToast({ kind: 'ok', text: r.changed ? '동기화했습니다 (다른 PC 의 변경을 받았습니다)' : '동기화했습니다' });
      } else if (r.state !== 'vault-mismatch') {
        pushToast({ kind: 'error', text: r.message || MSG[r.state] || '동기화하지 못했습니다' });
      }
    } catch (e) {
      const text = errorMessage(e);
      setLast({ state: 'error', message: text });
      pushToast({ kind: 'error', text: text || '동기화하지 못했습니다' });
    } finally {
      setRunning(false);
    }
  };

  const overwrite = async () => {
    const ok = await openConfirm({
      title: '이 PC 볼트로 덮어쓰기',
      body: '저장소의 기존 동기화 내용이 이 PC 의 볼트와 데이터로 대체됩니다. 다른 PC 는 새 마스터 비밀번호로 다시 받아야 합니다.',
      okText: '덮어쓰기',
      danger: true,
    });
    if (ok) void run('overwrite');
  };

  const mismatch = last?.state === 'vault-mismatch';
  const blocked = !draft.trim() || !vaultOpen || running;
  const errorText = last && last.state !== 'ok' && last.state !== 'vault-mismatch' ? last.message || MSG[last.state] : undefined;

  return (
    <>
      <p className="set-section-title" style={{ lineHeight: 1.6, margin: 0, color: 'var(--fg-2)' }}>
        비공개 git 저장소에 <b>볼트 키로 암호화한 파일 하나</b>만 올립니다. 서버 주소·계정명을 포함해 어떤 정보도 평문으로 남기지 않습니다. 다른 PC 에서는 같은 마스터 비밀번호로 그 파일을 엽니다. 인증은 이 PC 에 설치된 git 과 그 설정(SSH 키, 자격 증명 도우미)을 그대로 사용하며, 앱은 git 자격 증명을 저장하지 않습니다.
      </p>

      <Field label="저장소 주소" help="예: git@github.com:me/termgo-sync.git — 주소에 비밀번호·토큰을 넣지 마세요">
        <div style={{ display: 'flex', gap: 8 }}>
          <input
            className="input mono"
            aria-label="저장소 주소"
            placeholder="https://github.com/me/termgo-sync.git"
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onBlur={saveRepo}
          />
          <button className="btn" type="button" disabled={blocked} onClick={() => void run('')} style={{ flex: 'none' }}>
            <RefreshCw size={14} className={running ? 'spin' : undefined} /> 지금 동기화
          </button>
        </div>
      </Field>

      <Group>
        <ToggleRow
          title="자동 동기화"
          desc="볼트를 열 때, 데이터를 바꾼 약 20초 뒤, 그리고 10분마다 동기화합니다"
          checked={auto}
          onChange={setAuto}
        />
      </Group>

      <div className="set-note">
        {lastSync ? `마지막 동기화: ${fmt(lastSync)}` : '마지막 동기화: 아직 동기화한 적 없음'}
      </div>
      {gitMissing && <div className="danger-note">{MSG.nogit}</div>}
      {!vaultOpen && !gitMissing && <div className="set-note">{MSG.locked}</div>}
      {errorText && <div className="danger-note">{errorText}</div>}

      {mismatch && (
        <Group>
          <div className="cred-row" style={{ flexDirection: 'column', alignItems: 'stretch', gap: 8 }}>
            <div className="cred-main">
              <div style={{ fontWeight: 700 }}>저장소의 볼트가 이 PC 의 볼트와 다릅니다</div>
              <div className="cred-sub" style={{ fontSize: 12, lineHeight: 1.6 }}>
                처음 연결하는 새 PC 라면 "저장소 볼트 받기"를 선택하세요. 이 PC 의 데이터를 저장소의 기준으로 삼으려면 "이 PC 볼트로 덮어쓰기"를 선택하세요 (저장소의 기존 내용이 대체됩니다).
              </div>
            </div>
            <div style={{ display: 'flex', gap: 8 }}>
              <button className="btn primary" type="button" disabled={blocked} onClick={() => void run('pull-vault')}>저장소 볼트 받기</button>
              <button className="btn danger" type="button" disabled={blocked} onClick={() => void overwrite()}>이 PC 볼트로 덮어쓰기</button>
            </div>
          </div>
        </Group>
      )}
    </>
  );
}
