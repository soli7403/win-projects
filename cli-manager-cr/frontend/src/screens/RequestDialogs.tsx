// 인증·확인 요청 대화상자. 근거: spec/ui/dialogs.md §4.1~§4.3. 닫기(X·Esc·바깥)는 취소로 본다.
import { useState, type FormEvent } from 'react';
import { KeyRound, ShieldAlert, ShieldQuestion, User } from 'lucide-react';
import { useRequests, type UiRequest } from '../state/requests';
import { useApp } from '../state/app';
import { Modal } from '../ui/Modal';
import { Field } from '../ui/Controls';
import { Rich } from '../i18n/Rich';
import { t } from '../i18n';

type Answer = (v: unknown) => void;

export function RequestDialogs() {
  const first = useRequests((s) => s.queue[0]);
  const answer = useRequests((s) => s.answer);
  if (!first) return null;
  // 호스트 키 요청의 취소는 거부(false), 나머지는 입력 없음(null).
  const cancel = () => answer(first.kind === 'hostkey' ? false : null);
  return <RequestView key={first.requestId} req={first} onAnswer={answer} onCancel={cancel} />;
}

function RequestView({ req, onAnswer, onCancel }: { req: UiRequest; onAnswer: Answer; onCancel: () => void }) {
  switch (req.kind) {
    case 'hostkey': return <HostKeyView req={req} onAnswer={onAnswer} onCancel={onCancel} />;
    case 'password': return <PasswordView req={req} onAnswer={onAnswer} onCancel={onCancel} />;
    case 'keyboard': return <KeyboardView req={req} onAnswer={onAnswer} onCancel={onCancel} />;
    default: return <SingleView req={req} onAnswer={onAnswer} onCancel={onCancel} />;
  }
}

function HostKeyView({ req, onAnswer, onCancel }: { req: Extract<UiRequest, { kind: 'hostkey' }>; onAnswer: Answer; onCancel: () => void }) {
  if (!req.changed) {
    return (
      <Modal title={t('hk.newTitle')} icon={ShieldQuestion} iconColor="var(--accent)" size="sm" onClose={onCancel}
        foot={<><button className="btn" onClick={onCancel}>{t('hk.cancel')}</button><button className="btn primary" autoFocus onClick={() => onAnswer(true)}>{t('hk.trust')}</button></>}>
        <div style={{ lineHeight: 1.7 }}>
          <Rich text={t('hk.newBody', { session: req.sessionTitle, host: req.hostPort })} />
        </div>
        <Field label={t('hk.fingerprint', { type: req.keyType })}><div className="detail-box">{req.fingerprint}</div></Field>
      </Modal>
    );
  }
  return (
    <Modal title={t('hk.changedTitle')} icon={ShieldAlert} iconColor="var(--danger)" size="sm" onClose={onCancel}
      foot={<><button className="btn" onClick={onCancel}>{t('hk.cancel')}</button><button className="btn danger" onClick={() => onAnswer(true)}>{t('hk.connectAnyway')}</button></>}>
      <div className="notice danger">
        <Rich text={t('hk.changedBody', { session: req.sessionTitle, host: req.hostPort })} />
      </div>
      <Field label={t('hk.fingerprint', { type: req.keyType })}><div className="detail-box">{req.fingerprint}</div></Field>
      {req.oldFingerprint && <Field label={t('hk.oldFingerprint')}><div className="detail-box faint">{req.oldFingerprint}</div></Field>}
    </Modal>
  );
}

function PasswordView({ req, onAnswer, onCancel }: { req: Extract<UiRequest, { kind: 'password' }>; onAnswer: Answer; onCancel: () => void }) {
  const [pw, setPw] = useState('');
  const vaultOpen = useApp((s) => !!s.vault?.unlocked);
  const [save, setSave] = useState(vaultOpen);
  const submit = (e: FormEvent) => {
    e.preventDefault();
    onAnswer({ password: pw, save: vaultOpen && save && req.allowSave });
  };
  return (
    <Modal title={t('auth.pwTitle')} icon={KeyRound} size="sm" onClose={onCancel}
      foot={<><button className="btn" type="button" onClick={onCancel}>{t('common.cancel')}</button><button className="btn primary" type="submit" form="pw-form">{t('common.ok')}</button></>}>
      <form id="pw-form" onSubmit={submit} style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div className="faint">[{req.sessionTitle}]</div>
        {req.failed && <div className="err-text">{t('auth.failed')}</div>}
        <Field label={req.prompt || t('cred.password')}>
          <input className="input" type="password" autoFocus value={pw} onChange={(e) => setPw(e.target.value)} />
        </Field>
        {req.allowSave && (
          <label style={{ display: 'flex', gap: 8, alignItems: 'center', opacity: vaultOpen ? 1 : 0.5 }}>
            <input type="checkbox" disabled={!vaultOpen} checked={vaultOpen && save} onChange={(e) => setSave(e.target.checked)} />
            {t('auth.save')}{vaultOpen ? '' : t('auth.saveLocked')}
          </label>
        )}
      </form>
    </Modal>
  );
}

function KeyboardView({ req, onAnswer, onCancel }: { req: Extract<UiRequest, { kind: 'keyboard' }>; onAnswer: Answer; onCancel: () => void }) {
  const [vals, setVals] = useState<string[]>(req.prompts.map(() => ''));
  const submit = (e: FormEvent) => {
    e.preventDefault();
    onAnswer(vals);
  };
  return (
    <Modal title={req.name || t('auth.kbTitle')} icon={KeyRound} size="sm" onClose={onCancel}
      foot={<><button className="btn" type="button" onClick={onCancel}>{t('common.cancel')}</button><button className="btn primary" type="submit" form="kb-form">{t('common.ok')}</button></>}>
      <form id="kb-form" onSubmit={submit} style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div className="faint">[{req.sessionTitle}]</div>
        {req.instructions && <div>{req.instructions}</div>}
        {req.prompts.map((p, i) => (
          <Field key={i} label={p.prompt}>
            <input className="input" type={p.echo ? 'text' : 'password'} autoFocus={i === 0} value={vals[i]} onChange={(e) => setVals(vals.map((v, j) => (j === i ? e.target.value : v)))} />
          </Field>
        ))}
      </form>
    </Modal>
  );
}

function SingleView({ req, onAnswer, onCancel }: { req: Extract<UiRequest, { kind: 'passphrase' | 'username' }>; onAnswer: Answer; onCancel: () => void }) {
  const [val, setVal] = useState('');
  const isUser = req.kind === 'username';
  const submit = (e: FormEvent) => {
    e.preventDefault();
    onAnswer(val);
  };
  return (
    <Modal title={isUser ? t('auth.userTitle') : t('auth.passphraseTitle')} icon={isUser ? User : KeyRound} size="sm" onClose={onCancel}
      foot={<><button className="btn" type="button" onClick={onCancel}>{t('common.cancel')}</button><button className="btn primary" type="submit" form="single-form">{t('common.ok')}</button></>}>
      <form id="single-form" onSubmit={submit} style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div className="faint">[{req.sessionTitle}]</div>
        {!isUser && <div className="faint small">{t('auth.keyProtected')}</div>}
        <Field label={isUser ? t('auth.account') : undefined}>
          <input className="input" type={isUser ? 'text' : 'password'} autoFocus value={val} onChange={(e) => setVal(e.target.value)} />
        </Field>
      </form>
    </Modal>
  );
}
