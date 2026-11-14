// 설정 2 · 입력·붙여넣기. 근거: spec/ui/settings.md §2, spec/data/store.md §4 (기본 패턴), spec/acceptance G3.
import { useEffect, useState } from 'react';
import { ToggleRow, Field } from '../ui/Controls';
import { DANGEROUS_PATTERNS } from '../lib/defaults';
import { getBackend } from '../ipc/backend';
import { useApp } from '../state/app';
import { saveSettings } from '../state/settings';
import { useSetting, Group } from './Settings';
import { t } from '../i18n';

/** 플랫폼 조회(Ctrl+V 토글은 Windows 에서만 보인다). 알 수 없으면 보인다. */
function usePlatform(): string | undefined {
  const [p, setP] = useState<string | undefined>(undefined);
  useEffect(() => {
    getBackend()
      .invoke<{ platform?: string }>('app.info')
      .then((i) => setP(i?.platform))
      .catch(() => setP(undefined));
  }, []);
  return p;
}

export function SettingsInput() {
  const platform = usePlatform();
  const [copyOnSelect, setCopyOnSelect] = useSetting('copyOnSelect');
  const [rightClickPaste, setRightClickPaste] = useSetting('rightClickPaste');
  const [ctrlVPaste, setCtrlVPaste] = useSetting('ctrlVPaste');
  const [shiftEnterNewline, setShiftEnterNewline] = useSetting('shiftEnterNewline');
  const [osc52Copy, setOsc52Copy] = useSetting('osc52Copy');
  const [confirmMultilinePaste, setConfirmMultilinePaste] = useSetting('confirmMultilinePaste');
  const [guardDangerousOnProd, setGuardDangerousOnProd] = useSetting('guardDangerousOnProd');
  const patterns = useApp((s) => s.store?.settings.dangerousPatterns) ?? [];
  const [text, setText] = useState(patterns.join('\n'));
  const [prevKey, setPrevKey] = useState(patterns.join('\n'));
  // 외부 변경(기본값 복원 등)이 생기면 입력 칸을 맞춘다.
  if (prevKey !== patterns.join('\n')) {
    setPrevKey(patterns.join('\n'));
    setText(patterns.join('\n'));
  }

  const savePatterns = (raw: string) => {
    const list = raw.split(/\r?\n/).map((l) => l.trim()).filter((l) => l !== '');
    void saveSettings({ dangerousPatterns: list });
  };

  return (
    <>
      <Group>
        <ToggleRow
          title={t('input.copyOnSelect')}
          desc={t('input.copyOnSelectDesc')}
          checked={copyOnSelect}
          onChange={setCopyOnSelect}
        />
        <ToggleRow
          title={t('input.rightClickPaste')}
          desc={t('input.rightClickPasteDesc')}
          checked={rightClickPaste}
          onChange={setRightClickPaste}
        />
        {platform !== 'linux' && platform !== 'darwin' && (
          <ToggleRow
            title={t('input.ctrlVPaste')}
            desc={t('input.ctrlVPasteDesc')}
            checked={ctrlVPaste}
            onChange={setCtrlVPaste}
          />
        )}
        <ToggleRow
          title={t('input.shiftEnter')}
          desc={t('input.shiftEnterDesc')}
          checked={shiftEnterNewline}
          onChange={setShiftEnterNewline}
        />
        <ToggleRow
          title={t('input.osc52')}
          desc={t('input.osc52Desc')}
          checked={osc52Copy}
          onChange={setOsc52Copy}
        />
        <ToggleRow
          title={t('input.confirmMultiline')}
          desc={t('input.confirmMultilineDesc')}
          checked={confirmMultilinePaste}
          onChange={setConfirmMultilinePaste}
        />
        <ToggleRow
          title={t('input.guardProd')}
          desc={t('input.guardProdDesc')}
          checked={guardDangerousOnProd}
          onChange={setGuardDangerousOnProd}
        />
      </Group>

      <Field label={t('input.patterns')}>
        <textarea
          className="input mono"
          aria-label={t('input.patternsAria')}
          rows={7}
          style={{ height: 150, padding: '8px 10px', resize: 'vertical', fontSize: 12 }}
          value={text}
          onChange={(e) => setText(e.target.value)}
          onBlur={() => savePatterns(text)}
        />
      </Field>
      <div>
        <button
          className="btn"
          type="button"
          onClick={() => {
            setText(DANGEROUS_PATTERNS.join('\n'));
            void saveSettings({ dangerousPatterns: [...DANGEROUS_PATTERNS] });
          }}
        >
          {t('common.resetDefault')}
        </button>
      </div>
    </>
  );
}
