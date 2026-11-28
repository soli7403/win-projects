// 설정 · 출력 규칙: 터미널 출력에 반응하는 규칙을 한곳에 둔다 — 키워드 강조(색)와 출력 트리거(알림·소리·자동 응답).
// 근거: spec/ui/settings.md §2b, spec/behavior/terminal.md §9·§10 (D140).
import { useState } from 'react';
import { Trash2 } from 'lucide-react';
import type { TriggerRule } from '../ipc/types';
import { Field, Switch } from '../ui/Controls';
import { isValidPattern } from '../lib/highlight';
import { newTriggerRule, TRIGGER_MAX_RULES } from '../lib/trigger';
import { useApp } from '../state/app';
import { saveSettings } from '../state/settings';
import { SettingsHighlight } from './SettingsHighlight';
import { t } from '../i18n';

function Check({ label, checked, onChange }: { label: string; checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <label className="small" style={{ display: 'inline-flex', alignItems: 'center', gap: 4, whiteSpace: 'nowrap' }}>
      <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} />
      {label}
    </label>
  );
}

/**
 * 트리거 규칙 목록 편집(설정의 규칙과 서버 편집의 서버별 규칙이 같이 쓴다, D149).
 * onRows(next, commit): commit 이 참이면 바로 저장할 만한 변경(체크·추가·삭제), 거짓이면 글자 입력 중.
 * onCommit 은 글자 칸을 벗어날 때 부른다.
 */
export function TriggerRuleList({ rows, onRows, onCommit }: {
  rows: TriggerRule[];
  onRows: (next: TriggerRule[], commit: boolean) => void;
  onCommit?: () => void;
}) {
  const patch = (id: string, p: Partial<TriggerRule>, commit = true) => onRows(rows.map((r) => (r.id === id ? { ...r, ...p } : r)), commit);
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      {rows.map((r) => {
        const bad = r.pattern !== '' && !isValidPattern(r.pattern, r.regex);
        return (
          <div key={r.id} className="trig-row">
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
              <Switch checked={r.enabled} onChange={(v) => patch(r.id, { enabled: v })} label={t('hl.enabled')} />
              <input
                className="input mono"
                aria-label={t('hl.pattern')}
                aria-invalid={bad}
                placeholder={t('hl.pattern')}
                style={{ flex: '1 1 180px', minWidth: 120, borderColor: bad ? 'var(--danger)' : undefined }}
                value={r.pattern}
                onChange={(e) => patch(r.id, { pattern: e.target.value }, false)}
                onBlur={onCommit}
              />
              <Check label={t('hl.regex')} checked={r.regex} onChange={(v) => patch(r.id, { regex: v })} />
              <Check label={t('hl.case')} checked={r.caseSensitive} onChange={(v) => patch(r.id, { caseSensitive: v })} />
              <button type="button" className="icon-btn" aria-label={t('common.delete')} title={t('common.delete')} onClick={() => onRows(rows.filter((x) => x.id !== r.id), true)}>
                <Trash2 size={14} />
              </button>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap', paddingLeft: 44 }}>
              <Check label={t('trig.notify')} checked={r.notify} onChange={(v) => patch(r.id, { notify: v })} />
              <Check label={t('trig.sound')} checked={r.sound} onChange={(v) => patch(r.id, { sound: v })} />
              <Check label={t('trig.reply')} checked={r.replyOn} onChange={(v) => patch(r.id, { replyOn: v })} />
              {r.replyOn && (
                <>
                  <input
                    className="input mono"
                    aria-label={t('trig.replyText')}
                    placeholder={t('trig.replyText')}
                    style={{ flex: '1 1 140px', minWidth: 100 }}
                    value={r.reply}
                    onChange={(e) => patch(r.id, { reply: e.target.value }, false)}
                    onBlur={onCommit}
                  />
                  <Check label={t('trig.replyEnter')} checked={r.replyEnter} onChange={(v) => patch(r.id, { replyEnter: v })} />
                </>
              )}
            </div>
          </div>
        );
      })}
      <div>
        <button type="button" className="btn small-btn" disabled={rows.length >= TRIGGER_MAX_RULES} onClick={() => onRows([...rows, newTriggerRule()], true)}>
          {t('hl.add')}
        </button>
      </div>
    </div>
  );
}

export function SettingsTriggers() {
  const saved = useApp((s) => s.store?.settings.triggerRules);
  const osNotify = useApp((s) => s.store?.settings.triggerOsNotify !== false);
  const [rows, setRows] = useState<TriggerRule[]>(() => (Array.isArray(saved) ? saved : []));
  const [prev, setPrev] = useState(saved);
  // 밖에서 바뀌면(동기화·백업 가져오기) 화면을 맞춘다.
  if (prev !== saved) {
    setPrev(saved);
    setRows(Array.isArray(saved) ? saved : []);
  }

  /** 화면을 바꾸고, save 면 곧바로 저장한다(글자 칸은 벗어날 때 저장). */
  const apply = (next: TriggerRule[], save: boolean) => {
    setRows(next);
    if (save) void saveSettings({ triggerRules: next });
  };
  const saveNow = () => void saveSettings({ triggerRules: rows });

  return (
    <Field label={t('trig.title')} help={t('trig.help')}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        {/* 앱이 뒤에 있을 때 Windows 알림도 (D145) */}
        <div className="toggle-row">
          <div>
            <div className="t-title">{t('trig.osNotify')}</div>
            <div className="t-desc">{t('trig.osNotifyDesc')}</div>
          </div>
          <Switch checked={osNotify} onChange={(v) => void saveSettings({ triggerOsNotify: v })} label={t('trig.osNotify')} />
        </div>
        <TriggerRuleList rows={rows} onRows={apply} onCommit={saveNow} />
        <div className="t-desc small faint">{t('trig.hostHint')}</div>
      </div>
    </Field>
  );
}

export function SettingsRules() {
  return (
    <>
      <SettingsHighlight />
      <SettingsTriggers />
    </>
  );
}
