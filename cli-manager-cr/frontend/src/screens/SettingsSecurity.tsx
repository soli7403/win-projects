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
import { t } from '../i18n';
import { KeyGenerateDialog, PublicKeyDialog, type PublicKeyInfo } from './KeyDialogs';

export function SettingsSecurity() {
  const [autoLock] = useSetting('autoLockMinutes');
  const vault = useApp((s) => s.vault);
  const refreshVault = useApp((s) => s.refreshVault);
  const pushToast = useApp((s) => s.pushToast);
  const hosts = useApp((s) => s.store?.hosts ?? []);
  const confirm = useUi((s) => s.confirm);

  return (
    <>
      <Field label={t('sec.autoLock')} help={t('sec.autoLockHelp')}>
        <NumberField label={t('sec.autoLock')} value={autoLock} min={0} max={1440} fallback={0} onCommit={(v) => void saveSettings({ autoLockMinutes: v })} />
      </Field>

      {vault?.osUnlockAvailable && (
        <Group>
          <ToggleRow
            title={t('sec.osUnlock')}
            desc={t('sec.osUnlockDesc')}
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
        <div className="set-section-title">{t('sec.changePw')}</div>
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
    if (next.length < 8) return setResult({ ok: false, text: t('sec.pwMin') });
    if (next !== again) return setResult({ ok: false, text: t('sec.pwMismatch') });
    try {
      const ok = await getBackend().invoke<boolean>('vault.changePassword', [cur, next]);
      if (ok) {
        setCur(''); setNext(''); setAgain('');
        setResult({ ok: true, text: t('sec.changed') });
      } else {
        setCur(''); setNext(''); setAgain('');
        setResult({ ok: false, text: t('sec.wrongCurrent') });
      }
    } catch (e) {
      setResult({ ok: false, text: errorMessage(e) });
    }
  };

  return (
    <div className="set-stack">
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 8 }}>
        <input className="input" type="password" placeholder={t('sec.currentPw')} aria-label={t('sec.currentPw')} value={cur} onChange={(e) => setCur(e.target.value)} />
        <input className="input" type="password" placeholder={t('sec.newPw')} aria-label={t('sec.newPw')} value={next} onChange={(e) => setNext(e.target.value)} />
        <input className="input" type="password" placeholder={t('sec.newPwAgain')} aria-label={t('sec.newPwAgain')} value={again} onChange={(e) => setAgain(e.target.value)} />
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
        <button className="btn" type="button" onClick={() => void change()} disabled={!cur || !next || !again}>{t('sec.change')}</button>
        {result && <span className={result.ok ? 'ok-note' : 'danger-note'}>{result.text}</span>}
      </div>
    </div>
  );
}

function CredList({ hosts, confirm }: { hosts: Host[]; confirm: (o: { title: string; body: string; okText?: string; danger?: boolean }) => Promise<boolean> }) {
  const [creds, setCreds] = useState<CredMeta[]>([]);
  // SSH 키 만들기 창, 공개 키 창(만든 직후면 공개 키를 함께 넘긴다) (settings.md §6.3, D162).
  const [genOpen, setGenOpen] = useState(false);
  const [pubFor, setPubFor] = useState<{ id: string; name: string; info?: PublicKeyInfo } | null>(null);
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
    const name = await prompt({ title: t('cred.nameTitle'), value: c.name, okText: t('common.ok') });
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
      title: t('cred.deleteTitle'),
      body: t('cred.deleteBody', { name: c.name }) + (n > 0 ? ' ' + t('cred.deleteUsed', { count: n }) : ''),
      okText: t('common.delete'),
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
      <div className="set-section-title" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <span>{t('cred.saved', { count: creds.length })}</span>
        <button className="btn small-btn" type="button" onClick={() => setGenOpen(true)}>{t('key.generate')}</button>
      </div>
      <Group>
      {creds.length === 0 && <div className="cred-row"><span className="cred-sub">{t('cred.none')}</span></div>}
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
                <span className="pill-soft">{c.kind === 'key' ? t('cred.key') : t('cred.password')}</span>
              </div>
              <div className="cred-sub">
                {users.length > 0 ? t('cred.usedBy', { names: rest > 0 ? t('drop.andMore', { names: shown, count: rest }) : shown }) : t('cred.unused')}
              </div>
            </div>
            {c.kind === 'key' && <button className="btn small-btn" type="button" onClick={() => setPubFor({ id: c.id, name: c.name })}>{t('key.showPublic')}</button>}
            <button className="btn small-btn" type="button" onClick={() => void rename(c)}>{t('cred.rename')}</button>
            <button className="icon-btn" type="button" aria-label={t('common.delete')} title={t('common.delete')} onClick={() => void remove(c)}>
              <Trash2 size={15} />
            </button>
          </div>
        );
      })}
      </Group>
      {genOpen && (
        <KeyGenerateDialog
          onCancel={() => setGenOpen(false)}
          onDone={(k) => {
            setGenOpen(false);
            pushToast({ kind: 'ok', text: t('key.created', { name: k.cred.name }) });
            void load();
            setPubFor({ id: k.cred.id, name: k.cred.name, info: { publicKey: k.publicKey, fingerprint: k.fingerprint } });
          }}
        />
      )}
      {pubFor && <PublicKeyDialog credId={pubFor.id} name={pubFor.name} info={pubFor.info} onClose={() => setPubFor(null)} />}
    </div>
  );
}
