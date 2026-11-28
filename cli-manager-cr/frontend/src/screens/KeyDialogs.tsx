// SSH 키 만들기·공개 키 보기·서버에 등록 창. 근거: spec/ui/settings.md §6.3, spec/ui/dialogs.md §8, spec/behavior/ssh-keys.md (D162).
import { useEffect, useState, type FormEvent } from 'react';
import { Modal } from '../ui/Modal';
import { Field, Segmented, ToggleRow } from '../ui/Controls';
import { getBackend, errorMessage } from '../ipc/backend';
import { useApp } from '../state/app';
import type { CredMeta, Host } from '../ipc/types';
import { t } from '../i18n';

export interface PublicKeyInfo {
  publicKey: string;
  fingerprint: string;
}

export interface GeneratedKey extends PublicKeyInfo {
  cred: CredMeta;
}

/** 새 SSH 키 만들기. 만들면 보관함에 "키" 계정이 생기고 onDone 으로 결과(공개 키 포함)를 넘긴다. */
export function KeyGenerateDialog({ onCancel, onDone }: { onCancel: () => void; onDone: (key: GeneratedKey) => void }) {
  const [name, setName] = useState('');
  const [type, setType] = useState<'ed25519' | 'rsa'>('ed25519');
  const [pass, setPass] = useState('');
  const [again, setAgain] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const mismatch = pass !== again;
  const canCreate = !!name.trim() && !mismatch && !busy;
  const submit = async (e?: FormEvent) => {
    e?.preventDefault();
    if (!canCreate) return;
    setBusy(true);
    setError('');
    try {
      const key = await getBackend().invoke<GeneratedKey>('key.generate', [{ name: name.trim(), type, ...(type === 'rsa' ? { bits: 4096 } : {}), ...(pass ? { passphrase: pass } : {}) }]);
      onDone(key);
    } catch (err) {
      setError(errorMessage(err));
      setBusy(false);
    }
  };
  return (
    <Modal
      title={t('key.genTitle')}
      size="sm"
      onClose={onCancel}
      foot={
        <>
          <button className="btn" onClick={onCancel}>{t('common.cancel')}</button>
          <button className="btn primary" disabled={!canCreate} onClick={() => void submit()}>{busy ? t('key.creating') : t('key.create')}</button>
        </>
      }
    >
      <form onSubmit={(e) => void submit(e)} style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <Field label={t('key.name')}>
          <input className="input" autoFocus aria-label={t('key.name')} placeholder={t('key.namePh')} value={name} onChange={(e) => setName(e.target.value)} />
        </Field>
        <Field label={t('key.type')}>
          <Segmented label={t('key.type')} value={type} onChange={setType} options={[{ value: 'ed25519', label: t('key.typeEd') }, { value: 'rsa', label: t('key.typeRsa') }]} />
        </Field>
        <Field label={t('key.passphrase')} help={t('key.passphraseHelp')}>
          <input className="input" type="password" aria-label={t('key.passphrase')} value={pass} onChange={(e) => setPass(e.target.value)} />
        </Field>
        {pass && (
          <Field label={t('key.passphraseAgain')}>
            <input className="input" type="password" aria-label={t('key.passphraseAgain')} value={again} onChange={(e) => setAgain(e.target.value)} />
          </Field>
        )}
        {pass && mismatch && again && <div className="notice warn">{t('key.mismatch')}</div>}
        {error && <div className="notice error">{error}</div>}
        <button type="submit" style={{ display: 'none' }} aria-hidden="true" tabIndex={-1} />
      </form>
    </Modal>
  );
}

/** 공개 키 한 줄과 지문을 보이고 복사한다. info 가 없으면 credId 로 읽어 온다. */
export function PublicKeyDialog({ name, credId, info, onClose }: { name: string; credId: string; info?: PublicKeyInfo; onClose: () => void }) {
  const [pub, setPub] = useState<PublicKeyInfo | null>(info ?? null);
  const [error, setError] = useState('');
  const pushToast = useApp((s) => s.pushToast);
  useEffect(() => {
    if (info) return;
    getBackend().invoke<PublicKeyInfo>('key.publicKey', [credId]).then(setPub, (e) => setError(errorMessage(e)));
  }, [credId, info]);
  const copy = async () => {
    if (!pub) return;
    try {
      await getBackend().invoke('clip.writeText', [pub.publicKey]);
      pushToast({ kind: 'ok', text: t('key.copied') });
    } catch (e) {
      pushToast({ kind: 'error', text: errorMessage(e) });
    }
  };
  return (
    <Modal
      title={t('key.publicTitle', { name })}
      onClose={onClose}
      foot={
        <>
          <button className="btn" onClick={onClose}>{t('common.close')}</button>
          <button className="btn primary" disabled={!pub} onClick={() => void copy()}>{t('common.copy')}</button>
        </>
      }
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div className="faint small">{t('key.publicHelp')}</div>
        {pub && (
          <>
            <textarea className="input mono selectable" readOnly rows={4} style={{ height: 'auto', padding: 8, wordBreak: 'break-all' }} aria-label={t('key.showPublic')} value={pub.publicKey} onFocus={(e) => e.currentTarget.select()} />
            <Field label={t('key.fingerprint')}>
              <code className="mono selectable">{pub.fingerprint}</code>
            </Field>
          </>
        )}
        {error && <div className="notice error">{error}</div>}
      </div>
    </Modal>
  );
}

/** 열린 SSH 세션으로 공개 키를 서버에 등록한다. 저장된 서버의 세션이면 "다음부터 이 키로 로그인" 을 고를 수 있다. */
export function KeyInstallDialog({ sessionId, target, hostId, onClose }: { sessionId: string; target: string; hostId?: string | null; onClose: () => void }) {
  const [creds, setCreds] = useState<CredMeta[] | null>(null);
  const [credId, setCredId] = useState('');
  const [useKey, setUseKey] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [genOpen, setGenOpen] = useState(false);
  const pushToast = useApp((s) => s.pushToast);
  const store = useApp((s) => s.store);
  const patchStore = useApp((s) => s.patchStore);
  const host = hostId ? store?.hosts.find((h) => h.id === hostId) : undefined;
  const load = async (pick?: string) => {
    try {
      const list = (await getBackend().invoke<CredMeta[]>('cred.list')).filter((c) => c.kind === 'key');
      setCreds(list);
      setCredId((cur) => pick ?? (cur || list[0]?.id || ''));
    } catch (e) {
      setError(errorMessage(e));
    }
  };
  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  const install = async () => {
    if (!credId) return;
    setBusy(true);
    setError('');
    try {
      const res = await getBackend().invoke<{ status: 'added' | 'exists' }>('key.install', [sessionId, credId]);
      pushToast({ kind: 'ok', text: res.status === 'exists' ? t('key.installExists') : t('key.installed') });
      if (host && useKey && store) {
        const next: Host = { ...host, authType: 'key', credentialId: credId };
        await patchStore({ hosts: store.hosts.map((h) => (h.id === host.id ? next : h)) });
        pushToast({ kind: 'info', text: t('key.hostUpdated') });
      }
      onClose();
    } catch (e) {
      setError(errorMessage(e));
      setBusy(false);
    }
  };
  if (genOpen) {
    return <KeyGenerateDialog onCancel={() => setGenOpen(false)} onDone={(k) => { setGenOpen(false); void load(k.cred.id); }} />;
  }
  return (
    <Modal
      title={t('key.installTitle', { target })}
      size="sm"
      onClose={onClose}
      foot={
        <>
          <button className="btn" onClick={onClose}>{t('common.cancel')}</button>
          <button className="btn primary" disabled={!credId || busy} onClick={() => void install()}>{t('key.installOk')}</button>
        </>
      }
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div className="faint small">{t('key.installHelp')}</div>
        {creds && creds.length === 0 && <div className="notice info">{t('key.installNone')}</div>}
        {creds && creds.length > 0 && (
          <Field label={t('key.installPick')}>
            <select className="input" aria-label={t('key.installPick')} value={credId} onChange={(e) => setCredId(e.target.value)}>
              {creds.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
            </select>
          </Field>
        )}
        <div>
          <button type="button" className="btn small-btn" onClick={() => setGenOpen(true)}>{t('key.generate')}</button>
        </div>
        {host && creds && creds.length > 0 && <ToggleRow title={t('key.useForHost')} checked={useKey} onChange={setUseKey} />}
        {error && <div className="notice error">{error}</div>}
      </div>
    </Modal>
  );
}
