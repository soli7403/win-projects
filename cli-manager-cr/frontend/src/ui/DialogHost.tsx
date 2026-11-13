// 확인·입력 대화상자 호스트(큐에서 맨 앞 하나를 보여 준다). 근거: spec/ui/dialogs.md §1.1·§1.2.
import { useState, type FormEvent } from 'react';
import { useUi, type AskOptions, type ConfirmOptions, type PromptOptions } from '../state/ui';
import { Modal } from './Modal';

function ConfirmView({ opts }: { opts: ConfirmOptions }) {
  const settle = useUi((s) => s.settleDialog);
  const cancel = () => settle(false);
  const ok = () => settle(true);
  return (
    <Modal
      title={opts.title}
      size="sm"
      onClose={cancel}
      foot={
        <>
          <button className="btn" onClick={cancel} autoFocus={!!opts.danger}>취소</button>
          <button className={opts.danger ? 'btn danger' : 'btn primary'} onClick={ok} autoFocus={!opts.danger}>
            {opts.okText ?? '확인'}
          </button>
        </>
      }
    >
      <div className="selectable" style={{ lineHeight: 1.7 }}>{opts.body}</div>
      {opts.detail && <div className="detail-box">{opts.detail}</div>}
    </Modal>
  );
}

/** 세 갈래 질문: [취소](X·Esc·바깥 클릭과 같음, 기본 포커스) · [아니오] · [확인]. */
function AskView({ opts }: { opts: AskOptions }) {
  const settle = useUi((s) => s.settleDialog);
  const cancel = () => settle(null);
  return (
    <Modal
      title={opts.title}
      size="sm"
      onClose={cancel}
      foot={
        <>
          <button className="btn" onClick={cancel} autoFocus>취소</button>
          <button className="btn" onClick={() => settle('no')}>{opts.noText}</button>
          <button className={opts.danger ? 'btn danger' : 'btn primary'} onClick={() => settle('yes')}>
            {opts.okText ?? '확인'}
          </button>
        </>
      }
    >
      <div className="selectable" style={{ lineHeight: 1.7 }}>{opts.body}</div>
      {opts.detail && <div className="detail-box">{opts.detail}</div>}
    </Modal>
  );
}

function PromptView({ opts }: { opts: PromptOptions }) {
  const settle = useUi((s) => s.settleDialog);
  const [value, setValue] = useState(opts.value ?? '');
  const cancel = () => settle(null);
  const submit = (e: FormEvent) => {
    e.preventDefault();
    settle(value);
  };
  return (
    <Modal
      title={opts.title}
      size="sm"
      onClose={cancel}
      foot={
        <>
          <button className="btn" type="button" onClick={cancel}>취소</button>
          <button className="btn primary" type="submit" form="prompt-form">{opts.okText ?? '확인'}</button>
        </>
      }
    >
      <form id="prompt-form" onSubmit={submit} className="field">
        {opts.label && <label htmlFor="prompt-input">{opts.label}</label>}
        <input
          id="prompt-input"
          className="input"
          value={value}
          autoFocus
          onFocus={(e) => e.currentTarget.select()}
          onChange={(e) => setValue(e.target.value)}
        />
      </form>
    </Modal>
  );
}

export function DialogHost() {
  const first = useUi((s) => s.dialogs[0]);
  if (!first) return null;
  if (first.kind === 'confirm') return <ConfirmView opts={first.opts} />;
  if (first.kind === 'ask') return <AskView opts={first.opts} />;
  return <PromptView opts={first.opts} />;
}
