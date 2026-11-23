// 설정 1 · 모양. 근거: spec/ui/settings.md §1, spec/ui/tokens.md §6, spec/acceptance/04 G1·G2·G8.
import { useState } from 'react';
import { ToggleRow, Segmented, Field } from '../ui/Controls';
import { TERM_THEMES, resolveTermTheme } from '../lib/termThemes';
import { getBackend, errorMessage } from '../ipc/backend';
import { useApp } from '../state/app';
import { saveSettings } from '../state/settings';
import { NumberField, useSetting, Group } from './Settings';
import { SettingsHighlight } from './SettingsHighlight';
import { LANGS, isLang, t, useLang } from '../i18n';

export function SettingsAppearance() {
  const [theme, setTheme] = useSetting('theme');
  const [termTheme, setTermTheme] = useSetting('terminalTheme');
  const [fontFamily, setFontFamily] = useSetting('fontFamily');
  const [fontSize] = useSetting('fontSize');
  const [lineHeight] = useSetting('lineHeight');
  const [remember, setRemember] = useSetting('rememberWindow');
  const [cursorStyle] = useSetting('cursorStyle');
  const [scrollback] = useSetting('scrollback');
  const [cursorBlink, setCursorBlink] = useSetting('cursorBlink');
  const [bell, setBell] = useSetting('bell');
  const [maxSplits] = useSetting('maxVerticalSplits');
  const [, setLanguage] = useSetting('language');
  const lang = useLang((s) => s.lang);
  const pushToast = useApp((s) => s.pushToast);
  const current = resolveTermTheme(termTheme);

  // 앱 테마와 터미널 테마 짝 연동(G1): 짝 테마를 쓰고 있을 때만 바꾼다.
  const changeTheme = (next: 'dark' | 'light') => {
    setTheme(next);
    if (next === 'light' && termTheme === 'DariTerm Dark') setTermTheme('DariTerm Light');
    if (next === 'dark' && termTheme === 'DariTerm Light') setTermTheme('DariTerm Dark');
  };

  const resetWindow = async () => {
    try {
      await getBackend().invoke('app.resetWindow');
    } catch (e) {
      pushToast({ kind: 'error', text: errorMessage(e) });
    }
  };

  return (
    <>
      {/* D75: 화면 언어. 바꾸면 바로 적용(저장 → 언어 연결이 다시 그림) */}
      <Field label={t('lang.label')} help={t('lang.help')}>
        <select className="input" aria-label={t('lang.label')} value={lang} onChange={(e) => isLang(e.target.value) && setLanguage(e.target.value)}>
          {LANGS.map((l) => (
            <option key={l.id} value={l.id}>{l.name}</option>
          ))}
        </select>
      </Field>

      <Field label={t('appearance.theme')}>
        <Segmented
          label={t('appearance.theme')}
          value={theme}
          onChange={changeTheme}
          options={[
            { value: 'dark', label: t('appearance.dark') },
            { value: 'light', label: t('appearance.light') },
          ]}
        />
      </Field>

      <Field label={t('appearance.termColors')}>
        <div className="themes" role="radiogroup" aria-label={t('appearance.termColors')}>
          {TERM_THEMES.map((th) => {
            const on = current.name === th.name;
            return (
              <button
                key={th.name}
                type="button"
                role="radio"
                aria-checked={on}
                className={`theme-card${on ? ' on' : ''}`}
                style={{ background: th.background, color: th.foreground }}
                onClick={() => void saveSettings({ terminalTheme: th.name })}
              >
                <span className="tn">{th.name}</span>
                <span className="ts" style={{ fontSize: 12 }}>
                  <span style={{ color: th.green }}>user@host</span>:<span style={{ color: th.blue }}>~</span>$ <span style={{ color: th.yellow }}>ls</span> <span style={{ color: th.red }}>err</span>
                </span>
              </button>
            );
          })}
        </div>
      </Field>

      <div className="set-grid c3">
        <Field label={t('appearance.font')} help={t('appearance.fontHelp')}>
          <FontInput value={fontFamily} onCommit={setFontFamily} />
        </Field>
        <Field label={t('appearance.fontSize')}>
          <NumberField label={t('appearance.fontSize')} value={fontSize} min={8} max={32} fallback={14} onCommit={(v) => void saveSettings({ fontSize: v })} />
        </Field>
        <Field label={t('appearance.lineHeight')}>
          <NumberField label={t('appearance.lineHeight')} value={lineHeight} min={1} max={2} step={0.05} fallback={1.15} onCommit={(v) => void saveSettings({ lineHeight: v })} />
        </Field>
      </div>

      <div className="set-grid c2">
        <ToggleRow
          title={t('appearance.rememberWindow')}
          desc={t('appearance.rememberWindowDesc')}
          checked={remember}
          onChange={setRemember}
        />
        <Field label={t('appearance.windowSize')}>
          <div>
            <button className="btn" type="button" onClick={() => void resetWindow()}>{t('appearance.resetWindow')}</button>
          </div>
        </Field>
      </div>

      <div className="set-grid c2">
        <Field label={t('appearance.cursorStyle')}>
          <Segmented
            label={t('appearance.cursorStyle')}
            value={cursorStyle}
            onChange={(v) => void saveSettings({ cursorStyle: v })}
            options={[
              { value: 'block', label: t('appearance.cursorBlock') },
              { value: 'bar', label: t('appearance.cursorBar') },
              { value: 'underline', label: t('appearance.cursorUnderline') },
            ]}
          />
        </Field>
        <Field label={t('appearance.scrollback')}>
          <NumberField label={t('appearance.scrollback')} value={scrollback} min={500} max={200000} fallback={10000} onCommit={(v) => void saveSettings({ scrollback: v })} />
        </Field>
      </div>

      <Field label={t('appearance.maxSplits')} help={t('appearance.maxSplitsHelp')}>
        <Segmented
          label={t('appearance.maxSplits')}
          value={String(maxSplits ?? 3) as '1' | '2' | '3' | '4' | '5'}
          onChange={(v) => void saveSettings({ maxVerticalSplits: Number(v) })}
          options={[
            { value: '1', label: '1' },
            { value: '2', label: '2' },
            { value: '3', label: '3' },
            { value: '4', label: '4' },
            { value: '5', label: '5' },
          ]}
        />
      </Field>

      <Group>
        <ToggleRow title={t('appearance.cursorBlink')} checked={cursorBlink} onChange={setCursorBlink} />
        <ToggleRow title={t('appearance.bell')} checked={bell} onChange={setBell} />
      </Group>
      <SettingsHighlight />
    </>
  );
}

/** 글꼴 입력: 떠날 때 저장. */
function FontInput({ value, onCommit }: { value: string; onCommit: (v: string) => void }) {
  const [draft, setDraft] = useState(value);
  const [prev, setPrev] = useState(value);
  if (prev !== value) {
    setPrev(value);
    setDraft(value);
  }
  return (
    <input
      className="input"
      aria-label={t('appearance.font')}
      value={draft}
      onChange={(e) => setDraft(e.target.value)}
      onBlur={() => {
        const v = draft.trim();
        if (v !== value) onCommit(v || value);
      }}
    />
  );
}
