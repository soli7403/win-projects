// 매크로 편집 창: 이름과 단계 표(기다릴 글자 · 보낼 글자 · Enter). 근거: spec/ui/sftp-snippets.md §4.4 (D141).
import { useState } from 'react';
import { ArrowDown, ArrowUp, Trash2 } from 'lucide-react';
import { Modal } from '../ui/Modal';
import { Field } from '../ui/Controls';
import type { MacroStep, Snippet } from '../ipc/types';
import { cleanSteps, MACRO_EXPECT_TIMEOUT_MS, MACRO_QUIET_MS, MACRO_STEP_MAX, macroBody, newMacroStep } from '../lib/macro';
import { moveStep } from '../lib/loginScript';
import { t } from '../i18n';

export function MacroEditor({ initial, isNew, recorded, onCancel, onSave }: {
  initial: Snippet;
  isNew: boolean;
  /** 방금 녹화한 단계로 연 것인지(안내 문구가 달라진다). */
  recorded?: boolean;
  onCancel: () => void;
  onSave: (s: Snippet) => void;
}) {
  const [name, setName] = useState(initial.name);
  const [rows, setRows] = useState<MacroStep[]>(() => (initial.steps?.length ? initial.steps : [newMacroStep()]));
  const patch = (id: string, p: Partial<MacroStep>) => setRows((cur) => cur.map((r) => (r.id === id ? { ...r, ...p } : r)));
  const steps = cleanSteps(rows);
  const canSave = steps.length > 0;
  const submit = () => {
    if (!canSave) return;
    onSave({ id: initial.id, name: name.trim(), body: macroBody(steps), sendEnter: true, steps });
  };
  return (
    <Modal
      title={isNew ? t('macro.new') : t('macro.edit')}
      size="md"
      onClose={onCancel}
      dismissOnOverlay={false}
      foot={
        <>
          <button className="btn" onClick={onCancel}>{t('common.cancel')}</button>
          <button className="btn primary" onClick={submit} disabled={!canSave}>{t('common.save')}</button>
        </>
      }
    >
      {recorded && <div className="notice warn">{t('macro.recordedNote')}</div>}
      <Field label={t('snip.name')}>
        <input className="input" autoFocus placeholder={t('macro.namePlaceholder')} value={name} onChange={(e) => setName(e.target.value)} />
      </Field>
      <Field
        label={t('macro.steps')}
        help={t('macro.stepsHelp', {
          quiet: MACRO_QUIET_MS / 1000,
          sec: Math.round(MACRO_EXPECT_TIMEOUT_MS / 1000),
          var: '{{host}}',
          ctrl: '\\x03',
        })}
      >
        <table className="fwd-table" style={{ width: '100%', borderCollapse: 'separate', borderSpacing: '0 6px' }}>
          <thead>
            <tr className="faint small" style={{ textAlign: 'left' }}>
              <th style={{ width: 22 }}>#</th>
              <th style={{ width: '32%' }}>{t('macro.expect')}</th>
              <th>{t('macro.send')}</th>
              <th style={{ width: 52 }}>Enter</th>
              <th style={{ width: 96 }} />
            </tr>
          </thead>
          <tbody>
            {rows.map((r, i) => (
              <tr key={r.id}>
                <td className="faint small">{i + 1}</td>
                <td>
                  <input className="input mono" aria-label={t('macro.expectN', { n: i + 1 })} placeholder={t('macro.expectPlaceholder')} value={r.expect} onChange={(e) => patch(r.id, { expect: e.target.value })} />
                </td>
                <td>
                  <input className="input mono" aria-label={t('macro.sendN', { n: i + 1 })} value={r.send} onChange={(e) => patch(r.id, { send: e.target.value })} />
                </td>
                <td style={{ textAlign: 'center' }}>
                  <input type="checkbox" aria-label={t('macro.enterN', { n: i + 1 })} checked={r.enter} onChange={(e) => patch(r.id, { enter: e.target.checked })} />
                </td>
                <td style={{ whiteSpace: 'nowrap' }}>
                  <button type="button" className="icon-btn" aria-label={t('login.up')} title={t('login.up')} disabled={i === 0} onClick={() => setRows(moveStep(rows, i, -1))}>
                    <ArrowUp size={14} />
                  </button>
                  <button type="button" className="icon-btn" aria-label={t('login.down')} title={t('login.down')} disabled={i === rows.length - 1} onClick={() => setRows(moveStep(rows, i, 1))}>
                    <ArrowDown size={14} />
                  </button>
                  <button type="button" className="icon-btn" aria-label={t('common.delete')} title={t('common.delete')} onClick={() => setRows(rows.filter((x) => x.id !== r.id))}>
                    <Trash2 size={14} />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        <div>
          <button type="button" className="btn small-btn" disabled={rows.length >= MACRO_STEP_MAX} onClick={() => setRows([...rows, newMacroStep()])}>
            {t('login.add')}
          </button>
        </div>
      </Field>
    </Modal>
  );
}
