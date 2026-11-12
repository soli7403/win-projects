// 설정 1 · 모양. 근거: spec/ui/settings.md §1, spec/ui/tokens.md §6, spec/acceptance/04 G1·G2·G8.
import { useState } from 'react';
import { ToggleRow, Segmented, Field } from '../ui/Controls';
import { TERM_THEMES, resolveTermTheme } from '../lib/termThemes';
import { getBackend, errorMessage } from '../ipc/backend';
import { useApp } from '../state/app';
import { saveSettings } from '../state/settings';
import { NumberField, useSetting, Group } from './Settings';

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
  const pushToast = useApp((s) => s.pushToast);
  const current = resolveTermTheme(termTheme);

  // 앱 테마와 터미널 테마 짝 연동(G1): 짝 테마를 쓰고 있을 때만 바꾼다.
  const changeTheme = (t: 'dark' | 'light') => {
    setTheme(t);
    if (t === 'light' && termTheme === 'TermGo Dark') setTermTheme('TermGo Light');
    if (t === 'dark' && termTheme === 'TermGo Light') setTermTheme('TermGo Dark');
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
      <Field label="앱 테마">
        <Segmented
          label="앱 테마"
          value={theme}
          onChange={changeTheme}
          options={[
            { value: 'dark', label: '다크' },
            { value: 'light', label: '라이트' },
          ]}
        />
      </Field>

      <Field label="터미널 색 테마">
        <div className="themes" role="radiogroup" aria-label="터미널 색 테마">
          {TERM_THEMES.map((t) => {
            const on = current.name === t.name;
            return (
              <button
                key={t.name}
                type="button"
                role="radio"
                aria-checked={on}
                className={`theme-card${on ? ' on' : ''}`}
                style={{ background: t.background, color: t.foreground }}
                onClick={() => void saveSettings({ terminalTheme: t.name })}
              >
                <span className="tn">{t.name}</span>
                <span className="ts" style={{ fontSize: 12 }}>
                  <span style={{ color: t.green }}>user@host</span>:<span style={{ color: t.blue }}>~</span>$ <span style={{ color: t.yellow }}>ls</span> <span style={{ color: t.red }}>err</span>
                </span>
              </button>
            );
          })}
        </div>
      </Field>

      <div className="set-grid c3">
        <Field label="글꼴" help="D2Coding, Cascadia Mono 등 설치된 고정폭 글꼴">
          <FontInput value={fontFamily} onCommit={setFontFamily} />
        </Field>
        <Field label="글자 크기">
          <NumberField label="글자 크기" value={fontSize} min={8} max={32} fallback={14} onCommit={(v) => void saveSettings({ fontSize: v })} />
        </Field>
        <Field label="줄 간격">
          <NumberField label="줄 간격" value={lineHeight} min={1} max={2} step={0.05} fallback={1.15} onCommit={(v) => void saveSettings({ lineHeight: v })} />
        </Field>
      </div>

      <div className="set-grid c2">
        <ToggleRow
          title="창 크기·위치 기억"
          desc="다음 실행 때 마지막 크기, 위치, 최대화 상태로 엽니다 (모니터가 바뀌어 화면 밖이면 가운데로 엽니다)"
          checked={remember}
          onChange={setRemember}
        />
        <Field label="창 크기">
          <div>
            <button className="btn" type="button" onClick={() => void resetWindow()}>원래 크기로</button>
          </div>
        </Field>
      </div>

      <div className="set-grid c2">
        <Field label="커서 모양">
          <Segmented
            label="커서 모양"
            value={cursorStyle}
            onChange={(v) => void saveSettings({ cursorStyle: v })}
            options={[
              { value: 'block', label: '블록' },
              { value: 'bar', label: '막대' },
              { value: 'underline', label: '밑줄' },
            ]}
          />
        </Field>
        <Field label="스크롤백 (줄)">
          <NumberField label="스크롤백 (줄)" value={scrollback} min={500} max={200000} fallback={10000} onCommit={(v) => void saveSettings({ scrollback: v })} />
        </Field>
      </div>

      <Field label="최대 화면 분할 (상하)" help="탭 하나에 위아래로 쌓을 수 있는 칸의 최대 개수입니다. 이 개수에 이르면 아래로 분할 버튼이 꺼집니다.">
        <Segmented
          label="최대 화면 분할 (상하)"
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
        <ToggleRow title="커서 깜빡임" checked={cursorBlink} onChange={setCursorBlink} />
        <ToggleRow title="벨 소리" checked={bell} onChange={setBell} />
      </Group>
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
      aria-label="글꼴"
      value={draft}
      onChange={(e) => setDraft(e.target.value)}
      onBlur={() => {
        const v = draft.trim();
        if (v !== value) onCommit(v || value);
      }}
    />
  );
}
