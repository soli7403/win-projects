// 설정 1 · 화면의 "키워드 강조" 부분. 근거: spec/ui/settings.md §1 키워드 강조, spec/behavior/terminal.md §9 (D113).
import { useState } from 'react';
import { Trash2 } from 'lucide-react';
import type { HighlightRule } from '../ipc/types';
import { Field, Switch } from '../ui/Controls';
import { HIGHLIGHT_COLORS, HIGHLIGHT_MAX_RULES, isValidPattern, newHighlightRule } from '../lib/highlight';
import { useApp } from '../state/app';
import { saveSettings } from '../state/settings';
import { t, type MsgId } from '../i18n';

/** 작은 체크 상자 + 글자. */
function Check({ label, checked, onChange }: { label: string; checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <label className="small" style={{ display: 'inline-flex', alignItems: 'center', gap: 4, whiteSpace: 'nowrap' }}>
      <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} />
      {label}
    </label>
  );
}

export function SettingsHighlight() {
  const saved = useApp((s) => s.store?.settings.highlightRules);
  const [rows, setRows] = useState<HighlightRule[]>(() => (Array.isArray(saved) ? saved : []));
  const [prev, setPrev] = useState(saved);
  // 밖에서 바뀌면(동기화·백업 가져오기) 화면을 맞춘다. 이 화면이 저장한 것은 같은 배열이라 다시 맞추지 않는다.
  if (prev !== saved) {
    setPrev(saved);
    setRows(Array.isArray(saved) ? saved : []);
  }

  /** 화면을 바꾸고, save 면 곧바로 저장한다(패턴은 입력 칸을 벗어날 때 저장). */
  const apply = (next: HighlightRule[], save: boolean) => {
    setRows(next);
    if (save) void saveSettings({ highlightRules: next });
  };
  const patch = (id: string, p: Partial<HighlightRule>, save = true) => apply(rows.map((r) => (r.id === id ? { ...r, ...p } : r)), save);

  return (
    <Field label={t('hl.title')} help={t('hl.help')}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        {rows.map((r) => {
          const bad = r.pattern !== '' && !isValidPattern(r.pattern, r.regex);
          return (
            <div key={r.id} className="hl-row" style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
              <Switch checked={r.enabled} onChange={(v) => patch(r.id, { enabled: v })} label={t('hl.enabled')} />
              <input
                className="input mono"
                aria-label={t('hl.pattern')}
                aria-invalid={bad}
                placeholder={t('hl.pattern')}
                style={{ flex: '1 1 160px', minWidth: 120, borderColor: bad ? 'var(--danger)' : undefined }}
                value={r.pattern}
                onChange={(e) => patch(r.id, { pattern: e.target.value }, false)}
                onBlur={() => void saveSettings({ highlightRules: rows })}
              />
              <span role="radiogroup" aria-label={t('hl.color')} style={{ display: 'inline-flex', gap: 4 }}>
                {HIGHLIGHT_COLORS.map((c) => (
                  <button
                    key={c.id}
                    type="button"
                    role="radio"
                    aria-checked={r.color === c.hex}
                    title={t(`hl.color.${c.id}` as MsgId)}
                    aria-label={t(`hl.color.${c.id}` as MsgId)}
                    onClick={() => patch(r.id, { color: c.hex })}
                    style={{
                      width: 18,
                      height: 18,
                      borderRadius: 99,
                      background: c.hex,
                      border: r.color === c.hex ? '2px solid var(--fg-1)' : '2px solid transparent',
                      padding: 0,
                      cursor: 'pointer',
                    }}
                  />
                ))}
              </span>
              <Check label={t('hl.background')} checked={r.background} onChange={(v) => patch(r.id, { background: v })} />
              <Check label={t('hl.regex')} checked={r.regex} onChange={(v) => patch(r.id, { regex: v })} />
              <Check label={t('hl.case')} checked={r.caseSensitive} onChange={(v) => patch(r.id, { caseSensitive: v })} />
              <button type="button" className="icon-btn" aria-label={t('common.delete')} title={t('common.delete')} onClick={() => apply(rows.filter((x) => x.id !== r.id), true)}>
                <Trash2 size={14} />
              </button>
            </div>
          );
        })}
        <div>
          <button type="button" className="btn small-btn" disabled={rows.length >= HIGHLIGHT_MAX_RULES} onClick={() => apply([...rows, newHighlightRule()], true)}>
            {t('hl.add')}
          </button>
        </div>
      </div>
    </Field>
  );
}
