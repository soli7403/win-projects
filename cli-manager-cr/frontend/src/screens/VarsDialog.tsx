// 명령·매크로를 실행하기 전에 변수 값을 묻는 창. 근거: spec/ui/sftp-snippets.md §4.3 (D138).
import { useState, type FormEvent } from 'react';
import { Modal } from '../ui/Modal';
import { Field } from '../ui/Controls';
import type { SnippetVar } from '../lib/snippetVars';
import { t } from '../i18n';

export function VarsDialog({ title, vars, initial, preview, onCancel, onRun }: {
  title: string;
  vars: SnippetVar[];
  initial: Record<string, string>;
  /** 지금 값으로 채운 명령(미리 보기). */
  preview: (values: Record<string, string>) => string;
  onCancel: () => void;
  onRun: (values: Record<string, string>) => void;
}) {
  const [values, setValues] = useState<Record<string, string>>(initial);
  const submit = (e?: FormEvent) => {
    e?.preventDefault();
    onRun(values);
  };
  return (
    <Modal
      title={title}
      size="sm"
      onClose={onCancel}
      foot={
        <>
          <button className="btn" onClick={onCancel}>{t('common.cancel')}</button>
          <button className="btn primary" onClick={() => submit()}>{t('vars.run')}</button>
        </>
      }
    >
      <form onSubmit={submit} style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        {vars.map((v, i) => (
          <Field key={v.name} label={v.name}>
            <input
              className="input mono"
              autoFocus={i === 0}
              aria-label={v.name}
              placeholder={v.def}
              value={values[v.name] ?? ''}
              onChange={(e) => setValues((cur) => ({ ...cur, [v.name]: e.target.value }))}
            />
          </Field>
        ))}
        {/* Enter 로 실행되게 보이지 않는 제출 버튼을 둔다 */}
        <button type="submit" style={{ display: 'none' }} aria-hidden="true" tabIndex={-1} />
      </form>
      <Field label={t('vars.preview')}>
        <pre className="vars-preview mono selectable">{preview(values)}</pre>
      </Field>
    </Modal>
  );
}
