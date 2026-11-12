// 잠금 화면: 잠금 해제 / 최초 설정 / 복구 코드 표시 / 복구 코드로 재설정.
// 근거: spec/ui/lock.md 전체, spec/acceptance/01-lock-vault.md 의 L1~L10.
import { useState, type FormEvent } from 'react';
import { Check, Copy, KeyRound, Lock as LockIcon, ShieldCheck } from 'lucide-react';
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
        setError('비밀번호가 올바르지 않습니다');
        setPw('');
      } else {
        await refreshVault();
      }
    } catch (err) {
      // Q12: 거짓이 아니라 오류로 실패하면 입력창은 그대로 두고 오류를 붙여 보인다.
      setError(`잠금 해제에 실패했습니다: ${errorMessage(err)}`);
    } finally {
      setBusy(false);
    }
  };

  const unlockOs = async () => {
    setBusy(true);
    setError('');
    try {
      const ok = await backend().invoke<boolean>('vault.unlockOs');
      if (!ok) setError('Windows 자동 잠금해제에 실패했습니다');
      else await refreshVault();
    } catch {
      setError('Windows 자동 잠금해제에 실패했습니다');
    } finally {
      setBusy(false);
    }
  };

  const submitSetup = async (e: FormEvent) => {
    e.preventDefault();
    if (busy) return;
    if (pw.length < MIN_PW) return setError('8자 이상 입력하세요');
    if (pw !== pw2) return setError('비밀번호가 일치하지 않습니다');
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
      pushToast({ kind: 'ok', text: '복구 코드를 복사했습니다' });
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
    if (newPw.length < MIN_PW) return setError('새 비밀번호는 8자 이상이어야 합니다');
    if (newPw !== newPw2) return setError('새 비밀번호가 일치하지 않습니다');
    setBusy(true);
    setError('');
    try {
      const ok = await backend().invoke<boolean>('vault.recover', [normalizeRecoveryInput(code), newPw]);
      if (!ok) setError('복구 코드가 올바르지 않습니다');
      else await refreshVault();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  const score = passwordScore(pw);

  return (
    <div className={`lock-screen${overlay ? ' overlay-mode' : ''}`}>
      <div className="lock-card" role="region" aria-label="잠금 화면">
        <div className="center-row"><LogoMark size={40} /></div>

        {mode === 'unlock' && (
          <form key="unlock" className="lock-form" onSubmit={submitUnlock} style={formStyle}>
            <h1><LockIcon size={18} style={{ verticalAlign: '-3px', marginRight: 6 }} />TermGo 잠금</h1>
            <p className="desc">
              {overlay
                ? '세션은 계속 연결되어 있습니다. 마스터 비밀번호를 입력하면 다시 사용할 수 있습니다.'
                : '저장된 서버 계정을 열려면 마스터 비밀번호를 입력하세요.'}
            </p>
            <input
              className="input"
              type="password"
              placeholder="마스터 비밀번호"
              autoComplete="off"
              autoFocus
              value={pw}
              onChange={(e) => setPw(e.target.value)}
              aria-label="마스터 비밀번호"
            />
            <div className="err-text" role="alert">{error}</div>
            <div className="lock-actions">
              <button className="btn primary" type="submit" disabled={!pw || busy}>
                {busy ? '확인 중...' : '잠금 해제'}
              </button>
              {vault?.osUnlockEnabled && (
                <button className="btn" type="button" onClick={unlockOs} disabled={busy}>
                  <ShieldCheck size={15} />Windows 계정으로 잠금 해제
                </button>
              )}
            </div>
            <div className="center-row">
              <button className="btn link small-btn" type="button" onClick={() => go('recover')}>
                비밀번호를 잊으셨나요? 복구 코드로 재설정
              </button>
            </div>
          </form>
        )}

        {mode === 'setup' && (
          <form key="setup" onSubmit={submitSetup} style={formStyle}>
            <h1>TermGo 시작하기</h1>
            <p className="desc">
              서버 ID·비밀번호·키는 이 마스터 비밀번호로 AES-256 암호화되어 이 PC에만 저장됩니다. 마스터 비밀번호는 어디에도 전송되지 않습니다.
            </p>
            <input
              className="input"
              type="password"
              placeholder="마스터 비밀번호 (8자 이상)"
              autoComplete="new-password"
              autoFocus
              value={pw}
              onChange={(e) => setPw(e.target.value)}
              aria-label="마스터 비밀번호 (8자 이상)"
            />
            {pw && (
              <div>
                <div className="strength" aria-hidden="true">
                  <div style={{ width: strengthBarWidth(score), background: STRENGTH_COLORS[score] }} />
                </div>
                <div className="faint small" style={{ marginTop: 4 }}>강도: {STRENGTH_LABELS[score]}</div>
              </div>
            )}
            <input
              className="input"
              type="password"
              placeholder="비밀번호 확인"
              autoComplete="new-password"
              value={pw2}
              onChange={(e) => setPw2(e.target.value)}
              aria-label="비밀번호 확인"
            />
            <div className="err-text" role="alert">{error}</div>
            <button className="btn primary" type="submit" disabled={busy}>
              {busy ? '암호화 키 생성 중...' : '볼트 만들기'}
            </button>
          </form>
        )}

        {mode === 'recovery' && (
          <div key="recovery" style={formStyle}>
            <h1><KeyRound size={18} style={{ verticalAlign: '-3px', marginRight: 6 }} />복구 코드</h1>
            <p className="desc">
              마스터 비밀번호를 잊었을 때 이 코드로만 복구할 수 있습니다. 인쇄하거나 안전한 곳에 적어 두세요. 이 화면을 닫으면 다시 볼 수 없습니다.
            </p>
            <div className="recovery-box selectable">{formatRecoveryCode(recoveryCode)}</div>
            <div className="center-row">
              <button className="btn" type="button" onClick={copyCode}><Copy size={15} />복사</button>
            </div>
            <label className="toggle-row" style={{ cursor: 'pointer' }}>
              <span>복구 코드를 안전한 곳에 저장했습니다</span>
              <input type="checkbox" checked={saved} onChange={(e) => setSaved(e.target.checked)} />
            </label>
            <button className="btn primary" type="button" disabled={!saved} onClick={finishRecovery}>
              <Check size={15} />시작하기
            </button>
          </div>
        )}

        {mode === 'recover' && (
          <form key="recover" onSubmit={submitRecover} style={formStyle}>
            <h1>복구 코드로 재설정</h1>
            <p className="desc">
              볼트를 만들 때 받은 24자리 복구 코드와 새 마스터 비밀번호를 입력하세요. 저장된 계정은 그대로 유지됩니다.
            </p>
            <input
              className="input mono"
              placeholder="XXXX-XXXX-XXXX-XXXX-XXXX-XXXX"
              autoFocus
              value={code}
              onChange={(e) => setCode(e.target.value.toUpperCase())}
              aria-label="복구 코드"
            />
            <input
              className="input"
              type="password"
              placeholder="새 마스터 비밀번호"
              autoComplete="new-password"
              value={newPw}
              onChange={(e) => setNewPw(e.target.value)}
              aria-label="새 마스터 비밀번호"
            />
            <input
              className="input"
              type="password"
              placeholder="새 비밀번호 확인"
              autoComplete="new-password"
              value={newPw2}
              onChange={(e) => setNewPw2(e.target.value)}
              aria-label="새 비밀번호 확인"
            />
            <div className="err-text" role="alert">{error}</div>
            <button className="btn primary" type="submit" disabled={busy}>재설정</button>
            <div className="center-row">
              <button className="btn link small-btn" type="button" onClick={() => go('unlock')}>돌아가기</button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}

const formStyle = { display: 'flex', flexDirection: 'column' as const, gap: 14 };
