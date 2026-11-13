// 인증·확인 요청 대화상자. 근거: spec/ui/dialogs.md §4.1~§4.3. 닫기(X·Esc·바깥)는 취소로 본다.
import { useState, type FormEvent } from 'react';
import { KeyRound, ShieldAlert, ShieldQuestion, User } from 'lucide-react';
import { useRequests, type UiRequest } from '../state/requests';
import { useApp } from '../state/app';
import { Modal } from '../ui/Modal';
import { Field } from '../ui/Controls';

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
      <Modal title="처음 연결하는 서버입니다" icon={ShieldQuestion} iconColor="var(--accent)" size="sm" onClose={onCancel}
        foot={<><button className="btn" onClick={onCancel}>연결 취소</button><button className="btn primary" autoFocus onClick={() => onAnswer(true)}>믿고 연결</button></>}>
        <div style={{ lineHeight: 1.7 }}>
          [{req.sessionTitle}] <b>{req.hostPort}</b> 서버의 신원을 아직 확인하지 않았습니다. 아래 지문이 서버 관리자가 알려준 값과 같은지 확인하세요.
        </div>
        <Field label={`${req.keyType} 키 지문`}><div className="detail-box">{req.fingerprint}</div></Field>
      </Modal>
    );
  }
  return (
    <Modal title="경고: 서버의 호스트 키가 바뀌었습니다" icon={ShieldAlert} iconColor="var(--danger)" size="sm" onClose={onCancel}
      foot={<><button className="btn" onClick={onCancel}>연결 취소</button><button className="btn danger" onClick={() => onAnswer(true)}>위험을 알고 새 키로 연결</button></>}>
      <div className="notice danger">
        [{req.sessionTitle}] {req.hostPort}의 호스트 키가 이전에 저장한 키와 다릅니다. 서버를 재설치했거나 IP가 바뀐 경우가 아니라면 <b>중간자 공격</b>일 수 있습니다. 확실하지 않으면 연결하지 마세요.
      </div>
      <Field label={`${req.keyType} 키 지문`}><div className="detail-box">{req.fingerprint}</div></Field>
      {req.oldFingerprint && <Field label="이전에 저장된 지문"><div className="detail-box faint">{req.oldFingerprint}</div></Field>}
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
    <Modal title="비밀번호 입력" icon={KeyRound} size="sm" onClose={onCancel}
      foot={<><button className="btn" type="button" onClick={onCancel}>취소</button><button className="btn primary" type="submit" form="pw-form">확인</button></>}>
      <form id="pw-form" onSubmit={submit} style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div className="faint">[{req.sessionTitle}]</div>
        {req.failed && <div className="err-text">비밀번호가 틀렸거나 인증에 실패했습니다. 다시 입력하세요.</div>}
        <Field label={req.prompt || '비밀번호'}>
          <input className="input" type="password" autoFocus value={pw} onChange={(e) => setPw(e.target.value)} />
        </Field>
        {req.allowSave && (
          <label style={{ display: 'flex', gap: 8, alignItems: 'center', opacity: vaultOpen ? 1 : 0.5 }}>
            <input type="checkbox" disabled={!vaultOpen} checked={vaultOpen && save} onChange={(e) => setSave(e.target.checked)} />
            암호화해 저장하고 다음부터 자동 로그인{vaultOpen ? '' : ' (앱 잠김)'}
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
    <Modal title={req.name || '추가 인증'} icon={KeyRound} size="sm" onClose={onCancel}
      foot={<><button className="btn" type="button" onClick={onCancel}>취소</button><button className="btn primary" type="submit" form="kb-form">확인</button></>}>
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
    <Modal title={isUser ? '사용자 이름 입력' : '키 암호 입력'} icon={isUser ? User : KeyRound} size="sm" onClose={onCancel}
      foot={<><button className="btn" type="button" onClick={onCancel}>취소</button><button className="btn primary" type="submit" form="single-form">확인</button></>}>
      <form id="single-form" onSubmit={submit} style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div className="faint">[{req.sessionTitle}]</div>
        {!isUser && <div className="faint small">키 파일이 암호로 보호되어 있습니다</div>}
        <Field label={isUser ? '연결할 계정' : undefined}>
          <input className="input" type={isUser ? 'text' : 'password'} autoFocus value={val} onChange={(e) => setVal(e.target.value)} />
        </Field>
      </form>
    </Modal>
  );
}
