// 설정 6 · 보안·계정. 근거: spec/ui/settings.md §6, spec/ipc.md §2 (볼트·계정), spec/acceptance G4·G5.
import { useEffect, useState } from 'react';
import { Trash2 } from 'lucide-react';
import { ToggleRow, Field } from '../ui/Controls';
import { getBackend, errorMessage } from '../ipc/backend';
import { useApp } from '../state/app';
import { useUi } from '../state/ui';
import { saveSettings } from '../state/settings';
import type { CredMeta, Host } from '../ipc/types';
import { NumberField, useSetting, Group } from './Settings';

export function SettingsSecurity() {
  const [autoLock] = useSetting('autoLockMinutes');
  const vault = useApp((s) => s.vault);
  const refreshVault = useApp((s) => s.refreshVault);
  const pushToast = useApp((s) => s.pushToast);
  const hosts = useApp((s) => s.store?.hosts ?? []);
  const confirm = useUi((s) => s.confirm);

  return (
    <>
      <Field label="자동 잠금 (분)" help="PC를 이 시간 동안 사용하지 않거나, Windows를 잠그거나 절전하면 보관함을 잠급니다. 열린 세션은 유지됩니다. 0 = 사용 안 함">
        <NumberField label="자동 잠금 (분)" value={autoLock} min={0} max={1440} fallback={0} onCommit={(v) => void saveSettings({ autoLockMinutes: v })} />
      </Field>

      {vault?.osUnlockAvailable && (
        <Group>
          <ToggleRow
            title="Windows 로그인으로 열기"
            desc="잠금 화면에서 마스터 비밀번호 대신 Windows 로그인(DPAPI)으로 열 수 있습니다"
            checked={!!vault.osUnlockEnabled}
            onChange={async (on) => {
              try {
                await getBackend().invoke('vault.setOsUnlock', [on]);
                await refreshVault();
              } catch (e) {
                pushToast({ kind: 'error', text: errorMessage(e) });
                await refreshVault();
              }
            }}
          />
        </Group>
      )}

      <div>
        <div className="set-section-title">마스터 비밀번호 변경</div>
        <ChangePassword />
      </div>

      <CredList hosts={hosts} confirm={confirm} />
    </>
  );
}

function ChangePassword() {
  const [cur, setCur] = useState('');
  const [next, setNext] = useState('');
  const [again, setAgain] = useState('');
  const [result, setResult] = useState<{ ok: boolean; text: string } | null>(null);

  const change = async () => {
    if (next.length < 8) return setResult({ ok: false, text: '새 비밀번호는 8자 이상' });
    if (next !== again) return setResult({ ok: false, text: '새 비밀번호가 일치하지 않습니다' });
    try {
      const ok = await getBackend().invoke<boolean>('vault.changePassword', [cur, next]);
      if (ok) {
        setCur(''); setNext(''); setAgain('');
        setResult({ ok: true, text: '변경했습니다' });
      } else {
        setCur(''); setNext(''); setAgain('');
        setResult({ ok: false, text: '현재 비밀번호가 틀립니다' });
      }
    } catch (e) {
      setResult({ ok: false, text: errorMessage(e) });
    }
  };

  return (
    <div className="set-stack">
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 8 }}>
        <input className="input" type="password" placeholder="현재 비밀번호" aria-label="현재 비밀번호" value={cur} onChange={(e) => setCur(e.target.value)} />
        <input className="input" type="password" placeholder="새 비밀번호" aria-label="새 비밀번호" value={next} onChange={(e) => setNext(e.target.value)} />
        <input className="input" type="password" placeholder="새 비밀번호 확인" aria-label="새 비밀번호 확인" value={again} onChange={(e) => setAgain(e.target.value)} />
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
        <button className="btn" type="button" onClick={() => void change()} disabled={!cur || !next || !again}>변경</button>
        {result && <span className={result.ok ? 'ok-note' : 'danger-note'}>{result.text}</span>}
      </div>
    </div>
  );
}

function CredList({ hosts, confirm }: { hosts: Host[]; confirm: (o: { title: string; body: string; okText?: string; danger?: boolean }) => Promise<boolean> }) {
  const [creds, setCreds] = useState<CredMeta[]>([]);
  const prompt = useUi((s) => s.prompt);
  const pushToast = useApp((s) => s.pushToast);
  const refreshStore = useApp((s) => s.refreshStore);

  const load = async () => {
    try {
      setCreds(await getBackend().invoke<CredMeta[]>('cred.list'));
    } catch (e) {
      pushToast({ kind: 'error', text: errorMessage(e) });
    }
  };
  useEffect(() => {
    void load();
    // 계정 변경 알림이 오면 목록을 다시 읽는다.
    const off = getBackend().on('store.changed', () => void load());
    return off;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const usersOf = (id: string) => hosts.filter((h) => h.credentialId === id);

  const rename = async (c: CredMeta) => {
    const name = await prompt({ title: '계정 이름', value: c.name, okText: '확인' });
    if (name === null || !name.trim()) return;
    try {
      await getBackend().invoke('cred.save', [{ id: c.id, name: name.trim(), kind: c.kind }]);
      await load();
    } catch (e) {
      pushToast({ kind: 'error', text: errorMessage(e) });
    }
  };

  const remove = async (c: CredMeta) => {
    const n = usersOf(c.id).length;
    const ok = await confirm({
      title: '계정 삭제',
      body: `"${c.name}" 계정을 보관함에서 삭제할까요?${n > 0 ? ` ${n}개 서버가 연결할 때 비밀번호를 다시 묻게 됩니다.` : ''}`,
      okText: '삭제',
      danger: true,
    });
    if (!ok) return;
    try {
      await getBackend().invoke('cred.remove', [c.id]);
      await refreshStore();
      await load();
    } catch (e) {
      pushToast({ kind: 'error', text: errorMessage(e) });
    }
  };

  return (
    <div>
      <div className="set-section-title">저장된 계정 ({creds.length})</div>
      <Group>
      {creds.length === 0 && <div className="cred-row"><span className="cred-sub">저장된 계정이 없습니다</span></div>}
      {creds.map((c) => {
        const users = usersOf(c.id);
        const shown = users.slice(0, 4).map((h) => h.alias).join(', ');
        const rest = users.length - 4;
        return (
          <div className="cred-row" key={c.id}>
            <div className="cred-main">
              <div>
                <b>{c.name}</b>
                {c.username && <span className="faint"> ({c.username})</span>}
                <span className="pill-soft">{c.kind === 'key' ? '키' : '비밀번호'}</span>
              </div>
              <div className="cred-sub">
                {users.length > 0 ? `사용 중: ${shown}${rest > 0 ? ` 외 ${rest}` : ''}` : '사용하는 서버 없음'}
              </div>
            </div>
            <button className="btn small-btn" type="button" onClick={() => void rename(c)}>이름</button>
            <button className="icon-btn" type="button" aria-label="삭제" title="삭제" onClick={() => void remove(c)}>
              <Trash2 size={15} />
            </button>
          </div>
        );
      })}
      </Group>
    </div>
  );
}
