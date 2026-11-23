// 잠금 화면: 잠금 해제 / 최초 설정 / 복구 코드 표시 / 복구 코드로 재설정.
// 근거: spec/ui/lock.md 전체, spec/acceptance/01-lock-vault.md 의 L1~L10.
import { useState, type FormEvent } from 'react';
import { ArrowBigUpDash, Check, Copy, KeyRound, Lock as LockIcon, ShieldCheck } from 'lucide-react';
import { useCapsLock } from '../lib/useCapsLock';
import { getBackend, errorMessage } from '../ipc/backend';
import { useApp } from '../state/app';
import { LogoMark } from '../ui/Logo';
import { formatRecoveryCode, normalizeRecoveryInput } from '../lib/recovery';
import {
  passwordScore,
  STRENGTH_COLORS,
  STRENGTH_LABELS,
  strengthBarWidth,
} from '../lib/password';
import { t } from '../i18n';

type Mode = 'unlock' | 'setup' | 'recovery' | 'recover';

interface LockProps {
  /** 세션 화면 위에 덮는 오버레이 모드(자동 잠금 등). */
  overlay?: boolean;
}

const MIN_PW = 8;

export function Lock({ overlay = false }: LockProps) {
  const vault = useApp((s) => s.vault);
  const recoveryPending = useApp((s) => s.recoveryPending);
  const recoveryCode = useApp((s) => s.pendingRecoveryCode);
  const setRecoveryPending = useApp((s) => s.setRecoveryPending);
  const refreshVault = useApp((s) => s.refreshVault);
  const pushToast = useApp((s) => s.pushToast);

  const [mode, setMode] = useState<Mode>(
    recoveryPending ? 'recovery' : vault?.initialized ? 'unlock' : 'setup',
  );
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  // 입력값
  const [pw, setPw] = useState('');
  const [pw2, setPw2] = useState('');
  const [code, setCode] = useState('');
  const [newPw, setNewPw] = useState('');
  const [newPw2, setNewPw2] = useState('');
  const [saved, setSaved] = useState(false);

  const go = (next: Mode) => {
    setError('');
    setPw('');
    setPw2('');
    setCode('');
    setNewPw('');
    setNewPw2('');
    setMode(next);
  };

  const backend = () => getBackend();

  const submitUnlock = async (e: FormEvent) => {
    e.preventDefault();
    if (!pw || busy) return;
    setBusy(true);
    setError('');
    try {
      const ok = await backend().invoke<boolean>('vault.unlock', [pw]);
      if (!ok) {
        setError(t('lock.wrongPw'));
        setPw('');
      } else {
        await refreshVault();
      }
    } catch (err) {
      // Q12: 거짓이 아니라 오류로 실패하면 입력창은 그대로 두고 오류를 붙여 보인다.
      setError(t('lock.unlockFailed', { error: errorMessage(err) }));
    } finally {
      setBusy(false);
    }
  };

  const unlockOs = async () => {
    setBusy(true);
    setError('');
    try {
      const ok = await backend().invoke<boolean>('vault.unlockOs');
      if (!ok) setError(t('lock.osFailed'));
      else await refreshVault();
    } catch {
      setError(t('lock.osFailed'));
    } finally {
      setBusy(false);
    }
  };

  const submitSetup = async (e: FormEvent) => {
    e.preventDefault();
    if (busy) return;
    if (pw.length < MIN_PW) return setError(t('lock.min8'));
    if (pw !== pw2) return setError(t('lock.mismatch'));
    setBusy(true);
    setError('');
    try {
      const res = await backend().invoke<{ recoveryCode: string }>('vault.setup', [pw]);
      setRecoveryPending(true, res.recoveryCode);
      await refreshVault();
      go('recovery');
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  const copyCode = async () => {
    try {
      await backend().invoke('clip.writeText', [formatRecoveryCode(recoveryCode)]);
      pushToast({ kind: 'ok', text: t('lock.codeCopied') });
    } catch (err) {
      pushToast({ kind: 'error', text: errorMessage(err) });
    }
  };

  const finishRecovery = () => {
    setRecoveryPending(false, '');
  };

  const submitRecover = async (e: FormEvent) => {
    e.preventDefault();
    if (busy) return;
    if (newPw.length < MIN_PW) return setError(t('lock.newMin8'));
    if (newPw !== newPw2) return setError(t('sec.pwMismatch'));
    setBusy(true);
    setError('');
    try {
      const ok = await backend().invoke<boolean>('vault.recover', [normalizeRecoveryInput(code), newPw]);
      if (!ok) setError(t('lock.badCode'));
      else await refreshVault();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  const score = passwordScore(pw);
  // 비밀번호를 입력하는 화면(잠금 해제·처음 설정·복구)에서만 Caps Lock 을 살핀다 (D97).
  const capsOn = useCapsLock(mode !== 'recovery');
  const capsNote = capsOn && (
    <div className="caps-note" role="status"><ArrowBigUpDash size={14} />{t('lock.capsLock')}</div>
  );

  return (
    <div className={`lock-screen${overlay ? ' overlay-mode' : ''}`}>
      <div className="lock-card" role="region" aria-label={t('lock.screen')}>
        <div className="center-row"><LogoMark size={40} /></div>

        {mode === 'unlock' && (
          <form key="unlock" className="lock-form" onSubmit={submitUnlock} style={formStyle}>
            <h1><LockIcon size={18} style={{ verticalAlign: '-3px', marginRight: 6 }} />{t('lock.title')}</h1>
            <p className="desc">
              {overlay
                ? t('lock.overlayDesc')
                : t('lock.desc')}
            </p>
            <input
              className="input"
              type="password"
              placeholder={t('lock.pw')}
              autoComplete="off"
              autoFocus
              value={pw}
              onChange={(e) => setPw(e.target.value)}
              aria-label={t('lock.pw')}
            />
            {capsNote}
            <div className="err-text" role="alert">{error}</div>
            <div className="lock-actions">
              <button className="btn primary" type="submit" disabled={!pw || busy}>
                {busy ? t('lock.checking') : t('lock.open')}
              </button>
              {vault?.osUnlockEnabled && (
                <button className="btn" type="button" onClick={unlockOs} disabled={busy}>
                  <ShieldCheck size={15} />{t('lock.osUnlock')}
                </button>
              )}
            </div>
            <div className="center-row">
              <button className="btn link small-btn" type="button" onClick={() => go('recover')}>
                {t('lock.forgot')}
              </button>
            </div>
          </form>
        )}

        {mode === 'setup' && (
          <form key="setup" onSubmit={submitSetup} style={formStyle}>
            <h1>{t('setup.title')}</h1>
            <p className="desc">
              {t('setup.desc')}
            </p>
            <input
              className="input"
              type="password"
              placeholder={t('setup.pw')}
              autoComplete="new-password"
              autoFocus
              value={pw}
              onChange={(e) => setPw(e.target.value)}
              aria-label={t('setup.pw')}
            />
            {pw && (
              <div>
                <div className="strength" aria-hidden="true">
                  <div style={{ width: strengthBarWidth(score), background: STRENGTH_COLORS[score] }} />
                </div>
                <div className="faint small" style={{ marginTop: 4 }}>{t('setup.strength', { label: t(STRENGTH_LABELS[score]) })}</div>
              </div>
            )}
            <input
              className="input"
              type="password"
              placeholder={t('setup.pw2')}
              autoComplete="new-password"
              value={pw2}
              onChange={(e) => setPw2(e.target.value)}
              aria-label={t('setup.pw2')}
            />
            {capsNote}
            <div className="err-text" role="alert">{error}</div>
            <button className="btn primary" type="submit" disabled={busy}>
              {busy ? t('setup.creating') : t('setup.create')}
            </button>
          </form>
        )}

        {mode === 'recovery' && (
          <div key="recovery" style={formStyle}>
            <h1><KeyRound size={18} style={{ verticalAlign: '-3px', marginRight: 6 }} />{t('recovery.title')}</h1>
            <p className="desc">
              {t('recovery.desc')}
            </p>
            <div className="recovery-box selectable">{formatRecoveryCode(recoveryCode)}</div>
            <div className="center-row">
              <button className="btn" type="button" onClick={copyCode}><Copy size={15} />{t('common.copy')}</button>
            </div>
            <label className="toggle-row" style={{ cursor: 'pointer' }}>
              <span>{t('recovery.saved')}</span>
              <input type="checkbox" checked={saved} onChange={(e) => setSaved(e.target.checked)} />
            </label>
            <button className="btn primary" type="button" disabled={!saved} onClick={finishRecovery}>
              <Check size={15} />{t('recovery.start')}
            </button>
          </div>
        )}

        {mode === 'recover' && (
          <form key="recover" onSubmit={submitRecover} style={formStyle}>
            <h1>{t('recover.title')}</h1>
            <p className="desc">
              {t('recover.desc')}
            </p>
            <input
              className="input mono"
              placeholder="XXXX-XXXX-XXXX-XXXX-XXXX-XXXX"
              autoFocus
              value={code}
              onChange={(e) => setCode(e.target.value.toUpperCase())}
              aria-label={t('recovery.title')}
            />
            <input
              className="input"
              type="password"
              placeholder={t('recover.newPw')}
              autoComplete="new-password"
              value={newPw}
              onChange={(e) => setNewPw(e.target.value)}
              aria-label={t('recover.newPw')}
            />
            <input
              className="input"
              type="password"
              placeholder={t('sec.newPwAgain')}
              autoComplete="new-password"
              value={newPw2}
              onChange={(e) => setNewPw2(e.target.value)}
              aria-label={t('sec.newPwAgain')}
            />
            {capsNote}
            <div className="err-text" role="alert">{error}</div>
            <button className="btn primary" type="submit" disabled={busy}>{t('recover.submit')}</button>
            <div className="center-row">
              <button className="btn link small-btn" type="button" onClick={() => go('unlock')}>{t('common.back')}</button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}

const formStyle = { display: 'flex', flexDirection: 'column' as const, gap: 14 };
