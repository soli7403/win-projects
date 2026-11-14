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
import { langTag, t, type MsgId } from '../i18n';
import { Rich } from '../i18n/Rich';

export const CRED_IN_URL = /[a-z][a-z0-9+.-]*:\/\/[^/\s@]+:[^/\s@]*@/i;

function fmt(t?: number): string {
  return t ? new Date(t).toLocaleString(langTag()) : '';
}

/** 상태별 안내 문구 키(D75). */
const MSG: Record<string, MsgId> = {
  unconfigured: 'sync.unconfigured',
  locked: 'sync.locked',
  nogit: 'sync.nogit',
  'vault-mismatch': 'sync.mismatch',
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
      pushToast({ kind: 'error', text: t('sync.credInUrl') });
      return;
    }
    setRunning(true);
    try {
      const r = await getBackend().invoke<SyncRunResult>('sync.run', [mode]);
      setLast(r);
      if (r.state === 'ok') {
        pushToast({ kind: 'ok', text: r.changed ? t('sync.doneChanged') : t('sync.done') });
      } else if (r.state !== 'vault-mismatch') {
        pushToast({ kind: 'error', text: r.message || (MSG[r.state] ? t(MSG[r.state]) : t('sync.failed')) });
      }
    } catch (e) {
      const text = errorMessage(e);
      setLast({ state: 'error', message: text });
      pushToast({ kind: 'error', text: text || t('sync.failed') });
    } finally {
      setRunning(false);
    }
  };

  const overwrite = async () => {
    const ok = await openConfirm({
      title: t('sync.overwrite'),
      body: t('sync.overwriteBody'),
      okText: t('drop.overwrite'),
      danger: true,
    });
    if (ok) void run('overwrite');
  };

  const mismatch = last?.state === 'vault-mismatch';
  const blocked = !draft.trim() || !vaultOpen || running;
  const errorText = last && last.state !== 'ok' && last.state !== 'vault-mismatch' ? last.message || (MSG[last.state] ? t(MSG[last.state]) : undefined) : undefined;

  return (
    <>
      <p className="set-section-title" style={{ lineHeight: 1.6, margin: 0, color: 'var(--fg-2)' }}>
        <Rich text={t('sync.intro')} />
      </p>

      <Field label={t('sync.repo')} help={t('sync.repoHelp')}>
        <div style={{ display: 'flex', gap: 8 }}>
          <input
            className="input mono"
            aria-label={t('sync.repo')}
            placeholder="https://github.com/me/dariterm-sync.git"
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onBlur={saveRepo}
          />
          <button className="btn" type="button" disabled={blocked} onClick={() => void run('')} style={{ flex: 'none' }}>
            <RefreshCw size={14} className={running ? 'spin' : undefined} /> {t('sync.now')}
          </button>
        </div>
      </Field>

      <Group>
        <ToggleRow
          title={t('sync.auto')}
          desc={t('sync.autoDesc')}
          checked={auto}
          onChange={setAuto}
        />
      </Group>

      <div className="set-note">
        {lastSync ? t('sync.last', { time: fmt(lastSync) }) : t('sync.never')}
      </div>
      {gitMissing && <div className="danger-note">{MSG.nogit}</div>}
      {!vaultOpen && !gitMissing && <div className="set-note">{MSG.locked}</div>}
      {errorText && <div className="danger-note">{errorText}</div>}

      {mismatch && (
        <Group>
          <div className="cred-row" style={{ flexDirection: 'column', alignItems: 'stretch', gap: 8 }}>
            <div className="cred-main">
              <div style={{ fontWeight: 700 }}>{t('sync.mismatch')}</div>
              <div className="cred-sub" style={{ fontSize: 12, lineHeight: 1.6 }}>
                {t('sync.mismatchHelp')}
              </div>
            </div>
            <div style={{ display: 'flex', gap: 8 }}>
              <button className="btn primary" type="button" disabled={blocked} onClick={() => void run('pull-vault')}>{t('sync.pull')}</button>
              <button className="btn danger" type="button" disabled={blocked} onClick={() => void overwrite()}>{t('sync.overwrite')}</button>
            </div>
          </div>
        </Group>
      )}
    </>
  );
}
