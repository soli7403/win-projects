// HEX 보내기 창. 근거: spec/ui/dialogs.md §9, spec/behavior/serial-tools.md §2 (D163).
import { useState, type FormEvent } from 'react';
import { Modal } from '../ui/Modal';
import { Field } from '../ui/Controls';
import { getBackend, errorMessage } from '../ipc/backend';
import { t } from '../i18n';

/** "01 03 0a FF", "01030AFF", "0x01,0x03" 같은 입력의 바이트 수. 16진이 아니면 null (백엔드 ParseHex 와 같은 규칙). */
export function hexByteCount(text: string): number | null {
  const words = text.split(/[\s,:;]+/).filter(Boolean);
  let n = 0;
  for (let w of words) {
    w = w.replace(/^0x/i, '');
    if (w.length === 1) w = '0' + w;
    if (!w || w.length % 2 !== 0 || !/^[0-9a-f]+$/i.test(w)) return null;
    n += w.length / 2;
  }
  return n > 0 ? n : null;
}

export function HexSendDialog({ sessionId, onClose }: { sessionId: string; onClose: () => void }) {
  const [text, setText] = useState('');
  const [error, setError] = useState('');
  const count = hexByteCount(text);
  const send = async (e?: FormEvent) => {
    e?.preventDefault();
    if (count === null) return;
    try {
      await getBackend().invoke('session.sendHex', [sessionId, text]);
      setError('');
    } catch (err) {
      setError(errorMessage(err));
    }
  };
  return (
    <Modal
      title={t('serial.hexSendTitle')}
      size="sm"
      onClose={onClose}
      foot={
        <>
          <button className="btn" onClick={onClose}>{t('common.close')}</button>
          <button className="btn primary" disabled={count === null} onClick={() => void send()}>{t('serial.send')}</button>
        </>
      }
    >
      <form onSubmit={(e) => void send(e)} style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <Field label={t('serial.hexSendTitle')} help={t('serial.hexSendHelp')}>
          <input className="input mono" autoFocus aria-label={t('serial.hexSendTitle')} placeholder="01 03 00 00 00 0A C5 CD" value={text} onChange={(e) => setText(e.target.value)} />
        </Field>
        <div className="faint small">{text.trim() === '' ? '' : count === null ? t('be.hex.invalid') : t('serial.hexBytes', { count })}</div>
        {error && <div className="notice error">{error}</div>}
        <button type="submit" style={{ display: 'none' }} aria-hidden="true" tabIndex={-1} />
      </form>
    </Modal>
  );
}
